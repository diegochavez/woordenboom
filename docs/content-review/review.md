# Translation and Dutch-definition review

Reviewed October 1, 2026. Scope: all 9,500 rows were mechanically checked;
169 words received targeted AI editorial work. This is not a native-speaker
review of all 9,500 translations.

| Result | Count |
| --- | ---: |
| Words in the existing release | 9,500 |
| Words with Dutch definitions now | 7,675 |
| Words still missing a suitable definition | 1,825 |
| Definition pairs exposed | 8,897 |
| Highest-ranked words fully covered | 300 |
| Original lesson words fully covered | 190 |
| Targeted editorial drafts | 169 |
| Changed translation text | 58 |
| Changed glosses, excluding formatting | 39 |

## Findings and changes

**High severity, confirmed examples:** lemma-only definition lookup can attach an
unrelated homograph. `bij` teaches “at/near” but the source definition describes a
bee; `kom` teaches “come” but the source definition describes a bowl. English
gloss matching prevents those pairings. It cannot resolve every ambiguous
English gloss; `kan` / “can” requires an explicit verb explanation.

**High severity, localized examples:** dictionary ordering promotes obscure
meanings, such as `woord` starting with “drake”. The editorial overlay now uses
“word”. `grond` included “tereno”; its learner card now uses “ground; soil”.
`verhuurd` now says “rented out” to clarify the direction of renting.

**Medium severity, confirmed coverage gap:** the source has definitions for
7,593 words but some lack the displayed sense or contain unsuitable markup.
Targeted drafting and conservative matching produce 7,675 covered words.
Remaining gaps are preserved in a frequency-ordered review queue. No filler
explanations or definitions borrowed from another meaning are intentionally added.
Exact matching is a consistency check, not proof that dictionary wording is correct.

**Medium severity, pending language review:** 1,050 unedited words have multiple
source senses. Source descriptions can be technical or dated. Every imported row
remains a draft; independent Dutch-language review is still required. There is
no historical dataset here from which to infer a quality trend.

## Mobile behavior

After Flip, “Betekenis in het Nederlands” appears beneath the English translation
and Dutch word. Multiple meanings have separate English labels and Dutch
paragraphs. The word keeps its existing anchor, the card stays monochrome, and
the content scrolls above the fixed review controls. Definitions are not shown
before the reveal. Missing content produces no empty heading or placeholder.

The same content reaches practice and knowledge-map cards. Existing bilingual
examples remain available. Content changes do not reset progress or due dates.
Old sessions retain their wording and grading answer.

## Evidence and reproduction

- [Source release](../../data/content/release-9500/words.csv)
- [Definition manifest](../../data/content/definitions-v1/manifest.json)
- [Translation changes](../../data/content/definitions-v1/translation-changes.csv)
- [Review queue](../../data/content/definitions-v1/review-queue.csv)
- [Editorial drafts](../../data/content/definitions-v1/editorial.json)
- [Reproducible audit notebook](audit.ipynb)
- [Build/validation/import script](../../scripts/word_definitions.py)
- [Live browser results](browser-checks.json)

Source spot checks for ambiguous senses: [vanaf](https://nl.wiktionary.org/wiki/vanaf),
[bij](https://nl.wiktionary.org/wiki/bij), [kan](https://nl.wiktionary.org/wiki/kan),
and [leren](https://nl.wiktionary.org/wiki/leren). The current `vanaf` entry
separates the starting-place/time uses from the adverbial sense found in our
pinned dictionary export.

Verification covers deterministic output, membership, malformed/mismatched
content, first-300 and original-lesson coverage, idempotent import, API/map
agreement, session snapshots, and legacy translation preservation. Live Expo web
checks cover 320/375/414 px, reveal behavior, word position, long content scrolling,
multiple meanings, missing-definition fallback, and saving an answer. Native
VoiceOver/TalkBack behavior has not been verified.
