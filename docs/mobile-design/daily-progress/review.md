# Daily practice refresh — 2026-10-14

The daily summary counted session events, while Knowledge Map answers only updated word progress. A saved Map answer therefore increased reviewed words but left daily practice at zero. The regression test reproduced this before the fix.

Daily practice now counts distinct words with a saved answer today from `user_word_progress`, covering regular sessions and Map practice. Both remembered and missed answers count; repeating the same word today still counts once. The existing server-day boundary remains unchanged. Existing saved Map answers are included without a migration.

The dashboard refreshes on focus, on returning to the foreground, on pull-to-refresh, and after a confirmed answer save. The save notification also fires when the learner has already left the study screen. Failed saves do not increment the displayed total. Request generations prevent older responses from replacing newer data.

Summary, category, and Map responses update independently, so category latency no longer delays the daily total. Partial refresh failures preserve available data and offer a retry. Authenticated responses and client reads disable caching.

## Verification

- Go regression: Map/session answers, repeated words, previous-day exclusion, duplicate submissions, and authenticated response cache headers. Full API test suite passed against the local database.
- Live Playwright regression with a disposable account: completed stack → 10; answer saved after leaving → 11; foreground refresh after Map practice → 12; completed Map range and returned home → 13.
- Held category requests did not block the daily summary; a failed category request preserved the updated total and could be retried. No browser page errors.
- Mobile lint and TypeScript passed. iOS and Android exports passed; native device interactions were not exercised.
- Screenshot: `updated-dashboard.png`. Machine-readable checks: `checks.json`. Reusable browser check: `apps/mobile/scripts/daily-progress-qa.cjs`.

The Knowledge Map heading also uses centered row alignment and explicit text line height/padding, with a flexible title and non-shrinking range count. Browser checks at 320, 375, and 414 pixels, including enlarged title text, found no clipping or overlap. Those captures are under `docs/mobile-design/map-heading`.

## Range back-navigation follow-up

On iOS, opening a range such as 1200 and returning to the dashboard showed the native pull-to-refresh spinner above already-visible content. The dashboard set `RefreshControl.refreshing` for every automatic focus refresh, and the control remained visible until all dashboard reads finished. Automatic focus, foreground, and saved-progress refreshes now run without that spinner. An actual pull gesture still shows it until the request settles. The browser regression now exercises 1200 → back to Map → back to dashboard; native spinner appearance was checked from the reported screenshot and code path, not on an iOS simulator.
