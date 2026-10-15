#!/usr/bin/env bash
set -Eeuo pipefail

# Install as /usr/local/sbin/woordenboom-deploy-gateway, owned by root.
# The deployment key's authorized_keys entry must force this command.
PATH=/usr/sbin:/usr/bin:/sbin:/bin
export PATH

if [[ ! "${SSH_ORIGINAL_COMMAND:-}" =~ ^deploy\ ([0-9a-f]{40})$ ]]; then
  echo 'Only deploy <40-character commit SHA> is allowed.' >&2
  exit 1
fi
sha="${BASH_REMATCH[1]}"

exec 9>/run/lock/woordenboom-deploy.lock
if ! flock -n 9; then
  echo 'Another Woordenboom deployment is already running.' >&2
  exit 1
fi

repo=/opt/woordenboom/repo
origin=https://github.com/diegochavez/woordenboom.git
if [[ ! -d "$repo/.git" ]]; then
  install -d -m 0755 /opt/woordenboom
  git clone "$origin" "$repo"
fi
if [[ "$(git -C "$repo" remote get-url origin)" != "$origin" ]]; then
  echo 'Unexpected Git origin on the deployment server.' >&2
  exit 1
fi

git -C "$repo" fetch origin main
if [[ "$(git -C "$repo" rev-parse FETCH_HEAD)" != "$sha" ]]; then
  echo 'Deployment commit is not the current main commit.' >&2
  exit 1
fi
git -C "$repo" reset --hard "$sha"

export GITHUB_REF=refs/heads/main
export GITHUB_REPOSITORY=diegochavez/woordenboom
export GITHUB_SHA="$sha"
export BETA_ENV_FILE=/etc/woordenboom/beta.env
export BETA_BACKUP_DIR=/var/backups/woordenboom
exec bash "$repo/deploy/deploy-beta.sh"
