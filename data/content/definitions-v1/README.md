# Dutch definitions and translation review

This is a content overlay for `release-9500`. The original release remains
unchanged. Each row represents one Dutch word in that release; each meaning
pairs an English gloss with a Dutch explanation and its provenance.

The October 1, 2026 build contains definitions for 7,675 of 9,500 words, including
all first 300 ranked words and all 190 original authored lesson words. There are
8,897 definition pairs, with at most three displayed per word. 1,825 words still
have no usable matching definition. They keep their translation-only card.

## Translation review

The full dataset was checked for membership, duplicates, missing content,
English-gloss matches, malformed definition text, and provenance. The source
contains Dutch definitions for 7,593 words, but a definition may describe a
different homograph from the translation taught in the app. Matching uses exact
normalized English glosses, not a lemma-only join. It strips an optional English
infinitive `to` and recognizes semicolon / `or` separators. It does not infer
synonyms or certify semantic equivalence.

`editorial.json` contains 169 targeted AI editorial drafts, focusing on frequent
words, missing lesson explanations, and misleading homographs. Translation text
changed for 58 words: 39 changed glosses and 19 changed only separator formatting.
Examples include `woord` → `word`, `vanaf` → `from; starting from`, and separate
`learn` / `teach` explanations for `leren`. Dictionary gloss matching alone can
still fail when the English word is also ambiguous, such as `can`; that case has
an explicit editorial explanation for the Dutch verb form.

`translation-changes.csv` records every changed translation. `review-queue.csv`
is ordered by word rank and includes missing definitions, multiple senses, and
source cleanup flags. These are review candidates, not proof of language errors.
All rows remain `needs_content_review`. No native-speaker approval is claimed.
The 1,825 gaps and remaining dictionary wording require further language work.

## Rebuild and import

```sh
python3 scripts/word_definitions.py build
python3 scripts/word_definitions.py validate
python3 -m unittest discover -s scripts -p 'test_*.py'
docker compose run --rm migrate
python3 scripts/word_definitions.py import
docker compose up -d --build api
```

The build uses the pinned release CSV and explicit editorial edits without
network calls. The manifest pins checksums for source, edits, and output. The
import runs in one transaction, validates all 9,500 ranks/lemmas, serializes with
other vocabulary imports, and makes no changes on an identical second run.
Updated drafts cannot overwrite content marked `approved` in the database.

Migration 15 creates `word_learning_content`, keyed by batch and word. API
practice and map responses prefer its translation and expose optional `meanings`
with `english`, `dutch`, `partOfSpeech`, `source`, `sourceUrl`, and `license`.
New sessions snapshot this content. Existing snapshots and legacy sessions keep
their original wording. No progress, schedule, word ID, example, or rank changes.

## Source and license

Definitions come from the Dutch definitions retained in the pinned
FreeDict+WikDict Dutch–English TEI dictionary, version 2025.11.21, attributed to
Wiktionary contributors via DBnary and WikDict, maintained by Karl Bartel.
Dictionary content and this adaptation use CC BY-SA 3.0 as stated in the pinned
archive. Source URLs and licenses travel with each definition. Editorial text
and the selected/shortened definitions are modifications by Woordenboom.
See `../../../THIRD_PARTY_NOTICES.md` and `../../sources/wikdict/LICENSE.txt`.
