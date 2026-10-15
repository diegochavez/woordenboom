package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type config struct {
	vocabularyBatch string
	addr            string
	appEnv          string
	authSecret      string
	databaseURL     string
	smtpAddr        string
	smtpUser        string
	smtpPass        string
	emailFrom       string
}

type app struct {
	config                    config
	db                        *pgxpool.Pool
	logger                    *slog.Logger
	originalVocabularyContent bool // Legacy sessions without snapshots use the original translations.
}

type wordMeaning struct {
	English      string `json:"english"`
	Dutch        string `json:"dutch"`
	PartOfSpeech string `json:"partOfSpeech"`
	Source       string `json:"source"`
	SourceURL    string `json:"sourceUrl"`
	License      string `json:"license"`
}

type reviewWord struct {
	ID            string        `json:"id"`
	TopicSlug     string        `json:"-"`
	Topic         string        `json:"topic"`
	Word          string        `json:"word"`
	Pronunciation string        `json:"pronunciation"`
	Example       string        `json:"example"`
	Translation   string        `json:"translation"`
	Meanings      []wordMeaning `json:"meanings,omitempty"`
}

type topic struct {
	Slug        string `json:"slug"`
	Title       string `json:"title"`
	Description string `json:"description"`
}

type userProfile struct {
	Email       string `json:"email"`
	DisplayName string `json:"displayName"`
	AgeRange    string `json:"ageRange"`
}

type learningSession struct {
	ID           string            `json:"id"`
	TopicSlug    string            `json:"topicSlug"`
	TopicSlugs   []string          `json:"topicSlugs"`
	CEFRLevels   []string          `json:"cefrLevels"`
	Mode         string            `json:"mode"`
	Title        string            `json:"title"`
	Questions    []sessionQuestion `json:"questions"`
	CorrectCount int               `json:"correctCount"`
	StartedAt    time.Time         `json:"startedAt"`
}

type authUser struct {
	ID              string `json:"id"`
	Email           string `json:"email"`
	DisplayName     string `json:"displayName,omitempty"`
	AgeRange        string `json:"ageRange,omitempty"`
	ProfileComplete bool   `json:"profileComplete"`
}

type contextKey string

const userContextKey contextKey = "authenticated-user"

type sessionQuestion struct {
	ID            string        `json:"id"`
	WordID        string        `json:"wordId"`
	TopicSlug     string        `json:"-"`
	Type          string        `json:"type"`
	Topic         string        `json:"topic"`
	Prompt        string        `json:"prompt"`
	Word          string        `json:"word"`
	Pronunciation string        `json:"pronunciation"`
	Example       string        `json:"example"`
	Translation   string        `json:"translation"`
	Meanings      []wordMeaning `json:"meanings,omitempty"`
	Options       []string      `json:"options"`
	CorrectAnswer string        `json:"-"`
}

var learningTopics = []topic{
	{Slug: "general-vocabulary", Title: "General vocabulary", Description: "A wider Dutch vocabulary, ordered by frequency."},
	{Slug: "daily-basics", Title: "Daily Basics", Description: "Core words for everyday interaction."},
	{Slug: "housing-renting", Title: "Housing & Renting", Description: "Contracts, repairs, utilities, and moving."},
	{Slug: "supermarket-food", Title: "Supermarket & Food", Description: "Shopping, ingredients, meals, and labels."},
	{Slug: "public-transport", Title: "Public Transport", Description: "Trains, trams, buses, tickets, and delays."},
	{Slug: "healthcare", Title: "Healthcare", Description: "Appointments, symptoms, pharmacy, and insurance."},
	{Slug: "government-municipality", Title: "Government & Municipality", Description: "Forms, registration, taxes, and official letters."},
	{Slug: "immigration-documents", Title: "Immigration & Documents", Description: "Permits, identity documents, and applications."},
	{Slug: "work-office", Title: "Work & Office", Description: "Meetings, contracts, colleagues, and planning."},
	{Slug: "banking-money", Title: "Banking & Money", Description: "Payments, accounts, invoices, and budgeting."},
	{Slug: "social-life", Title: "Social Life", Description: "Invitations, relationships, hobbies, and small talk."},
	{Slug: "weather-small-talk", Title: "Weather & Small Talk", Description: "Everyday conversation starters."},
	{Slug: "home-repair", Title: "Home Repair", Description: "Tools, leaks, electricity, paint, and furniture."},
	{Slug: "emergencies", Title: "Emergencies", Description: "Urgent help, safety, and reporting problems."},
	{Slug: "school-childcare", Title: "School & Childcare", Description: "Children, school communication, and care."},
	{Slug: "politics-news", Title: "Politics & News", Description: "Government, elections, society, and media."},
	{Slug: "engineering-technical", Title: "Engineering & Technical Work", Description: "Systems, materials, measurements, and maintenance."},
	{Slug: "software-it", Title: "Software & IT", Description: "Digital work, support, security, and products."},
	{Slug: "legal-contracts", Title: "Legal & Contracts", Description: "Agreements, rights, obligations, and disputes."},
	{Slug: "utilities-bills", Title: "Utilities & Bills", Description: "Energy, water, internet, invoices, and meters."},
	{Slug: "appointments-phone", Title: "Appointments & Phone Calls", Description: "Planning, rescheduling, calls, and confirmations."},
}

var commonDutchWords = []reviewWord{
	{ID: "afspraak", Word: "afspraak", Pronunciation: "af-spraak", Example: "Ik heb morgen een afspraak.", Translation: "appointment"},
	{ID: "aanvraag", Word: "aanvraag", Pronunciation: "aan-vraag", Example: "De aanvraag is ontvangen.", Translation: "application"},
	{ID: "rekening", Word: "rekening", Pronunciation: "re-ke-ning", Example: "De rekening is betaald.", Translation: "bill or account"},
	{ID: "bewijs", Word: "bewijs", Pronunciation: "be-wijs", Example: "Neem een bewijs mee.", Translation: "proof"},
	{ID: "bericht", Word: "bericht", Pronunciation: "be-richt", Example: "Ik heb een bericht gekregen.", Translation: "message"},
	{ID: "formulier", Word: "formulier", Pronunciation: "for-mu-lier", Example: "Vul het formulier in.", Translation: "form"},
	{ID: "verzekering", Word: "verzekering", Pronunciation: "ver-ze-ke-ring", Example: "Mijn verzekering dekt dit.", Translation: "insurance"},
	{ID: "contract", Word: "contract", Pronunciation: "con-tract", Example: "Lees het contract goed.", Translation: "contract"},
	{ID: "kosten", Word: "kosten", Pronunciation: "kos-ten", Example: "Wat zijn de kosten?", Translation: "costs"},
	{ID: "termijn", Word: "termijn", Pronunciation: "ter-mijn", Example: "De termijn is twee weken.", Translation: "deadline or period"},
	{ID: "toestemming", Word: "toestemming", Pronunciation: "toe-stem-ming", Example: "Je hebt toestemming nodig.", Translation: "permission"},
	{ID: "melding", Word: "melding", Pronunciation: "mel-ding", Example: "Maak een melding online.", Translation: "report or notification"},
	{ID: "adres", Word: "adres", Pronunciation: "a-dres", Example: "Wat is je adres?", Translation: "address"},
	{ID: "nummer", Word: "nummer", Pronunciation: "num-mer", Example: "Bel dit nummer.", Translation: "number"},
	{ID: "wachttijd", Word: "wachttijd", Pronunciation: "wacht-tijd", Example: "De wachttijd is kort.", Translation: "waiting time"},
	{ID: "openingstijden", Word: "openingstijden", Pronunciation: "o-pe-nings-tij-den", Example: "Controleer de openingstijden.", Translation: "opening hours"},
	{ID: "vertraging", Word: "vertraging", Pronunciation: "ver-tra-ging", Example: "De trein heeft vertraging.", Translation: "delay"},
	{ID: "betaling", Word: "betaling", Pronunciation: "be-ta-ling", Example: "De betaling is gelukt.", Translation: "payment"},
	{ID: "onderhoud", Word: "onderhoud", Pronunciation: "on-der-houd", Example: "Er is onderhoud gepland.", Translation: "maintenance"},
	{ID: "spoed", Word: "spoed", Pronunciation: "spoed", Example: "Dit is spoed.", Translation: "urgent"},
}

