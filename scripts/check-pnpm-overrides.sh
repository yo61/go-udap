#!/usr/bin/env bash

# Enable xtrace if the DEBUG environment variable is set
if [[ ${DEBUG-} =~ ^(1|yes|true)$ ]]; then
  set -o xtrace # Trace the execution of the script (debug)
fi

set -o errexit  # Exit on most errors (see the manual)
set -o nounset  # Disallow expansion of unset variables
set -o pipefail # Use last non-zero exit code in a pipeline

check_file() {
  local file=$1
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
        reason = "has no comment naming the advisory it mitigates (GHSA-... or CVE-...)"
        printf "%s:%d: override '\''%s'\'' %s\n", file, NR, key, reason
        failed = 1
      }
      comment = ""
    }
    END { exit failed }
  ' "$file"
}

main() {
  if [[ $# -eq 0 ]]; then
    echo "usage: $0 pnpm-workspace.yaml..." >&2
    exit 2
  fi

  local file status=0
  for file in "$@"; do
    if [[ ! -f "$file" ]]; then
      echo "$file: no such file" >&2
      exit 2
    fi
    check_file "$file" || status=1
  done
  exit "$status"
}

# Invoke main with args if not sourced (https://stackoverflow.com/a/28776166/8787985)
if ! (return 0 2> /dev/null); then
  main "$@"
fi
