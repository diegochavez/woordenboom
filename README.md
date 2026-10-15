# Woordenboom

Dutch vocabulary learning app for English-speaking expats.

## Product Goal

Build a mobile-first Dutch vocabulary app inspired by WordUp, focused on practical Dutch, topic-based mastery, and spaced repetition.

## Target Audience

English-speaking expats living in the Netherlands or Belgium.

## MVP Scope

- React Native mobile app with Expo
- Go backend API
- Postgres database
- Docker Compose local development
- Passwordless email code login
- 9,500 draft Dutch words for the private beta; editorial review remains in progress
- English translations
- Topic packs for expat life
- Spaced repetition review system
- User progress tracking

## Deferred

- Admin panel
- Magic links
- OAuth
- Payments
- Social features
- AI chat
- Advanced audio generation
- Full offline sync

## Stack

- Mobile: React Native, Expo, TypeScript
- Backend: Go
- Database: Postgres
- Local email: Mailpit
- Production email: authenticated SMTP on port 587 (Amazon SES is the current example)
- Dutch word source: OpenTaal word list 2.20.23 under BSD-3-Clause
- Local infra: Docker Compose
- Beta hosting plan: Hetzner Cloud with Go, PostgreSQL, Caddy, and external SMTP ([details](docs/launch-plan.md))
- Beta API delivery: GitHub Actions verification and opt-in Hetzner deployment ([setup](docs/deployment-pipeline.md))

## Initial Topic Packs

- Daily Basics
- Housing & Renting
- Supermarket & Food
- Public Transport
- Healthcare
- Government & Municipality
- Immigration & Documents
- Work & Office
- Banking & Money
- Social Life
- Weather & Small Talk
- Home Repair
- Emergencies
- School & Childcare
- Politics & News
- Engineering & Technical Work
- Software & IT
- Legal & Contracts
- Utilities & Bills
- Appointments & Phone Calls

## MVP Tasks

### Phase 1: Foundation

- [x] Initialize git repository
- [x] Create monorepo structure
- [x] Add Docker Compose
- [x] Add Postgres service
- [x] Add Mailpit service
- [x] Add Go API skeleton
- [x] Add Expo React Native app
- [x] Add environment configuration

### Phase 2: Auth

- [x] Add users table
- [x] Add auth email codes table
- [x] Add auth sessions table
- [x] Implement `POST /auth/request-code`
- [x] Implement `POST /auth/verify-code`
- [x] Implement `POST /auth/refresh`
- [x] Implement `POST /auth/logout`
- [x] Implement `GET /me`
- [x] Store hashed email codes only
- [x] Add 10-minute code expiry
- [x] Add 5-attempt verification limit
- [x] Add refresh token rotation
- [x] Add Expo SecureStore token storage

### Phase 3: Vocabulary Data

- [x] Choose the initial Dutch spelling source
- [x] Import the complete 413,937-entry OpenTaal word list
- [x] Match all 9,500 release lemmas to imported OpenTaal words
- [x] Add English translations for 9,500 unique Dutch words
- [ ] Extend example sentences beyond the existing 190 bilingual pairs
- [x] Preserve dictionary part-of-speech and sense metadata where available
- [ ] Finish topic review: 9,048 release entries still need review
- [x] Store source/license metadata and per-entry dictionary links

### Phase 4: Vocabulary API

- [x] Add words table
- [x] Add word translations table
- [x] Add word examples table
- [x] Add topics table
- [x] Add word topics table
- [ ] Implement word list endpoint
- [ ] Implement word detail endpoint
- [x] Implement topic list endpoint
- [x] Implement topic words endpoint

### Phase 5: Spaced Repetition

- [x] Add user word state table
- [x] Add review events table
- [ ] Implement SM-2 style scheduler
- [x] Implement due reviews endpoint
- [x] Implement review answer endpoint
- [ ] Track status, ease factor, interval, repetitions, lapses, and due date

### Phase 6: Mobile MVP

- [x] Build welcome screen
- [x] Build email login screen
- [x] Build code verification screen
- [x] Build home dashboard
- [x] Build topic selection screen
- [ ] Build topic progress screen
- [ ] Build learn new words screen
- [x] Build review card screen
- [ ] Build word detail screen
- [ ] Build profile/settings screen

### Phase 7: Polish

- [ ] Add streaks
- [ ] Add daily goal
- [ ] Add loading states
- [ ] Add error states
- [ ] Add empty states
- [ ] Add basic analytics events
- [ ] Add local caching strategy
- [x] Prepare Hetzner deployment plan

## Auth Decision

Use email code login only for MVP.

Magic links, OAuth, and passkeys are deferred.

## Email Code Defaults

- 6-digit numeric code
- 10-minute expiry
- 5 max verification attempts
- 1 active code per email
- Short-lived access token, around 15 minutes
- Refresh token expiry, around 30 days
- Refresh token rotation

## Review Buttons

- Again
- Hard
- Good
- Easy

## Local Development Goal

Run backend dependencies locally with:

