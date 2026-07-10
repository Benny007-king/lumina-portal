# Lumina NetOS — Productization & Upgrade Plan

This document plans how to ship Lumina NetOS as a real product across three
delivery models and how to manage customer upgrades safely.

## 1. Component split (the key decision)

The codebase already separates cleanly into three deployables:

| Component | What it is | Ships as |
|-----------|-----------|----------|
| **Desktop app** (`src/` + `src-tauri/` + `core/` sidecar) | Tauri GUI + Go engine for a single operator workstation | Signed installer (.msi / .dmg / .AppImage) |
| **Core engine** (`core/`) | Headless discovery + API server (`:8080`) | Docker image **or** FreeBSD appliance |
| **Portal** (`portal/`) | Public landing page + licensing/billing (`:8090`) | Docker image (SaaS) |

The Go engine is **pure-Go (no CGO)** — `modernc.org/sqlite`, `gosnmp`, `go-ldap`
are all pure-Go — so it cross-compiles to any OS/arch with one command. That is
what makes both the container and the FreeBSD appliance cheap to build.

## 2. Delivery model A — Docker (recommended first step)

Artifacts already added under `deploy/`:
- `Dockerfile.core` — static distroless image of the headless engine.
- `Dockerfile.portal` — static distroless image of the portal.
- `docker-compose.yml` — runs both; `core` uses `network_mode: host` so the
  discovery engine can read ARP and reach the LAN.

```bash
cd deploy && docker compose up --build -d
```

DB + logs persist on named volumes (`APPDATA=/data` inside the container).
This is the fastest path to a server install and to a SaaS portal.

## 3. Delivery model B — FreeBSD appliance (the "pfSense-style" product)

Goal: a hardened image the customer boots on a box/VM; the engine runs as a
service and serves the React UI on `:443`. FreeBSD is a good fit (stable ABI,
pf firewall, ZFS, jails).

**Build the engine for FreeBSD (no CGO, so trivial):**
```bash
cd core && GOOS=freebsd GOARCH=amd64 CGO_ENABLED=0 \
  /c/Program\ Files/Go/bin/go.exe build -o lumina-core-freebsd .
```

**Appliance layout:**
- Run `lumina-core` under an `rc.d` service (`/usr/local/etc/rc.d/lumina`),
  unprivileged user `lumina`, DB on ZFS dataset `/var/db/lumina`.
- Serve the built React bundle (static) via the same Go process (add a static
  file route) or via nginx in front, terminating TLS.
- Lock down with `pf`: only management VLAN may reach `:443`.
- Package with `poudriere`/`pkg` or ship a ZFS image (mfsBSD-style) for
  bare-metal/VM install.

**Why a separate static-serve route:** today the React app is served by Vite in
dev and by Tauri in the desktop build. For the appliance, add a Go route that
serves `dist/` (embed via `go:embed`) so the engine is self-contained — one
binary, no nginx required for an MVP.

## 4. Customer upgrades (must be designed before first sale)

### Versioning
- Semantic version on the engine (`/api/version` → `{version, channel}`).
- Channels: `stable`, `beta`. License ties a customer to a channel.

### Database migrations (already half-built)
- `db.go` uses an idempotent `ensureColumn(...)` pattern — every release adds
  forward-only migrations there. Formalize it: a `schema_version` row + an
  ordered migration list applied on boot. **Never** drop columns in a minor
  release; deprecate then remove in a major.
- Back up the SQLite DB (WAL checkpoint + copy) before applying migrations so a
  failed upgrade can roll back.

### Update delivery per model
- **Desktop:** Tauri's built-in updater (signed release feed). The desktop polls
  the portal for the latest signed build for its license channel; user clicks
  "Update". Keep the current sidecar-rebuild discipline (kill old `core` before
  swapping the binary).
- **Docker:** new image tag; `docker compose pull && up -d`. Migrations run on
  container start. Roll back by pinning the previous tag.
- **FreeBSD appliance:** dual-slot (A/B) root datasets on ZFS — write the new
  release to the inactive slot, switch the boot environment (`bectl`), reboot,
  and auto-rollback if health check fails. This gives atomic, reversible
  upgrades like a real appliance.

### License-gated upgrades
- The portal issues Ed25519-signed licenses (already implemented). Extend the
  token with `channel` + `maxVersion` so an expired subscription still runs but
  won't pull new feature releases. The engine verifies offline.

### Telemetry / update check (opt-in)
- A lightweight `GET portal/api/latest?channel=stable&v=<cur>` the engine calls
  to surface an "update available" banner. No data leaves the customer beyond
  current version + channel; keep it opt-out friendly for air-gapped sites.

## 5. Suggested rollout order
1. **Docker** portal (SaaS landing + billing) — already buildable.
2. **Docker** core for server installs.
3. **Tauri auto-update** for the desktop.
4. **FreeBSD appliance** with A/B ZFS upgrades for enterprise/air-gapped.
