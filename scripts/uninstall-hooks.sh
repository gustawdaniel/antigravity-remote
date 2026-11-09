#!/usr/bin/env bash
CONFIG_DIR="$HOME/.gemini/config"
HOOKS_FILE="$CONFIG_DIR/hooks.json"

if [ -f "$HOOKS_FILE" ]; then
  rm "$HOOKS_FILE"
  echo "✅ Removed $HOOKS_FILE"
  if [ -f "$HOOKS_FILE.bak" ]; then
    mv "$HOOKS_FILE.bak" "$HOOKS_FILE"
    echo "Restored previous $HOOKS_FILE from backup"
  fi
else
  echo "No $HOOKS_FILE found."
fi
