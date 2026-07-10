# Organization Sync — Design

Lumina NetOS is **organization-aware**: several people install the desktop app
and share an appliance, all activated with the **same license** (one org). They
should never each start from scratch — discovered assets and admin settings
converge across every instance.

This document is the design of record for that subsystem (`core/orgsync.go`).

## The core tension

- **Layer-2 discovery (ARP/MAC, Wi-Fi, NetScaler VIP folding) must run on the
  LAN** — i.e. locally, on each person's machine.
- **Organization sync needs one shared source of truth.**

These pull in opposite directions, so the design is **hybrid**: a central hub
holds the shared state; each desktop scans locally and **pushes up / pulls down**.

## Roles

| Role       | Who                              | Behaviour                                                    |
|------------|----------------------------------|-------------------------------------------------------------|
| **Hub**    | The shared appliance (on-prem)   | Stores the org's assets + settings; serves the union.       |
| **Client** | Each desktop app                 | Scans its LAN; pushes findings; pulls + applies org state.  |

An engine is a **client** when an *Org hub URL* is set (Settings → Organization
Sync) and the **hub** when it is blank. The hub keeps its own scans in the org
store too, so the appliance's view is part of the shared map.

Storage on the hub is local SQLite by default (data stays on-prem); an org may
opt into a cloud Postgres/Supabase backend instead — its choice.

## Authentication — the license token IS the credential

Every member activates with an **Ed25519-signed license** that carries the org.
On every sync call the client presents that token; the hub verifies the
signature **offline** (the same public key it activated with) and trusts the
`org` claim. No extra secret to provision, and a token signed by an unknown key
is rejected (`TestOrgSyncRoundTrip`).

## Data model (hub)

All tables are partitioned by `org` for multi-tenant safety:

- `org_assets(org, node_id, data, updated)` — merged nodes (JSON).
- `org_links(org, link_id, data, updated)` — merged links (JSON).
- `org_config(org, data, updated)` — shared settings blob (LDAP + idle timeout).
- `org_user_mfa(org, username, totp_secret, enabled)` — org-wide MFA enrollments.

## API

| Endpoint                     | Auth            | Purpose                              |
|------------------------------|-----------------|--------------------------------------|
| `POST /api/org/push`         | license token   | Client uploads discovered assets.    |
| `GET  /api/org/pull`         | `X-License-Token` | Client downloads the merged map.   |
| `POST /api/org/settings/push`| license token   | Client uploads LDAP/idle/MFA.        |
| `GET  /api/org/settings/pull`| `X-License-Token` | Client downloads + applies settings.|
| `POST /api/settings/org-sync`| session         | Set this engine's hub URL.           |

## Sync flow

- **Assets (stage 1):** after each scan a client pushes its full view; the hub
  merges by node/link ID (a later push updates a node). `TopologyHandler`
  overlays the org map onto the local one — local discoveries win on ID
  collision — so a scan on one machine appears for everyone.
- **Settings (stage 2):** the hub is authoritative. Changing LDAP, the idle
  timeout, or enrolling MFA pushes the new value up; clients pull on an interval
  (30 s) + on boot and **apply** it. Org-wide MFA means a user's TOTP secret is
  identical on every instance — the permanent fix for "OTP works in the browser
  but not the app".

## Conflict policy

- **Assets:** union by ID; last writer updates a node's fields. Local (live)
  discoveries always win over the cached org copy in the display overlay.
- **Settings:** hub-authoritative, last-writer-wins. Local edits are pushed
  immediately so the next pull is consistent; the 30 s pull window is the only
  race, acceptable for admin-changed config.

## Security notes

- All sync traffic should run over the appliance's HTTPS (self-signed or real).
- The bind password and TOTP secrets travel over that TLS channel and live on
  the org's own appliance — not a vendor cloud — in the default on-prem mode.
- Tokens are signature-verified; an attacker cannot push into another org
  without a validly-signed license for it.

## Staging

1. **Assets** — push + hub overlay (done).
2. **Settings** — LDAP, idle timeout, org-wide MFA (done).
3. **This design doc** + client pull-to-display + conflict policy (done).

### Roadmap / not yet

- Delta sync (currently full-snapshot push/pull) for very large estates.
- Per-field merge timestamps for finer conflict resolution.
- A cloud (Supabase) hub option wired end-to-end (the portal already supports
  Postgres; extending the core hub to Postgres is the remaining work).
