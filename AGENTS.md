# AGENTS.md

## Repository

| Path | Description |
|------|-------------|
| `nas-manager/` | Flask service management dashboard (active project) |
| `README.md` | Bilingual docs (Chinese + English) |

Root `index.html` referenced in old docs no longer exists — only `nas-manager/` matters.

## nas-manager

**Stack:** Python Flask + YAML config + vanilla HTML/CSS/JS (no bundler)

**Entrypoint:** `nas-manager/app.py` — creates `ConfigManager` + `ServiceManager` at module level

**Dev run:** `cd nas-manager && pip install -r requirements.txt && python app.py`
- Serves on `http://0.0.0.0:5000`, **debug mode is off** (`app.run(debug=False)` at `app.py:103`)
- Terminal page at `/terminal`; WS PTY server at `:5001` (`ws_server.py`)
- On Windows, `ws_server.py` falls back to `asyncio.create_subprocess_exec` pipe mode (no PTY fork) — `ws_server.py:21`
- No hot reload — restart manually after code changes

**Dependencies:** `flask>=3.0`, `pyyaml>=6.0`, `websockets>=14.0`, `ptyprocess>=0.7`

**Prod deploy (Debian 13 only):** `sudo bash nas-manager/install.sh`
- Installs to `/opt/nasmanager/`, runs as `nasmanager` user, two systemd services: `nasmanager.service` (Flask :5000) + `nasmanager-ws.service` (WS :5001)

No tests, no linter, no CI, no typecheck.

## Architecture

- **Service discovery:** `systemctl list-unit-files` + `/etc/systemd/system/` scan, merged with `config.yaml` definitions
- **Management** (start/stop/restart): `sudo systemctl` via subprocess (`sudo=True`)
- **Config hot-reload:** `config_manager.py:30-36` checks file mtime on each `list_services()` call — change `config.yaml` without restart
- **Demo mode:** On non-systemd platforms (Windows, containers), falls back to 11 hardcoded mock services at `service_manager.py:5-17`; no action needed
- **WS terminal:** Separate asyncio `websockets` server. Each connection spawns `/bin/bash` via PTY fork (Linux) or subprocess pipe (Windows). Max 10 concurrent connections. Auto-reconnect every 3s on the frontend (`terminal.js:81-87`). **No idle timeout** implemented despite being planned.

## API routes (`app.py`)

| Method | Path | Purpose |
|--------|------|---------|
| GET | `/api/services` | All services + categories |
| GET | `/api/services/<name>` | Single service detail |
| POST | `/api/services/<name>/start` | Start service |
| POST | `/api/services/<name>/stop` | Stop service |
| POST | `/api/services/<name>/restart` | Restart service |
| GET | `/api/services/<name>/logs?lines=50` | journalctl output |
| GET | `/api/host/info` | Hostname + IP |
| GET/PUT | `/api/config` | Read/write raw config (PUT saves via `config_manager.save()`) |

## Config (`config.yaml`)

- `services`: list — manual definitions override auto-discovered ones (deep-merged per key at `service_manager.py:63`)
- `exclude_services`: list — names hidden from panel
- `categories`: map — display names for section headings

**Known quirk:** `exclude_services` entries use **literal string comparison**, not glob. `systemd-*.service` matches only if literally named `systemd-*.service`. The example config at `config.yaml:134` contains `systemd-*.service` which is misleading.

## Frontend

- Static files in `static/`, served by Flask (no build step)
- `app.js` polls `GET /api/services` every 15s
- Dynamic button event binding via `MutationObserver` on `#service-container` (`app.js:192-195`), not delegated events
- Toast notifications: 3s auto-dismiss via CSS opacity transition (`app.js:197-208`)
- Search debounces at 200ms (`app.js:152-156`)
- Terminal page uses `@wterm/dom` from jsdelivr CDN importmap (`terminal.html:17-23`)

## Conventions

- Config/script comments in Chinese; Python source files have **no comments or docstrings**
- `PLAN.md` at `nas-manager/PLAN.md` documents the WS terminal design rationale (independent services, no Flask-SocketIO)