var topicSeeds = map[string][]reviewWord{
	"housing-renting":         {{ID: "huurcontract", Word: "huurcontract", Pronunciation: "huur-contract", Example: "Ik moet mijn huurcontract ondertekenen.", Translation: "rental contract"}, {ID: "borg", Word: "borg", Pronunciation: "borg", Example: "De borg is een maand huur.", Translation: "deposit"}, {ID: "verhuurder", Word: "verhuurder", Pronunciation: "ver-huur-der", Example: "Bel de verhuurder over de lekkage.", Translation: "landlord"}},
	"healthcare":              {{ID: "huisarts", Word: "huisarts", Pronunciation: "huis-arts", Example: "Ik bel de huisarts.", Translation: "general practitioner"}, {ID: "apotheek", Word: "apotheek", Pronunciation: "a-po-theek", Example: "De apotheek is open.", Translation: "pharmacy"}, {ID: "klacht", Word: "klacht", Pronunciation: "klacht", Example: "Welke klacht heeft u?", Translation: "complaint or symptom"}},
	"home-repair":             {{ID: "lekkage", Word: "lekkage", Pronunciation: "lek-ka-ge", Example: "Er is lekkage onder de gootsteen.", Translation: "leak"}, {ID: "schroevendraaier", Word: "schroevendraaier", Pronunciation: "schroe-ven-draai-er", Example: "Ik heb een schroevendraaier nodig.", Translation: "screwdriver"}, {ID: "stopcontact", Word: "stopcontact", Pronunciation: "stop-con-tact", Example: "Het stopcontact werkt niet.", Translation: "power outlet"}},
	"public-transport":        {{ID: "halte", Word: "halte", Pronunciation: "hal-te", Example: "De halte is om de hoek.", Translation: "stop"}, {ID: "perron", Word: "perron", Pronunciation: "per-ron", Example: "De trein vertrekt van perron vier.", Translation: "platform"}, {ID: "overstappen", Word: "overstappen", Pronunciation: "o-ver-stap-pen", Example: "Je moet overstappen in Utrecht.", Translation: "to transfer"}},
	"government-municipality": {{ID: "gemeente", Word: "gemeente", Pronunciation: "ge-meen-te", Example: "Ik sta ingeschreven bij de gemeente.", Translation: "municipality"}, {ID: "inschrijving", Word: "inschrijving", Pronunciation: "in-schrij-ving", Example: "De inschrijving is verplicht.", Translation: "registration"}, {ID: "uittreksel", Word: "uittreksel", Pronunciation: "uit-trek-sel", Example: "Ik heb een uittreksel nodig.", Translation: "official extract"}},
}

func main() {
	cfg := loadConfig()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := cfg.validate(); err != nil {
		logger.Error("invalid api configuration", "error", err)
		os.Exit(1)
	}

	db, err := pgxpool.New(context.Background(), cfg.databaseURL)
	if err != nil {
		logger.Error("connect database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	if err := db.Ping(context.Background()); err != nil {
		logger.Error("ping database", "error", err)
		os.Exit(1)
	}

	application := &app{config: cfg, db: db, logger: logger}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthHandler)
	mux.HandleFunc("GET /readyz", application.readyHandler)
	mux.HandleFunc("POST /auth/request-code", application.requestCodeHandler)
	mux.HandleFunc("POST /auth/verify-code", application.verifyCodeHandler)
	mux.HandleFunc("POST /auth/refresh", application.refreshHandler)
	mux.HandleFunc("POST /auth/logout", application.logoutHandler)
	mux.Handle("GET /me", application.requireAuth(http.HandlerFunc(application.meHandler)))
	mux.Handle("GET /topics/preview", application.requireAuth(http.HandlerFunc(application.topicPreviewHandler)))
	mux.Handle("GET /topics", application.requireAuth(http.HandlerFunc(application.topicsHandler)))
	mux.Handle("GET /topics/{slug}/words", application.requireAuth(http.HandlerFunc(application.topicWordsHandler)))
	mux.Handle("GET /knowledge-map", application.requireAuth(http.HandlerFunc(application.knowledgeMapHandler)))
	mux.Handle("GET /knowledge-map/ranges/{start}", application.requireAuth(http.HandlerFunc(application.knowledgeMapRangeHandler)))
	mux.Handle("POST /knowledge-map/words/{id}/mark", application.requireAuth(http.HandlerFunc(application.knowledgeMapMarkWordHandler)))
	mux.Handle("GET /progress/summary", application.requireAuth(http.HandlerFunc(application.progressSummaryHandler)))
	mux.Handle("POST /profile", application.requireAuth(http.HandlerFunc(application.profileHandler)))
	mux.Handle("GET /words/{id}/review-timing", application.requireAuth(http.HandlerFunc(application.reviewTimingHandler)))
	mux.Handle("GET /sessions/next", application.requireAuth(http.HandlerFunc(application.nextSessionHandler)))
	mux.Handle("POST /sessions/next", application.requireAuth(http.HandlerFunc(application.createSessionHandler)))
	mux.Handle("POST /sessions/{id}/answer", application.requireAuth(http.HandlerFunc(application.sessionAnswerHandler)))

	server := &http.Server{
		Addr:              cfg.addr,
		Handler:           cors(requestLogger(logger, mux)),
		ReadHeaderTimeout: 5 * time.Second,
	}

	logger.Info("starting api", "addr", cfg.addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func loadConfig() config {
	return config{
		vocabularyBatch: env("VOCABULARY_BATCH", "release-9500"),
		addr:            env("API_ADDR", ":8080"),
		appEnv:          env("APP_ENV", "development"),
		authSecret:      env("AUTH_SECRET", "local-development-secret-change-before-production"),
		databaseURL:     env("DATABASE_URL", ""),
		smtpAddr:        env("SMTP_ADDR", env("MAILPIT_SMTP_ADDR", "localhost:1025")),
		smtpUser:        os.Getenv("SMTP_USER"),
		smtpPass:        os.Getenv("SMTP_PASSWORD"),
		emailFrom:       env("AUTH_EMAIL_FROM", "login@woordenboom.local"),
	}
}

func (cfg config) validate() error {
	if len(cfg.authSecret) < 32 {
		return fmt.Errorf("AUTH_SECRET must be at least 32 characters")
	}
	if cfg.appEnv == "development" {
		return nil
	}
	if cfg.appEnv != "production" && cfg.appEnv != "staging" {
		return fmt.Errorf("APP_ENV must be development, staging, or production")
	}
	if cfg.authSecret == "local-development-secret-change-before-production" || cfg.authSecret == "replace-with-at-least-32-random-characters" {
		return fmt.Errorf("AUTH_SECRET must be changed before deployment")
	}
	if cfg.databaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required outside development")
	}
	databaseURL, err := url.Parse(cfg.databaseURL)
	if err != nil || (databaseURL.Scheme != "postgres" && databaseURL.Scheme != "postgresql") || databaseURL.Hostname() == "" {
		return fmt.Errorf("DATABASE_URL must be a PostgreSQL URL outside development")
	}
	sslMode := databaseURL.Query().Get("sslmode")
	if sslMode != "verify-full" && !(databaseURL.Hostname() == "postgres" && sslMode == "disable") {
		return fmt.Errorf("DATABASE_URL requires sslmode=verify-full, except for the local Compose postgres service")
	}
	smtpHost, _, err := net.SplitHostPort(cfg.smtpAddr)
	if err != nil || smtpHost == "" || strings.EqualFold(smtpHost, "localhost") || strings.EqualFold(smtpHost, "mailpit") || net.ParseIP(smtpHost) != nil && net.ParseIP(smtpHost).IsLoopback() {
		return fmt.Errorf("SMTP_ADDR must point to a delivery service outside development")
	}
	if cfg.smtpUser == "" || cfg.smtpPass == "" {
		return fmt.Errorf("SMTP_USER and SMTP_PASSWORD are required outside development")
	}
	if cfg.emailFrom == "" || strings.HasSuffix(cfg.emailFrom, ".local") {
		return fmt.Errorf("AUTH_EMAIL_FROM must use a verified sending domain outside development")
	}
	return nil
}

func env(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *app) readyHandler(w http.ResponseWriter, r *http.Request) {
	if err := a.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "database_unavailable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *app) requestCodeHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
	}
	if !readJSON(w, r, &payload) {
		return
	}

	email := normalizeEmail(payload.Email)
	if email == "" || !strings.Contains(email, "@") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "valid_email_required"})
		return
	}

	code, err := generateCode()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "code_generation_failed"})
		return
	}

	var recentRequests int
	if err := a.db.QueryRow(r.Context(), `SELECT count(*) FROM auth_email_codes WHERE email = $1 AND created_at > now() - interval '10 minutes'`, email).Scan(&recentRequests); err != nil {
		a.serverError(w, "count email codes", err)
		return
	}
	if recentRequests >= 5 {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too_many_code_requests"})
		return
	}

	_, err = a.db.Exec(r.Context(), `UPDATE auth_email_codes SET consumed_at = now() WHERE email = $1 AND consumed_at IS NULL`, email)
	if err != nil {
		a.serverError(w, "expire email codes", err)
		return
	}
	_, err = a.db.Exec(r.Context(), `
		INSERT INTO auth_email_codes (email, code_hash, expires_at, user_agent) VALUES ($1, $2, now() + interval '10 minutes', $3)
	`, email, a.hashAuth(email+":"+code), r.UserAgent())
	if err != nil {
		a.serverError(w, "store email code", err)
		return
	}

	if err := a.sendLoginCode(email, code); err != nil {
		a.serverError(w, "send email code", err)
		return
	}
	if a.config.appEnv == "development" {
		a.logger.Info("dev email login code", "email", email, "code", code, "expires_in_minutes", 10)
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "code_sent"})
}

