package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "Postgres connection URL")
	limit := flag.Int("limit", 50, "number of candidates to print")
	withFrequency := flag.Bool("with-frequency", true, "require wordfreq evidence")
	topic := flag.String("topic", "", "require a topic label")
	excludeTopic := flag.String("exclude-topic", "", "exclude a topic label")
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

	args := []any{*limit}
	where := "WHERE is_learning_candidate"
	if *withFrequency {
		where += " AND zipf_score IS NOT NULL"
	}
	if *topic != "" {
		args = append(args, *topic)
		where += fmt.Sprintf(" AND $%d = ANY(topic_labels)", len(args))
	}
	if *excludeTopic != "" {
		args = append(args, *excludeTopic)
		where += fmt.Sprintf(" AND NOT ($%d = ANY(topic_labels))", len(args))
	}
	rows, err := db.Query(ctx, fmt.Sprintf(`
		SELECT lemma, entry_type, COALESCE(cefr_level, ''), COALESCE(frequency_rank, 0), COALESCE(zipf_score, 0), candidate_score, topic_labels
		FROM candidate_word_scores
		%s
		ORDER BY candidate_score DESC, frequency_rank NULLS LAST, lemma
		LIMIT $1
	`, where), args...)
	if err != nil {
		log.Fatalf("load candidate scores: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var lemma, entryType, cefr string
		var rank int
		var zipf, score float64
		var topics []string
		if err := rows.Scan(&lemma, &entryType, &cefr, &rank, &zipf, &score, &topics); err != nil {
			log.Fatalf("scan candidate score: %v", err)
		}
		fmt.Printf("%s\t%s\t%s\trank=%d\tzipf=%.2f\tscore=%.2f\ttopics=%s\n", lemma, entryType, cefr, rank, zipf, score, strings.Join(topics, ","))
	}
	if err := rows.Err(); err != nil {
		log.Fatalf("candidate rows: %v", err)
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
