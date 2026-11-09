#!/usr/bin/env bash
# Antigravity Lifecycle Hook Client
# Receives JSON on stdin from Antigravity, sends to Go Remote server, outputs decision to stdout.

REMOTE_SERVER_URL="${AGY_REMOTE_URL:-http://127.0.0.1:8765}"
EVENT_TYPE="${1:-pre-tool}" # pre-tool, post-tool, stop

# Read stdin
INPUT_JSON=$(cat)

# If server is not responding, gracefully allow to prevent blocking local workflow
RESPONSE=$(curl -s -m 310 -X POST "${REMOTE_SERVER_URL}/api/hook/${EVENT_TYPE}" \
  -H "Content-Type: application/json" \
  -d "$INPUT_JSON" 2>/dev/null)

if [ $? -eq 0 ] && [ -n "$RESPONSE" ]; then
  echo "$RESPONSE"
else
  # Default fallback if remote server is offline
  if [ "$EVENT_TYPE" = "pre-tool" ]; then
    echo '{"decision": "allow"}'
  else
    echo '{}'
  fi
fi