func (a *app) verifyCodeHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if !readJSON(w, r, &payload) {
		return
	}

	email := normalizeEmail(payload.Email)
	code := strings.TrimSpace(payload.Code)

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		a.serverError(w, "begin verify transaction", err)
		return
	}
	defer tx.Rollback(r.Context())

	var codeID, storedHash string
	var attempts int
	var expiresAt time.Time
	err = tx.QueryRow(r.Context(), `
		SELECT id::text, code_hash, attempts, expires_at
		FROM auth_email_codes
		WHERE email = $1 AND consumed_at IS NULL
		ORDER BY created_at DESC LIMIT 1 FOR UPDATE
	`, email).Scan(&codeID, &storedHash, &attempts, &expiresAt)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "code_expired_or_missing"})
		return
	}
	if err != nil {
		a.serverError(w, "load email code", err)
		return
	}
	if time.Now().After(expiresAt) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "code_expired_or_missing"})
		return
	}
	if attempts >= 5 {
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "too_many_attempts"})
		return
	}
	if !hmac.Equal([]byte(a.hashAuth(email+":"+code)), []byte(storedHash)) {
		_, _ = tx.Exec(r.Context(), `UPDATE auth_email_codes SET attempts = attempts + 1 WHERE id = $1`, codeID)
		_ = tx.Commit(r.Context())
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_code"})
		return
	}

	_, err = tx.Exec(r.Context(), `UPDATE auth_email_codes SET consumed_at = now() WHERE id = $1`, codeID)
	if err != nil {
		a.serverError(w, "consume email code", err)
		return
	}
	var userID string
	err = tx.QueryRow(r.Context(), `
		INSERT INTO users (email, email_verified_at) VALUES ($1, now())
		ON CONFLICT (email) DO UPDATE SET email_verified_at = COALESCE(users.email_verified_at, now()), updated_at = now()
		RETURNING id::text
	`, email).Scan(&userID)
	if err != nil {
		a.serverError(w, "upsert user", err)
		return
	}
	refreshToken, err := randomToken(32)
	if err != nil {
		a.serverError(w, "generate refresh token", err)
		return
	}
	_, err = tx.Exec(r.Context(), `INSERT INTO auth_sessions (user_id, refresh_token_hash, expires_at) VALUES ($1, $2, now() + interval '30 days')`, userID, a.hashAuth(refreshToken))
	if err != nil {
		a.serverError(w, "store auth session", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		a.serverError(w, "commit verification", err)
		return
	}
	accessToken, err := a.createAccessToken(userID, email)
	if err != nil {
		a.serverError(w, "create access token", err)
		return
	}
	user, err := a.loadUser(r.Context(), userID)
	if err != nil {
		a.serverError(w, "load user", err)
		return
	}
	a.logger.Info("login verified", "user_id", userID)
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"user":         user,
	})
}

