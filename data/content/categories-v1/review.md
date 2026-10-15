# Category inspection

October 1, 2026. This was an agent review, not independent verification by a
Dutch language editor. Every new assignment remains `needs_content_review`.
No accuracy percentage or calibrated confidence score is claimed.

The 500-word stratified pilot initially proposed categories for 66 words. The
proposed definitions exposed errors that English gloss matching alone misses:

| Word / sense | Problem | Correction |
| --- | --- | --- |
| kam / bridge | Musical bridge assigned to public transport | Require domain evidence for ambiguous bridge |
| pupil / pupil | Eye anatomy assigned to school | Keep anatomy and legal ward senses in their matching categories |
| hozen / bail | Emptying a boat assigned to legal contracts | Ambiguous bail requires domain evidence |
| beslist / firm | Adjective assigned to work as a company | Require domain evidence for firm |
| dragend / bearing | Supporting adjective treated as a machine bearing | Require domain evidence for bearing |
| tafel / table | A table/list sense included with furniture | Leave ambiguous table to further review |
| paar / couple | Indefinite pronoun included as a relationship | Exclude unsupported parts of speech and ambiguous couple |
| dentaal / dental | Phonetics assigned to healthcare | Respect conflicting dictionary domains |
| lager | Truncated dictionary explanation | Exclude visibly incomplete definitions |
| kut; idioot | Vulgar or outdated dictionary wording unsuitable for neutral category lessons | Exclude vulgar domains and flagged entries |

A second inspection read four deterministically sampled new assignments per
specific topic, 76 assignments. It caught confirmation as a religious rite,
flood as an influx, policy as an insurance document, sewer as a person who sews,
and consumption without a utility context. Those glosses now require domain
evidence. `wegloper` and `houtwerk` were held out because their paired glosses
need language review. A generic queue sense was also held out.

After these corrections, 43 pilot words have new proposals. The full release
has 1,372 additional memberships covering 1,290 previously uncategorized words.
There are 1,763 distinct specifically categorized words including the original
473. Category totals overlap and must not be summed to obtain vocabulary size.

Definitions marked as anatomy, medicine and food give those categories the
largest increases. Immigration, phone appointments and utility bills still have
limited coverage. Adding useful words there requires more reviewed sense
mappings and context, not broad matching of words such as call, bill or form.

Remaining risks include inherited translation/definition pairing errors, broad
or specialist dictionary senses, source part-of-speech errors, and missing Dutch
definitions for 1,825 words. The classifier leaves uncertain words in General
Vocabulary. These rules are a conservative first release, not an exhaustive
semantic classification of the corpus. Original category assignments were
preserved and were not relabeled or certified by this inspection.

Automated regressions cover the ambiguous examples, category-specific meanings,
maximum three memberships including original assignments, reproducible output,
legacy pools, unique union counts, cross-category progress and future review
exclusion. Full before/after counts are in `manifest.json`.