```bash
docker compose up
```

Run mobile with:

```bash
cd apps/mobile
npx expo start
```

## Current Project Structure

```txt
apps/mobile       Expo React Native app
services/api      Go API service
db/migrations     Postgres migrations
data/seeds        Future vocabulary seed data
THIRD_PARTY_NOTICES.md  Data attribution and license notices
```

## Verification

Current checks:

```bash
docker compose config
docker compose build api
sh scripts/smoke-test.sh
cd apps/mobile && npx tsc --noEmit
cd apps/mobile && npx expo lint
cd apps/mobile && npx expo-doctor
```

## Vocabulary import

The OpenTaal importer downloads a pinned copy of `wordlist.txt`, checks its SHA-256, and imports all 413,937 entries with source metadata. It is safe to run again.

```bash
docker compose up --build -d
docker compose run --rm api /app/import-opentaal
```

Classify imported words with:

```bash
docker compose run --rm api /app/classify-vocabulary
```

The current classifier is deterministic and conservative. It assigns entry type, rough CEFR level, topic labels, and learner-priority scores for all imported words. It preserves labels produced by later classifiers such as TypeSafe/Jev.

Import Dutch frequency evidence from `wordfreq` by exporting candidate lemmas to TSV and piping them into the importer:

```bash
docker compose exec -T postgres psql -U woordenboom -d woordenboom -tA -c "SELECT lemma FROM candidate_word_scores WHERE is_learning_candidate ORDER BY candidate_score DESC, lemma LIMIT 50000" \
  | docker run --rm -i -v "$PWD":/src -w /src python:3.12-slim sh -c 'pip install -q wordfreq && python scripts/export-wordfreq-nl.py' \
  | docker compose run --rm -T api /app/import-wordfreq --input -
```

Generate a ranked candidate report with:

```bash
docker compose run --rm api /app/report-candidates --limit 40 --with-frequency=true --exclude-topic daily-basics
```

Frequency evidence improves candidate ordering but does not provide definitions, translations, examples, or final learning suitability by itself.

The original 2,000-word candidate workspace is retained for reference and existing sessions. To create a separate candidate batch, use a new slug with:

```bash
docker compose run --rm api /app/generate-word-batch --slug release-2000 --title "Release 2,000 Candidate Vocabulary" --size 2000 --per-topic 100 --daily-limit 350 --topic-min-zipf 3.0
```

Export the full batch, or one category at a time, with:

```bash
docker compose run --rm api /app/export-word-batch --slug release-2000 > data/exports/release-2000/all.csv
docker compose run --rm api /app/export-word-batch --slug release-2000 --topic healthcare > data/exports/release-2000/healthcare.csv
docker compose run --rm api /app/export-word-batch --slug release-2000 --topic unassigned > data/exports/release-2000/unassigned.csv
```

The exported CSV includes empty `english_translation`, `dutch_example`, and `english_example` columns for batch content work. Words marked `needs_topic_review` should be assigned or rejected before publishing.

Import completed or generated content CSVs back into the database with:

```bash
docker compose run --rm -T api /app/import-word-content --batch release-2000 --status needs_content_review --input - < data/content/release-2000/practical-core.csv
```

The importer replaces prior `woordenboom-generated-v1` translation/example rows for the imported words, then marks those batch words as `needs_content_review` or `approved`. Use `needs_content_review` for AI-generated content until it has been checked.

Prepare a full 2,000-word translation workspace from the current release batch with:

```bash
docker run --rm -v "$PWD":/src -w /src/services/api golang:1.23-alpine \
  go run ./cmd/prepare-content-workqueue \
  --input ../../data/exports/release-2000/all.csv \
  --output-dir ../../data/content/release-2000/workqueue
```

The generated workspace reserves all 2,000 release words and creates:

- `all-work-queue.csv` for the complete release review/translation queue.
- `topic-review.csv` for words that need topic assignment or rejection before translation.
- `<topic>.csv` files that are import-ready once translations and examples are filled in.

Topic-reviewed and generated rows should still be imported as `needs_content_review` until checked.

OpenTaal provides spelling data rather than definitions, frequency ranks, or translations. Imported words are kept out of lessons until the categorization process marks them as suitable and adds learning metadata.

The selected BSD-3-Clause attribution and disclaimer are available in `THIRD_PARTY_NOTICES.md` and on the mobile app's Data sources and licenses page.

## Import the 9,500-word translated release

Python 3, Docker Compose, and the existing OpenTaal word import are required. No translation service or API key is used.

```bash
docker compose up -d postgres
docker compose run --rm migrate
python3 scripts/vocabulary_release.py validate
python3 scripts/vocabulary_release.py import
docker compose up -d --build api
```

The importer reads the checked-in dataset, verifies the checksum and all 9,500
rows, then imports everything in one transaction. Repeat imports add no duplicate
translations, examples, or batch memberships. It preserves existing authored
content, older batches, review decisions, and user progress. Set
`VOCABULARY_BATCH=release-2000` to use the previous release.

