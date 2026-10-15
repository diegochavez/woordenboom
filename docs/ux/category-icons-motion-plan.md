# Category coverage, Iconoir and motion plan

Status: implemented locally, October 1, 2026. See [delivery and verification](../mobile-design/icons-motion/review.md). Native device tuning remains unverified. Scope: the existing Expo mobile app and its
vocabulary backend. Preserve monochrome screens, black study cards, compact map
pills, readable typography and the stable word position.

## What exists

The local release has 9,500 words. Of these, 9,027 have General Vocabulary as
their sole release topic; 261 are Daily Basics and 212 are spread across the
other 19 topics. Examples: Healthcare 11, Work & Office 8, Software & IT 5.
These counts come from `vocabulary_batch_words` joined to `vocabulary_batches`
for `release-9500`, grouped by `topic_slug`.

The source word-label model supports multiple labels, but the active release
and API expose one topic per word. Classification tools exist, while release
building takes one label and otherwise falls back to General Vocabulary.
Existing classifier inputs do not include our new paired Dutch definitions.
`LineIcon.tsx` currently renders text glyphs. Study content already has a 160 ms
opacity transition and live Reduce Motion support.

## 1. Make categories useful

General Vocabulary should be the all-words practice collection, rather than a
bucket that shrinks whenever a word gets categorized. Start practice continues
to open this collection. Category practice filters that same vocabulary.

- Keep the current topic taxonomy for the first pass. Do not spread a small
  number of words across more new categories.
- Classify the meaning taught on the card using its Dutch word, paired English
  gloss, Dutch definition, part of speech and available example. Avoid assigning
  a word using only spelling or a different dictionary sense.
- Extend the existing classification tooling with those inputs. Record proposed
  assignments, evidence/reason, classifier version and review state. Treat model
  confidence as an uncalibrated signal until evaluated on a reviewed sample.
- Begin with a stratified 500-word pilot covering frequent unassigned words,
  different parts of speech, and ambiguous meanings. Inspect errors before
  processing the remaining words in resumable batches.
- Allow a primary category and up to two genuinely relevant secondary categories.
  Keep ordinary function words and uncertain assignments in general practice.
  Do not force topic membership to meet a numerical coverage target.
- Preserve existing curated assignments. Store new memberships separately and
  version the release-specific assignment import so it can be inspected and
  rolled back without changing word IDs, map ranks or learning progress.

Use batch-scoped word/category memberships, adapting existing word labels where
possible. Keep a primary topic for display and legacy data, but use memberships
for category filtering and counts. General practice selects all eligible words
in the active release. Store the selected meaning/topic with new session
snapshots; old sessions retain their content.

Update category word counts, started counts, previews, unique selected-word
counts and availability together. Category counts can overlap. The overall
vocabulary total must count distinct word IDs rather than sum category totals.
Preserve deduplication when mixing categories and keep spaced-repetition state
per word across all entry points.

In the selector, show General Vocabulary as a separate all-words choice.
Choosing it clears narrower selections; choosing a specific category switches
to a custom mix. This avoids adding an all-words category to its own subsets.
Keep the current maximum of three specific categories and existing default CTA.

Acceptance: General practice reaches all 9,500 eligible words; selecting
Healthcare returns words relevant to the displayed healthcare meanings;
overlapping categories produce one card per word; category reassignment never
resets progress, changes a due date, or alters an existing session. Report actual
before/after category counts, uncertain assignments and reviewed-sample errors.

## 2. Introduce a restrained icon system

Install `iconoir-react-native` and Expo-compatible `react-native-svg` through
Expo's dependency installer. Verify peer dependencies and pin the resolved
versions. Replace the implementation of the existing LineIcon component with a
small, explicitly imported set of regular Iconoir components. Keep semantic
names at call sites. Do not construct dynamic imports for the complete catalog.
Keep the package's MIT notice in third-party attribution.

