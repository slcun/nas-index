# AGENTS.md — nas-index / nas-manager

## Repository structure

Two independent projects coexist at root:

| Path | Description |
|------|-------------|
| `index.html` | Static NAS navigation page (pre-existing, manual-edit links) |
| `nas-manager/` | Flask service management dashboard (active project) |

## nas-manager quick reference

**Stack:** Python Flask + YAML config + vanilla HTML/CSS/JS (no bundler)

**Entrypoint:** `nas-manager/app.py`

**Dev run:** `cd nas-manager && python app.py` → http://localhost:5000
  - Terminal available at `/terminal`; WebSocket PTY server at `:5001` (`ws_server.py`)
  - On Windows, `ws_server.py` won't work (pty fork unavailable) — terminal page still loads, WS will fail gracefully

**Prod deploy (Debian 13 only):** `sudo bash nas-manager/install.sh`
  - Installs to `/opt/nasmanager/`, runs as `nasmanager` user, systemd-managed
  - Two services: `nasmanager.service` (Flask :5000) + `nasmanager-ws.service` (WebSocket :5001)
  - Requires sudoers at `/etc/sudoers.d/nasmanager` (created by install.sh)

**Dependencies:** `flask>=3.0`, `pyyaml>=6.0`, `websockets>=14.0`, `ptyprocess>=0.7` (`requirements.txt`)

No tests, no linter config, no CI.

## Service management architecture

- **Service discovery:** `systemctl list-unit-files` + `/etc/systemd/system/` scan, merged with `config.yaml` definitions
- **Management** (start/stop/restart): `sudo systemctl` via subprocess, called with `sudo=True`
- **Config hot-reload:** `config_manager.py` checks file mtime on each `list_services()` call — no restart needed
- **Demo mode:** On non-systemd systems (Windows, containers), falls back to `DEMO_SERVICES` mock data in `service_manager.py:5-17`; no action needed

## Frontend

- Static files in `static/`, served by Flask (no build step)
- `app.js` polls `GET /api/services` every 15s for status refresh
- Dynamic button event binding via `MutationObserver` (not delegated events), at `app.js:192-195`
- Toast notifications auto-dismiss after 3s (CSS opacity transition), at `app.js:197-208`
- Search debounces at 200ms, at `app.js:152-156`

## WebSocket terminal (`ws_server.py`)

- Separate asyncio server on port `:5001` using `websockets` library
- Each WS connection spawns a `/bin/bash` via `pty.fork()` with PTY resize support
- Terminal page at `/terminal` loads `@wterm/dom` from CDN (jsdelivr importmap, no build step)
- Connection status indicator + auto-reconnect every 3s
- Max 10 concurrent connections, idle timeout 15 min (configurable via env)
- WS server won't start on non-systemd platforms (Windows) — terminal page still renders, WS fails silently

## API routes (all in `app.py`)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/services` | All services + categories |
| POST | `/api/services/<name>/start` | Start service |
| POST | `/api/services/<name>/stop` | Stop service |
| POST | `/api/services/<name>/restart` | Restart service |
| GET | `/api/services/<name>/logs?lines=50` | journalctl output |
| GET | `/api/host/info` | Hostname + IP |

## Config file (`config.yaml`)

- `services`: list — manual service definitions (overrides auto-discovered)
- `exclude_services`: list — names to hide from panel
- `categories`: map — display names for group headings

**Known quirk:** `config.yaml:exclude_services` entries like `systemd-*.service` are **literal string comparisons**, not glob patterns. To match a wildcard, use Python fnmatch or regex — or just add exact names.

## Conventions

- Comments in Chinese (from `CLAUDE.md`)
- Function-level doc comments expected (from `CLAUDE.md`)
