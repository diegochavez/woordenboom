package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	classifierName    = "woordenboom-typesafe"
	classifierVersion = "2026-09-28.1"
	defaultModel      = "jev-latest"
	sourceID          = "opentaal-wordlist"
	typesafeEndpoint  = "https://api.typesafe.ai/v1/systemone"
)

type candidateWord struct {
	ID                  string         `json:"id"`
	Lemma               string         `json:"word"`
	Language            string         `json:"language"`
	EntryType           string         `json:"entry_type"`
	CEFRLevel           sql.NullString `json:"-"`
	LearnerPriority     int            `json:"learner_priority"`
	UsefulnessScore     float64        `json:"usefulness_score"`
	ComplexityScore     float64        `json:"complexity_score"`
	HasTranslation      bool           `json:"has_translation"`
	HasExample          bool           `json:"has_example"`
	TopicLabels         []string       `json:"candidate_topics"`
	HeuristicConfidence float64        `json:"heuristic_confidence"`
}

type requestState struct {
	Word            string         `json:"word"`
	Language        string         `json:"language"`
	KnownHeuristics map[string]any `json:"known_heuristics"`
	LearnerContext  string         `json:"learner_context"`
}

type systemOneRequest struct {
	State     requestState        `json:"state"`
	Model     string              `json:"model"`
	Questions map[string]question `json:"questions"`
}

type question struct {
	Type         string `json:"type"`
	Instructions any    `json:"instructions"`
	Criteria     any    `json:"criteria,omitempty"`
}