The dataset and its source archive carry CC BY-SA 3.0 attribution in
`THIRD_PARTY_NOTICES.md` and the app's Data and licenses screen.

## Practice selection

Practice sessions use each learner's saved progress. Unseen words retain their
frequency order and the selected categories remain interleaved. Up to five of a
ten-word stack are due reviews when there are enough unseen words; unused places
are filled from the other group. Recently answered words wait at least ten
minutes. A small remaining set produces a shorter stack. Future-dated and paused
words never fill a practice stack early. When none are due and no unseen words
remain, the app shows “You’re up to date”.

Every new session stores its exact questions and grading fields. Answer handling
uses that snapshot, so recording progress or changing the active vocabulary
release cannot change a session already in progress. Sessions created before
migration 13 retain the legacy first-ten behavior until completed. Mobile creates
sessions with POST, including single-category practice and “Practice again”.
Session responses use `Cache-Control: no-store`.

## Dutch definitions and translation review

Practice and knowledge-map cards now reveal Dutch definitions alongside the
English translation. The batch-scoped content overlay covers 7,675 words,
including all first 300 ranked words and all 190 original lesson words.
58 translation strings were updated, including 39 changes to glosses.
1,825 words still lack a suitable definition and retain translation-only cards.
All content remains pending native-speaker review.

Apply migration 15, then run `python3 scripts/word_definitions.py import` before
rebuilding the API. Existing progress and saved session wording are preserved.
See the [content review](docs/content-review/review.md) and
[build/import instructions](data/content/definitions-v1/README.md).

## Review timing

After revealing a word, the card shows successful recalls and the proposed next
review. “Know” records one recall with the recommended timing. “Change” opens a
monochrome sheet with 10 minutes; 1 or 3 days; 1 or 2 weeks; 1, 2, 3, or 6 months;
1 year; or Pause reviews. Confirming
the sheet records the recall, schedules it, and advances. Cancel makes no change.
“Not yet” resets the recall streak and schedules a review in ten minutes.

The default interval ladder is 1, 3, 7, 14, 30, 60, 90, 180, then 365 days, based
on consecutive successful recalls. Further successful recalls stay at 365 days. This is a simple scheduling policy, not a calibrated memory
model. Intervals are elapsed durations from the saved answer, so “In 1 day” means
24 hours. Month choices mean 30, 60, 90, or 180 days; a year means 365 days.
Existing saved due dates are unchanged. A paused word stays on the knowledge map; explicitly choose an interval
there to resume. Session answer retries do not add recalls or change a saved date.

`GET /words/{id}/review-timing` returns the user's recall count, recommendation,
next review timestamp, and paused state. Session answers and knowledge-map marks
accept `reviewInterval`: `auto`, `10m`, `1d`, `2d`, `3d`, `7d`, `14d`, `30d`, `60d`, `90d`, `180d`, `365d`, or
`paused`. The legacy `2d` choice remains accepted but is no longer a default step.
Older clients may omit the field. Apply migration 14 before updating
the API. No notifications, images, or audio are involved.

## Current Dev Flow

Authentication, profiles, sessions, answers, progress, and vocabulary are persisted in Postgres. Docker Compose serves `release-9500`: 9,500 unique Dutch words with English translations across 95 knowledge-map ranges. The original 190 bilingual example pairs are preserved. Examples are optional for practice. Dictionary senses, topic assignments, and learning suitability still need human review; this is a broad vocabulary collection, not a certified 9,500-word beginner syllabus. General vocabulary includes all 9,500 words. Practice also supports mixes of up to three specific categories, deduplicated by word ID. Existing sessions retain their original vocabulary batch. See [the release notes](data/content/release-9500/README.md) for counts, sources, import steps, and remaining editorial tasks.

Email code test flow:

```bash
open http://localhost:8025
```

Run the complete authenticated API flow with:

```bash
sh scripts/smoke-test.sh
```

## Prototype Implementation Order

1. Local Docker setup with Postgres and Mailpit
2. Go API foundation
3. Email-code auth backend
4. React Native login screens
5. Token storage and authenticated API client
6. Vocabulary database/import
7. Topics
8. Spaced repetition review flow
9. Progress tracking

## Category memberships and mobile icons

Apply migration 16 and run `python3 scripts/category_release.py import` after
the vocabulary and definition imports. The category overlay preserves original
assignments and adds sense-specific memberships. See
[data and review notes](data/content/categories-v1/README.md).

The category selector uses `GET /topics/preview?topics=healthcare,supermarket-food`
for an exact unique count and at most six preview words. All-words General
Vocabulary is exclusive with narrower categories. Counts and progress follow
word membership, while existing sessions retain their snapshots.

The Expo app uses pinned Iconoir 7.12.1 and Expo-compatible react-native-svg
15.15.4. Run `npm install` in `apps/mobile` after pulling these dependency changes.
The icon registry and shared motion tokens follow [design.md](design.md).
Implementation and verification notes: [icons and categories](docs/mobile-design/icons-motion/review.md).
