package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

const sessionSize = 10
const recentPracticeCooldown = 10 * time.Minute

type wordReviewState struct {
	Paused     bool
	LastSeen   *time.Time
	NextReview *time.Time
}

func (a *app) nextPracticeWords(ctx context.Context, userID string, pool []reviewWord) ([]reviewWord, error) {
	rows, err := a.db.Query(ctx, `SELECT word_key, last_seen_at, next_review_at, review_paused
        FROM user_word_progress WHERE user_id=$1 AND correct_count + wrong_count > 0`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	progress := map[string]wordReviewState{}
	for rows.Next() {
		var id string
		var state wordReviewState
		if err := rows.Scan(&id, &state.LastSeen, &state.NextReview, &state.Paused); err != nil {
			return nil, err
		}
		progress[id] = state
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return selectPracticeWords(pool, progress, time.Now(), sessionSize), nil
}

// Preserve frequency/category order for unseen words. Reserve up to half a stack
// for due reviews, then fill with new words. Recently answered words wait until
// they become due, including words marked "Not yet" in the last stack.
func selectPracticeWords(pool []reviewWord, progress map[string]wordReviewState, now time.Time, limit int) []reviewWord {
	if limit <= 0 {
		return nil
	}
	var unseen, due []reviewWord
	cutoff := now.Add(-recentPracticeCooldown)
	for _, word := range pool {
		state, seen := progress[word.ID]
		if state.Paused {
			continue
		}
		if !seen {
			unseen = append(unseen, word)
			continue
		}
		if state.NextReview != nil && !state.NextReview.After(now) && (state.LastSeen == nil || !state.LastSeen.After(cutoff)) {
			due = append(due, word)
		}
	}
	sort.SliceStable(due, func(i, j int) bool { return progress[due[i].ID].NextReview.Before(*progress[due[j].ID].NextReview) })
	selected := make([]reviewWord, 0, min(limit, len(pool)))
	reviews := min(limit/2, len(due))
	selected = append(selected, due[:reviews]...)
	selected = append(selected, unseen[:min(limit-len(selected), len(unseen))]...)
	selected = append(selected, due[reviews:min(len(due), reviews+limit-len(selected))]...)
	return selected
}

// Private grading fields are stored separately because sessionQuestion omits
// them from client JSON. Persist the wording as well as the exact selected IDs.
type storedSessionQuestion struct {
	Question      sessionQuestion `json:"question"`
	TopicSlug     string          `json:"topicSlug"`
	CorrectAnswer string          `json:"correctAnswer"`
}

func encodeSessionQuestions(questions []sessionQuestion) ([]byte, error) {
	snapshot := make([]storedSessionQuestion, len(questions))
	for i, q := range questions {
		snapshot[i] = storedSessionQuestion{q, q.TopicSlug, q.CorrectAnswer}
	}
	return json.Marshal(snapshot)
}

func decodeSessionQuestions(snapshot []byte, count int) ([]sessionQuestion, error) {
	var stored []storedSessionQuestion
	if err := json.Unmarshal(snapshot, &stored); err != nil {
		return nil, err
	}
	if len(stored) != count {
		return nil, fmt.Errorf("session snapshot has %d questions, expected %d", len(stored), count)
	}
	questions := make([]sessionQuestion, len(stored))
	for i, item := range stored {
		q := item.Question
		q.TopicSlug = item.TopicSlug
		q.CorrectAnswer = item.CorrectAnswer
		if q.ID == "" || q.WordID == "" || q.TopicSlug == "" {
			return nil, fmt.Errorf("incomplete stored question at %d", i)
		}
		questions[i] = q
	}
	return questions, nil
}
