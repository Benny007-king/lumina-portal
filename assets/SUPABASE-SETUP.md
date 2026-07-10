# Connecting an external user database (Supabase / Postgres)

The Lumina NetOS licensing portal stores **registered users**, **login
sessions**, and **billing/payment state** in SQL. By default it uses a local
SQLite file (on the `portal-data` volume in Docker). For a production launch —
where you want to *see and manage* who registered and who paid from a hosted
dashboard — point the portal at a managed **Postgres** database such as
**Supabase**.

No code changes are required: set one environment variable and the portal
creates its schema automatically on first start.

---

## 1. Create the database (Supabase)

1. Create a project at <https://supabase.com> (the free tier is enough to start).
2. Go to **Project → Settings → Database → Connection string → URI**.
3. Copy the **Connection pooling** URI (port **6543**) — it is the right choice
   for long-lived / serverless apps:

   ```
   postgres://postgres.<project-ref>:<password>@aws-0-<region>.pooler.supabase.com:6543/postgres
   ```

> Any managed Postgres works too (Neon, Railway, RDS, self-hosted). Just use its
> standard `postgres://user:pass@host:port/dbname` URI.

## 2. Point the portal at it

Set `DATABASE_URL` for the portal process:

- **Docker compose:** uncomment and fill the `DATABASE_URL` line under the
  `portal` service in `deploy/docker-compose.yml`, then `docker compose up -d`.
- **Bare process:** `DATABASE_URL=postgres://... ./portal`

When `DATABASE_URL` is set the portal connects to Postgres; when it is unset it
falls back to the local SQLite file. The schema is identical either way, so you
can develop on SQLite and run production on Supabase.

## 3. Verify

On first start the portal creates four tables: `users`, `sessions`, `payments`,
and `meta` (which holds the Ed25519 license-signing key — **back this up**, it is
how desktop installs verify licenses offline).

Open the Supabase **Table editor** and register a test account on `/signup`; the
new row appears under `users`.

---

## What lives where

| Table      | Purpose                                                              |
|------------|---------------------------------------------------------------------|
| `users`    | Every registered account — email, org, license token, created date. |
| `sessions` | Cookie-backed login sessions (`lumina_session`).                    |
| `payments` | Billing state per account (populated when payment is wired in).      |
| `meta`     | The license-signing keypair seed. **Keep a secure backup.**          |

- **Registered users** = rows in `users`.
- **Paying users** = rows in `payments` with `status = 'active'`.

## Next: wiring payments

The `payments` table is ready for a payment provider (e.g. Stripe). When you
connect checkout, a webhook handler calls the portal's `recordPayment(...)` to
upsert a row (plan, status, Stripe customer/subscription IDs, amount). The
portal can then distinguish free vs. paid accounts via `userPlan(email)`.

> Security: treat `DATABASE_URL` and the Stripe keys as secrets — inject them as
> environment variables / Docker secrets, never commit them. The repo's
> `.gitignore` already excludes secret/DB artifacts.
