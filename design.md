<!-- Hallmark review: philosophy 5, hierarchy 4, execution 4, specificity 4, restraint 5, consistency 5. -->
# Woordenboom mobile design

A minimal, typography-led system for the existing Expo app. The current direction is black and white, with color reserved for knowledge-map progress. Keep large Dutch words, generous space and simple pill controls from the reference. Study cards are black on both sides; all other screens are white.

## Shared foundation

Native tokens live in `apps/mobile/src/lib/theme.ts`. This React Native project uses `StyleSheet` tokens rather than CSS or a new font dependency. System typography keeps native text scaling and familiar letterforms. Use the same system on every route.

| Role | Value | Usage |
| --- | --- | --- |
| Canvas | `#FFFFFF` | Dashboard, categories, map, sign-in, profile, licenses |
| Ink | `#111111` | Headings, body emphasis, primary actions |
| Body | `#383838` | Paragraphs and examples |
| Muted | `#686868` | Secondary labels on white |
| Rule | `#E5E5E5` | List separators, fields and map cells |
| Study | `#111111` | Both sides of the word card, with white text |
| Complete | `#FFFFFF` | A successfully finished stack or range |
| Known | `#C9E89B` | Map progress only |
| Learning | `#FFE047` | Map progress only |
| Secondary on black | `#BDBDBD` | Card metadata and original Dutch |

Do not introduce color outside the knowledge map. Use neutral grays for hierarchy. Both sides of a study card share the same black background and white controls.

## Typography and spacing

- Display: 48/54, weight 600. Study words: up to 46/52, sized down to 28 for long words on narrow screens. Dashboard count: 88/96.
- Screen title: 36/41, weight 600. Section title: 22/28, weight 600.
- Body: 16/24, regular. Caption: 14/20, regular. Button: 16, weight 500.
- Preserve native font scaling. Do not truncate words, examples or primary instructions.
- Use 4, 8, 12, 16, 24, 32 and 40-point spacing tokens. Screen gutters are 24 points, with a 600-point content maximum on tablets.
- Structure sections with space and occasional hairline rules. Avoid wrapping ordinary content in bordered cards.

## Controls

Primary actions use dark filled pills. Secondary actions use a one-point outline. Tertiary navigation uses text. Standard buttons are at least 56 points tall; icon and interactive map targets are at least 44 points; age choices are at least 48 points. Height can grow with larger text.

Category rows show a checkbox only when selecting. Navigation rows show an arrow. Map cells include rank ranges and status in their accessibility labels. On black cards, primary buttons invert to white with black labels. Secondary controls use white outlines. Card metadata uses a readable neutral gray.

Do not add simulated audio buttons, noninteractive tabs, confetti, or spatial movement between words. A successful rating advances immediately. Haptics are optional, and failure to play them must not block progress.

## Screen rules

| Route | Structure |
| --- | --- |
| `/login` | Open white page, welcome heading, labeled email/code input, one primary action and recovery links |
| `/onboarding` | Same form system, name, age choices, Continue |
| `/home` | Greeting, daily count, Start practice in General vocabulary and secondary Choose categories, unboxed map, metrics, learning strength, divided category rows |
| `/topics` | Heading, available-level information, selectable rows, word preview, persistent Start practice action |
| `/knowledge-map` | Short explanation, unboxed totals, responsive map grid and legend |
| `/review` | Black card, Dutch first, English on reveal, labeled Not yet / Know decisions |
| `/knowledge-range` | Same card and decisions, with available example translation |
| Completion | White canvas, saved-progress explanation, known / still learning counts, return and restart actions |
| `/disclaimer` | Same header and spacing, readable original attribution and license text |
| Loading / error / empty | Shared open layout, plain explanation, retry or exit where applicable |

## Word position and transitions

Keep words left-aligned at a stable vertical anchor slightly above the midpoint. Never vertically center the whole word/example group: revealing examples changes its height and moves the word. Reserve a metadata region based on usable screen height, independent of the footer and examples. Large accessibility text can expand it rather than clip.

Use the same display size for Dutch and English within each pair and a consistent 54-point line height. Examples flow below the word. The card scrolls when content is long. The secondary-action row reserves space even before reveal, and controls stay in their bottom area.

Fade the new word and supporting content in over 160 ms using opacity only. No morphing, sliding, scaling or animated layout changes. Stop animation on unmount; skip it with Reduce Motion, including preference changes while the screen is open. Changing side or card resets the reading scroll position.

This anchor is a product-specific recommendation for repeated recall. The reduced-motion behavior follows [W3C guidance on interaction-triggered animation](https://www.w3.org/WAI/WCAG22/Understanding/animation-from-interactions.html). It is implemented with [React Native Animated](https://reactnative.dev/docs/animated).

## State and navigation

A zero-word response is empty, never complete. Saving locks the answer controls immediately. Failed saves keep the current word and allow retry. Outstanding requests cannot advance a reloaded or unmounted stack. Returning from completion dismisses to the existing dashboard/map; both refresh on focus. Authentication restoration must not discard the requested route.

Safe-area insets come from `react-native-safe-area-context`. Scrollable content owns its padding; bottom controls sit inside the safe area. No fixed-height completion screens or absolutely positioned answer docks.

## Verification boundaries

Expo web at mobile widths verifies shared layouts and interactions. An iOS bundle export verifies compilation, not native device behavior. VoiceOver, TalkBack, the software keyboard, Dynamic Type and real device safe areas still require device testing.

Knowledge-map pills use a 32-point minimum visual height. Dashboard rows have 4-point gaps; interactive map rows retain separate 44-point minimum touch targets around each pill. Labels and rows may grow with larger text.

## Icons and feedback

Use Iconoir regular icons at 20 points in text controls and category rows, 24
points for navigation, with a 1.5-point stroke. All icons inherit monochrome
foreground colors. Keep icons decorative to screen readers; label the containing
control. The learning icon identifies Start practice, ListSelect identifies
Choose categories, Map identifies the map heading, and Clock identifies timing.
Only knowledge-map progress uses color.

The semantic registry lives in `LineIcon.tsx`. Explicit imports in
`src/lib/iconoir.js` select only the used icons from pinned version 7.12.1; avoid
the package barrel, which adds the entire catalog to Metro builds.

Motion tokens live in `useMotion.ts`: press opacity 110 ms, selection check
120 ms, word reveal 160 ms, review sheet entry and saved completion 180 ms,
updated progress fills 200 ms. Initial progress renders at its actual value.
There is no staggered map entrance. Sheet dismissal and actions remain immediate.
One shared native preference subscription disables motion when Reduce Motion
is enabled, including changes while the app is open.

General Vocabulary is the all-words collection. In the selector it appears
separately above specific categories. Choosing it clears a custom mix; choosing
a specific category clears General. Allow up to three specific categories.
Preview counts use unique word IDs from the API rather than adding category
totals. Missing previews never prevent starting an otherwise valid practice.
