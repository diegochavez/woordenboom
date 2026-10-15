package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestDailyPracticeIncludesMapAndSession(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("requires release-9500 database")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var userID string
	err = db.QueryRow(ctx, `INSERT INTO users(email) VALUES($1) RETURNING id::text`, fmt.Sprintf("daily-progress-qa-%d@example.test", time.Now().UnixNano())).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := db.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID); err != nil {
			t.Error(err)
		}
	}()
	a := &app{config: config{vocabularyBatch: "release-9500", authSecret: "test"}, db: db, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	request := func(method, body string) *http.Request {
		r := httptest.NewRequest(method, "/", strings.NewReader(body))
		return r.WithContext(context.WithValue(ctx, userContextKey, authUser{ID: userID}))
	}
	daily := func(want int) {
		t.Helper()
		w := httptest.NewRecorder()
		a.progressSummaryHandler(w, request("GET", ""))
		var result struct{ DailyDone int }
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &result) != nil || result.DailyDone != want {
			t.Fatalf("want dailyDone=%d, got %s", want, w.Body.String())
		}
	}
	var wordID string
	if err := db.QueryRow(ctx, `SELECT bw.word_id::text FROM vocabulary_batch_words bw JOIN vocabulary_batches b ON b.id=bw.batch_id WHERE b.slug='release-9500' ORDER BY bw.batch_rank LIMIT 1`).Scan(&wordID); err != nil {
		t.Fatal(err)
	}
	mark := func(action string) {
		t.Helper()
		r := request("POST", fmt.Sprintf(`{"action":%q}`, action))
		r.SetPathValue("id", wordID)
		w := httptest.NewRecorder()
		a.knowledgeMapMarkWordHandler(w, r)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	daily(0)
	mark("already_knew")
	daily(1)
	mark("should_learn")
	daily(1) // Distinct words, not button taps.
	// A map word last practiced yesterday must not count today.
	if _, err := db.Exec(ctx, `UPDATE user_word_progress SET last_seen_at=date_trunc('day',now())-interval '1 second' WHERE user_id=$1`, userID); err != nil {
		t.Fatal(err)
	}
	daily(0)
	w := httptest.NewRecorder()
	a.createLearningSession(w, request("POST", ""), []string{"general-vocabulary"}, []string{}, "balanced")
	var session learningSession
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &session) != nil {
		t.Fatal(w.Body.String())
	}
	q := session.Questions[0]
	body, _ := json.Marshal(map[string]string{"questionId": q.ID, "wordId": q.WordID, "answer": "Already Knew"})
	answer := func() {
		r := request("POST", string(body))
		r.SetPathValue("id", session.ID)
		w := httptest.NewRecorder()
		a.sessionAnswerHandler(w, r)
		if w.Code != 200 {
			t.Fatal(w.Body.String())
		}
	}
	answer()
	daily(1)
	answer()
	daily(1)
	wordID = q.WordID
	mark("already_knew")
	daily(1) // Same word through both paths counts once.
	// Progress responses must not reuse a cached total.
	token, err := a.createAccessToken(userID, "test@example.test")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("GET", "/progress/summary", nil)
	r.Header.Set("Authorization", "Bearer "+token)
	w = httptest.NewRecorder()
	a.requireAuth(http.HandlerFunc(a.progressSummaryHandler)).ServeHTTP(w, r)
	if !strings.Contains(w.Header().Get("Cache-Control"), "no-store") {
		t.Fatal("authenticated progress must not be cached")
	}
}
