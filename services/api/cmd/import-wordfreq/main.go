package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const sourceID = "wordfreq-nl"

type frequencyRow struct {
	Lemma string
	Zipf  float64
	Rank  int
}

type frequencySource struct {
	items []frequencyRow
	index int
}

func (s *frequencySource) Next() bool {
	return s.index < len(s.items)
}

func (s *frequencySource) Values() ([]any, error) {
	item := s.items[s.index]
	s.index++
	return []any{item.Lemma, item.Zipf, item.Rank}, nil
}

func (s *frequencySource) Err() error { return nil }

func main() {
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "Postgres connection URL")
	input := flag.String("input", "-", "TSV input path, or - for stdin. Format: lemma<TAB>zipf")
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	rows, err := readRows(*input)
	if err != nil {
		log.Fatalf("read frequency rows: %v", err)
	}
	if len(rows) == 0 {
		log.Fatal("no frequency rows found")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	matched, err := save(ctx, db, rows)
	if err != nil {
		log.Fatalf("save word frequencies: %v", err)
	}
	log.Printf("imported %d wordfreq rows, matched %d words", len(rows), matched)
}

func readRows(path string) ([]frequencyRow, error) {
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

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	rows := make([]frequencyRow, 0)
	seen := map[string]bool{}
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid row %q", line)
		}
		lemma := strings.TrimSpace(parts[0])
		if lemma == "" || seen[lemma] {
			continue
		}
		zipf, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err != nil {
			return nil, fmt.Errorf("parse zipf for %q: %w", lemma, err)
		}
		seen[lemma] = true
		rows = append(rows, frequencyRow{Lemma: lemma, Zipf: zipf})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Zipf == rows[j].Zipf {
			return rows[i].Lemma < rows[j].Lemma
		}
		return rows[i].Zipf > rows[j].Zipf
	})
	for index := range rows {
		rows[index].Rank = index + 1
	}
	return rows, nil
}

func save(ctx context.Context, db *pgxpool.Pool, rows []frequencyRow) (int64, error) {
	tx, err := db.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `
		INSERT INTO lexical_sources (id, title, creator, source_url, version, license, attribution, safe_for_commercial_app, requires_attribution, share_alike_risk, notes)
		VALUES ($1, 'wordfreq Dutch frequency estimates', 'Robyn Speer and source contributors', 'https://github.com/rspeer/wordfreq', '3.0.2 data snapshot through about 2021', 'Apache-2.0 code; bundled data includes CC BY-SA and other attributed sources', 'wordfreq by Robyn Speer; see https://github.com/rspeer/wordfreq and bundled notices.', true, true, true, 'Use for ranking/frequency evidence. Preserve attribution and do not redistribute stripped data without notices.')
		ON CONFLICT (id) DO NOTHING
	`, sourceID); err != nil {
		return 0, err
	}
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE imported_wordfreq (lemma TEXT PRIMARY KEY, zipf_score DOUBLE PRECISION NOT NULL, frequency_rank INTEGER NOT NULL) ON COMMIT DROP`); err != nil {
		return 0, err
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"imported_wordfreq"}, []string{"lemma", "zipf_score", "frequency_rank"}, &frequencySource{items: rows}); err != nil {
		return 0, err
	}
	commandTag, err := tx.Exec(ctx, `
		INSERT INTO word_frequencies (word_id, source_id, frequency_rank, zipf_score, raw_score, imported_at)
		SELECT w.id, $1, iw.frequency_rank, iw.zipf_score, iw.zipf_score, now()
		FROM imported_wordfreq iw
		JOIN words w ON w.lemma = iw.lemma
		ON CONFLICT (word_id, source_id) DO UPDATE SET
			frequency_rank = EXCLUDED.frequency_rank,
			zipf_score = EXCLUDED.zipf_score,
			raw_score = EXCLUDED.raw_score,
			imported_at = now()
	`, sourceID)
	if err != nil {
		return 0, err
	}
	return commandTag.RowsAffected(), tx.Commit(ctx)
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
