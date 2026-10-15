package main

import (
	"context"
	"encoding/csv"
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "Postgres connection URL")
	slug := flag.String("slug", "release-2000", "batch slug")
	topic := flag.String("topic", "", "optional topic slug, or unassigned")
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

	where := "WHERE vb.slug = $1"
	args := []any{*slug}
	if *topic == "unassigned" {
		where += " AND vbw.topic_slug IS NULL"
	} else if *topic != "" {
		args = append(args, *topic)
		where += " AND vbw.topic_slug = $2"
	}

	rows, err := db.Query(ctx, `
		SELECT vbw.batch_rank, COALESCE(vbw.topic_slug, ''), COALESCE(vbw.topic_rank, 0), w.lemma,
			vbw.entry_type, COALESCE(vbw.cefr_level, ''), COALESCE(vbw.frequency_rank, 0), COALESCE(vbw.zipf_score, 0),
			vbw.candidate_score, vbw.review_status, vbw.needs_topic_review, vbw.notes,
			COALESCE(wt.translation, ''), COALESCE(we.dutch_sentence, ''), COALESCE(we.english_sentence, '')
		FROM vocabulary_batches vb
		JOIN vocabulary_batch_words vbw ON vbw.batch_id = vb.id
		JOIN words w ON w.id = vbw.word_id
		LEFT JOIN LATERAL (
			SELECT translation FROM word_translations
			WHERE word_id = w.id AND language_code = 'en'
			ORDER BY CASE WHEN source = 'woordenboom-generated-v1' THEN 0 ELSE 1 END, sort_order, created_at
			LIMIT 1
		) wt ON true
		LEFT JOIN LATERAL (
			SELECT dutch_sentence, english_sentence FROM word_examples
			WHERE word_id = w.id
			ORDER BY CASE WHEN source = 'woordenboom-generated-v1' THEN 0 ELSE 1 END, sort_order, created_at
			LIMIT 1
		) we ON true
		`+where+`
		ORDER BY vbw.batch_rank
	`, args...)
	if err != nil {
		log.Fatalf("load batch: %v", err)
	}
	defer rows.Close()

	writer := csv.NewWriter(os.Stdout)
	defer writer.Flush()
	if err := writer.Write([]string{"rank", "topic", "topic_rank", "lemma", "entry_type", "cefr", "frequency_rank", "zipf", "candidate_score", "review_status", "needs_topic_review", "notes", "english_translation", "dutch_example", "english_example"}); err != nil {
		log.Fatalf("write header: %v", err)
	}
	for rows.Next() {
		var rank, topicRank, frequencyRank int
		var topic, lemma, entryType, cefr, status, notes, translation, dutchExample, englishExample string
		var zipf, score float64
		var needsTopicReview bool
		if err := rows.Scan(&rank, &topic, &topicRank, &lemma, &entryType, &cefr, &frequencyRank, &zipf, &score, &status, &needsTopicReview, &notes, &translation, &dutchExample, &englishExample); err != nil {
			log.Fatalf("scan batch row: %v", err)
		}
		if err := writer.Write([]string{
			strconv.Itoa(rank), topic, strconv.Itoa(topicRank), lemma, entryType, cefr, strconv.Itoa(frequencyRank), strconv.FormatFloat(zipf, 'f', 2, 64), strconv.FormatFloat(score, 'f', 2, 64), status, strconv.FormatBool(needsTopicReview), notes, translation, dutchExample, englishExample,
		}); err != nil {
			log.Fatalf("write row: %v", err)
		}
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("batch rows: %v", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