Use 20-point icons within text controls and category rows, 24-point navigation
icons, stroke width 1.5, and existing foreground colors. Invert icons on black
surfaces. Keep icon-only controls at least 44 points and label their actions for
screen readers. Decorative icons must not duplicate spoken text.

| Placement | Iconoir regular icon | Role |
| --- | --- | --- |
| Start practice | Learning | The supplied head/learning motif beside the label |
| Choose categories | ListSelect | Custom practice selection |
| General Vocabulary | BookStack | All-words collection |
| Knowledge map heading | Map | Identify the progress view; leave individual pills number-only |
| Next review control | Clock | Scheduling |
| Saved stack completion | Check | Confirm completed work |
| Back / forward / close | NavArrowLeft / NavArrowRight / Xmark | Replace text-glyph navigation |
| Housing | HomeSimple | Category recognition |
| Healthcare | HealthShield | Category recognition |
| Food & supermarket | Cart | Category recognition |
| Public transport | Bus | Category recognition |
| Work & office | Suitcase | Category recognition |
| Banking | Bank | Category recognition |
| School | GraduationCap | Category recognition |
| Repairs | Tools | Category recognition |
| Software & IT | Computer | Category recognition |

Map remaining categories to the same semantic registry during implementation.
Use one icon per category row with a stable text gutter. Keep text labels on
primary actions and the Know / Not yet decisions. Avoid decorative icon badges
around numbers or repeated icons beside every heading.

## 3. Add purposeful motion

These durations are proposed design tokens, to be tuned on devices.

| Interaction | Behavior | Duration |
| --- | --- | --- |
| Button press | Small opacity change, restored on release | 100–120 ms |
| Category selection | Immediate checked state, brief check opacity transition | 120 ms |
| Word flip / next word | Preserve the existing opacity-only reveal and fixed anchor | 160 ms |
| Timing sheet | Fade the backdrop and sheet together | 160–180 ms |
| Saved progress | Animate only the changed progress fill after confirmed data arrives | 200 ms |
| Stack complete | Show the check and completion text once after successful save | 180 ms |

Keep rating and navigation responsive; animations must never delay a save,
advance a card twice or hide a failure. Preserve the retry behavior. Respect
Reduce Motion and changes to that setting while the app is open. Use a shared
motion hook/tokens and the existing React Native Animated APIs for this scope;
no additional animation library is needed unless device testing identifies a
specific requirement. Optional light haptics can use the already installed Expo
Haptics package and must never block an action.

Avoid looping icon motion, bouncing counters, confetti, animated text layouts,
3D flips and staggered animation of all 95 map pills. Color remains reserved for
knowledge-map progress.

## Delivery order and checks

1. Category pilot and data model/API work, followed by a reviewed assignment
   import and correct counts/default collection semantics.
2. Iconoir integration through LineIcon, then navigation, category rows and the
   two dashboard actions.
3. Motion tokens and selected feedback transitions, followed by device tuning.

Check small screens, large text, long category names, mixed-category duplicates,
empty/due-only practice, old sessions, failure/retry, and rapid taps. Check icon
contrast and accessibility labels with VoiceOver/TalkBack. Test Reduce Motion,
low-end Android responsiveness, and Expo Go/development-build support. Run lint,
type checks, native bundle exports, database/session regressions and the existing
mobile browser checks. Compare bundle sizes after icon integration.

## References

- [Iconoir React Native documentation](https://iconoir.com/docs/packages/iconoir-react-native)
- [Iconoir package metadata](https://github.com/iconoir-icons/iconoir/blob/main/packages/iconoir-react-native/package.json)
- [Regular icon source files](https://github.com/iconoir-icons/iconoir/tree/main/icons/regular)
- [W3C interaction-animation guidance](https://www.w3.org/WAI/WCAG22/Understanding/animation-from-interactions.html)

The inspected package metadata is version 7.12.1 and supports React 18/19,
React Native >=0.78, and react-native-svg ^15.12.0. Our app uses React 19.2.3,
React Native 0.86.3 and Expo 57. Actual installed dependency compatibility and
bundle behavior remain implementation checks, not verified results of this plan.
