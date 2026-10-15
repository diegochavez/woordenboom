package main

import (
	"context"
	"flag"
	"log"
	"math"
	"os"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	classifierName    = "woordenboom-heuristic"
	classifierVersion = "2026-09-28.2"
	sourceID          = "opentaal-wordlist"
)

var symbolicPattern = regexp.MustCompile(`[0-9@€&+/²³₂]`)

type wordRow struct {
	ID             string
	Lemma          string
	HasTranslation bool
	HasExample     bool
}

type classification struct {
	WordID         string
	EntryType      string
	CEFRLevel      string
	UsageRank      *int
	UsageScore     *float64
	LearnerRank    int
	Usefulness     float64
	Complexity     float64
	Candidate      bool
	HasTranslation bool
	HasExample     bool
	Status         string
	Confidence     float64
}

type wordLabel struct {
	WordID      string
	Dimension   string
	Slug        string
	Confidence  float64
	Priority    int
	NeedsReview bool
}

type classificationSource struct {
	items []classification
	index int
	runID string
}

func (s *classificationSource) Next() bool {
	return s.index < len(s.items)
}

func (s *classificationSource) Values() ([]any, error) {
	item := s.items[s.index]
	s.index++
	return []any{
		item.WordID, item.EntryType, item.CEFRLevel, item.UsageRank, item.UsageScore,
		item.LearnerRank, item.Usefulness, item.Complexity, item.Candidate,
		item.HasTranslation, item.HasExample, item.Status, item.Confidence, s.runID,
	}, nil
}

func (s *classificationSource) Err() error { return nil }

type labelSource struct {
	items []wordLabel
	index int
	runID string
}

func (s *labelSource) Next() bool {
	return s.index < len(s.items)
}

func (s *labelSource) Values() ([]any, error) {
	item := s.items[s.index]
	s.index++
	return []any{item.WordID, item.Dimension, item.Slug, item.Confidence, item.Priority, s.runID, item.NeedsReview}, nil
}

func (s *labelSource) Err() error { return nil }

func main() {
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "Postgres connection URL")
	limit := flag.Int("limit", 0, "optional max words to classify")
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

	words, err := loadWords(ctx, db, *limit)
	if err != nil {
		log.Fatalf("load words: %v", err)
	}

	classifications := make([]classification, 0, len(words))
	labels := make([]wordLabel, 0, len(words)*3)
	for _, word := range words {
		item, wordLabels := classify(word)
		classifications = append(classifications, item)
		labels = append(labels, wordLabels...)
	}

	if err := save(ctx, db, classifications, labels); err != nil {
		log.Fatalf("save classifications: %v", err)
	}
	log.Printf("classified %d words with %d labels", len(classifications), len(labels))
}

