package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Run after importing release-9500. The test user and all its progress are removed.
func TestVocabularyReleaseIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("set TEST_DATABASE_URL to the migrated test database containing release-9500")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var userID string
	err = db.QueryRow(ctx, `INSERT INTO users(email) VALUES($1) RETURNING id::text`,
		fmt.Sprintf("vocabulary-qa-%d@example.test", time.Now().UnixNano())).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := db.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID); err != nil {
			t.Error(err)
		}
	}()
	a := &app{config: config{vocabularyBatch: "release-9500"}, db: db, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	request := func(method, body string) *http.Request {
		r := httptest.NewRequest(method, "/", strings.NewReader(body))
		return r.WithContext(context.WithValue(r.Context(), userContextKey, authUser{ID: userID}))
	}
	decode := func(w *httptest.ResponseRecorder, result any) {
		t.Helper()
		if w.Code != http.StatusOK {
			t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
		}
		if err := json.Unmarshal(w.Body.Bytes(), result); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("map covers all 9500 words", func(t *testing.T) {
		w := httptest.NewRecorder()
		a.knowledgeMapHandler(w, request("GET", ""))
		var result struct {
			TotalWords, AvailableWords int
			BatchSlug                  string
			Ranges                     []json.RawMessage
		}
		decode(w, &result)
		if result.TotalWords != 9500 || result.AvailableWords != 9500 || len(result.Ranges) != 95 || result.BatchSlug != "release-9500" {
			t.Fatalf("unexpected map: %+v", result)
		}
		// Check every range for duplicates, translations and missing words, including optional examples.
		seen := map[string]bool{}
		withoutExamples := 0
		for start := 1; start <= 9500; start += 100 {
			r := request("GET", "")
			r.SetPathValue("start", fmt.Sprint(start))
			w = httptest.NewRecorder()
			a.knowledgeMapRangeHandler(w, r)
			var rangeResult struct {
				Words []struct{ ID, Word, Translation, Example string }
			}
			decode(w, &rangeResult)
			if len(rangeResult.Words) != 100 {
				t.Fatalf("range %d has %d words", start, len(rangeResult.Words))
			}
			for _, word := range rangeResult.Words {
				if seen[word.ID] || strings.TrimSpace(word.Translation) == "" {
					t.Fatalf("duplicate/missing translation: %+v", word)
				}
				seen[word.ID] = true
				if word.Example == "" {
					withoutExamples++
				}
			}
		}
		if len(seen) != 9500 || withoutExamples != 9310 {
			t.Fatalf("words %d, without examples %d", len(seen), withoutExamples)
		}
	})
	t.Run("Dutch definitions match the displayed content and survive snapshots", func(t *testing.T) {
		words, err := a.wordsForTopic(ctx, "general-vocabulary")
		if err != nil {
			t.Fatal(err)
		}
		var chosen reviewWord
		for _, word := range words {
			if word.Word == "woord" {
				chosen = word
				break
			}
		}
		if chosen.Translation != "word" || len(chosen.Meanings) != 1 || chosen.Meanings[0].English != "word" || chosen.Meanings[0].Dutch == "" {
			t.Fatalf("incorrect matched content: %+v", chosen)
		}
		question := buildQuestion(0, chosen, words)
		encoded, err := encodeSessionQuestions([]sessionQuestion{question})
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := decodeSessionQuestions(encoded, 1)
		if err != nil || len(decoded[0].Meanings) != 1 || decoded[0].Meanings[0] != chosen.Meanings[0] {
			t.Fatal("snapshot lost definitions", err)
		}
		legacy := *a
		legacy.originalVocabularyContent = true
		oldWords, err := legacy.wordsForTopic(ctx, "general-vocabulary")
		if err != nil {
			t.Fatal(err)
		}
		for _, word := range oldWords {
			if word.ID == chosen.ID && (word.Translation == "word" || len(word.Meanings) != 0) {
				t.Fatal("legacy content was rewritten")
			}
		}
		var rank int
		if err := db.QueryRow(ctx, `SELECT bw.batch_rank FROM vocabulary_batch_words bw JOIN vocabulary_batches b ON b.id=bw.batch_id WHERE b.slug='release-9500' AND bw.word_id=$1`, chosen.ID).Scan(&rank); err != nil {
			t.Fatal(err)
		}
		r := request("GET", "")
		r.SetPathValue("start", fmt.Sprint(rank))
		w := httptest.NewRecorder()
		a.knowledgeMapRangeHandler(w, r)
		var result struct{ Words []reviewWord }
		decode(w, &result)
		if len(result.Words) == 0 || result.Words[0].Translation != chosen.Translation || len(result.Words[0].Meanings) != 1 || result.Words[0].Meanings[0] != chosen.Meanings[0] {
			t.Fatal("map and practice content differ")
		}
	})

	t.Run("topic availability agrees with map", func(t *testing.T) {
		counts, err := a.availableWordCountsByTopic(ctx)
		if err != nil {
			t.Fatal(err)
		}
		seen := map[string]bool{}
		for slug, count := range counts {
			words, err := a.wordsForTopic(ctx, slug)
			if err != nil {
				t.Fatal(err)
			}
			if len(words) != count {
				t.Fatalf("%s: count %d vs %d", slug, count, len(words))
			}
			for _, word := range words {
				seen[word.ID] = true
			}
		}
		if len(seen) != 9500 || counts["general-vocabulary"] != 9500 {
			t.Fatalf("topic union total %d", len(seen))
		}
	})
	t.Run("session stays on original release and completes once", func(t *testing.T) {
		old := *a
		old.config.vocabularyBatch = "release-2000"
		w := httptest.NewRecorder()
		old.createLearningSession(w, request("POST", ""), []string{"daily-basics"}, []string{}, "balanced")
		var session learningSession
		decode(w, &session)
		if len(session.Questions) != 10 {
			t.Fatal("expected ten questions")
		}
		// Simulate an in-flight session created before the snapshot migration.
		if _, err := db.Exec(ctx, `UPDATE learning_sessions SET question_snapshot=NULL WHERE id=$1`, session.ID); err != nil {
			t.Fatal(err)
		}
		for _, q := range session.Questions {
			body, _ := json.Marshal(map[string]string{"questionId": q.ID, "wordId": q.WordID, "answer": "Already Knew"})
			r := request("POST", string(body))
			r.SetPathValue("id", session.ID)
			w = httptest.NewRecorder()
			a.sessionAnswerHandler(w, r)
			var result map[string]any
			decode(w, &result)
		}
		q := session.Questions[0]
		body, _ := json.Marshal(map[string]string{"questionId": q.ID, "wordId": q.WordID, "answer": "Already Knew"})
		r := request("POST", string(body))
		r.SetPathValue("id", session.ID)
		w = httptest.NewRecorder()
		a.sessionAnswerHandler(w, r)
		var result struct{ AnsweredCount int }
		decode(w, &result)
		if result.AnsweredCount != 10 {
			t.Fatalf("repeat changed answer count: %d", result.AnsweredCount)
		}
		var completed bool
		var batch string
		if err := db.QueryRow(ctx, `SELECT completed_at IS NOT NULL,vocabulary_batch FROM learning_sessions WHERE id=$1`, session.ID).Scan(&completed, &batch); err != nil {
			t.Fatal(err)
		}
		if !completed || batch != "release-2000" {
			t.Fatalf("completion %v, batch %s", completed, batch)
		}
	})
	t.Run("new sessions use expanded release", func(t *testing.T) {
		w := httptest.NewRecorder()
		a.createLearningSession(w, request("POST", ""), []string{"general-vocabulary"}, []string{}, "balanced")
		var session learningSession
		decode(w, &session)
		if len(session.Questions) != 10 {
			t.Fatal("expected ten dictionary words")
		}
		var batch string
		if err := db.QueryRow(ctx, `SELECT vocabulary_batch FROM learning_sessions WHERE id=$1`, session.ID).Scan(&batch); err != nil {
			t.Fatal(err)
		}
		if batch != "release-9500" {
			t.Fatal(batch)
		}
	})
	t.Run("successive general-vocabulary stacks advance for this learner", func(t *testing.T) {
		newSession := func(get bool) learningSession {
			w := httptest.NewRecorder()
			if get {
				r := request("GET", "")
				r.URL.RawQuery = "topic=general-vocabulary"
				a.nextSessionHandler(w, r)
			} else {
				a.createSessionHandler(w, request("POST", `{"topicSlugs":["general-vocabulary"],"mode":"balanced"}`))
			}
			var session learningSession
			decode(w, &session)
			if len(session.Questions) != 10 {
				t.Fatalf("expected ten words: %d", len(session.Questions))
			}
			if w.Header().Get("Cache-Control") != "no-store" {
				t.Fatal("session creation must not be cached")
			}
			return session
		}
		seen := map[string]bool{}
		for stack := 0; stack < 3; stack++ {
			session := newSession(stack%2 == 0)
			for _, q := range session.Questions {
				if seen[q.WordID] {
					t.Fatalf("word repeated in immediate stack %d: %s", stack, q.Word)
				}
				seen[q.WordID] = true
				answer := "Already Knew"
				if stack == 1 {
					answer = "Should Learn"
				}
				body, _ := json.Marshal(map[string]string{"questionId": q.ID, "wordId": q.WordID, "answer": answer})
				r := request("POST", string(body))
				r.SetPathValue("id", session.ID)
				w := httptest.NewRecorder()
				a.sessionAnswerHandler(w, r)
				var result struct{ CorrectAnswer string }
				decode(w, &result)
				if result.CorrectAnswer != q.Translation {
					t.Fatalf("graded a different word: %q vs %q", result.CorrectAnswer, q.Translation)
				}
			}
			var recorded int
			if err := db.QueryRow(ctx, `SELECT count(*) FROM learning_events WHERE session_id=$1`, session.ID).Scan(&recorded); err != nil {
				t.Fatal(err)
			}
			if recorded != 10 {
				t.Fatalf("events: %d", recorded)
			}
		}
		if len(seen) != 30 {
			t.Fatalf("expected 30 distinct words, got %d", len(seen))
		}
		// Make only this subtest's words due. General now also includes prior category practice.
		seenIDs := make([]string, 0, len(seen))
		for id := range seen {
			seenIDs = append(seenIDs, id)
		}
		if _, err := db.Exec(ctx, `UPDATE user_word_progress SET last_seen_at=now()-interval '2 days', next_review_at=now()-interval '1 day' WHERE user_id=$1 AND word_key=ANY($2)`, userID, seenIDs); err != nil {
			t.Fatal(err)
		}
		session := newSession(false)
		reviews := 0
		for _, q := range session.Questions {
			if seen[q.WordID] {
				reviews++
			}
		}
		if reviews != 5 {
			t.Fatalf("expected 5 due and 5 unseen, got %d due", reviews)
		}
		// The app configuration and other answers cannot change a stored session.
		other := *a
		other.config.vocabularyBatch = "release-2000"
		q := session.Questions[9]
		body, _ := json.Marshal(map[string]string{"questionId": q.ID, "wordId": q.WordID, "answer": "Already Knew"})
		r := request("POST", string(body))
		r.SetPathValue("id", session.ID)
		w := httptest.NewRecorder()
		other.sessionAnswerHandler(w, r)
		var result struct{ CorrectAnswer string }
		decode(w, &result)
		if result.CorrectAnswer != q.Translation {
			t.Fatal("snapshot not used")
		}
		bad := request("POST", fmt.Sprintf(`{"questionId":%q,"wordId":"another-word","answer":"Already Knew"}`, q.ID))
		bad.SetPathValue("id", session.ID)
		w = httptest.NewRecorder()
		a.sessionAnswerHandler(w, bad)
		if w.Code != http.StatusConflict {
			t.Fatalf("wrong-word answer accepted: %d", w.Code)
		}
	})

}
