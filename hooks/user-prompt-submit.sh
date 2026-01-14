#!/bin/bash
# Juggle: Track activity on each user message

JUGGLE_BIN="${JUGGLE_BIN:-juggle}"

# Silently update activity (errors are ignored)
$JUGGLE_BIN track-activity 2>/dev/null || true
