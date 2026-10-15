# 9,500 Dutch words with English translations

Status: imported for local development, pending human language review.

| Check | Result |
| --- | ---: |
| Unique Dutch lemmas | 9,500 |
| Nonempty English translations | 9,500 |
| Existing bilingual example pairs retained | 190 |
| Dictionary translations | 9,270 |
| Editorial translation corrections | 40 |
| Existing authored draft translations | 190 |
| Dictionary entries with retained senses | 9,446 |
| Words needing topic review | 9,048 |
| Words with multiple dictionary senses | 1,086 |
| Knowledge-map ranges of 100 words | 95 |

`words.csv` is the portable, attributed dataset and review queue. It contains
translation provenance, dictionary entry URLs, parts of speech, frequency
estimates, topic-review flags, examples where available, and full dictionary
senses as JSON. `manifest.json` records counts and source/output checksums.
`overrides.csv` records common-sense corrections and the reason for each.

## Selection and quality limits

We matched exact Dutch dictionary lemmas to the existing OpenTaal candidate pool.
The pinned WikDict dictionary contains 40,061 headwords. Proper-name, prefix,
suffix, and abbreviation entries were excluded. This yielded 9,590 eligible
candidates including the preserved authored words. We kept the 190 existing
translations/examples and filled the remaining places by descending wordfreq
Zipf score, with lemma as the tie-breaker. The cutoff is 1.07, so the wider set
includes uncommon, specialist, dated, and potentially sensitive vocabulary.
This is not a list of the 9,500 most common Dutch words or 9,500 distinct concepts.
Inflected words can remain because the upstream candidate classifier is heuristic.

Displayed dictionary translations include up to three senses/synonyms. Full
selected dictionary senses remain available in the CSV and in
`word_dictionary_entries`. A dictionary can omit a common sense or contain a bad
translation. Forty common cases were corrected, including `mee`, `hen`, and
`week`. Automated checks establish completeness and consistency, not fluency or
pedagogical correctness. All rows remain `needs_content_review`; none were
promoted to `approved` by the importer. CEFR estimates are intentionally omitted
from this release because they have not been verified.

Unclassified words appear under General vocabulary. Existing topic assignments
are retained, but the `needs_topic_review` flag remains where applicable.
There are 9,310 words without example sentences; the app supports translation-only
cards and never inserts fabricated placeholder sentences.

## Import and verification

From the repository root, after the original OpenTaal import:

```bash
docker compose run --rm migrate
python3 scripts/vocabulary_release.py validate
python3 -m unittest discover -s scripts -p 'test_*.py'
python3 scripts/vocabulary_release.py import
docker compose up -d --build api
```

The importer refuses missing Dutch lemmas and checksum mismatches. It locks
concurrent release imports and wraps all writes in one transaction. Re-running an
unchanged release adds zero rows. If the release already exists with a different
checksum, it refuses to overwrite it. Create a new version for changed membership
or ranking. Existing word IDs and learner progress are preserved.

For the live database integration checks, after import:

```bash
docker run --rm --network woordenboom_default \
  -e 'TEST_DATABASE_URL=postgres://woordenboom:woordenboom@postgres:5432/woordenboom?sslmode=disable' \
  -v "$PWD/services/api:/src" -w /src golang:1.23-alpine go test ./...
```

These checks inspect all 95 ranges, compare category counts, exercise session
completion and duplicate-answer handling, and verify that sessions remain tied
to their original release. They create and remove a dedicated test user without
sending email.

## Rebuild the dataset

The original snapshot of the candidate view was exported before import. To make
a new candidate snapshot:

```bash
docker compose exec -T postgres psql -U woordenboom -d woordenboom -c \
  "COPY (SELECT c.* FROM candidate_word_scores c WHERE c.is_learning_candidate AND c.zipf_score IS NOT NULL ORDER BY c.zipf_score DESC,c.lemma) TO STDOUT WITH CSV HEADER" \
  > /tmp/woordenboom-candidates.csv
python3 scripts/vocabulary_release.py build --candidates /tmp/woordenboom-candidates.csv
```

The builder reads the pinned compressed TEI source under `data/sources/wikdict`,
checks its uncompressed SHA-256, and uses the original release-2000 export to
preserve its authored content. It uses only the Python standard library and makes
no network calls. The candidate-snapshot hash changes when its view changes;
only columns used for selection affect the dataset checksum.

## Remaining backend content tasks

- [x] Reach at least 2,000 translated words, expanded to 9,500 at the user's request.
- [x] Import the release and expose translation-only cards through the API.
- [x] Preserve attribution, dictionary senses, and repeatable import checks.
- [x] Preserve older session batches and learner progress.
- [ ] Review translations, ambiguous senses, inflections, and learning suitability.
- [ ] Assign or confirm topics for 9,048 flagged entries.
- [ ] Add checked bilingual examples for the remaining 9,310 entries.
- [ ] Validate CEFR levels before exposing level-based study.

## Attribution and license

Dictionary: Nederlands-English FreeDict+WikDict, version 2025.11.21, by Wiktionary
contributors via DBnary and WikDict, maintained by Karl Bartel.
[Source download](https://download.wikdict.com/dictionaries/tei/recommended/nld-eng.tei).
This archive's TEI header states
[CC BY-SA 3.0 Unported](https://creativecommons.org/licenses/by-sa/3.0/legalcode).
The adapted release is provided under that license. Preserve attribution and
notice of modifications when sharing it. See `../../../THIRD_PARTY_NOTICES.md`
for OpenTaal and wordfreq attribution, and `../../sources/wikdict/LICENSE.txt`
for the dictionary's license text.
