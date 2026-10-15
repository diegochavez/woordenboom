#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"

if [[ "${GITHUB_REF:-}" != 'refs/heads/main' || "${GITHUB_REPOSITORY:-}" != 'diegochavez/woordenboom' ]]; then
  echo 'Beta deployment only accepts the main branch of diegochavez/woordenboom.' >&2
  exit 1
fi
: "${GITHUB_SHA:?GITHUB_SHA is required}"
: "${BETA_ENV_FILE:?BETA_ENV_FILE is required}"
: "${BETA_BACKUP_DIR:?BETA_BACKUP_DIR is required}"

if [[ ! -r "$BETA_ENV_FILE" || ! -d "$BETA_BACKUP_DIR" || ! -w "$BETA_BACKUP_DIR" ]]; then
  echo 'The beta environment file must be readable and the backup directory writable by the runner.' >&2
  exit 1
fi

umask 077
export COMPOSE_FILE="$repo_root/docker-compose.beta.yml"
export COMPOSE_ENV_FILES="$BETA_ENV_FILE"

docker compose config --quiet
docker compose build api
timeout 180 docker compose up -d --wait postgres

# Preserve the database state before running migrations or content imports.
backup_tmp="$(mktemp "$BETA_BACKUP_DIR/.predeploy-XXXXXX")"
trap 'rm -f "$backup_tmp"' EXIT
docker compose exec -T postgres pg_dump -Fc -U woordenboom woordenboom > "$backup_tmp"
backup_file="$BETA_BACKUP_DIR/predeploy-$(date -u +%Y%m%dT%H%M%SZ)-${GITHUB_SHA:0:12}.dump"
mv "$backup_tmp" "$backup_file"
trap - EXIT
echo "Database backup saved to $backup_file"

docker compose run --rm migrate

# The pinned OpenTaal universe is needed only on a fresh database. The other
# imports check their revisions and remain safe to rerun for content updates.
opentaal_present="$(docker compose exec -T postgres psql -X -q -tA -U woordenboom -d woordenboom -c \
  "SELECT EXISTS (SELECT 1 FROM vocabulary_sources WHERE id = 'opentaal-wordlist')")"
if [[ "$opentaal_present" != 't' ]]; then
  docker compose run --rm api /app/import-opentaal
fi

python3 scripts/vocabulary_release.py import
python3 scripts/word_definitions.py import
python3 scripts/category_release.py import

docker compose up -d api caddy

api_domain="$(docker compose config --format json | python3 -c \
  'import json,sys; print(json.load(sys.stdin)["services"]["caddy"]["environment"]["API_DOMAIN"])')"
for attempt in {1..12}; do
  if curl --silent --show-error --fail --max-time 5 \
      --resolve "$api_domain:443:127.0.0.1" "https://$api_domain/readyz" > /dev/null; then
    echo "Deployed $GITHUB_SHA to https://$api_domain"
    exit 0
  fi
  sleep 5
done

echo "API did not become ready at https://$api_domain/readyz" >&2
exit 1
