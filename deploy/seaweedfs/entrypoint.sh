#!/bin/sh
# Builds the SeaweedFS S3 identity config from Docker secrets
# (/run/secrets/seaweedfs_access_key, /run/secrets/seaweedfs_secret_key) or,
# if absent, from SEAWEEDFS_ACCESS_KEY / SEAWEEDFS_SECRET_KEY env vars — the
# same lookup order the backend uses — then hands off to the image entrypoint.
# Without -s3.config SeaweedFS would accept anonymous S3 requests.
set -eu

secret() {
  file="/run/secrets/$(printf '%s' "$1" | tr 'A-Z' 'a-z')"
  if [ -f "$file" ]; then
    tr -d '\r\n' < "$file"
  else
    printenv "$1" || true
  fi
}

ACCESS_KEY=$(secret SEAWEEDFS_ACCESS_KEY)
SECRET_KEY=$(secret SEAWEEDFS_SECRET_KEY)

if [ -z "$ACCESS_KEY" ] || [ -z "$SECRET_KEY" ]; then
  echo "seaweedfs: SEAWEEDFS_ACCESS_KEY and SEAWEEDFS_SECRET_KEY are required" >&2
  exit 1
fi
case "$ACCESS_KEY$SECRET_KEY" in
  *'"'* | *'\'*)
    echo 'seaweedfs: keys must not contain " or \' >&2
    exit 1
    ;;
esac

CONFIG=/tmp/s3.json
umask 077
cat > "$CONFIG" <<JSON
{
  "identities": [
    {
      "name": "app",
      "credentials": [{"accessKey": "$ACCESS_KEY", "secretKey": "$SECRET_KEY"}],
      "actions": ["Admin", "Read", "List", "Tagging", "Write"]
    }
  ]
}
JSON
# The image entrypoint drops privileges to the "seaweed" user.
if [ "$(id -u)" = "0" ]; then
  chown seaweed:seaweed "$CONFIG"
fi

exec /entrypoint.sh "$@" -s3.config="$CONFIG"
