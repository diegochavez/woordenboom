package main

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestPracticeSelection(t *testing.T) {
	now := time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)
	recent, old, due, future := now.Add(-time.Minute), now.Add(-48*time.Hour), now.Add(-time.Hour), now.Add(24*time.Hour)
	pool := make([]reviewWord, 30)
	for i := range pool {
		pool[i] = reviewWord{ID: fmt.Sprint(i), TopicSlug: []string{"general-vocabulary", "daily-basics"}[i%2]}
	}
	ids := func(words []reviewWord) []string {
		result := []string{}
		for _, w := range words {
			result = append(result, w.ID)
		}
		return result
	}
	t.Run("new learner keeps balanced category and frequency order", func(t *testing.T) {
		got := selectPracticeWords(pool, nil, now, 10)
		if !reflect.DeepEqual(ids(got), ids(pool[:10])) {
			t.Fatal(ids(got))
		}
	})
	t.Run("known and recently missed words do not repeat while unseen remain", func(t *testing.T) {
		progress := map[string]wordReviewState{}
		for i := 0; i < 10; i++ {
			next := future
			if i%2 == 0 {
				next = due
			}
			progress[pool[i].ID] = wordReviewState{LastSeen: &recent, NextReview: &next}
		}
		got := selectPracticeWords(pool, progress, now, 10)
		if !reflect.DeepEqual(ids(got), ids(pool[10:20])) {
			t.Fatal(ids(got))
		}
	})
	t.Run("overdue reviews reserve room for new words", func(t *testing.T) {
		progress := map[string]wordReviewState{}
		for i := 0; i < 20; i++ {
			next := due.Add(time.Duration(i) * time.Minute)
			progress[pool[i].ID] = wordReviewState{LastSeen: &old, NextReview: &next}
		}
		got := selectPracticeWords(pool, progress, now, 10)
		expected := append(append([]reviewWord{}, pool[:5]...), pool[20:25]...)
		if !reflect.DeepEqual(ids(got), ids(expected)) {
			t.Fatal(ids(got))
		}
	})
	t.Run("due-only selection fills ten", func(t *testing.T) {
		progress := map[string]wordReviewState{}
		for i := range pool {
			progress[pool[i].ID] = wordReviewState{LastSeen: &old, NextReview: &due}
		}
		if got := selectPracticeWords(pool, progress, now, 10); len(got) != 10 {
			t.Fatal(len(got))
		}
	})
	t.Run("small remaining set does not pad with recently known words", func(t *testing.T) {
		progress := map[string]wordReviewState{}
		for i := 0; i < 28; i++ {
			progress[pool[i].ID] = wordReviewState{LastSeen: &recent, NextReview: &future}
		}
		if got := selectPracticeWords(pool, progress, now, 10); !reflect.DeepEqual(ids(got), ids(pool[28:])) {
			t.Fatal(ids(got))
		}
	})
	t.Run("future review dates are respected even after all words are seen", func(t *testing.T) {
		progress := map[string]wordReviewState{}
		for i := range pool {
			seen := recent.Add(-time.Duration(i) * time.Minute)
			progress[pool[i].ID] = wordReviewState{LastSeen: &seen, NextReview: &future}
		}
		got := selectPracticeWords(pool, progress, now, 10)
		if len(got) != 0 {
			t.Fatal(ids(got))
		}
	})
	t.Run("empty pool", func(t *testing.T) {
		if got := selectPracticeWords(nil, nil, now, 10); len(got) != 0 {
			t.Fatal(got)
		}
	})
}

func TestSessionSnapshotPreservesPrivateGradingFields(t *testing.T) {
	original := []sessionQuestion{{ID: "q1", WordID: "w1", Word: "kennen", Translation: "know", Type: "reveal", TopicSlug: "general-vocabulary", CorrectAnswer: "know"}}
	snapshot, err := encodeSessionQuestions(original)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := decodeSessionQuestions(snapshot, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(original, decoded) {
		t.Fatalf("snapshot changed questions: %#v", decoded)
	}
	if _, err := decodeSessionQuestions(snapshot, 2); err == nil {
		t.Fatal("accepted incorrect question count")
	}
}
