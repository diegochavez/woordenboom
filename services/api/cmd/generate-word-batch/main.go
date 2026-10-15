package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

var topicOrder = []string{
	"daily-basics",
	"housing-renting",
	"supermarket-food",
	"public-transport",
	"healthcare",
	"government-municipality",
	"immigration-documents",
	"work-office",
	"banking-money",
	"social-life",
	"weather-small-talk",
	"home-repair",
	"emergencies",
	"school-childcare",
	"politics-news",
	"engineering-technical",
	"software-it",
	"legal-contracts",
	"utilities-bills",
	"appointments-phone",
}

type candidate struct {
	WordID         string
	Lemma          string
	EntryType      string
	CEFRLevel      string
	FrequencyRank  int
	ZipfScore      float64
	CandidateScore float64
	TopicLabels    []string
	AssignedTopic  string
	BatchRank      int
	TopicRank      int
	NeedsReview    bool
}

func main() {
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "Postgres connection URL")
	slug := flag.String("slug", "release-2000", "batch slug")
	title := flag.String("title", "Release 2,000 Candidate Vocabulary", "batch title")
	size := flag.Int("size", 2000, "target batch size")
	perTopic := flag.Int("per-topic", 100, "target per non-daily topic")
	dailyLimit := flag.Int("daily-limit", 350, "max daily-basics words")
	topicMinZipf := flag.Float64("topic-min-zipf", 3.0, "minimum Zipf score for assigning a practical topic")
	dryRun := flag.Bool("dry-run", false, "print summary without writing")
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	candidates, err := loadCandidates(ctx, db)
	if err != nil {
		log.Fatalf("load candidates: %v", err)
	}
	selected := selectBatch(candidates, *size, *perTopic, *dailyLimit, *topicMinZipf)
	printSummary(selected)
	if *dryRun {
		return
	}
	if err := saveBatch(ctx, db, *slug, *title, *size, selected); err != nil {
		log.Fatalf("save batch: %v", err)
	}
	log.Printf("saved batch %q with %d words", *slug, len(selected))
}

func loadCandidates(ctx context.Context, db *pgxpool.Pool) ([]candidate, error) {
	rows, err := db.Query(ctx, `
		SELECT word_id::text, lemma, entry_type, COALESCE(cefr_level, ''), COALESCE(frequency_rank, 0), COALESCE(zipf_score, 0), candidate_score, topic_labels
		FROM candidate_word_scores
		WHERE is_learning_candidate
			AND zipf_score IS NOT NULL
			AND entry_type IN ('base_word', 'compound', 'multiword')
			AND COALESCE(cefr_level, '') IN ('A1', 'A2', 'B1', 'B2')
		ORDER BY candidate_score DESC, frequency_rank NULLS LAST, lemma
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]candidate, 0)
	for rows.Next() {
		var item candidate
		if err := rows.Scan(&item.WordID, &item.Lemma, &item.EntryType, &item.CEFRLevel, &item.FrequencyRank, &item.ZipfScore, &item.CandidateScore, &item.TopicLabels); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func selectBatch(candidates []candidate, target, perTopic, dailyLimit int, topicMinZipf float64) []candidate {
	selected := make([]candidate, 0, target)
	seen := map[string]bool{}
	topicCounts := map[string]int{}

	selectForTopic := func(topic string, limit int) {
		for _, item := range candidates {
			if len(selected) >= target || topicCounts[topic] >= limit || seen[item.WordID] || item.ZipfScore < topicMinZipf || !hasTopic(item.TopicLabels, topic) {
				continue
			}
			item.AssignedTopic = topic
			item.TopicRank = topicCounts[topic] + 1
			item.NeedsReview = len(item.TopicLabels) > 1
			selected = append(selected, item)
			seen[item.WordID] = true
			topicCounts[topic]++
		}
	}

	selectForTopic("daily-basics", dailyLimit)
	for _, topic := range topicOrder {
		if topic != "daily-basics" {
			selectForTopic(topic, perTopic)
		}
	}
	for _, item := range candidates {
		if len(selected) >= target || seen[item.WordID] {
			continue
		}
		item.AssignedTopic = bestFallbackTopic(item.TopicLabels)
		item.TopicRank = topicCounts[item.AssignedTopic] + 1
		item.NeedsReview = item.AssignedTopic == "" || len(item.TopicLabels) != 1
		selected = append(selected, item)
		seen[item.WordID] = true
		topicCounts[item.AssignedTopic]++
	}
	for index := range selected {
		selected[index].BatchRank = index + 1
	}
	return selected
}

func hasTopic(topics []string, topic string) bool {
	for _, item := range topics {
		if item == topic {
			return true
		}
	}
	return false
}

func bestFallbackTopic(topics []string) string {
	for _, preferred := range topicOrder {
		if preferred != "daily-basics" && hasTopic(topics, preferred) {
			return preferred
		}
	}
	return ""
}

func printSummary(selected []candidate) {
	counts := map[string]int{}
	review := 0
	for _, item := range selected {
		counts[item.AssignedTopic]++
		if item.NeedsReview {
			review++
		}
	}
	topics := make([]string, 0, len(counts))
	for topic := range counts {
		topics = append(topics, topic)
	}
	sort.Strings(topics)
	for _, topic := range topics {
		label := topic
		if label == "" {
			label = "unassigned"
		}
		log.Printf("topic %s: %d", label, counts[topic])
	}
	log.Printf("selected %d words, needs topic review %d", len(selected), review)
}

func saveBatch(ctx context.Context, db *pgxpool.Pool, slug, title string, target int, selected []candidate) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var batchID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO vocabulary_batches (slug, title, target_size, status, notes)
		VALUES ($1, $2, $3, 'translation_pending', 'Generated from candidate_word_scores using wordfreq evidence. Translate and review before publishing.')
		ON CONFLICT (slug) DO UPDATE SET title = EXCLUDED.title, target_size = EXCLUDED.target_size, status = EXCLUDED.status, updated_at = now()
		RETURNING id::text
	`, slug, title, target).Scan(&batchID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM vocabulary_batch_words WHERE batch_id = $1`, batchID); err != nil {
		return err
	}
	for _, item := range selected {
		reviewStatus := "needs_translation"
		notes := ""
		if item.NeedsReview {
			reviewStatus = "needs_topic_review"
			if len(item.TopicLabels) == 0 {
				notes = "no topic assigned"
			} else {
				notes = fmt.Sprintf("candidate topics: %s", strings.Join(item.TopicLabels, ","))
			}
		}
		var topic any
		var topicRank any
		if item.AssignedTopic != "" {
			topic = item.AssignedTopic
			topicRank = item.TopicRank
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO vocabulary_batch_words (batch_id, word_id, batch_rank, topic_slug, topic_rank, candidate_score, frequency_rank, zipf_score, cefr_level, entry_type, review_status, needs_topic_review, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		`, batchID, item.WordID, item.BatchRank, topic, topicRank, item.CandidateScore, item.FrequencyRank, item.ZipfScore, item.CEFRLevel, item.EntryType, reviewStatus, item.NeedsReview, notes); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
