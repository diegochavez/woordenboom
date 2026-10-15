package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReviewIntervals(t *testing.T) {
	for streak, want := range []string{"1d", "3d", "7d", "14d", "30d", "60d", "90d", "180d", "365d", "365d"} {
		if got := recommendedInterval(streak); got != want {
			t.Fatalf("streak %d: %s", streak, got)
		}
	}
	if recommendedInterval(-1) != "1d" || recommendedInterval(10000) != "365d" {
		t.Fatal("recommendation outside supported bounds")
	}
	for choice, minutes := range map[string]int{"10m": 10, "1d": 1440, "2d": 2880, "3d": 4320, "7d": 10080, "14d": 20160, "30d": 43200, "60d": 86400, "90d": 129600, "180d": 259200, "365d": 525600} {
		if !validReviewInterval(choice) || intervalMinutes(choice) != minutes {
			t.Fatalf("%s: expected %d minutes, got %d", choice, minutes, intervalMinutes(choice))
		}
	}
	for _, choice := range []string{"-1d", "366d", "999d", "forever", "1.5d"} {
		if validReviewInterval(choice) {
			t.Fatal(choice)
		}
	}
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	due := now.Add(-time.Hour)
	words := []reviewWord{{ID: "paused"}, {ID: "due"}}
	progress := map[string]wordReviewState{"paused": {Paused: true, LastSeen: &old, NextReview: &due}, "due": {LastSeen: &old, NextReview: &due}}
	selected := selectPracticeWords(words, progress, now, 10)
	if len(selected) != 1 || selected[0].ID != "due" {
		t.Fatal(selected)
	}
}

func TestReviewTimingIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var userID string
	if err := db.QueryRow(ctx, `INSERT INTO users(email) VALUES($1) RETURNING id::text`, fmt.Sprintf("timing-qa-%d@example.test", time.Now().UnixNano())).Scan(&userID); err != nil {
		t.Fatal(err)
	}
	defer db.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID)
	a := &app{config: config{vocabularyBatch: "release-9500"}, db: db, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	words, err := a.wordsForTopic(ctx, "general-vocabulary")
	if err != nil {
		t.Fatal(err)
	}
	word := words[0]
	mark := func(correct bool, choice string) {
		t.Helper()
		tx, err := db.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer tx.Rollback(ctx)
		if err := a.updateWordProgressWithTiming(ctx, tx, userID, word.TopicSlug, word.ID, correct, choice); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(ctx); err != nil {
			t.Fatal(err)
		}
	}
	check := func(minutes, streak, remembered int, paused bool) {
		t.Helper()
		var actualMinutes *float64
		var actualStreak, count int
		var actualPaused bool
		err := db.QueryRow(ctx, `SELECT EXTRACT(epoch FROM(next_review_at-last_seen_at))/60,review_streak,correct_count,review_paused FROM user_word_progress WHERE user_id=$1 AND word_key=$2`, userID, word.ID).Scan(&actualMinutes, &actualStreak, &count, &actualPaused)
		if err != nil {
			t.Fatal(err)
		}
		if actualStreak != streak || count != remembered || actualPaused != paused {
			t.Fatalf("streak/count/paused: %d/%d/%v", actualStreak, count, actualPaused)
		}
		if paused {
			if actualMinutes != nil {
				t.Fatal("paused word has a due date")
			}
		} else if actualMinutes == nil || int(*actualMinutes) != minutes {
			t.Fatalf("minutes: %v, expected %d", actualMinutes, minutes)
		}
	}
	for i, choice := range []string{"1d", "3d", "7d", "14d", "30d", "60d", "90d", "180d", "365d", "365d"} {
		mark(true, "auto")
		check(intervalMinutes(choice), i+1, i+1, false)
	}
	mark(false, "365d")
	check(10, 0, 10, false)
	mark(true, "7d")
	check(10080, 1, 11, false)
	mark(true, "paused")
	check(0, 2, 12, true)
	mark(true, "auto")
	check(0, 3, 13, true)
	mark(true, "2d")
	check(2880, 4, 14, false)
	mark(true, "paused")
	mark(false, "auto")
	check(10, 0, 15, false)
	r := httptest.NewRequest("GET", "/", nil).WithContext(context.WithValue(ctx, userContextKey, authUser{ID: userID}))
	r.SetPathValue("id", word.ID)
	w := httptest.NewRecorder()
	a.reviewTimingHandler(w, r)
	var timing reviewTiming
	if err := json.Unmarshal(w.Body.Bytes(), &timing); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || timing.RememberedCount != 15 || timing.RecommendedInterval != "1d" || timing.Paused {
		t.Fatalf("timing: %s", w.Body.String())
	}
	// Invalid API input does not change memory counts or schedules.
	bad := httptest.NewRequest("POST", "/", strings.NewReader(`{"action":"already_knew","reviewInterval":"-1d"}`)).WithContext(r.Context())
	bad.SetPathValue("id", word.ID)
	w = httptest.NewRecorder()
	a.knowledgeMapMarkWordHandler(w, bad)
	if w.Code != 400 {
		t.Fatal(w.Code)
	}
	check(10, 0, 15, false)
	// A category fully scheduled in the future should return an empty, successful session.
	_, err = db.Exec(ctx, `INSERT INTO user_word_progress(user_id,word_key,topic_slug,correct_count,next_review_at,last_seen_at)
        SELECT $1,bw.word_id::text,bw.topic_slug,1,now()+interval '1 day',now() FROM vocabulary_topic_memberships bw JOIN vocabulary_batches b ON b.id=bw.batch_id
        WHERE b.slug='release-9500' AND bw.topic_slug='immigration-documents' ON CONFLICT DO NOTHING`, userID)
	if err != nil {
		t.Fatal(err)
	}
	w = httptest.NewRecorder()
	a.createLearningSession(w, r, []string{"immigration-documents"}, nil, "balanced")
	var result struct {
		Questions []sessionQuestion
		Status    string
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || result.Status != "up_to_date" || len(result.Questions) != 0 {
		t.Fatal(w.Body.String())
	}
}
