#!/usr/bin/env bash
# Installs and starts the antigravity-remote user systemd service.
set -euo pipefail
DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

mkdir -p ~/.config/systemd/user
cat > ~/.config/systemd/user/antigravity-remote.service <<EOF
[Unit]
Description=Antigravity Remote Control Server
After=network.target

[Service]
Type=simple
WorkingDirectory=${DIR}
ExecStart=${DIR}/bin/server -port 8765 -static ${DIR}/frontend/build
Restart=always
RestartSec=3
Environment=PORT=8765

[Install]
WantedBy=default.target
EOF
systemctl --user daemon-reload
systemctl --user enable --now antigravity-remote.service
loginctl enable-linger "$USER" 2>/dev/null || true
systemctl --user status antigravity-remote.service --no-pager | head -5
