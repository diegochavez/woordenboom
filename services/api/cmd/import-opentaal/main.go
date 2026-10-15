package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultCommit  = "b250510dda431785f962019167d1415198ff3905"
	defaultSHA256  = "12e5fb5e3c73840b583b30016926d1f63a75e9bf1652a3a6634b2ba7c49ad7be"
	defaultVersion = "2.20.23"
	sourceID       = "opentaal-wordlist"
	sourceURL      = "https://github.com/OpenTaal/opentaal-wordlist"
)

type scannerSource struct {
	scanner *bufio.Scanner
	value   []any
	err     error
}

func (s *scannerSource) Next() bool {
	for s.scanner.Scan() {
		word := strings.TrimSuffix(s.scanner.Text(), "\r")
		if word == "" {
			continue
		}
		s.value = []any{word}
		return true
	}
	s.err = s.scanner.Err()
	return false
}

func (s *scannerSource) Values() ([]any, error) {
	return s.value, nil
}

func (s *scannerSource) Err() error {
	return s.err
}

func main() {
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "Postgres connection URL")
	commit := flag.String("commit", defaultCommit, "OpenTaal git commit")
	version := flag.String("version", defaultVersion, "OpenTaal list version")
	expectedSHA := flag.String("sha256", defaultSHA256, "expected wordlist SHA-256")
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	downloadURL := fmt.Sprintf("https://raw.githubusercontent.com/OpenTaal/opentaal-wordlist/%s/wordlist.txt", *commit)
	file, actualSHA, err := download(downloadURL)
	if err != nil {
		log.Fatalf("download OpenTaal word list: %v", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()
	if !strings.EqualFold(actualSHA, *expectedSHA) {
		log.Fatalf("word list checksum mismatch: expected %s, got %s", *expectedSHA, actualSHA)
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()
	if err := db.Ping(ctx); err != nil {
		log.Fatalf("ping database: %v", err)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		log.Fatalf("begin import transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO vocabulary_sources (id, title, creator, source_url, version, commit_sha, license, content_sha256)
		VALUES ($1, 'Dutch word list', 'OpenTaal', $2, $3, $4, 'BSD-3-Clause', $5)
		ON CONFLICT (id) DO UPDATE SET version = EXCLUDED.version, commit_sha = EXCLUDED.commit_sha,
			license = EXCLUDED.license, content_sha256 = EXCLUDED.content_sha256
	`, sourceID, sourceURL, *version, *commit, actualSHA)
	if err != nil {
		log.Fatalf("save vocabulary source: %v", err)
	}
	if _, err := tx.Exec(ctx, `CREATE TEMP TABLE imported_opentaal_words (lemma TEXT PRIMARY KEY) ON COMMIT DROP`); err != nil {
		log.Fatalf("create import table: %v", err)
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		log.Fatalf("rewind word list: %v", err)
	}
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	count, err := tx.CopyFrom(ctx, pgx.Identifier{"imported_opentaal_words"}, []string{"lemma"}, &scannerSource{scanner: scanner})
	if err != nil {
		log.Fatalf("copy word list: %v", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO words (lemma, display_word, language_code, source)
		SELECT lemma, lemma, 'nl', $1 FROM imported_opentaal_words
		ON CONFLICT (lemma) DO NOTHING
	`, sourceID); err != nil {
		log.Fatalf("insert words: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO word_sources (word_id, source_id)
		SELECT w.id, $1 FROM imported_opentaal_words i JOIN words w ON w.lemma = i.lemma
		ON CONFLICT (word_id, source_id) DO NOTHING
	`, sourceID); err != nil {
		log.Fatalf("link word sources: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM word_sources ws WHERE ws.source_id = $1
		AND NOT EXISTS (
			SELECT 1 FROM imported_opentaal_words i JOIN words w ON w.lemma = i.lemma WHERE w.id = ws.word_id
		)
	`, sourceID); err != nil {
		log.Fatalf("remove stale source links: %v", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE vocabulary_sources SET word_count = $2, imported_at = now() WHERE id = $1
	`, sourceID, count); err != nil {
		log.Fatalf("finish vocabulary source: %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("commit import: %v", err)
	}

	log.Printf("imported %d OpenTaal words, version %s, commit %s", count, *version, *commit)
}

func download(url string) (*os.File, string, error) {
	client := &http.Client{Timeout: 2 * time.Minute}
	response, err := client.Get(url)
	if err != nil {
		return nil, "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected HTTP status %s", response.Status)
	}
	file, err := os.CreateTemp("", "opentaal-wordlist-*.txt")
	if err != nil {
		return nil, "", err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(file, hash), response.Body); err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, "", err
	}
	return file, hex.EncodeToString(hash.Sum(nil)), nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
