#!/usr/bin/env bash
# sync-quickstart.sh
#
# Copies internal/cli/quickstart.md into the README.md between the
# <!-- BEGIN QUICKSTART ... --> and <!-- END QUICKSTART --> markers.
#
# Usage: ./scripts/sync-quickstart.sh
#
# The single source of truth for quickstart content is:
#   internal/cli/quickstart.md
#
# This script should be run whenever that file is updated.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
QUICKSTART_SRC="$REPO_ROOT/internal/cli/quickstart.md"
README="$REPO_ROOT/README.md"

if [ ! -f "$QUICKSTART_SRC" ]; then
  echo "Error: $QUICKSTART_SRC not found" >&2
  exit 1
fi

if [ ! -f "$README" ]; then
  echo "Error: $README not found" >&2
  exit 1
fi

# Strip the leading "## Quick Start" heading and any following blank line
# since the README already has its own heading above the marker
QUICKSTART_BODY=$(sed '1{/^## Quick Start$/d;}' "$QUICKSTART_SRC" | sed '1{/^$/d;}')

# Use awk to replace everything between the markers.
# The quickstart content is read from a temp file to avoid shell escaping issues.
TMPFILE=$(mktemp)
echo "$QUICKSTART_BODY" > "$TMPFILE"

awk -v bodyfile="$TMPFILE" '
    /^<!-- BEGIN QUICKSTART/ {
        print
        print ""
        while ((getline line < bodyfile) > 0) print line
        close(bodyfile)
        print ""
        skip = 1
        next
    }
    /^<!-- END QUICKSTART/ {
        print
        skip = 0
        next
    }
    !skip { print }
' "$README" > "$README.tmp"

rm -f "$TMPFILE"
mv "$README.tmp" "$README"

echo "Synced quickstart content from $QUICKSTART_SRC into $README"
