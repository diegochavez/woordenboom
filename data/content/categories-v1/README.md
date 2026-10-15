# Category memberships, version 1

This overlay adds 1,372 draft category memberships to the existing 9,500-word
release. It preserves all 473 original specific assignments. There are now
1,763 distinct words with specific categories, and 7,737 words available only
through General Vocabulary. General Vocabulary includes all 9,500 words.

The source is the paired English gloss and Dutch definition in
`../definitions-v1/words.csv`. The classifier uses exact English glosses and
explicit Dutch dictionary domain labels within the same meaning. It checks
part of speech, rejects ambiguous glosses without domain evidence, and excludes
figurative, vulgar, outdated and visibly incomplete definitions. It does not
infer a meaning from the Dutch spelling or force a category for every word.
Source meanings retain their CC-BY-SA-3.0 attribution in `meanings_json`.

## Build and import

From the project root, after importing the base release and definition overlay:

```sh
python3 scripts/category_release.py pilot
python3 scripts/category_release.py build
python3 scripts/category_release.py validate
docker compose run --rm migrate
python3 scripts/category_release.py import
```

`rules.json` holds the versioned rules. `pilot.csv` has 500 words sampled across
frequency bands, parts of speech and ambiguous meanings. `assignments.csv`
records each proposed topic, selected translation, paired meanings and evidence.
`manifest.json` holds source/rules/output checksums and before/after counts.
See `review.md` for inspection findings and limitations.

The build is deterministic and can be resumed by rerunning it. The transactional
import uses the vocabulary import lock, checks release IDs/ranks and definition
revisions, and replaces only draft assignments owned by `category-senses-v1`.
Human-approved assignments and original topic assignments are preserved. An
unchanged reimport performs zero updates. The import never writes user progress,
review dates, session snapshots, word IDs or map ranks.

Migration 16 adds `vocabulary_word_topics` and a view combining original topics,
valid additional memberships and General Vocabulary. If the definition revision
changes, affected additional assignments stop appearing until reclassified.
Narrow-category cards receive only their matched definitions and translation.
Original examples are omitted for newly selected senses because those examples
have not been checked against the category meaning. General and original-category
content keeps its existing examples.

For rollback, delete draft rows for this classifier and batch in a transaction:

```sql
DELETE FROM vocabulary_word_topics m USING vocabulary_batches b
WHERE b.id=m.batch_id AND b.slug='release-9500'
  AND m.source='category-senses-v1' AND m.review_status='needs_content_review';
```

This removes the extra categories; it does not remove words or reset learning.
Use an earlier saved rules/assignment release to rebuild and import a previous
version. Do not drop migration 16 while the updated API is running.
