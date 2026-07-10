# Lumina Portal

The public **licensing portal & website** for Lumina NetOS — split out from the
appliance monorepo so it can be hosted independently and always-on.

It serves:
- the marketing landing page, `/docs`, and `/releases`;
- `/api/latest` — the release feed the desktop app polls for updates;
- sign-up / login (email+password and Google/GitHub OAuth) and issues
  **Ed25519-signed license keys** the desktop verifies offline;
- `/api/pubkey` (the verify key) and `/api/revoked` (the revocation denylist
  hubs poll).

Pure Go, no CGO. Runs as a single static binary / distroless container on `:8090`.

## ⚠️ Before you deploy: the signing key

The portal signs every license with an Ed25519 key stored in its database
(`meta.ed25519_seed`). A **fresh** deployment generates a **new** key — which
would make every license your current portal already issued fail to verify, and
break already-activated desktops.

**So a new deployment must reuse the existing signing key.** Two ways:

- **Recommended — shared database.** Point this deployment at the *same*
  `DATABASE_URL` (Postgres/Supabase) your current portal uses. The seed, users,
  and sessions all live there, so signing is identical. See `assets/SUPABASE-SETUP.md`.
- **SQLite copy.** If you're on the built-in SQLite, copy the old `portal.db`
  (or just its `meta.ed25519_seed` row) into the new deployment's data volume
  before first start.

If this is a brand-new product with no issued licenses yet, ignore the above.

## Local dev

```sh
go run .                       # SQLite at ./portal.db (or %APPDATA% on Windows), :8090
# or with a shared DB:
DATABASE_URL=postgres://... go run .
```

## Deploy to Fly.io

```sh
fly launch --no-deploy          # creates the app; edit `app =` in fly.toml to a unique name

# Pick ONE persistence strategy:
#  (a) shared Postgres/Supabase (recommended — carries the signing key):
fly secrets set DATABASE_URL='postgres://...'
#      → then DELETE the [[mounts]] block from fly.toml.
#  (b) built-in SQLite on a Fly volume:
fly volume create portal_data --size 1

# OAuth + links (optional):
fly secrets set GOOGLE_CLIENT_ID=... GOOGLE_CLIENT_SECRET=... \
               GITHUB_CLIENT_ID=... GITHUB_CLIENT_SECRET=... \
               PAYMENT_URL=... DEMO_DOWNLOAD_URL=...

fly deploy
```

Your portal is now at `https://<app>.fly.dev`.

## Point the desktop app / appliance at it

In the appliance/desktop set **`PUBLIC_PORTAL_URL`** to the hosted URL so
"Upgrade Now", sign-up, and update downloads open the public site:

```
PUBLIC_PORTAL_URL=https://<app>.fly.dev
```

(The engine's server-to-server pubkey/latest fetch uses `PORTAL_URL` separately.)

## Environment

| Var | Purpose |
|---|---|
| `PORTAL_ADDR` | listen address (default `0.0.0.0:8090`) |
| `DATA_DIR` | SQLite dir (default `/data` in the image) |
| `DATABASE_URL` | managed Postgres/Supabase; overrides SQLite (carries the signing key) |
| `GOOGLE_/GITHUB_CLIENT_ID/SECRET` | enable OAuth login |
| `PAYMENT_URL`, `DEMO_DOWNLOAD_URL` | checkout + demo links shown on the site |

## Keeping releases in sync

`/releases` and `/api/latest` come from `releases` in `releases.go`. On each
Lumina NetOS release, prepend the new entry here and redeploy (`fly deploy`).
