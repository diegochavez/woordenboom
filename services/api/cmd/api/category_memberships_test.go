package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestCategoryMembershipIntegration(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("requires imported category release")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	a := &app{config: config{vocabularyBatch: "release-9500"}, db: db, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	all, err := a.wordsForTopic(ctx, "general-vocabulary")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 9500 || all[0].Word != "de" {
		t.Fatalf("general must contain frequency ordered complete corpus: %d, %s", len(all), all[0].Word)
	}
	legacy := *a
	legacy.originalVocabularyContent = true
	old, err := legacy.wordsForTopic(ctx, "general-vocabulary")
	if err != nil || len(old) != 9027 {
		t.Fatalf("legacy pool changed: %d %v", len(old), err)
	}
	healthcare, err := a.wordsForTopic(ctx, "healthcare")
	if err != nil {
		t.Fatal(err)
	}
	school, err := a.wordsForTopic(ctx, "school-childcare")
	if err != nil {
		t.Fatal(err)
	}
	var pupil reviewWord
	for _, word := range healthcare {
		if word.Word == "pupil" {
			pupil = word
		}
	}
	if pupil.ID == "" || len(pupil.Meanings) != 1 || pupil.Translation != "pupil" {
		t.Fatal("missing anatomical sense", pupil)
	}
	for _, word := range school {
		if word.ID == pupil.ID {
			t.Fatal("anatomical pupil leaked into school")
		}
	}
	food, err := a.wordsForTopic(ctx, "supermarket-food")
	if err != nil {
		t.Fatal(err)
	}
	union := map[string]bool{}
	for _, word := range append(healthcare, food...) {
		union[word.ID] = true
	}
	if len(union) >= len(healthcare)+len(food) {
		t.Fatal("fixture needs category overlap")
	}
	mixed, err := a.wordsForTopics(ctx, []string{"healthcare", "supermarket-food"})
	if err != nil || len(mixed) != len(union) {
		t.Fatalf("mixed count: %d vs %d %v", len(mixed), len(union), err)
	}
	slugs, err := a.validTopicSlugs(ctx, []string{"healthcare", "general-vocabulary"})
	if err != nil || len(slugs) != 1 || slugs[0] != "general-vocabulary" {
		t.Fatal("general collection not normalized", slugs, err)
	}
	r := httptest.NewRequest("GET", "/topics/preview?topics=healthcare,supermarket-food", nil)
	w := httptest.NewRecorder()
	a.topicPreviewHandler(w, r)
	var preview struct {
		WordCount int
		Words     []reviewWord
	}
	if json.Unmarshal(w.Body.Bytes(), &preview) != nil || w.Code != 200 || preview.WordCount != len(union) || len(preview.Words) != 6 {
		t.Fatal("bad bounded unique preview", w.Body.String())
	}
	var userID string
	err = db.QueryRow(ctx, `INSERT INTO users(email) VALUES($1) RETURNING id::text`, fmt.Sprintf("categories-qa-%d@example.test", time.Now().UnixNano())).Scan(&userID)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := db.Exec(ctx, `DELETE FROM users WHERE id=$1`, userID); err != nil {
			t.Error(err)
		}
	}()
	_, err = db.Exec(ctx, `INSERT INTO user_word_progress(user_id,word_key,topic_slug,correct_count,next_review_at,last_seen_at) VALUES($1,$2,'general-vocabulary',1,now()+interval '1 year',now())`, userID, pupil.ID)
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", path, nil)
		r = r.WithContext(context.WithValue(ctx, userContextKey, authUser{ID: userID}))
		w := httptest.NewRecorder()
		if path == "/topics" {
			a.topicsHandler(w, r)
		} else {
			a.progressSummaryHandler(w, r)
		}
		if w.Code != 200 {
			t.Fatalf("%s: %s", path, w.Body.String())
		}
		return w
	}
	var result struct {
		Topics []struct {
			Slug         string
			StartedCount int
		}
	}
	if err := json.Unmarshal(request("/topics").Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	started := map[string]int{}
	for _, topic := range result.Topics {
		started[topic.Slug] = topic.StartedCount
	}
	if started["general-vocabulary"] != 1 || started["healthcare"] != 1 || started["school-childcare"] != 0 {
		t.Fatal("progress must follow word membership", started)
	}
	var summary struct{ TotalWords, ReviewedWords, TopicsStarted, TopicsTotal int }
	if err := json.Unmarshal(request("/progress/summary").Body.Bytes(), &summary); err != nil {
		t.Fatal(err)
	}
	if summary.TotalWords != 9500 || summary.ReviewedWords != 1 || summary.TopicsStarted != 2 || summary.TopicsTotal != 20 {
		t.Fatalf("bad unique summary: %+v", summary)
	} // pupil also has the legal ward sense
	selected, err := a.nextPracticeWords(ctx, userID, healthcare)
	if err != nil {
		t.Fatal(err)
	}
	for _, word := range selected {
		if word.ID == pupil.ID {
			t.Fatal("category switch reset future review")
		}
	}
}
