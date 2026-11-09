# 📱 Antigravity Remote Control (AgyRemote)

> Remote control, tool approval, and multi-window monitoring for Google Antigravity directly from your smartphone over Tailscale.

Inspired by Claude Code's Remote Control (`claude rc`), built with **Go** (backend) + **SvelteKit 2 / Svelte 5** (PWA frontend).

---

## ✨ Features

- **Mobile First PWA**: Add to home screen on iOS and Android. Works standalone with full screen and custom dark theme.
- **Multi-Window Support**: Seamlessly tracks multiple open Antigravity windows (`well-done`, `brain`, etc.) and lets you filter approvals and live logs per window or all combined.
- **Synchronous Tool Approvals**:
  - `run_command`: Terminal preview with syntax styling and one-tap copy button.
  - `replace_file_content` / `write_to_file`: Unified diff viewer showing green additions (`+`) and red deletions (`-`).
  - Large thumb-friendly touch buttons: `[ ✓ APPROVE ]`, `[ ✕ REJECT ]`, `[ 💬 FEEDBACK ]`.
- **Feedback & Guidance**: Reject with custom instructions (e.g. "run tests first", "use pnpm instead of npm") so the agent dynamically corrects its approach.
- **Audio & Haptic Feedback**:
  - Web Audio synthesized chime on new approval requests (zero audio assets required).
  - Haptic vibration patterns on tap and approval alerts.
  - Native browser push notifications.
- **Live Activity Feed**: Real-time streaming of agent thoughts and executed tools from `transcript.jsonl`.
- **Fault-Tolerant & Lightweight**:
  - Go single binary backend with low memory footprint (~15 MB RAM).
  - If the remote server is offline, the bash hook gracefully falls back to `allow` after a brief check so it never locks your local workflow.

---

## 🚀 Quick Start

### 1. Start the Remote Server

Runs as a user systemd service on port **8765** (auto-starts on boot, restarts on crash):
```bash
./scripts/install-service.sh                      # one-time install (symlinks systemd/antigravity-remote.service)
systemctl --user restart antigravity-remote       # after rebuilding bin/server
journalctl --user -u antigravity-remote -f        # logs
```
For a manual foreground run: `./scripts/start.sh`.

### 2. Connect from your Phone

Open your browser (or install as PWA via "Add to Home Screen"):
- **Via HTTPS (needed for PWA/notifications):** `https://<your-device-host>:8765`
- **Via LAN / VPN IP:** `http://<device-ip>:8765`
- **Locally on PC:** `http://127.0.0.1:8765`

### 3. Enable Antigravity Hooks

To have Antigravity ask your phone for approvals before running tools:

#### Option A: Global (Recommended — covers all projects)
```bash
./scripts/install-hooks.sh
```
This creates `~/.gemini/config/hooks.json`.

#### Option B: Per-Repository
Add to `.agents/hooks.json` in any repository:
```json
{
  "preToolUse": [
    {
      "command": "/path/to/antigravity-remote/hook/agy-remote-hook.sh pre-tool"
    }
  ],
  "postToolUse": [
    {
      "command": "/path/to/antigravity-remote/hook/agy-remote-hook.sh post-tool"
    }
  ],
  "stop": [
    {
      "command": "/path/to/antigravity-remote/hook/agy-remote-hook.sh stop"
    }
  ]
}
```

---

## 🛠 Project Architecture

```
apps/antigravity-remote/
├── bin/
│   └── server                 # Compiled Go binary
├── cmd/server/
│   └── main.go                # HTTP, WebSocket & SPA server
├── pkg/
│   ├── approvals/             # Approval state manager & diff builder
│   ├── hub/                   # Thread-safe WebSocket broadcast hub
│   ├── models/                # Hook & domain types
│   └── transcript/            # Reader for transcript.jsonl
├── hook/
│   └── agy-remote-hook.sh     # Executable bash hook for Antigravity
├── frontend/                  # SvelteKit 2 + Svelte 5 PWA
│   ├── src/
│   │   ├── lib/
│   │   │   ├── audio.ts       # Web Audio API chime synthesizer
│   │   │   ├── components/    # Navbar, ApprovalCard, LiveFeed, etc.
│   │   │   └── types.ts       # TypeScript models
│   │   └── routes/            # Main SPA dashboard (+page.svelte)
│   └── static/                # PWA manifest, icons & SVGs
└── scripts/
    ├── install-hooks.sh       # Activates global Antigravity hooks
    ├── uninstall-hooks.sh     # Deactivates hooks
    └── start.sh               # Runs server
```
