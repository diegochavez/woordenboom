# Private beta launch plan

The first release is a private beta. The Hetzner server and GitHub pipeline are prepared, but the hosted API and signed builds are not live yet.

## Hosting choice

The beta server is already running at `2.29.102.249`. It has 1 vCPU, 1 GB RAM, 20 GB disk, and 2 GB swap. Docker and Compose are installed, and the API image built on this host. That establishes build feasibility, but 1 GB leaves little room for PostgreSQL, the API, Caddy, and imports at the same time. Watch memory and disk during the first deployment; move to a 4 GB server if it swaps heavily or restarts containers. Confirm the live resize price in Hetzner Console.

Run the current Go API, PostgreSQL 16, and Caddy in separate containers with [`docker-compose.beta.yml`](../docker-compose.beta.yml). Caddy provides HTTPS at `api.woordenboom.app` and serves the Mailjet verification file on `woordenboom.app`; the database has no published port, and the API listens only on the private Compose network. Keep the database on the server's local disk rather than an attached Hetzner Volume so the [server Backups include it](https://docs.hetzner.com/cloud/servers/backups-snapshots/overview/). Use Mailjet's Send API for email login codes. This is a low-cost, single-machine beta with no high availability; move the database to managed Postgres if uptime or growth demands it.

The choice follows the existing code and beta reliability needs. The long-running local development database is **643 MB**, but a clean beta bootstrap is **182 MB**, so it fits [Supabase Free's 500 MB allowance](https://supabase.com/pricing). The free project [can pause after inactivity](https://supabase.com/docs/guides/platform/free-project-pausing) and [has no automatic backups](https://supabase.com/docs/guides/platform/backups). Vercel's [Go Functions runtime is still beta](https://vercel.com/docs/functions/runtimes/go), would require converting the current long-running API to a function, and its [free Hobby plan is limited to personal, non-commercial use](https://vercel.com/docs/plans/hobby). Vercel + Supabase is the lower cash-cost experiment if those tradeoffs suit a small beta; Hetzner uses the Compose stack already tested locally and has a straightforward backup path.

```text
Expo beta app -> https://api.<domain> -> Caddy :443 -> Go API :8080
                                                    |       |
                                             private Postgres  Mailjet HTTPS
```

## Local preparation

- [`docker-compose.beta.yml`](../docker-compose.beta.yml) keeps Postgres on an internal network, exposes only Caddy's ports 80/443, and has a separate migration job. [`deploy/Caddyfile`](../deploy/Caddyfile) reverse-proxies the API. [`deploy/beta.env.example`](../deploy/beta.env.example) lists the required settings without real secrets.
- The API defaults to `release-9500`. Production mode rejects development auth secrets, requires both Mailjet credentials for the beta provider, and requires verified TLS for remote PostgreSQL. It allows plaintext database traffic only to the `postgres` service on the same private Compose network. Local development still uses Mailpit over SMTP. The container includes CA certificates for outbound TLS.
- EAS `preview` and `production` build profiles are present. A build hook rejects a missing or non-HTTPS `EXPO_PUBLIC_API_URL`, and the app refuses a local API URL in non-development mode.
- Go tests, mobile lint/typecheck, all 21 Expo Doctor checks, Expo iOS/Android exports, the API image build, and Compose/Caddy configuration checks passed locally. On a fresh isolated beta stack, all 16 migrations, the OpenTaal and three content imports, and the API `/readyz` check passed; it contained 9,500 release words, 9,500 content records, and 1,372 category memberships. The test volume was removed afterward. A compatible `npm audit fix` removed the critical advisory; 29 audit findings remain in the dependency tree and need triage against Expo-supported versions before public release. Signed-device and hosted API checks remain outstanding.
- The [GitHub Actions beta pipeline](deployment-pipeline.md) verifies pushes to `main` and, once enabled, deploys from a GitHub-hosted runner through a restricted SSH key. The hosted API check remains outstanding.

## Deployment order

1. Keep the existing server updated and enable Hetzner Backups. Apply a Hetzner Cloud Firewall that allows TCP 80/443 publicly and TCP 22 for GitHub-hosted deployments. Leave outbound HTTPS open for Mailjet and package updates. Set a project [cost alert](https://docs.hetzner.com/cloud/billing/faq/). Point both `api.woordenboom.app` and `woordenboom.app` at `2.29.102.249`; Caddy will request and renew their TLS certificates.
2. Verify the sending domain in Mailjet using the zero-byte file served at the root of `woordenboom.app`. Add Mailjet's SPF and DKIM DNS records, create an API key and secret key, and store them in the GitHub `beta` environment. Send and verify a real login email before inviting testers.
3. Create `/etc/woordenboom/beta.env` from `deploy/beta.env.example`, outside the Git checkout. Generate independent random `POSTGRES_PASSWORD` and `AUTH_SECRET` values; URL-encode the database password in `DATABASE_URL`. Keep `APP_ENV=production` (set in Compose), `VOCABULARY_BATCH=release-9500`, and `API_DOMAIN=api.woordenboom.app`. Never place these secrets in Expo or a Git commit. The two environment values for the database password must match. Keep the file root-owned and mode `0600`.
4. Install the restricted deployment SSH key on the server, create its backup directory, and configure the GitHub `beta` environment. Follow the [deployment pipeline setup](deployment-pipeline.md), then enable `BETA_DEPLOY_ENABLED` and run the workflow from `main`. The deploy script checks Compose, saves a predeploy database dump, runs migrations and content imports, starts the API and Caddy, and verifies HTTPS readiness. Check 9,500 release words and 95 map ranges after the first run.
5. Start the API and Caddy, test `/readyz` over the public HTTPS hostname, and exercise code request/verification, practice, review scheduling, knowledge map, and daily progress. Keep Docker logs and a simple uptime check. The API has per-address email request limits, but before broadening beta access add an invitation allowlist or an IP-based edge limit to prevent email-sending abuse.
6. Keep [Hetzner's seven daily Backups](https://docs.hetzner.com/cloud/servers/backups-snapshots/overview/) enabled and periodic encrypted `pg_dump` copies outside the server and Hetzner project. The deployment script writes a local predeploy dump, but that alone is not an off-server backup. Restore one dump into a disposable database before inviting testers. Server Backups are disk copies, not a substitute for database-level backups; attached Hetzner Volumes are excluded. No zero-downtime or automatic failover is promised on this single server.
7. Confirm the final `ios.bundleIdentifier` and `android.package` in `apps/mobile/app.json`. Set `EXPO_PUBLIC_API_URL=https://api.woordenboom.app` in both EAS `preview` and `production` environments. Use `preview` for an Android internal APK or an iOS ad hoc build for registered devices. For TestFlight, use the store-distribution `production` profile and submit that iOS build through the existing Apple account. Test login and navigation on actual iPhone and Android hardware before sending invitations.

## Content and later public release

The 9,500 words have English translations, but this remains a **draft learning corpus**: 9,048 topic assignments need review, 1,825 Dutch definitions are missing, and most words lack bilingual examples. Use a small, informed beta group and collect corrections. Do not describe the full set as editorially verified. See the [content review](content-review/review.md) and [release notes](../data/content/release-9500/README.md).

For public release, finish content review, native accessibility checks, privacy policy/support contact, account deletion, and store listing requirements. [Apple](https://developer.apple.com/support/offering-account-deletion-in-your-app/) and [Google Play](https://support.google.com/googleplay/android-developer/answer/13327111?hl=en) require deletion flows for account-based apps. Move away from the single instance when uptime or user volume demands it.

## Inputs still needed

- Mailjet domain validation and its API key and secret key in GitHub's `beta` environment. The sender is `login@woordenboom.app`; the actual email delivery remains untested until those values are in place.
- Confirmation of Hetzner Backups and the server firewall. The current 1 GB server needs monitoring during the first full deployment.
- The final iOS bundle ID and Android package name, and access to the existing Expo project and Apple Developer account. Expo and Apple access have been confirmed; the identifiers remain to be chosen.
- The initial commit is pushed to [GitHub](https://github.com/diegochavez/woordenboom). Choose whether beta testers should start with clean progress or whether any existing local accounts should be migrated.