type systemOneResponse struct {
	Model   string                     `json:"model"`
	Answers map[string]json.RawMessage `json:"answers"`
	Usage   usage                      `json:"usage"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type choiceAnswer struct {
	Type          string             `json:"type"`
	Choice        string             `json:"choice"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

type scoreAnswer struct {
	Type          string             `json:"type"`
	Score         float64            `json:"score"`
	Probabilities map[string]float64 `json:"probabilities"`
	Confidence    float64            `json:"confidence"`
}

type noulAnswer struct {
	Type string  `json:"type"`
	Noul float64 `json:"noul"`
}

type appliedJudgment struct {
	EntryType   string
	CEFRLevel   string
	Usefulness  float64
	Candidate   bool
	Topic       string
	Confidence  float64
	NeedsReview bool
	Applied     bool
	ApplyCEFR   bool
	ApplyTopic  bool
}

func main() {
	databaseURL := flag.String("database-url", env("DATABASE_URL", ""), "Postgres connection URL")
	apiKey := flag.String("api-key", env("TYPESAFE_API_KEY", env("typesafe_ai_api_key", "")), "TypeSafe API key")
	model := flag.String("model", defaultModel, "TypeSafe model")
	set := flag.String("set", "release", "candidate set: release, balanced, seeded-topic, common-core, tricky, or heuristic")
	limit := flag.Int("limit", 100, "maximum words to classify")
	dryRun := flag.Bool("dry-run", true, "print selected words without calling TypeSafe or writing judgments")
	apply := flag.Bool("apply", true, "apply high-confidence judgments to word metadata when not dry-running")
	flag.Parse()

	if *databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}
	if !*dryRun && *apiKey == "" {
		log.Fatal("TYPESAFE_API_KEY is required when --dry-run=false")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, *databaseURL)
	if err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer db.Close()

	words, err := loadCandidates(ctx, db, *set, *limit)
	if err != nil {
		log.Fatalf("load candidates: %v", err)
	}
	if len(words) == 0 {
		log.Printf("no candidates found for set %q", *set)
		return
	}

	if *dryRun {
		for _, word := range words {
			log.Printf("dry-run candidate word=%q entry_type=%s cefr=%s topics=%s priority=%d", word.Lemma, word.EntryType, nullString(word.CEFRLevel), strings.Join(word.TopicLabels, ","), word.LearnerPriority)
		}
		log.Printf("dry-run selected %d candidates", len(words))
		return
	}

	runID, err := startRun(ctx, db, *model, fmt.Sprintf("TypeSafe candidate set=%s limit=%d apply=%t", *set, *limit, *apply))
	if err != nil {
		log.Fatalf("start classifier run: %v", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	processed := 0
	totalInput := 0
	totalOutput := 0
	for _, word := range words {
		state := stateFor(word)
		resp, err := askTypeSafe(ctx, client, *apiKey, *model, state, word.TopicLabels)
		if err != nil {
			_ = failRun(ctx, db, runID, processed, err)
			log.Fatalf("classify %q: %v", word.Lemma, err)
		}
		judgment, err := interpret(resp, word.Lemma, word.TopicLabels)
		if err != nil {
			_ = failRun(ctx, db, runID, processed, err)
			log.Fatalf("interpret %q: %v", word.Lemma, err)
		}
		judgment.Applied = *apply && judgment.Applied
		if err := saveJudgment(ctx, db, runID, *model, word, state, resp, judgment); err != nil {
			_ = failRun(ctx, db, runID, processed, err)
			log.Fatalf("save %q: %v", word.Lemma, err)
		}
		processed++
		totalInput += resp.Usage.InputTokens
		totalOutput += resp.Usage.OutputTokens
		log.Printf("classified word=%q entry_type=%s cefr=%s topic=%s confidence=%.2f applied=%t review=%t", word.Lemma, judgment.EntryType, judgment.CEFRLevel, judgment.Topic, judgment.Confidence, judgment.Applied, judgment.NeedsReview)
	}
	if err := completeRun(ctx, db, runID, processed, totalInput, totalOutput); err != nil {
		log.Fatalf("complete classifier run: %v", err)
	}
	if err := reportRun(ctx, db, runID); err != nil {
		log.Printf("quality report failed: %v", err)
	}
	log.Printf("classified %d words with TypeSafe input_tokens=%d output_tokens=%d", processed, totalInput, totalOutput)
}

func loadCandidates(ctx context.Context, db *pgxpool.Pool, set string, limit int) ([]candidateWord, error) {
	if limit <= 0 {
		limit = 100
	}
	if set == "balanced" {
		return loadBalancedCandidates(ctx, db, limit)
	}
	query := `
		SELECT w.id::text, w.lemma, w.language_code,
			COALESCE(wlm.entry_type, 'unknown'), wlm.cefr_level,
			COALESCE(wlm.learner_priority, 0), COALESCE(wlm.usefulness_score, 0), COALESCE(wlm.complexity_score, 0),
			EXISTS (SELECT 1 FROM word_translations wt WHERE wt.word_id = w.id),
			EXISTS (SELECT 1 FROM word_examples we WHERE we.word_id = w.id),
			COALESCE(wlm.classification_confidence, 0),
			COALESCE(array_remove(array_agg(ll.slug ORDER BY wl.priority_score DESC) FILTER (WHERE ld.slug = 'topic'), NULL), '{}')
		FROM words w
		LEFT JOIN word_learning_metadata wlm ON wlm.word_id = w.id
		LEFT JOIN word_labels wl ON wl.word_id = w.id
		LEFT JOIN learning_labels ll ON ll.id = wl.label_id
		LEFT JOIN learning_dimensions ld ON ld.id = ll.dimension_id AND ld.slug = 'topic'
	`
	args := []any{}
	switch set {
	case "release":
		query += ` WHERE lower(w.lemma) = ANY($1) `
		args = append(args, releaseCandidateWords)
	case "common-core":
		query += ` WHERE lower(w.lemma) = ANY($1) `
		args = append(args, commonCoreWords)
	case "tricky":
		query += ` WHERE lower(w.lemma) = ANY($1) `
		args = append(args, trickyWords)
	case "seeded-topic":
		query += ` WHERE w.lemma = lower(w.lemma) AND lower(w.lemma) = ANY($1) `
		args = append(args, practicalTopicWords)
	case "heuristic":
		query += ` WHERE wlm.is_learning_candidate `
	default:
		return nil, fmt.Errorf("unsupported candidate set %q", set)
	}
	limitPlaceholder := len(args) + 1
	query += fmt.Sprintf(`
		GROUP BY w.id, w.lemma, w.language_code, wlm.entry_type, wlm.cefr_level, wlm.learner_priority, wlm.usefulness_score, wlm.complexity_score, wlm.classification_confidence, wlm.has_translation, wlm.has_example
		ORDER BY COALESCE(wlm.has_translation, false) DESC, COALESCE(wlm.has_example, false) DESC,
			COALESCE(wlm.learner_priority, 0) DESC, w.lemma
		LIMIT $%d
	`, limitPlaceholder)
	args = append(args, limit)

	rows, err := db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	words := make([]candidateWord, 0)
	for rows.Next() {
		var word candidateWord
		if err := rows.Scan(&word.ID, &word.Lemma, &word.Language, &word.EntryType, &word.CEFRLevel, &word.LearnerPriority, &word.UsefulnessScore, &word.ComplexityScore, &word.HasTranslation, &word.HasExample, &word.HeuristicConfidence, &word.TopicLabels); err != nil {
			return nil, err
		}
		words = append(words, word)
	}
	return words, rows.Err()
}

func loadBalancedCandidates(ctx context.Context, db *pgxpool.Pool, limit int) ([]candidateWord, error) {
	seededLimit := maxInt(1, limit/2)
	commonLimit := maxInt(1, limit*3/10)
	trickyLimit := maxInt(1, limit-seededLimit-commonLimit)

	seen := map[string]bool{}
	result := make([]candidateWord, 0, limit)
	for _, bucket := range []struct {
		set   string
		limit int
	}{
		{set: "seeded-topic", limit: seededLimit},
		{set: "common-core", limit: commonLimit},
		{set: "tricky", limit: trickyLimit},
	} {
		items, err := loadCandidates(ctx, db, bucket.set, bucket.limit)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			if seen[item.ID] {
				continue
			}
			seen[item.ID] = true
			result = append(result, item)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func startRun(ctx context.Context, db *pgxpool.Pool, model, notes string) (string, error) {
	var runID string
	err := db.QueryRow(ctx, `
		INSERT INTO classifier_runs (classifier_name, classifier_version, model_name, source_id, notes)
		VALUES ($1, $2, $3, $4, $5) RETURNING id::text
	`, classifierName, classifierVersion, model, sourceID, notes).Scan(&runID)
	return runID, err
}

func stateFor(word candidateWord) requestState {
	cefr := nullString(word.CEFRLevel)
	if cefr == "" {
		cefr = "unknown"
	}
	return requestState{
		Word:     word.Lemma,
		Language: "Dutch",
		KnownHeuristics: map[string]any{
			"entry_type":              word.EntryType,
			"current_cefr":            cefr,
			"candidate_topics":        word.TopicLabels,
			"learner_priority":        word.LearnerPriority,
			"usefulness_score":        word.UsefulnessScore,
			"complexity_score":        word.ComplexityScore,
			"has_english_translation": word.HasTranslation,
			"has_example":             word.HasExample,
		},
		LearnerContext: "English-speaking expats learning practical Dutch for life in the Netherlands or Belgium. Interpret the word as Dutch, not as an English word or English proper name. Prefer useful standalone learner vocabulary over names, inflections, symbols, and very rare specialist terms.",
	}
}

func askTypeSafe(ctx context.Context, client *http.Client, apiKey, model string, state requestState, candidateTopics []string) (systemOneResponse, error) {
	payload := systemOneRequest{State: state, Model: model, Questions: questions(candidateTopics)}
	body, err := json.Marshal(payload)
	if err != nil {
		return systemOneResponse{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, typesafeEndpoint, bytes.NewReader(body))
	if err != nil {
		return systemOneResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)
	if err != nil {
		return systemOneResponse{}, err
	}
	defer res.Body.Close()
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		return systemOneResponse{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return systemOneResponse{}, fmt.Errorf("typesafe status %d: %s", res.StatusCode, string(resBody))
	}
	var out systemOneResponse
	if err := json.Unmarshal(resBody, &out); err != nil {
		return systemOneResponse{}, err
	}
	return out, nil
}

func questions(candidateTopics []string) map[string]question {
	items := map[string]question{
		"entry_type": {
			Type:         "choice",
			Instructions: "What kind of Dutch vocabulary entry is `word`? Interpret it as Dutch and use its capitalization as given. Prefer base_word for normal standalone dictionary headwords and useful function words that a learner may study directly. Do not mark a lowercase Dutch word as a proper name just because it is also an English name. Do not choose inflection only because the word ends in -s or -en; many Dutch base nouns and verbs have those endings. Choose inflection only when the entry is mainly a plural, conjugated, declined, comparative, or other form that is better replaced by a clearer lemma.",
			Criteria: map[string]string{
				"base_word":    "A normal standalone learner word.",
				"inflection":   "A plural, conjugated, declined, or otherwise inflected form that is not the best dictionary form to teach directly.",
				"compound":     "A compound word that is meaningful as a single vocabulary item.",
				"proper_name":  "A person, place, organization, brand, or other name.",
				"abbreviation": "An abbreviation, acronym, clipped form, or initialism.",
				"multiword":    "A phrase or expression containing multiple words.",
				"symbolic":     "Contains digits, formulas, symbols, or non-word notation.",
				"unknown":      "Cannot be categorized confidently.",
			},
		},
		"cefr_level": {
			Type:         "choice",
			Instructions: "Which CEFR level best fits `word` for an English-speaking adult learning practical Dutch? Interpret homographs by their Dutch meaning. Do not assign C1 or C2 just because a practical compound is long. Use B1 or B2 for common practical housing, healthcare, transport, repair, government, money, or work terms. Use C2 only for rare, archaic, very formal, or highly specialized vocabulary.",
			Criteria: map[string]string{
				"A1": "First essential everyday words.",
				"A2": "Common survival vocabulary for everyday interactions.",
				"B1": "Independent daily-life vocabulary.",
				"B2": "Work, administration, and more abstract vocabulary.",
				"C1": "Advanced, formal, or lower-frequency vocabulary.",
				"C2": "Rare, archaic, highly specialized, or not useful as learner vocabulary.",
			},
		},
		"learner_usefulness": {
			Type:         "score",
			Instructions: "How useful is `word` as a lesson candidate for the stated learner context?",
			Criteria: []string{
				"Not useful; reject from lessons.",
				"Low usefulness; only keep with human review.",
				"Moderately useful for a specific situation.",
				"Useful practical vocabulary.",
				"Very useful core practical vocabulary.",
			},
		},
		"good_lesson_candidate": {
			Type:         "noul",
			Instructions: "Should `word` be eligible for a vocabulary lesson once it has an English translation and example sentence?",
			Criteria: map[string]string{
				"true":  "A useful Dutch learner item, not merely a name, spelling variant, inflection, symbol, or very rare specialist term.",
				"false": "Not appropriate as a direct lesson vocabulary item.",
			},
		},
		"primary_topic": {
			Type:         "choice",
			Instructions: "Choose the single best practical topic for `word`, interpreting it as Dutch. If `known_heuristics.candidate_topics` contains a plausible topic, prefer that topic over an unrelated English interpretation. Choose none when no topic fits well.",
			Criteria:     topicCriteria(),
		},
	}
	seen := map[string]bool{}
	for _, topic := range candidateTopics {
		if topic == "" || topic == "none" || seen[topic] {
			continue
		}
		seen[topic] = true
		description, ok := topicCriteria()[topic]
		if !ok {
			continue
		}
		items[topicFitKey(topic)] = question{
			Type:         "noul",
			Instructions: fmt.Sprintf("Interpreting `word` as Dutch, does it fit the practical topic %q? Topic meaning: %s", topic, description),
			Criteria: map[string]string{
				"true":  "The Dutch word is useful vocabulary for this topic.",
				"false": "The Dutch word does not belong in this topic, or only matches by accidental spelling overlap.",
			},
		}
	}
	return items
}

func topicCriteria() map[string]string {
	return map[string]string{
		"none":                    "No practical topic fits well.",
		"daily-basics":            "Core words for everyday interaction.",
		"housing-renting":         "Contracts, repairs, utilities, and moving.",
		"supermarket-food":        "Shopping, ingredients, meals, and labels.",
		"public-transport":        "Trains, trams, buses, tickets, and delays.",
		"healthcare":              "Appointments, symptoms, pharmacy, and insurance.",
		"government-municipality": "Forms, registration, taxes, and official letters.",
		"immigration-documents":   "Permits, identity documents, and applications.",
		"work-office":             "Meetings, contracts, colleagues, and planning.",
		"banking-money":           "Payments, accounts, invoices, and budgeting.",
		"social-life":             "Invitations, relationships, hobbies, and small talk.",
		"weather-small-talk":      "Everyday conversation starters.",
		"home-repair":             "Tools, leaks, electricity, paint, and furniture.",
		"emergencies":             "Urgent help, safety, and reporting problems.",
		"school-childcare":        "Children, school communication, and care.",
		"politics-news":           "Government, elections, society, and media.",
		"engineering-technical":   "Systems, materials, measurements, and maintenance.",
		"software-it":             "Digital work, support, security, and products.",
		"legal-contracts":         "Agreements, rights, obligations, and disputes.",
		"utilities-bills":         "Energy, water, internet, invoices, and meters.",
		"appointments-phone":      "Planning, rescheduling, calls, and confirmations.",
	}
}

func topicFitKey(topic string) string {
	return "topic_fit_" + strings.ReplaceAll(topic, "-", "_")
}

func interpret(resp systemOneResponse, lemma string, candidateTopics []string) (appliedJudgment, error) {
	entry, err := decodeChoice(resp.Answers, "entry_type")
	if err != nil {
		return appliedJudgment{}, err
	}
	cefr, err := decodeChoice(resp.Answers, "cefr_level")
	if err != nil {
		return appliedJudgment{}, err
	}
	usefulness, err := decodeScore(resp.Answers, "learner_usefulness")
	if err != nil {
		return appliedJudgment{}, err
	}
	candidate, err := decodeNoul(resp.Answers, "good_lesson_candidate")
	if err != nil {
		return appliedJudgment{}, err
	}
	topic, err := decodeChoice(resp.Answers, "primary_topic")
	if err != nil {
		return appliedJudgment{}, err
	}
	topicChoice := topic.Choice
	topicConfidence := topic.Confidence
	for _, candidateTopic := range candidateTopics {
		candidateTopic = strings.TrimSpace(candidateTopic)
		if candidateTopic == "" || candidateTopic == "none" {
			continue
		}
		fit, err := decodeNoul(resp.Answers, topicFitKey(candidateTopic))
		if err != nil {
			continue
		}
		if fit.Noul >= 0.72 && fit.Noul > topicConfidence {
			topicChoice = candidateTopic
			topicConfidence = fit.Noul
		}
	}
	if overrideTopic, ok := lexicalTopicOverrides[strings.ToLower(lemma)]; ok && (topicChoice == "none" || topicConfidence < 0.70) {
		topicChoice = overrideTopic
		topicConfidence = 0.86
	}
	if overrideEntryType, ok := lexicalEntryTypeOverrides[strings.ToLower(lemma)]; ok && entry.Confidence < 0.75 {
		entry.Choice = overrideEntryType
		entry.Confidence = 0.86
	}
	confidence := minFloat(entry.Confidence, candidate.Noul)
	applyCandidate := candidate.Noul >= 0.72 || candidate.Noul <= 0.28
	applyCEFR := cefr.Confidence >= 0.70
	applyTopic := topicChoice != "none" && topicConfidence >= 0.70
	applied := entry.Confidence >= 0.75 && applyCandidate
	needsReview := applied && (!applyCEFR || (topic.Choice != "none" && !applyTopic) || usefulness.Confidence < 0.55)
	return appliedJudgment{
		EntryType:   entry.Choice,
		CEFRLevel:   cefr.Choice,
		Usefulness:  math.Round(usefulness.Score*250) / 100,
		Candidate:   candidate.Noul >= 0.72,
		Topic:       topicChoice,
		Confidence:  math.Round(confidence*100) / 100,
		NeedsReview: needsReview,
		Applied:     applied,
		ApplyCEFR:   applyCEFR,
		ApplyTopic:  applyTopic,
	}, nil
}

func saveJudgment(ctx context.Context, db *pgxpool.Pool, runID, model string, word candidateWord, state requestState, resp systemOneResponse, judgment appliedJudgment) error {
	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	stateJSON, err := json.Marshal(state)
	if err != nil {
		return err
	}
	answersJSON, err := json.Marshal(resp.Answers)
	if err != nil {
		return err
	}
	finalJSON, err := json.Marshal(map[string]any{
		"entry_type":   judgment.EntryType,
		"cefr_level":   judgment.CEFRLevel,
		"usefulness":   judgment.Usefulness,
		"candidate":    judgment.Candidate,
		"topic":        judgment.Topic,
		"confidence":   judgment.Confidence,
		"needs_review": judgment.NeedsReview,
		"applied":      judgment.Applied,
		"apply_cefr":   judgment.ApplyCEFR,
		"apply_topic":  judgment.ApplyTopic,
	})
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO typesafe_word_judgments (word_id, classifier_run_id, model_name, request_state, raw_answers, final_judgment, input_tokens, output_tokens, applied, needs_review)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, word.ID, runID, model, stateJSON, answersJSON, finalJSON, resp.Usage.InputTokens, resp.Usage.OutputTokens, judgment.Applied, judgment.NeedsReview); err != nil {
		return err
	}
	if judgment.Applied {
		status := "classified"
		if judgment.NeedsReview {
			status = "needs_review"
		}
		if !judgment.Candidate {
			status = "rejected"
		}
		if _, err := tx.Exec(ctx, `
			UPDATE word_learning_metadata
			SET entry_type = $2, cefr_level = CASE WHEN $9 THEN $3 ELSE cefr_level END, usefulness_score = GREATEST(usefulness_score, $4),
				is_learning_candidate = CASE WHEN $5 AND has_translation THEN true WHEN NOT $5 THEN false ELSE is_learning_candidate END,
				classification_status = $6, classification_confidence = $7,
				classifier_run_id = $8, classified_at = now(), updated_at = now()
			WHERE word_id = $1
		`, word.ID, judgment.EntryType, judgment.CEFRLevel, judgment.Usefulness, judgment.Candidate, status, judgment.Confidence, runID, judgment.ApplyCEFR); err != nil {
			return err
		}
		if err := upsertLabel(ctx, tx, word.ID, "entry_type", judgment.EntryType, judgment.Confidence, word.LearnerPriority, runID, judgment.NeedsReview); err != nil {
			return err
		}
		if judgment.ApplyCEFR {
			if err := upsertLabel(ctx, tx, word.ID, "cefr", judgment.CEFRLevel, judgment.Confidence, word.LearnerPriority, runID, judgment.NeedsReview); err != nil {
				return err
			}
		}
		if judgment.ApplyTopic {
			if err := upsertLabel(ctx, tx, word.ID, "topic", judgment.Topic, judgment.Confidence, word.LearnerPriority, runID, judgment.NeedsReview); err != nil {
				return err
			}
		}
	}
	return tx.Commit(ctx)
}

type execer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

func upsertLabel(ctx context.Context, tx execer, wordID, dimension, slug string, confidence float64, priority int, runID string, needsReview bool) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO word_labels (word_id, label_id, confidence, priority_score, classifier_run_id, needs_review)
		SELECT $1, ll.id, $4, $5, $6, $7
		FROM learning_labels ll
		JOIN learning_dimensions d ON d.id = ll.dimension_id
		WHERE d.slug = $2 AND ll.slug = $3
		ON CONFLICT (word_id, label_id) DO UPDATE SET confidence = EXCLUDED.confidence,
			priority_score = EXCLUDED.priority_score, classifier_run_id = EXCLUDED.classifier_run_id,
			needs_review = EXCLUDED.needs_review, updated_at = now()
	`, wordID, dimension, slug, confidence, priority, runID, needsReview)
	return err
}

func completeRun(ctx context.Context, db *pgxpool.Pool, runID string, processed, inputTokens, outputTokens int) error {
	notes := fmt.Sprintf("input_tokens=%d output_tokens=%d", inputTokens, outputTokens)
	_, err := db.Exec(ctx, `
		UPDATE classifier_runs
		SET status = 'completed', words_processed = $2, completed_at = now(), notes = COALESCE(notes, '') || '; ' || $3
		WHERE id = $1
	`, runID, processed, notes)
	return err
}

func reportRun(ctx context.Context, db *pgxpool.Pool, runID string) error {
	log.Printf("quality report for run %s", runID)
	if err := reportApplyCounts(ctx, db, runID); err != nil {
		return err
	}
	for _, item := range []struct {
		title string
		expr  string
	}{
		{title: "entry_type", expr: "final_judgment->>'entry_type'"},
		{title: "cefr_level", expr: "final_judgment->>'cefr_level'"},
		{title: "primary_topic", expr: "final_judgment->>'topic'"},
	} {
		if err := reportDistribution(ctx, db, runID, item.title, item.expr); err != nil {
			return err
		}
	}
	return reportLowConfidence(ctx, db, runID)
}

func reportApplyCounts(ctx context.Context, db *pgxpool.Pool, runID string) error {
	rows, err := db.Query(ctx, `
		SELECT applied, needs_review, count(*)
		FROM typesafe_word_judgments
		WHERE classifier_run_id = $1
		GROUP BY applied, needs_review
		ORDER BY applied DESC, needs_review
	`, runID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var applied, needsReview bool
		var count int
		if err := rows.Scan(&applied, &needsReview, &count); err != nil {
			return err
		}
		log.Printf("quality apply_count applied=%t needs_review=%t count=%d", applied, needsReview, count)
	}
	return rows.Err()
}

func reportDistribution(ctx context.Context, db *pgxpool.Pool, runID, title, expr string) error {
	rows, err := db.Query(ctx, fmt.Sprintf(`
		SELECT COALESCE(%s, 'missing') AS label, count(*)
		FROM typesafe_word_judgments
		WHERE classifier_run_id = $1
		GROUP BY label
		ORDER BY count(*) DESC, label
	`, expr), runID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var label string
		var count int
		if err := rows.Scan(&label, &count); err != nil {
			return err
		}
		log.Printf("quality distribution dimension=%s label=%s count=%d", title, label, count)
	}
	return rows.Err()
}

func reportLowConfidence(ctx context.Context, db *pgxpool.Pool, runID string) error {
	rows, err := db.Query(ctx, `
		SELECT w.lemma,
			COALESCE(final_judgment->>'entry_type', 'missing'),
			COALESCE(final_judgment->>'cefr_level', 'missing'),
			COALESCE(final_judgment->>'topic', 'missing'),
			COALESCE((final_judgment->>'confidence')::double precision, 0) AS confidence
		FROM typesafe_word_judgments tj
		JOIN words w ON w.id = tj.word_id
		WHERE classifier_run_id = $1
		ORDER BY confidence ASC, w.lemma
		LIMIT 10
	`, runID)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var lemma, entryType, cefr, topic string
		var confidence float64
		if err := rows.Scan(&lemma, &entryType, &cefr, &topic, &confidence); err != nil {
			return err
		}
		log.Printf("quality low_confidence word=%q entry_type=%s cefr=%s topic=%s confidence=%.2f", lemma, entryType, cefr, topic, confidence)
	}
	return rows.Err()
}

func failRun(ctx context.Context, db *pgxpool.Pool, runID string, processed int, cause error) error {
	_, err := db.Exec(ctx, `
		UPDATE classifier_runs
		SET status = 'failed', words_processed = $2, completed_at = now(), notes = COALESCE(notes, '') || '; failed: ' || $3
		WHERE id = $1
	`, runID, processed, cause.Error())
	return err
}

func decodeChoice(answers map[string]json.RawMessage, key string) (choiceAnswer, error) {
	raw, ok := answers[key]
	if !ok {
		return choiceAnswer{}, fmt.Errorf("missing answer %q", key)
	}
	var out choiceAnswer
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	if out.Type != "choice" || out.Choice == "" {
		return out, fmt.Errorf("answer %q is not a choice", key)
	}
	return out, nil
}

func decodeScore(answers map[string]json.RawMessage, key string) (scoreAnswer, error) {
	raw, ok := answers[key]
	if !ok {
		return scoreAnswer{}, fmt.Errorf("missing answer %q", key)
	}
	var out scoreAnswer
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	if out.Type != "score" {
		return out, fmt.Errorf("answer %q is not a score", key)
	}
	return out, nil
}

func decodeNoul(answers map[string]json.RawMessage, key string) (noulAnswer, error) {
	raw, ok := answers[key]
	if !ok {
		return noulAnswer{}, fmt.Errorf("missing answer %q", key)
	}
	var out noulAnswer
	if err := json.Unmarshal(raw, &out); err != nil {
		return out, err
	}
	if out.Type != "noul" {
		return out, fmt.Errorf("answer %q is not a noul", key)
	}
	return out, nil
}

func nullString(value sql.NullString) string {
	if value.Valid {
		return value.String
	}
	return ""
}

func minFloat(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	result := values[0]
	for _, value := range values[1:] {
		if value < result {
			result = value
		}
	}
	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

var releaseCandidateWords = []string{
	"afspraak", "aanvraag", "rekening", "bewijs", "bericht", "formulier", "verzekering", "contract", "kosten", "termijn", "toestemming", "melding", "adres", "nummer", "wachttijd", "openingstijden", "vertraging", "betaling", "onderhoud", "spoed",
	"huurcontract", "borg", "verhuurder", "huisarts", "apotheek", "klacht", "lekkage", "schroevendraaier", "stopcontact", "halte", "perron", "overstappen", "gemeente", "inschrijving", "uittreksel",
}

var commonCoreWords = []string{
	"de", "het", "een", "ik", "je", "jij", "u", "hij", "zij", "wij", "we", "ze", "zijn", "is", "ben", "bent", "heb", "hebt", "heeft", "hebben", "kan", "kunt", "wil", "moet", "mag", "zal", "ga", "gaan", "kom", "komen",
	"doe", "doen", "maak", "maken", "zie", "zien", "hoor", "horen", "zeg", "zeggen", "vraag", "vragen", "antwoord", "ja", "nee", "niet", "geen", "wel", "ook", "en", "of", "maar", "want", "dat", "dit", "die", "deze", "hier", "daar", "waar",
}

var practicalTopicWords = []string{
	"adres", "afspraak", "aanvraag", "apotheek", "arts", "baan", "bank", "betaling", "bewijs", "bericht", "borg", "contract", "formulier", "gemeente", "geld", "halte", "huis", "huisarts", "huurcontract", "inschrijving",
	"klacht", "kosten", "lekkage", "melding", "nummer", "onderhoud", "overstappen", "perron", "rekening", "school", "schroevendraaier", "spoed", "stopcontact", "termijn", "toestemming", "uittreksel", "verhuurder", "verzekering", "vertraging", "wachttijd",
	"water", "werk", "ziek", "zorg", "paspoort", "visum", "vergunning", "belasting", "factuur", "salaris", "rooster", "telefoon", "bericht", "internet", "energie", "gas", "stroom", "kraan", "reparatie",
}

var lexicalTopicOverrides = map[string]string{
	"arts": "healthcare",
}

var lexicalEntryTypeOverrides = map[string]string{
	"kosten":  "base_word",
	"salaris": "base_word",
}

var trickyWords = []string{
	"openingstijden", "kosten", "huisarts", "overstappen", "schroevendraaier", "huurcontract", "wachttijd", "bewijs", "met", "dan", "ben", "was", "had", "heeft", "gaat", "komt", "maakt", "zegt", "hoort",
	"belastingkamer", "belastingkamers", "belastingfactuur", "belastingpolitiek", "internetbelasting", "ziekenhuisfactuur", "ziekenhuisrekening", "rechtswetenschappen", "kameraadschappelijk", "belastingsamenwerking",
}
