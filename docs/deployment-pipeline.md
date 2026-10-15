# Beta API deployment pipeline

The [Beta API workflow](../.github/workflows/beta-api.yml) tests and builds on GitHub-hosted runners after pushes to `main`. The deploy job also runs on GitHub-hosted infrastructure. It connects to the Hetzner server with a separate SSH key stored as a GitHub `beta` environment secret. Your personal 1Password SSH key is not used by Actions.

The deployment key has a forced command in `/root/.ssh/authorized_keys`. It can request only `deploy <commit SHA>`. The [gateway script](../deploy/ssh-deploy-gateway.sh) fetches the public repository and accepts the request only when the SHA is the current `main` commit. It then runs [the deployment script](../deploy/deploy-beta.sh). This avoids a persistent self-hosted runner on the public repository. [GitHub advises against self-hosted runners for public repositories](https://docs.github.com/en/actions/reference/security/secure-use).

`BETA_DEPLOY_ENABLED` starts as `false`, so verification runs without deploying. The `beta` environment accepts deployments only from `main`.

## One-time setup

1. In Hetzner DNS, create the `woordenboom.app` zone and an A record for `api` pointing to `2.29.102.249`. The registrar must delegate to the authoritative nameservers for that zone. Check `dig NS woordenboom.app` and `dig A api.woordenboom.app` before enabling deployment. Caddy needs working public DNS and inbound TCP 80/443 to obtain a certificate.
2. Keep Docker Engine, Compose, Git, Python 3, and curl installed on the server. Install the gateway script as `/usr/local/sbin/woordenboom-deploy-gateway`, owned by root and mode `0755`. Add the dedicated deployment public key to `/root/.ssh/authorized_keys` with `restrict,command="/usr/local/sbin/woordenboom-deploy-gateway"` before the key. Keep SSH password authentication disabled. The deployment key must be different from your personal 1Password key.
3. Keep `/etc/woordenboom/beta.env` outside the checkout, root-owned and mode `0600`. Copy the required names from [`deploy/beta.env.example`](../deploy/beta.env.example). Set `API_DOMAIN=api.woordenboom.app`, independent random `POSTGRES_PASSWORD` and `AUTH_SECRET` values, a matching password in `DATABASE_URL`, and working SMTP credentials. Keep `/var/backups/woordenboom` root-owned and mode `0700`. Do not commit either file or backup directory.
4. In the repository, create the `beta` environment with deployment branches restricted to `main`. Store the deployment private key as its `BETA_DEPLOY_SSH_KEY` secret. Set repository variables `BETA_DEPLOY_HOST=2.29.102.249` and `BETA_DEPLOY_HOST_KEY` to the server's verified Ed25519 host public key. The workflow pins this host key instead of trusting an SSH key scan during each run. Leave `BETA_DEPLOY_ENABLED=false` until DNS, email, server configuration, and backup arrangements are ready.
5. Set `BETA_DEPLOY_ENABLED=true` and run **Actions > Beta API > Run workflow** from `main`. Verify `https://api.woordenboom.app/readyz`, email login, practice, knowledge map, and daily progress on a device. Later pushes to `main` deploy after the verification job passes.

The GitHub-hosted deploy job needs inbound TCP 22 on the server. GitHub-hosted runner IP ranges change, so a fixed source-IP rule is not practical. The forced command limits what the dedicated key can request. Keep the server firewall restricted to ports 22, 80, and 443, and keep the SSH daemon's password login disabled.

## What each deployment does

The workflow runs Go tests, validates the three release datasets, checks Compose configuration, and builds the API image. The server checks that the requested SHA is the current `main` commit, rebuilds the API image, starts Postgres, writes a custom-format `pg_dump` under `/var/backups/woordenboom`, runs migrations, imports the pinned vocabulary on a fresh database, reruns the revision-checked release imports, starts the API and Caddy, and checks the HTTPS readiness endpoint. A failed command stops the deployment job. A migration can affect the running API before the new image starts, so deploy during a quiet period.

The local predeploy dump is a recovery point for a bad migration. It does not replace encrypted off-server backups or a restore drill. Keep Hetzner daily Backups enabled and arrange an off-server `pg_dump` copy before inviting testers. If a migration or import damages data, restore a known-good database dump before redeploying. Reverting code alone cannot reverse a schema change.
