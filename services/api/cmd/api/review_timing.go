package main

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type reviewTiming struct {
	RememberedCount     int        `json:"rememberedCount"`
	RecommendedInterval string     `json:"recommendedInterval"`
	NextReviewAt        *time.Time `json:"nextReviewAt"`
	Paused              bool       `json:"paused"`
}

func recommendedInterval(streak int) string {
	// Elapsed durations, capped at 365 days. Existing due dates are unchanged.
	steps := []string{"1d", "3d", "7d", "14d", "30d", "60d", "90d", "180d", "365d"}
	return steps[min(max(streak, 0), len(steps)-1)]
}

func validReviewInterval(value string) bool {
	switch value {
	case "", "auto", "10m", "1d", "2d", "3d", "7d", "14d", "30d", "60d", "90d", "180d", "365d", "paused":
		return true
	}
	return false
}

func intervalMinutes(value string) int {
	switch value {
	case "10m":
		return 10
	case "1d":
		return 1440
	case "2d":
		return 2880
	case "3d":
		return 4320
	case "7d":
		return 10080
	case "14d":
		return 20160
	case "30d":
		return 43200
	case "60d":
		return 86400
	case "90d":
		return 129600
	case "180d":
		return 259200
	case "365d":
		return 525600
	}
	return 0
}

func (a *app) reviewTimingHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	id := r.PathValue("id")
	var timing reviewTiming
	var streak int
	err := a.db.QueryRow(r.Context(), `SELECT COALESCE(p.correct_count,0),COALESCE(p.review_streak,0),p.next_review_at,COALESCE(p.review_paused,false)
        FROM words w JOIN vocabulary_batch_words bw ON bw.word_id=w.id JOIN vocabulary_batches b ON b.id=bw.batch_id
        LEFT JOIN user_word_progress p ON p.word_key=w.id::text AND p.user_id=$2
        WHERE w.id::text=$1 AND b.slug=$3`, id, user.ID, a.config.vocabularyBatch).Scan(&timing.RememberedCount, &streak, &timing.NextReviewAt, &timing.Paused)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "word_not_found"})
		return
	}
	if err != nil {
		a.serverError(w, "load review timing", err)
		return
	}
	timing.RecommendedInterval = recommendedInterval(streak)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, timing)
}

// The progress-row lock serializes concurrent answers from different sessions.
// Session answer deduplication happens before this function in the same transaction.
func (a *app) updateWordProgressWithTiming(ctx context.Context, tx pgx.Tx, userID, topicSlug, wordID string, correct bool, choice string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO user_word_progress(user_id,word_key,topic_slug,status) VALUES($1,$2,$3,'new') ON CONFLICT(user_id,word_key) DO NOTHING`, userID, wordID, topicSlug); err != nil {
		return err
	}
	var streak int
	var paused bool
	if err := tx.QueryRow(ctx, `SELECT review_streak,review_paused FROM user_word_progress WHERE user_id=$1 AND word_key=$2 FOR UPDATE`, userID, wordID).Scan(&streak, &paused); err != nil {
		return err
	}
	if !correct {
		// Forgetting resumes paused words and resets consecutive recall growth.
		choice = "10m"
	} else if choice == "" || choice == "auto" {
		choice = recommendedInterval(streak)
		if paused {
			choice = "paused"
		}
	}
	minutes := intervalMinutes(choice)
	_, err := tx.Exec(ctx, `UPDATE user_word_progress SET
        topic_slug=$3,
        strength=CASE WHEN $4 THEN LEAST(100,strength+25) ELSE GREATEST(0,strength-10) END,
        correct_count=correct_count+CASE WHEN $4 THEN 1 ELSE 0 END,
        wrong_count=wrong_count+CASE WHEN $4 THEN 0 ELSE 1 END,
        lapses=lapses+CASE WHEN $4 THEN 0 ELSE 1 END,
        review_streak=CASE WHEN $4 THEN review_streak+1 ELSE 0 END,
        status=CASE WHEN $4 AND strength+25>=80 THEN 'mastered' WHEN $4 THEN 'review' ELSE 'learning' END,
        review_paused=$5,interval_days=$6/1440,
        last_seen_at=now(),next_review_at=CASE WHEN $5 THEN NULL ELSE now()+$6*interval '1 minute' END,updated_at=now()
        WHERE user_id=$1 AND word_key=$2`, userID, wordID, topicSlug, correct, choice == "paused", minutes)
	return err
}
