#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
HOOK_SCRIPT="${DIR}/hook/agy-remote-hook.sh"
CONFIG_DIR="$HOME/.gemini/config"
HOOKS_FILE="$CONFIG_DIR/hooks.json"

chmod +x "$HOOK_SCRIPT"
mkdir -p "$CONFIG_DIR"

if [ -f "$HOOKS_FILE" ]; then
  echo "⚠️  $HOOKS_FILE already exists. Creating backup at $HOOKS_FILE.bak"
  cp "$HOOKS_FILE" "$HOOKS_FILE.bak"
fi

cat <<EOF > "$HOOKS_FILE"
{
  "antigravity-remote": {
    "PreToolUse": [
      {
        "matcher": "run_command|replace_file_content|multi_replace_file_content|write_to_file",
        "hooks": [
          {
            "type": "command",
            "command": "$HOOK_SCRIPT pre-tool",
            "timeout": 300
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "$HOOK_SCRIPT post-tool",
            "timeout": 30
          }
        ]
      }
    ],
    "Stop": [
      {
        "type": "command",
        "command": "$HOOK_SCRIPT stop",
        "timeout": 30
      }
    ]
  }
}
EOF

echo "✅ Antigravity Remote hook installed globally to $HOOKS_FILE"