func (a *app) refreshHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !readJSON(w, r, &payload) {
		return
	}
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		a.serverError(w, "begin refresh transaction", err)
		return
	}
	defer tx.Rollback(r.Context())
	var sessionID, userID, email string
	err = tx.QueryRow(r.Context(), `
		SELECT s.id::text, u.id::text, u.email FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.refresh_token_hash = $1 AND s.revoked_at IS NULL AND s.expires_at > now()
		FOR UPDATE
	`, a.hashAuth(payload.RefreshToken)).Scan(&sessionID, &userID, &email)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_refresh_token"})
		return
	}
	if err != nil {
		a.serverError(w, "load auth session", err)
		return
	}
	newRefreshToken, err := randomToken(32)
	if err != nil {
		a.serverError(w, "generate refresh token", err)
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE auth_sessions SET revoked_at = now(), last_used_at = now() WHERE id = $1`, sessionID)
	if err == nil {
		_, err = tx.Exec(r.Context(), `INSERT INTO auth_sessions (user_id, refresh_token_hash, expires_at) VALUES ($1, $2, now() + interval '30 days')`, userID, a.hashAuth(newRefreshToken))
	}
	if err != nil {
		a.serverError(w, "rotate refresh token", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		a.serverError(w, "commit refresh", err)
		return
	}
	accessToken, err := a.createAccessToken(userID, email)
	if err != nil {
		a.serverError(w, "create access token", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"accessToken":  accessToken,
		"refreshToken": newRefreshToken,
	})
}

func (a *app) logoutHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		RefreshToken string `json:"refreshToken"`
	}
	if !readJSON(w, r, &payload) {
		return
	}
	_, err := a.db.Exec(r.Context(), `UPDATE auth_sessions SET revoked_at = now() WHERE refresh_token_hash = $1 AND revoked_at IS NULL`, a.hashAuth(payload.RefreshToken))
	if err != nil {
		a.serverError(w, "logout", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) topicsHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	classifiedByTopic, err := a.classifiedWordCounts(r.Context(), "topic")
	if err != nil {
		a.serverError(w, "load classified topic counts", err)
		return
	}
	classifiedByCEFR, err := a.classifiedWordCounts(r.Context(), "cefr")
	if err != nil {
		a.serverError(w, "load classified CEFR counts", err)
		return
	}
	rows, err := a.db.Query(r.Context(), `
        SELECT m.topic_slug, count(DISTINCT p.word_key)
        FROM user_word_progress p
        JOIN vocabulary_topic_memberships m ON m.word_id::text=p.word_key
        JOIN vocabulary_batches b ON b.id=m.batch_id AND b.slug=$2
        JOIN vocabulary_batch_words bw ON bw.batch_id=m.batch_id AND bw.word_id=m.word_id
        WHERE p.user_id=$1 AND bw.review_status IN ('needs_content_review','approved')
        GROUP BY m.topic_slug`, user.ID, a.config.vocabularyBatch)
	if err != nil {
		a.serverError(w, "load topic progress", err)
		return
	}
	defer rows.Close()
	progress := map[string]int{}
	for rows.Next() {
		var slug string
		var count int
		if err := rows.Scan(&slug, &count); err != nil {
			a.serverError(w, "scan topic progress", err)
			return
		}
		progress[slug] = count
	}
	availableByTopic, err := a.availableWordCountsByTopic(r.Context())
	if err != nil {
		a.serverError(w, "load available topic counts", err)
		return
	}
	topics := make([]map[string]any, 0, len(learningTopics))
	for _, item := range learningTopics {
		wordCount := availableByTopic[item.Slug]
		if wordCount == 0 {
			wordCount = len(staticWordsForTopic(item.Slug))
		}
		startedCount := min(progress[item.Slug], wordCount)
		topics = append(topics, map[string]any{
			"slug": item.Slug, "title": item.Title, "description": item.Description,
			"wordCount": wordCount, "startedCount": startedCount, "isAvailable": wordCount > 0,
			"classifiedWordCount": classifiedByTopic[item.Slug],
		})
	}
	cefrLevels := make([]map[string]any, 0, 6)
	for _, level := range []string{"A1", "A2", "B1", "B2", "C1", "C2"} {
		cefrLevels = append(cefrLevels, map[string]any{
			"level": level, "wordCount": classifiedByCEFR[level], "isAvailable": false,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"topics": topics, "cefrLevels": cefrLevels})
}

func (a *app) classifiedWordCounts(ctx context.Context, dimension string) (map[string]int, error) {
	rows, err := a.db.Query(ctx, `
		SELECT ll.slug, count(*)
		FROM word_labels wl
		JOIN learning_labels ll ON ll.id = wl.label_id
		JOIN learning_dimensions ld ON ld.id = ll.dimension_id
		JOIN word_learning_metadata wlm ON wlm.word_id = wl.word_id
		WHERE ld.slug = $1 AND wlm.is_learning_candidate
		GROUP BY ll.slug
	`, dimension)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	counts := map[string]int{}
	for rows.Next() {
		var slug string
		var count int
		if err := rows.Scan(&slug, &count); err != nil {
			return nil, err
		}
		counts[slug] = count
	}
	return counts, rows.Err()
}

// Return a small preview and the exact union count, rather than sending thousands
// of complete definitions to the category picker on every selection.
func (a *app) topicPreviewHandler(w http.ResponseWriter, r *http.Request) {
	slugs := strings.Split(r.URL.Query().Get("topics"), ",")
	if len(slugs) > 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "too_many_topics"})
		return
	}
	for _, slug := range slugs {
		if !topicExists(slug) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unknown_topic"})
			return
		}
	}
	for _, slug := range slugs {
		if slug == "general-vocabulary" {
			slugs = []string{slug}
			break
		}
	}
	words, err := a.wordsForTopics(r.Context(), slugs)
	if err != nil {
		a.serverError(w, "load category preview", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"wordCount": len(words), "words": words[:min(6, len(words))]})
}

func (a *app) topicWordsHandler(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	for _, topic := range learningTopics {
		if topic.Slug == slug {
			words, err := a.wordsForTopic(r.Context(), slug)
			if err != nil {
				a.serverError(w, "load topic words", err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"topic": map[string]any{"slug": topic.Slug, "title": topic.Title, "description": topic.Description, "wordCount": len(words)},
				"words": words,
			})
			return
		}
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "topic_not_found"})
}

func (a *app) knowledgeMapHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	rows, err := a.db.Query(r.Context(), `
		WITH batch_words AS (
			SELECT
				vbw.word_id,
				vbw.batch_rank,
				((vbw.batch_rank - 1) / 100) * 100 + 1 AS start_rank,
				((vbw.batch_rank - 1) / 100) * 100 + 100 AS end_rank,
				EXISTS (SELECT 1 FROM word_translations wt WHERE wt.word_id = vbw.word_id AND wt.language_code = 'en' AND btrim(wt.translation) <> '')
					AND vbw.review_status IN ('needs_content_review', 'approved') AS available
			FROM vocabulary_batches vb
			JOIN vocabulary_batch_words vbw ON vbw.batch_id = vb.id
			WHERE vb.slug = $2
		)
		SELECT
			bw.start_rank,
			bw.end_rank,
			count(*) AS total,
			count(*) FILTER (WHERE bw.available) AS available,
			count(*) FILTER (WHERE COALESCE(uwp.correct_count, 0) > 0) AS known,
			count(*) FILTER (WHERE COALESCE(uwp.correct_count, 0) = 0 AND COALESCE(uwp.wrong_count, 0) > 0) AS learning
		FROM batch_words bw
		LEFT JOIN user_word_progress uwp ON uwp.user_id = $1 AND uwp.word_key = bw.word_id::text
		GROUP BY bw.start_rank, bw.end_rank
		ORDER BY bw.start_rank
	`, user.ID, a.config.vocabularyBatch)
	if err != nil {
		a.serverError(w, "load knowledge map", err)
		return
	}
	defer rows.Close()

	type rangeSummary struct {
		StartRank int  `json:"startRank"`
		EndRank   int  `json:"endRank"`
		Total     int  `json:"total"`
		Available int  `json:"available"`
		Known     int  `json:"known"`
		Learning  int  `json:"learning"`
		Locked    bool `json:"locked"`
	}

	ranges := make([]rangeSummary, 0)
	totalWords := 0
	availableWords := 0
	knownWords := 0
	learningWords := 0
	for rows.Next() {
		var item rangeSummary
		if err := rows.Scan(&item.StartRank, &item.EndRank, &item.Total, &item.Available, &item.Known, &item.Learning); err != nil {
			a.serverError(w, "scan knowledge map", err)
			return
		}
		item.Locked = item.Available == 0
		ranges = append(ranges, item)
		totalWords += item.Total
		availableWords += item.Available
		knownWords += item.Known
		learningWords += item.Learning
	}
	if err := rows.Err(); err != nil {
		a.serverError(w, "iterate knowledge map", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"batchSlug":      a.config.vocabularyBatch,
		"rangeSize":      100,
		"totalWords":     totalWords,
		"availableWords": availableWords,
		"knownWords":     knownWords,
		"learningWords":  learningWords,
		"unseenWords":    max(0, availableWords-knownWords-learningWords),
		"ranges":         ranges,
	})
}

func (a *app) knowledgeMapRangeHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	startRank, err := strconv.Atoi(r.PathValue("start"))
	if err != nil || startRank < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_range_start"})
		return
	}
	endRank := startRank + 99

	rows, err := a.db.Query(r.Context(), `
		SELECT
			w.id::text,
			w.lemma,
			COALESCE(vbw.topic_slug, ''),
			vbw.batch_rank,
			COALESCE(vbw.cefr_level, ''),
			COALESCE(lc.translation, wt.translation, ''),
			COALESCE(lc.meanings, '[]'::jsonb),
			COALESCE(we.dutch_sentence, ''),
			COALESCE(we.english_sentence, ''),
			COALESCE(uwp.status, 'new')
		FROM vocabulary_batches vb
		JOIN vocabulary_batch_words vbw ON vbw.batch_id = vb.id
		JOIN words w ON w.id = vbw.word_id
		LEFT JOIN word_learning_content lc ON lc.batch_id=vb.id AND lc.word_id=w.id
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
		LEFT JOIN user_word_progress uwp ON uwp.user_id = $3 AND uwp.word_key = w.id::text
		WHERE vb.slug = $4
			AND vbw.batch_rank BETWEEN $1 AND $2
			AND vbw.review_status IN ('needs_content_review', 'approved')
			AND NULLIF(btrim(wt.translation), '') IS NOT NULL
		ORDER BY vbw.batch_rank
	`, startRank, endRank, user.ID, a.config.vocabularyBatch)
	if err != nil {
		a.serverError(w, "load knowledge range", err)
		return
	}
	defer rows.Close()

	type rangeWord struct {
		ID             string        `json:"id"`
		Word           string        `json:"word"`
		TopicSlug      string        `json:"topicSlug"`
		Topic          string        `json:"topic"`
		Rank           int           `json:"rank"`
		CEFRLevel      string        `json:"cefrLevel,omitempty"`
		Translation    string        `json:"translation"`
		Meanings       []wordMeaning `json:"meanings,omitempty"`
		Example        string        `json:"example"`
		EnglishExample string        `json:"englishExample"`
		Status         string        `json:"status"`
	}

	words := make([]rangeWord, 0)
	for rows.Next() {
		var word rangeWord
		if err := rows.Scan(&word.ID, &word.Word, &word.TopicSlug, &word.Rank, &word.CEFRLevel, &word.Translation, &word.Meanings, &word.Example, &word.EnglishExample, &word.Status); err != nil {
			a.serverError(w, "scan knowledge range", err)
			return
		}
		word.Topic = titleForTopic(word.TopicSlug)
		words = append(words, word)
	}
	if err := rows.Err(); err != nil {
		a.serverError(w, "iterate knowledge range", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"startRank": startRank,
		"endRank":   endRank,
		"words":     words,
	})
}

func (a *app) knowledgeMapMarkWordHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	wordID := strings.TrimSpace(r.PathValue("id"))
	if wordID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "word_id_required"})
		return
	}

	var payload struct {
		Action         string `json:"action"`
		ReviewInterval string `json:"reviewInterval"`
	}
	if !readJSON(w, r, &payload) {
		return
	}
	if !validReviewInterval(payload.ReviewInterval) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_review_interval"})
		return
	}
	action := strings.ToLower(strings.TrimSpace(payload.Action))
	if action != "already_knew" && action != "should_learn" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_action"})
		return
	}

	var topicSlug string
	if err := a.db.QueryRow(r.Context(), `
		SELECT COALESCE(vbw.topic_slug, '')
		FROM vocabulary_batches vb
		JOIN vocabulary_batch_words vbw ON vbw.batch_id = vb.id
		WHERE vb.slug = $2 AND vbw.word_id = $1
	`, wordID, a.config.vocabularyBatch).Scan(&topicSlug); err != nil {
		if err == pgx.ErrNoRows {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "word_not_found"})
			return
		}
		a.serverError(w, "load knowledge word", err)
		return
	}
	if topicSlug == "" {
		topicSlug = "daily-basics"
	}

	tx, err := a.db.Begin(r.Context())
	if err != nil {
		a.serverError(w, "begin mark word", err)
		return
	}
	defer tx.Rollback(r.Context())

	correct := action == "already_knew"
	if err := a.updateWordProgressWithTiming(r.Context(), tx, user.ID, topicSlug, wordID, correct, payload.ReviewInterval); err != nil {
		a.serverError(w, "mark knowledge word", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		a.serverError(w, "commit mark word", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":    "recorded",
		"wordId":    wordID,
		"topicSlug": topicSlug,
		"known":     correct,
	})
}

func (a *app) progressSummaryHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	var reviewedCount, masteredCount, dueCount, topicsStarted, strengthScore, dailyDone, dailyGoal int
	err := a.db.QueryRow(r.Context(), `
		SELECT count(*), count(*) FILTER (WHERE status = 'mastered'),
		       count(*) FILTER (WHERE NOT review_paused AND next_review_at IS NOT NULL AND next_review_at <= now()),
		       count(DISTINCT topic_slug), COALESCE(round(avg(strength)), 0)::int
		FROM user_word_progress WHERE user_id = $1
	`, user.ID).Scan(&reviewedCount, &masteredCount, &dueCount, &topicsStarted, &strengthScore)
	if err != nil {
		a.serverError(w, "load progress summary", err)
		return
	}
	err = a.db.QueryRow(r.Context(), `
        SELECT count(DISTINCT m.topic_slug) FROM user_word_progress p
        JOIN vocabulary_topic_memberships m ON m.word_id::text=p.word_key
        JOIN vocabulary_batches b ON b.id=m.batch_id AND b.slug=$2
        WHERE p.user_id=$1 AND m.topic_slug<>'general-vocabulary'
    `, user.ID, a.config.vocabularyBatch).Scan(&topicsStarted)
	if err != nil {
		a.serverError(w, "load active categories", err)
		return
	}
	// Both session answers and map practice update this per-word row in their
	// save transaction. Session events alone omit every map practice answer.
	err = a.db.QueryRow(r.Context(), `
		SELECT count(*) FROM user_word_progress
		WHERE user_id = $1 AND last_seen_at >= date_trunc('day', now())
		  AND correct_count + wrong_count > 0
	`, user.ID).Scan(&dailyDone)
	if err != nil {
		a.serverError(w, "load daily progress", err)
		return
	}
	if err := a.db.QueryRow(r.Context(), `SELECT COALESCE((SELECT daily_goal FROM user_profiles WHERE user_id = $1), 10)`, user.ID).Scan(&dailyGoal); err != nil {
		a.serverError(w, "load daily goal", err)
		return
	}
	totalWords, err := a.totalAvailableWordCount(r.Context())
	if err != nil {
		a.serverError(w, "load available word count", err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"totalWords":    totalWords,
		"reviewedWords": reviewedCount,
		"masteredWords": masteredCount,
		"dueWords":      dueCount,
		"topicsTotal":   len(learningTopics) - 1, // General is the all-words collection.
		"topicsStarted": topicsStarted,
		"dailyGoal":     dailyGoal,
		"dailyDone":     dailyDone,
		"strengthScore": strengthScore,
	})
}

func (a *app) profileHandler(w http.ResponseWriter, r *http.Request) {
	var payload userProfile
	if !readJSON(w, r, &payload) {
		return
	}
	payload.DisplayName = strings.TrimSpace(payload.DisplayName)
	payload.AgeRange = strings.TrimSpace(payload.AgeRange)
	if payload.DisplayName == "" || payload.AgeRange == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "display_name_and_age_range_required"})
		return
	}
	user := currentUser(r.Context())
	_, err := a.db.Exec(r.Context(), `
		INSERT INTO user_profiles (user_id, display_name, age_range) VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET display_name = EXCLUDED.display_name, age_range = EXCLUDED.age_range, updated_at = now()
	`, user.ID, payload.DisplayName, payload.AgeRange)
	if err != nil {
		a.serverError(w, "save profile", err)
		return
	}
	payload.Email = user.Email
	a.logger.Info("profile saved", "user_id", user.ID)
	writeJSON(w, http.StatusOK, map[string]any{"status": "saved", "profile": payload})
}

func (a *app) nextSessionHandler(w http.ResponseWriter, r *http.Request) {
	topicSlug := r.URL.Query().Get("topic")
	if topicSlug == "" {
		topicSlug = "daily-basics"
	}
	words, err := a.wordsForTopic(r.Context(), topicSlug)
	if err != nil {
		a.serverError(w, "load topic words", err)
		return
	}
	if !topicExists(topicSlug) || len(words) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "topic_not_found"})
		return
	}
	a.createLearningSession(w, r, []string{topicSlug}, []string{}, "balanced")
}

func (a *app) createSessionHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		TopicSlugs []string `json:"topicSlugs"`
		CEFRLevels []string `json:"cefrLevels"`
		Mode       string   `json:"mode"`
	}
	if !readJSON(w, r, &payload) {
		return
	}
	var err error
	payload.TopicSlugs, err = a.validTopicSlugs(r.Context(), payload.TopicSlugs)
	if err != nil {
		a.serverError(w, "validate topic slugs", err)
		return
	}
	if len(payload.TopicSlugs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "at_least_one_available_topic_required"})
		return
	}
	if len(payload.TopicSlugs) > 3 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "maximum_three_topics"})
		return
	}
	if len(payload.CEFRLevels) > 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "cefr_classification_not_available"})
		return
	}
	if payload.Mode == "" {
		payload.Mode = "balanced"
	}
	if payload.Mode != "balanced" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "ranking_mode_not_available"})
		return
	}
	if payload.CEFRLevels == nil {
		payload.CEFRLevels = []string{}
	}
	a.createLearningSession(w, r, payload.TopicSlugs, payload.CEFRLevels, payload.Mode)
}

func (a *app) createLearningSession(w http.ResponseWriter, r *http.Request, topicSlugs, cefrLevels []string, mode string) {
	w.Header().Set("Cache-Control", "no-store")
	user := currentUser(r.Context())
	pool, err := a.wordsForTopics(r.Context(), topicSlugs)
	if err != nil {
		a.serverError(w, "load session words", err)
		return
	}
	words, err := a.nextPracticeWords(r.Context(), user.ID, pool)
	if err != nil {
		a.serverError(w, "select session words", err)
		return
	}
	questionCount := len(words)
	if questionCount == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"questions": []sessionQuestion{}, "status": "up_to_date"})
		return
	}
	questions := make([]sessionQuestion, 0, questionCount)
	for i, word := range words[:questionCount] {
		questions = append(questions, buildQuestion(i, word, pool))
	}

	snapshot, err := encodeSessionQuestions(questions)
	if err != nil {
		a.serverError(w, "encode session questions", err)
		return
	}
	session := &learningSession{
		TopicSlug:  topicSlugs[0],
		TopicSlugs: topicSlugs,
		CEFRLevels: cefrLevels,
		Mode:       mode,
		Title:      titleForTopics(topicSlugs),
		Questions:  questions,
		StartedAt:  time.Now(),
	}
	if err := a.db.QueryRow(r.Context(), `
		INSERT INTO learning_sessions (user_id, topic_slug, topic_slugs, cefr_levels, mode, question_count, vocabulary_batch, question_snapshot)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id::text, started_at
	`, user.ID, topicSlugs[0], topicSlugs, cefrLevels, mode, len(questions), a.config.vocabularyBatch, snapshot).Scan(&session.ID, &session.StartedAt); err != nil {
		a.serverError(w, "create learning session", err)
		return
	}
	writeJSON(w, http.StatusOK, session)
}

func (a *app) sessionAnswerHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.PathValue("id")
	var payload struct {
		QuestionID     string `json:"questionId"`
		WordID         string `json:"wordId"`
		Answer         string `json:"answer"`
		ReviewInterval string `json:"reviewInterval"`
	}
	if !readJSON(w, r, &payload) {
		return
	}
	if !validReviewInterval(payload.ReviewInterval) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_review_interval"})
		return
	}

	user := currentUser(r.Context())
	var topicSlugs []string
	var totalCount int
	var batchSlug string
	var snapshot []byte
	err := a.db.QueryRow(r.Context(), `SELECT topic_slugs, question_count, vocabulary_batch, question_snapshot FROM learning_sessions WHERE id = $1 AND user_id = $2`, sessionID, user.ID).Scan(&topicSlugs, &totalCount, &batchSlug, &snapshot)
	if err == pgx.ErrNoRows {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "session_not_found"})
		return
	}
	if err != nil {
		a.serverError(w, "load learning session", err)
		return
	}
	var questions []sessionQuestion
	if snapshot != nil {
		questions, err = decodeSessionQuestions(snapshot, totalCount)
		if err != nil {
			a.serverError(w, "decode session questions", err)
			return
		}
	} else {
		// Sessions created before snapshots used a fixed first-ten selection.
		// Reconstruct only that legacy selection, never apply current progress.
		sessionApp := *a
		sessionApp.config.vocabularyBatch = batchSlug
		sessionApp.originalVocabularyContent = true
		words, err := sessionApp.wordsForTopics(r.Context(), topicSlugs)
		if err != nil {
			a.serverError(w, "load legacy session words", err)
			return
		}
		if totalCount > len(words) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "session_content_changed"})
			return
		}
		for i, word := range words[:totalCount] {
			questions = append(questions, buildQuestion(i, word, words))
		}
	}
	question, found := findSessionQuestion(&learningSession{Questions: questions}, payload.QuestionID)
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "question_not_found"})
		return
	}
	if payload.WordID != "" && payload.WordID != question.WordID {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "question_word_mismatch"})
		return
	}
	isCorrect := isAnswerCorrect(question, payload.Answer)
	wordID := question.WordID
	tx, err := a.db.Begin(r.Context())
	if err != nil {
		a.serverError(w, "begin answer transaction", err)
		return
	}
	defer tx.Rollback(r.Context())
	result, err := tx.Exec(r.Context(), `
		INSERT INTO learning_events (session_id, user_id, question_id, word_key, exercise_type, answer, correct_answer, was_correct, review_interval)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (session_id, question_id) DO NOTHING
	`, sessionID, user.ID, question.ID, wordID, question.Type, payload.Answer, question.CorrectAnswer, isCorrect, payload.ReviewInterval)
	if err != nil {
		a.serverError(w, "store learning event", err)
		return
	}
	if result.RowsAffected() > 0 {
		if err := a.updateWordProgressWithTiming(r.Context(), tx, user.ID, question.TopicSlug, wordID, isCorrect, payload.ReviewInterval); err != nil {
			a.serverError(w, "update word progress", err)
			return
		}
	}
	var answeredCount, correctCount int
	if err := tx.QueryRow(r.Context(), `SELECT count(*), count(*) FILTER (WHERE was_correct) FROM learning_events WHERE session_id = $1`, sessionID).Scan(&answeredCount, &correctCount); err != nil {
		a.serverError(w, "count session answers", err)
		return
	}
	_, err = tx.Exec(r.Context(), `UPDATE learning_sessions SET correct_count = $2, completed_at = CASE WHEN $3 >= question_count THEN COALESCE(completed_at, now()) ELSE completed_at END WHERE id = $1`, sessionID, correctCount, answeredCount)
	if err != nil {
		a.serverError(w, "update learning session", err)
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		a.serverError(w, "commit learning answer", err)
		return
	}

	a.logger.Info("dev session answer", "session_id", sessionID, "question_id", payload.QuestionID, "word_id", wordID, "correct", isCorrect)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":        "recorded",
		"answeredCount": answeredCount,
		"correctCount":  correctCount,
		"totalCount":    totalCount,
		"isComplete":    answeredCount >= totalCount,
		"isCorrect":     isCorrect,
		"correctAnswer": question.CorrectAnswer,
	})
}

func (a *app) meHandler(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r.Context())
	loaded, err := a.loadUser(r.Context(), user.ID)
	if err != nil {
		a.serverError(w, "load current user", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user": loaded})
}

func (a *app) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "private, no-store")
		header := r.Header.Get("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication_required"})
			return
		}
		user, err := a.parseAccessToken(strings.TrimPrefix(header, "Bearer "))
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid_access_token"})
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userContextKey, user)))
	})
}

func currentUser(ctx context.Context) authUser {
	user, _ := ctx.Value(userContextKey).(authUser)
	return user
}

func (a *app) loadUser(ctx context.Context, userID string) (authUser, error) {
	var user authUser
	var displayName, ageRange *string
	err := a.db.QueryRow(ctx, `
		SELECT u.id::text, u.email, p.display_name, p.age_range
		FROM users u LEFT JOIN user_profiles p ON p.user_id = u.id WHERE u.id = $1
	`, userID).Scan(&user.ID, &user.Email, &displayName, &ageRange)
	if err != nil {
		return authUser{}, err
	}
	if displayName != nil {
		user.DisplayName = *displayName
		user.ProfileComplete = true
	}
	if ageRange != nil {
		user.AgeRange = *ageRange
	}
	return user, nil
}

func (a *app) createAccessToken(userID, email string) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
	if err != nil {
		return "", err
	}
	claims, err := json.Marshal(map[string]any{
		"sub":   userID,
		"email": email,
		"iat":   time.Now().Unix(),
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
	})
	if err != nil {
		return "", err
	}
	unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(claims)
	mac := hmac.New(sha256.New, []byte(a.config.authSecret))
	_, _ = mac.Write([]byte(unsigned))
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (a *app) parseAccessToken(token string) (authUser, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return authUser{}, fmt.Errorf("malformed token")
	}
	unsigned := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(a.config.authSecret))
	_, _ = mac.Write([]byte(unsigned))
	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(signature, mac.Sum(nil)) {
		return authUser{}, fmt.Errorf("invalid signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return authUser{}, err
	}
	var claims struct {
		Subject string `json:"sub"`
		Email   string `json:"email"`
		Expires int64  `json:"exp"`
	}
	if err := json.Unmarshal(payload, &claims); err != nil {
		return authUser{}, err
	}
	if claims.Subject == "" || claims.Expires <= time.Now().Unix() {
		return authUser{}, fmt.Errorf("expired token")
	}
	return authUser{ID: claims.Subject, Email: claims.Email}, nil
}

func (a *app) hashAuth(value string) string {
	mac := hmac.New(sha256.New, []byte(a.config.authSecret))
	_, _ = mac.Write([]byte(value))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomToken(size int) (string, error) {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(value), nil
}

func (a *app) sendLoginCode(email, code string) error {
	message := []byte("From: " + a.config.emailFrom + "\r\n" +
		"To: " + email + "\r\n" +
		"Subject: Your Woordenboom login code\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n\r\n" +
		"Your Woordenboom code is " + code + ". It expires in 10 minutes.\r\n")
	var auth smtp.Auth
	if a.config.smtpUser != "" {
		host, _, err := net.SplitHostPort(a.config.smtpAddr)
		if err != nil {
			return fmt.Errorf("parse SMTP address: %w", err)
		}
		auth = smtp.PlainAuth("", a.config.smtpUser, a.config.smtpPass, host)
	}
	return smtp.SendMail(a.config.smtpAddr, auth, a.config.emailFrom, []string{email}, message)
}

func (a *app) updateWordProgress(ctx context.Context, tx pgx.Tx, userID, topicSlug, wordID string, correct bool) error {
	return a.updateWordProgressWithTiming(ctx, tx, userID, topicSlug, wordID, correct, "auto")
}

func topicExists(slug string) bool {
	for _, topic := range learningTopics {
		if topic.Slug == slug {
			return true
		}
	}
	return false
}

func (a *app) serverError(w http.ResponseWriter, operation string, err error) {
	a.logger.Error(operation, "error", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_server_error"})
}

func requestLogger(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Info("request", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func readJSON(w http.ResponseWriter, r *http.Request, payload any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		slog.Error("write response", "error", err)
	}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func hashSecret(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

func (a *app) availableWordCountsByTopic(ctx context.Context) (map[string]int, error) {
	rows, err := a.db.Query(ctx, `
		SELECT m.topic_slug, count(DISTINCT vbw.word_id)
		FROM vocabulary_batches vb
		JOIN vocabulary_batch_words vbw ON vbw.batch_id = vb.id
        JOIN vocabulary_topic_memberships m ON m.batch_id=vb.id AND m.word_id=vbw.word_id
		WHERE vb.slug = $1
			AND m.topic_slug IS NOT NULL
			AND vbw.review_status IN ('needs_content_review', 'approved')
			AND EXISTS (SELECT 1 FROM word_translations wt WHERE wt.word_id = vbw.word_id AND wt.language_code = 'en' AND btrim(wt.translation) <> '')
		GROUP BY m.topic_slug
	`, a.config.vocabularyBatch)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := map[string]int{}
	for rows.Next() {
		var slug string
		var count int
		if err := rows.Scan(&slug, &count); err != nil {
			return nil, err
		}
		counts[slug] = count
	}
	return counts, rows.Err()
}

func (a *app) wordsForTopic(ctx context.Context, slug string) ([]reviewWord, error) {
	rows, err := a.db.Query(ctx, `
		SELECT w.id::text, w.lemma, COALESCE(m.translation, lc.translation, wt.translation, ''), CASE WHEN m.translation IS NULL THEN COALESCE(we.dutch_sentence, '') ELSE '' END, COALESCE(m.meanings, lc.meanings, '[]'::jsonb)
		FROM vocabulary_batches vb
		JOIN vocabulary_batch_words vbw ON vbw.batch_id = vb.id
		JOIN words w ON w.id = vbw.word_id
        LEFT JOIN vocabulary_topic_memberships m ON m.batch_id=vb.id AND m.word_id=w.id AND m.topic_slug=$1 AND NOT $3
		LEFT JOIN word_learning_content lc ON lc.batch_id=vb.id AND lc.word_id=w.id AND NOT $3
		LEFT JOIN LATERAL (
			SELECT translation FROM word_translations
			WHERE word_id = w.id AND language_code = 'en'
			ORDER BY CASE WHEN source = 'woordenboom-generated-v1' THEN 0 ELSE 1 END, sort_order, created_at
			LIMIT 1
		) wt ON true
		LEFT JOIN LATERAL (
			SELECT dutch_sentence FROM word_examples
			WHERE word_id = w.id
			ORDER BY CASE WHEN source = 'woordenboom-generated-v1' THEN 0 ELSE 1 END, sort_order, created_at
			LIMIT 1
		) we ON true
		WHERE vb.slug = $2
			AND (($3 AND vbw.topic_slug=$1) OR (NOT $3 AND m.word_id IS NOT NULL))
			AND vbw.review_status IN ('needs_content_review', 'approved')
			AND NULLIF(btrim(wt.translation), '') IS NOT NULL
		ORDER BY CASE WHEN $1='general-vocabulary' AND NOT $3 THEN vbw.batch_rank ELSE vbw.topic_rank END NULLS LAST, vbw.batch_rank
	`, slug, a.config.vocabularyBatch, a.originalVocabularyContent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	words := make([]reviewWord, 0)
	for rows.Next() {
		var word reviewWord
		if err := rows.Scan(&word.ID, &word.Word, &word.Translation, &word.Example, &word.Meanings); err != nil {
			return nil, err
		}
		word.TopicSlug = slug
		word.Topic = titleForTopic(slug)
		words = append(words, word)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(words) > 0 {
		return words, nil
	}
	return staticWordsForTopic(slug), nil
}

func staticWordsForTopic(slug string) []reviewWord {
	var source []reviewWord
	if slug == "daily-basics" {
		source = commonDutchWords
	} else {
		source = topicSeeds[slug]
	}
	words := make([]reviewWord, 0, len(source))
	for _, sourceWord := range source {
		word := sourceWord
		word.TopicSlug = slug
		word.Topic = titleForTopic(slug)
		words = append(words, word)
	}
	return words
}

func (a *app) wordsForTopics(ctx context.Context, slugs []string) ([]reviewWord, error) {
	buckets := make([][]reviewWord, 0, len(slugs))
	longest := 0
	for _, slug := range slugs {
		words, err := a.wordsForTopic(ctx, slug)
		if err != nil {
			return nil, err
		}
		buckets = append(buckets, words)
		longest = max(longest, len(words))
	}
	result := make([]reviewWord, 0)
	seen := map[string]bool{}
	for index := 0; index < longest; index++ {
		for _, words := range buckets {
			if index >= len(words) || seen[words[index].ID] {
				continue
			}
			seen[words[index].ID] = true
			result = append(result, words[index])
		}
	}
	return result, nil
}

func (a *app) validTopicSlugs(ctx context.Context, slugs []string) ([]string, error) {
	// The all-words collection supersedes narrower categories.
	for _, slug := range slugs {
		if slug == "general-vocabulary" {
			slugs = []string{slug}
			break
		}
	}
	result := make([]string, 0, len(slugs))
	seen := map[string]bool{}
	for _, slug := range slugs {
		words, err := a.wordsForTopic(ctx, slug)
		if err != nil {
			return nil, err
		}
		if seen[slug] || !topicExists(slug) || len(words) == 0 {
			continue
		}
		seen[slug] = true
		result = append(result, slug)
	}
	return result, nil
}

func titleForTopics(slugs []string) string {
	titles := make([]string, 0, len(slugs))
	for _, slug := range slugs {
		titles = append(titles, titleForTopic(slug))
	}
	return strings.Join(titles, " + ")
}

func (a *app) totalAvailableWordCount(ctx context.Context) (int, error) {
	counts, err := a.availableWordCountsByTopic(ctx)
	if err != nil {
		return 0, err
	}
	if counts["general-vocabulary"] > 0 {
		return counts["general-vocabulary"], nil
	}
	seen := map[string]bool{}
	for _, item := range learningTopics {
		for _, word := range staticWordsForTopic(item.Slug) {
			seen[word.ID] = true
		}
	}
	return len(seen), nil
}

func titleForTopic(slug string) string {
	for _, topic := range learningTopics {
		if topic.Slug == slug {
			return topic.Title
		}
	}
	return slug
}

func buildQuestion(index int, word reviewWord, pool []reviewWord) sessionQuestion {
	exerciseTypes := []string{"reveal"}
	exerciseType := exerciseTypes[index%len(exerciseTypes)]
	question := sessionQuestion{
		ID:            fmt.Sprintf("q_%02d_%s", index+1, word.ID),
		WordID:        word.ID,
		TopicSlug:     word.TopicSlug,
		Type:          exerciseType,
		Topic:         word.Topic,
		Word:          word.Word,
		Pronunciation: word.Pronunciation,
		Example:       word.Example,
		Translation:   word.Translation,
		Meanings:      word.Meanings,
		CorrectAnswer: word.Translation,
	}

	switch exerciseType {
	case "multiple_choice":
		question.Prompt = "What does this Dutch word mean?"
		question.Options = answerOptions(word, pool)
	case "reverse_recall":
		question.Prompt = "Which Dutch word matches this meaning?"
		question.Word = word.Translation
		question.Options = wordOptions(word, pool)
		question.CorrectAnswer = word.Word
	case "sentence_gap":
		question.Prompt = "Which word completes the sentence?"
		question.Example = strings.Replace(word.Example, word.Word, "____", 1)
		question.Options = wordOptions(word, pool)
		question.CorrectAnswer = word.Word
	default:
		question.Prompt = "Study this word, then rate your memory."
	}

	return question
}

func findSessionQuestion(session *learningSession, questionID string) (sessionQuestion, bool) {
	for _, question := range session.Questions {
		if question.ID == questionID {
			return question, true
		}
	}
	return sessionQuestion{}, false
}

func isAnswerCorrect(question sessionQuestion, answer string) bool {
	normalizedAnswer := strings.ToLower(strings.TrimSpace(answer))
	if normalizedAnswer == "already knew" {
		return true
	}
	if normalizedAnswer == "should learn" {
		return false
	}
	if question.Type == "reveal" {
		return normalizedAnswer == "hard" || normalizedAnswer == "good" || normalizedAnswer == "easy"
	}
	return normalizedAnswer == strings.ToLower(strings.TrimSpace(question.CorrectAnswer))
}

func answerOptions(correct reviewWord, pool []reviewWord) []string {
	options := []string{correct.Translation}
	for _, candidate := range pool {
		if candidate.ID != correct.ID && candidate.Translation != correct.Translation {
			options = append(options, candidate.Translation)
		}
		if len(options) == 4 {
			return options
		}
	}
	return options
}

func wordOptions(correct reviewWord, pool []reviewWord) []string {
	options := []string{correct.Word}
	for _, candidate := range pool {
		if candidate.ID != correct.ID && candidate.Word != correct.Word {
			options = append(options, candidate.Word)
		}
		if len(options) == 4 {
			return options
		}
	}
	return options
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
