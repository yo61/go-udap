#!/usr/bin/env bash
set -euo pipefail

if [[ $# -eq 0 ]]; then
  echo "usage: $0 pnpm-workspace.yaml..." >&2
  exit 2
fi

status=0
for file in "$@"; do
  if [[ ! -f "$file" ]]; then
    echo "$file: no such file" >&2
    exit 2
  fi
  awk -v file="$file" '
    /^overrides:/ { in_overrides = 1; comment = ""; next }
    in_overrides && /^[^[:space:]#]/ { in_overrides = 0 }
    !in_overrides { next }
    /^[[:space:]]*$/ { comment = ""; next }
    /^[[:space:]]*#/ { comment = comment " " $0; next }
    {
      key = $0
      sub(/^[[:space:]]+/, "", key)
      sub(/:[[:space:]].*$/, "", key)
      if (comment !~ /GHSA-[a-z0-9]{4}-[a-z0-9]{4}-[a-z0-9]{4}|CVE-[0-9]{4}-[0-9]+/) {
        printf "%s:%d: override '\''%s'\'' has no comment naming the advisory it mitigates (GHSA-... or CVE-...)\n", file, NR, key
        failed = 1
      }
      comment = ""
    }
    END { exit failed }
  ' "$file" || status=1
done
exit "$status"