func loadWords(ctx context.Context, db *pgxpool.Pool, limit int) ([]wordRow, error) {
	query := `
		SELECT w.id::text, w.lemma,
			EXISTS (SELECT 1 FROM word_translations wt WHERE wt.word_id = w.id),
			EXISTS (SELECT 1 FROM word_examples we WHERE we.word_id = w.id)
		FROM words w ORDER BY w.lemma
	`
	if limit > 0 {
		query += ` LIMIT $1`
	}
	var rows pgx.Rows
	var err error
	if limit > 0 {
		rows, err = db.Query(ctx, query, limit)
	} else {
		rows, err = db.Query(ctx, query)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	words := make([]wordRow, 0)
	for rows.Next() {
		var word wordRow
		if err := rows.Scan(&word.ID, &word.Lemma, &word.HasTranslation, &word.HasExample); err != nil {
			return nil, err
		}
		words = append(words, word)
	}
	return words, rows.Err()
}

func save(ctx context.Context, db *pgxpool.Pool, classifications []classification, labels []wordLabel) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var runID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO classifier_runs (classifier_name, classifier_version, source_id, notes)
		VALUES ($1, $2, $3, $4) RETURNING id::text
	`, classifierName, classifierVersion, sourceID, "Deterministic structural classifier. Usage rank is present only for a small built-in common-word seed list; remaining words require a licensed frequency source.").Scan(&runID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		CREATE TEMP TABLE classified_words (
			word_id UUID PRIMARY KEY,
			entry_type TEXT NOT NULL,
			cefr_level TEXT,
			usage_rank INTEGER,
			usage_score DOUBLE PRECISION,
			learner_priority INTEGER NOT NULL,
			usefulness_score DOUBLE PRECISION NOT NULL,
			complexity_score DOUBLE PRECISION NOT NULL,
			is_learning_candidate BOOLEAN NOT NULL,
			has_translation BOOLEAN NOT NULL,
			has_example BOOLEAN NOT NULL,
			classification_status TEXT NOT NULL,
			classification_confidence DOUBLE PRECISION NOT NULL,
			classifier_run_id UUID NOT NULL
		) ON COMMIT DROP
	`); err != nil {
		return err
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"classified_words"}, []string{"word_id", "entry_type", "cefr_level", "usage_rank", "usage_score", "learner_priority", "usefulness_score", "complexity_score", "is_learning_candidate", "has_translation", "has_example", "classification_status", "classification_confidence", "classifier_run_id"}, &classificationSource{items: classifications, runID: runID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO word_learning_metadata (word_id, entry_type, cefr_level, usage_rank, usage_score, learner_priority, usefulness_score, complexity_score, is_learning_candidate, has_translation, has_example, classification_status, classification_confidence, classifier_run_id)
		SELECT word_id, entry_type, cefr_level, usage_rank, usage_score, learner_priority, usefulness_score, complexity_score, is_learning_candidate, has_translation, has_example, classification_status, classification_confidence, classifier_run_id FROM classified_words
		ON CONFLICT (word_id) DO UPDATE SET
			entry_type = EXCLUDED.entry_type,
			cefr_level = EXCLUDED.cefr_level,
			usage_rank = EXCLUDED.usage_rank,
			usage_score = EXCLUDED.usage_score,
			learner_priority = EXCLUDED.learner_priority,
			usefulness_score = EXCLUDED.usefulness_score,
			complexity_score = EXCLUDED.complexity_score,
			is_learning_candidate = EXCLUDED.is_learning_candidate,
			has_translation = EXCLUDED.has_translation,
			has_example = EXCLUDED.has_example,
			classification_status = EXCLUDED.classification_status,
			classification_confidence = EXCLUDED.classification_confidence,
			classifier_run_id = EXCLUDED.classifier_run_id,
			classified_at = now(),
			updated_at = now()
	`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		CREATE TEMP TABLE classified_labels (
			word_id UUID NOT NULL,
			dimension_slug TEXT NOT NULL,
			label_slug TEXT NOT NULL,
			confidence DOUBLE PRECISION NOT NULL,
			priority_score INTEGER NOT NULL,
			classifier_run_id UUID NOT NULL,
			needs_review BOOLEAN NOT NULL
		) ON COMMIT DROP
	`); err != nil {
		return err
	}
	if _, err := tx.CopyFrom(ctx, pgx.Identifier{"classified_labels"}, []string{"word_id", "dimension_slug", "label_slug", "confidence", "priority_score", "classifier_run_id", "needs_review"}, &labelSource{items: labels, runID: runID}); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		DELETE FROM word_labels wl USING learning_labels ll, learning_dimensions d, classifier_runs cr
		WHERE wl.label_id = ll.id
			AND ll.dimension_id = d.id
			AND wl.classifier_run_id = cr.id
			AND cr.classifier_name = $1
			AND d.slug IN ('topic', 'cefr', 'entry_type')
	`, classifierName); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO word_labels (word_id, label_id, confidence, priority_score, classifier_run_id, needs_review)
		SELECT cl.word_id, ll.id, cl.confidence, cl.priority_score, cl.classifier_run_id, cl.needs_review
		FROM classified_labels cl
		JOIN learning_dimensions d ON d.slug = cl.dimension_slug
		JOIN learning_labels ll ON ll.dimension_id = d.id AND ll.slug = cl.label_slug
		ON CONFLICT (word_id, label_id) DO UPDATE SET confidence = EXCLUDED.confidence,
			priority_score = EXCLUDED.priority_score, classifier_run_id = EXCLUDED.classifier_run_id,
			needs_review = EXCLUDED.needs_review, updated_at = now()
		WHERE word_labels.classifier_run_id IS NULL
			OR EXISTS (
				SELECT 1 FROM classifier_runs cr
				WHERE cr.id = word_labels.classifier_run_id AND cr.classifier_name = $1
			)
	`, classifierName); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE classifier_runs SET status = 'completed', words_processed = $2, completed_at = now() WHERE id = $1`, runID, len(classifications)); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func classify(word wordRow) (classification, []wordLabel) {
	lemma := strings.TrimSpace(word.Lemma)
	lower := strings.ToLower(lemma)
	length := utf8.RuneCountInString(lemma)
	entryType := entryTypeFor(lemma)
	complexity := complexityScore(lemma, entryType)
	cefr := cefrFor(lower, length, complexity, entryType)
	usageRank, usageScore := usageFor(lower)
	usefulness := usefulnessScore(entryType, complexity, usageScore)
	learnerPriority := int(math.Round(usefulness*1000 - complexity*35))
	candidate := entryType == "base_word" || entryType == "compound" || entryType == "multiword"
	if length > 28 || entryType == "proper_name" || entryType == "abbreviation" || entryType == "symbolic" || cefr == "C2" {
		candidate = false
	}
	status := "needs_frequency"
	if !candidate {
		status = "rejected"
	} else if usageRank == nil {
		status = "needs_frequency"
	} else {
		status = "classified"
	}
	confidence := 0.62
	if usageRank != nil {
		confidence = 0.78
	}
	if entryType == "unknown" {
		confidence = 0.4
	}

	item := classification{
		WordID: word.ID, EntryType: entryType, CEFRLevel: cefr, UsageRank: usageRank, UsageScore: usageScore,
		LearnerRank: learnerPriority, Usefulness: usefulness, Complexity: complexity, Candidate: candidate,
		HasTranslation: word.HasTranslation, HasExample: word.HasExample, Status: status, Confidence: confidence,
	}
	labels := []wordLabel{
		{WordID: word.ID, Dimension: "entry_type", Slug: entryType, Confidence: confidence, Priority: learnerPriority, NeedsReview: confidence < 0.5},
		{WordID: word.ID, Dimension: "cefr", Slug: cefr, Confidence: confidence, Priority: learnerPriority, NeedsReview: confidence < 0.5},
	}
	for _, topic := range topicLabelsFor(lower, entryType, usageRank != nil) {
		labels = append(labels, wordLabel{WordID: word.ID, Dimension: "topic", Slug: topic.slug, Confidence: topic.confidence, Priority: learnerPriority, NeedsReview: topic.confidence < 0.55})
	}
	return item, labels
}

type topicMatch struct {
	slug       string
	confidence float64
}

func entryTypeFor(word string) string {
	if strings.Contains(word, " ") {
		return "multiword"
	}
	if symbolicPattern.MatchString(word) {
		return "symbolic"
	}
	if strings.ContainsAny(word, ".'") {
		return "abbreviation"
	}
	if isAllUpper(word) && utf8.RuneCountInString(word) > 1 {
		return "abbreviation"
	}
	if startsWithUpper(word) {
		return "proper_name"
	}
	if strings.Contains(word, "-") || utf8.RuneCountInString(word) >= 15 {
		return "compound"
	}
	if looksInflected(strings.ToLower(word)) {
		return "inflection"
	}
	if isLettersOnly(word) {
		return "base_word"
	}
	return "unknown"
}

func cefrFor(word string, length int, complexity float64, entryType string) string {
	if rank, _ := usageFor(word); rank != nil {
		switch {
		case *rank <= 40:
			return "A1"
		case *rank <= 100:
			return "A2"
		default:
			return "B1"
		}
	}
	if entryType == "symbolic" || entryType == "abbreviation" || entryType == "proper_name" {
		return "C2"
	}
	if entryType == "multiword" || entryType == "compound" {
		if length <= 14 {
			return "B1"
		}
		if length <= 22 {
			return "B2"
		}
		return "C1"
	}
	switch {
	case length <= 5 && complexity < 2.5:
		return "A2"
	case length <= 8:
		return "B1"
	case length <= 12:
		return "B2"
	case length <= 18:
		return "C1"
	default:
		return "C2"
	}
}

func complexityScore(word, entryType string) float64 {
	score := float64(utf8.RuneCountInString(word)) / 4
	if strings.Contains(word, "-") {
		score += 1.2
	}
	if strings.Contains(word, " ") {
		score += 1.5
	}
	if !isASCII(word) {
		score += 0.4
	}
	switch entryType {
	case "proper_name", "abbreviation", "symbolic":
		score += 4
	case "compound", "multiword":
		score += 1
	case "inflection":
		score += 0.5
	}
	return math.Round(score*100) / 100
}

func usefulnessScore(entryType string, complexity float64, usageScore *float64) float64 {
	score := 6.0 - complexity
	if usageScore != nil {
		score += *usageScore * 5
	}
	switch entryType {
	case "base_word":
		score += 1.2
	case "compound", "multiword":
		score += 0.3
	case "inflection":
		score -= 1.2
	case "proper_name", "abbreviation", "symbolic":
		score -= 4
	}
	if score < 0 {
		return 0
	}
	return math.Round(score*100) / 100
}

func topicLabelsFor(word, entryType string, common bool) []topicMatch {
	matches := make([]topicMatch, 0, 2)
	if common && entryType == "base_word" {
		matches = append(matches, topicMatch{slug: "daily-basics", confidence: 0.82})
	}
	for _, rule := range topicRules {
		for _, needle := range rule.needles {
			if topicNeedleMatches(word, needle) {
				matches = append(matches, topicMatch{slug: rule.slug, confidence: rule.confidence})
				break
			}
		}
	}
	if len(matches) == 0 && entryType == "base_word" && utf8.RuneCountInString(word) <= 7 {
		matches = append(matches, topicMatch{slug: "daily-basics", confidence: 0.48})
	}
	return dedupeTopics(matches)
}

func topicNeedleMatches(word, needle string) bool {
	if utf8.RuneCountInString(needle) <= 4 {
		return word == needle
	}
	return strings.Contains(word, needle)
}

func dedupeTopics(matches []topicMatch) []topicMatch {
	seen := map[string]topicMatch{}
	for _, match := range matches {
		if previous, ok := seen[match.slug]; !ok || match.confidence > previous.confidence {
			seen[match.slug] = match
		}
	}
	result := make([]topicMatch, 0, len(seen))
	for _, match := range seen {
		result = append(result, match)
	}
	return result
}

func usageFor(word string) (*int, *float64) {
	if rank, ok := commonWordRank[word]; ok {
		score := 1 - (float64(rank-1) / float64(len(commonWordRank)))
		return &rank, &score
	}
	return nil, nil
}

func looksInflected(word string) bool {
	if utf8.RuneCountInString(word) <= 5 {
		return false
	}
	for _, suffix := range []string{"tjes", "pjes", "jes", "eren", "heden", "ende", "den", "ten", "en", "s"} {
		if strings.HasSuffix(word, suffix) {
			return true
		}
	}
	return false
}

func startsWithUpper(word string) bool {
	r, _ := utf8.DecodeRuneInString(word)
	return unicode.IsUpper(r)
}

func isAllUpper(word string) bool {
	foundLetter := false
	for _, r := range word {
		if unicode.IsLetter(r) {
			foundLetter = true
			if unicode.IsLower(r) {
				return false
			}
		}
	}
	return foundLetter
}

func isLettersOnly(word string) bool {
	for _, r := range word {
		if !unicode.IsLetter(r) && r != '-' {
			return false
		}
	}
	return true
}

func isASCII(word string) bool {
	for _, r := range word {
		if r > unicode.MaxASCII {
			return false
		}
	}
	return true
}

type topicRule struct {
	slug       string
	confidence float64
	needles    []string
}

var topicRules = []topicRule{
	{slug: "housing-renting", confidence: 0.72, needles: []string{"huur", "woning", "huis", "kamer", "appartement", "verhuur", "makelaar", "borg", "adres", "verhuis"}},
	{slug: "supermarket-food", confidence: 0.70, needles: []string{"brood", "melk", "kaas", "appel", "eten", "drink", "winkel", "boodschap", "groente", "fruit", "vlees"}},
	{slug: "public-transport", confidence: 0.76, needles: []string{"trein", "tram", "bus", "halte", "station", "perron", "kaart", "ticket", "vertraging", "reis"}},
	{slug: "healthcare", confidence: 0.76, needles: []string{"arts", "huisarts", "apotheek", "ziek", "klacht", "pijn", "medic", "zorg", "verzekering", "spoed"}},
	{slug: "government-municipality", confidence: 0.72, needles: []string{"gemeente", "belasting", "formulier", "aanvraag", "bewijs", "melding", "vergunning", "loket"}},
	{slug: "immigration-documents", confidence: 0.72, needles: []string{"paspoort", "visum", "verblijf", "identiteit", "document", "inschrijving", "uittreksel", "nationaliteit"}},
	{slug: "work-office", confidence: 0.68, needles: []string{"werk", "baan", "kantoor", "collega", "vergadering", "contract", "salaris", "dienst", "rooster"}},
	{slug: "banking-money", confidence: 0.74, needles: []string{"bank", "rekening", "betaling", "geld", "factuur", "prijs", "kosten", "pin", "saldo", "budget"}},
	{slug: "social-life", confidence: 0.62, needles: []string{"vriend", "feest", "familie", "hobby", "uitnod", "afspraak", "samen", "club"}},
	{slug: "weather-small-talk", confidence: 0.72, needles: []string{"weer", "regen", "zon", "wind", "koud", "warm", "temperatuur", "mist", "sneeuw"}},
	{slug: "home-repair", confidence: 0.76, needles: []string{"lekkage", "repar", "schroef", "stopcontact", "kraan", "verf", "onderhoud", "monteur", "gereedschap"}},
	{slug: "emergencies", confidence: 0.78, needles: []string{"nood", "brand", "politie", "ambulance", "spoed", "veilig", "gevaar", "ongeval", "hulp"}},
	{slug: "school-childcare", confidence: 0.72, needles: []string{"school", "kind", "opvang", "leraar", "les", "ouder", "klas", "huiswerk"}},
	{slug: "politics-news", confidence: 0.66, needles: []string{"politiek", "verkiez", "nieuws", "minister", "partij", "kamer", "wet", "regering"}},
	{slug: "engineering-technical", confidence: 0.68, needles: []string{"techn", "meter", "systeem", "machine", "druk", "materiaal", "motor", "construct"}},
	{slug: "software-it", confidence: 0.72, needles: []string{"software", "computer", "data", "bestand", "server", "wachtwoord", "internet", "app", "beveilig"}},
	{slug: "legal-contracts", confidence: 0.72, needles: []string{"recht", "jurid", "contract", "overeenkomst", "plicht", "bezwaar", "advocaat", "clausule"}},
	{slug: "utilities-bills", confidence: 0.74, needles: []string{"energie", "water", "gas", "stroom", "internet", "meter", "factuur", "rekening", "aansluiting"}},
	{slug: "appointments-phone", confidence: 0.70, needles: []string{"afspraak", "bel", "telefoon", "bericht", "nummer", "mail", "bevestig", "planning"}},
}

var commonWordRank = func() map[string]int {
	words := []string{"de", "het", "een", "ik", "je", "jij", "u", "hij", "zij", "we", "wij", "ze", "zijn", "is", "ben", "bent", "was", "waren", "heb", "hebt", "heeft", "hebben", "had", "kan", "kun", "kunt", "wil", "wilt", "moet", "mag", "zal", "zou", "ga", "gaat", "gaan", "kom", "komt", "komen", "doe", "doet", "doen", "maak", "maakt", "maken", "zie", "ziet", "zien", "hoor", "hoort", "horen", "zeg", "zegt", "zeggen", "vraag", "vragen", "antwoord", "ja", "nee", "niet", "geen", "wel", "ook", "en", "of", "maar", "want", "dat", "dit", "die", "deze", "hier", "daar", "waar", "wie", "wat", "wanneer", "hoe", "waarom", "naar", "van", "voor", "met", "zonder", "in", "op", "bij", "over", "onder", "tussen", "door", "tot", "als", "dan", "nu", "vandaag", "morgen", "gisteren", "tijd", "dag", "week", "maand", "jaar", "huis", "werk", "school", "kind", "man", "vrouw", "mens", "naam", "adres", "nummer", "geld", "water", "eten", "drinken", "goed", "slecht", "nieuw", "oud", "groot", "klein", "veel", "weinig", "meer", "minder", "eerste", "laatste"}
	result := make(map[string]int, len(words))
	for index, word := range words {
		result[word] = index + 1
	}
	return result
}()

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
