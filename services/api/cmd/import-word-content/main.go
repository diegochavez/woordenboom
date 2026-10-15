package main

import (
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

const generatedSource = "woordenboom-generated-v1"

type contentRow struct {
	Lemma              string
	EnglishTranslation string
	DutchExample       string
	EnglishExample     string
}

func main() {
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "Postgres connection URL")
	input := flag.String("input", "-", "CSV input path, or - for stdin")
	batchSlug := flag.String("batch", "release-2000", "batch slug")
	status := flag.String("status", "needs_content_review", "review status to set for imported rows")
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if *status != "needs_content_review" && *status != "approved" {
		log.Fatal("status must be needs_content_review or approved")
	}

	rows, err := readRows(*input)
	if err != nil {
		log.Fatalf("read content rows: %v", err)
	}
	if len(rows) == 0 {
		log.Fatal("no importable content rows found")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	imported, err := save(ctx, db, *batchSlug, *status, rows)
	if err != nil {
		log.Fatalf("save content rows: %v", err)
	}
	log.Printf("imported content for %d words into batch %q", imported, *batchSlug)
}

func readRows(path string) ([]contentRow, error) {
	var reader io.Reader
	var file *os.File
	if path == "-" {
		reader = os.Stdin
	} else {
		var err error
		file, err = os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		reader = file
	}

	csvReader := csv.NewReader(reader)
	csvReader.FieldsPerRecord = -1
	header, err := csvReader.Read()
	if err != nil {
		return nil, err
	}
	columns := map[string]int{}
	for index, name := range header {
		columns[strings.TrimSpace(name)] = index
	}
	for _, required := range []string{"lemma", "english_translation", "dutch_example", "english_example"} {
		if _, ok := columns[required]; !ok {
			return nil, fmt.Errorf("missing required column %q", required)
		}
	}

	result := make([]contentRow, 0)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		row := contentRow{
			Lemma:              cell(record, columns["lemma"]),
			EnglishTranslation: cell(record, columns["english_translation"]),
			DutchExample:       cell(record, columns["dutch_example"]),
			EnglishExample:     cell(record, columns["english_example"]),
		}
		if row.Lemma == "" || row.EnglishTranslation == "" || row.DutchExample == "" || row.EnglishExample == "" {
			continue
		}
		if !strings.Contains(strings.ToLower(row.DutchExample), strings.ToLower(row.Lemma)) {
			return nil, fmt.Errorf("example for %q does not contain lemma: %q", row.Lemma, row.DutchExample)
		}
		result = append(result, row)
	}
	return result, nil
}

func cell(record []string, index int) string {
	if index >= len(record) {
		return ""
	}
	return strings.TrimSpace(record[index])
}

func save(ctx context.Context, db *pgxpool.Pool, batchSlug, status string, rows []contentRow) (int, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	imported := 0
	for _, row := range rows {
		var wordID string
		if err := tx.QueryRow(ctx, `
			SELECT w.id::text
			FROM vocabulary_batches vb
			JOIN vocabulary_batch_words vbw ON vbw.batch_id = vb.id
			JOIN words w ON w.id = vbw.word_id
			WHERE vb.slug = $1 AND w.lemma = $2
		`, batchSlug, row.Lemma).Scan(&wordID); err != nil {
			return imported, fmt.Errorf("find batch word %q: %w", row.Lemma, err)
		}
		if _, err := tx.Exec(ctx, `DELETE FROM word_translations WHERE word_id = $1 AND source = $2`, wordID, generatedSource); err != nil {
			return imported, err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM word_examples WHERE word_id = $1 AND source = $2`, wordID, generatedSource); err != nil {
			return imported, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO word_translations (word_id, language_code, translation, sort_order, source)
			VALUES ($1, 'en', $2, 0, $3)
		`, wordID, row.EnglishTranslation, generatedSource); err != nil {
			return imported, err
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO word_examples (word_id, dutch_sentence, english_sentence, source, sort_order)
			VALUES ($1, $2, $3, $4, 0)
		`, wordID, row.DutchExample, row.EnglishExample, generatedSource); err != nil {
			return imported, err
		}
		if _, err := tx.Exec(ctx, `
			UPDATE vocabulary_batch_words vbw
			SET review_status = $3, updated_at = now()
			FROM vocabulary_batches vb
			WHERE vbw.batch_id = vb.id AND vb.slug = $1 AND vbw.word_id = $2
		`, batchSlug, wordID, status); err != nil {
			return imported, err
		}
		imported++
	}
	return imported, tx.Commit(ctx)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
