package main

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // Postgres / Supabase driver (pure Go)
	_ "modernc.org/sqlite"             // SQLite driver (pure Go)
)

/*
   ======================================================================
   PORTAL DATA STORE + LICENSE SIGNING
   ----------------------------------------------------------------------
   - Users (email + PBKDF2 password hash) belong to an organization.
   - Each organization gets one Ed25519-signed license token.
   - The signing keypair is generated once and persisted; the public key
     is served at /api/pubkey so the desktop app can verify offline.
   ======================================================================
*/

var pdb *sql.DB
var signingPriv ed25519.PrivateKey
var signingPub ed25519.PublicKey

// dbDialect is "sqlite" (default, local file) or "postgres" (external —
// Supabase/managed Postgres, selected when DATABASE_URL is set). It drives
// the small dialect differences: id type and the ?/$N placeholder syntax.
var dbDialect = "sqlite"

// rb rewrites the portable `?` placeholders to `$1,$2,...` for Postgres.
// (Queries here never contain a literal `?`, so the scan is safe.)
func rb(q string) string {
	if dbDialect != "postgres" {
		return q
	}
	var b strings.Builder
	n := 0
	for i := 0; i < len(q); i++ {
		if q[i] == '?' {
			n++
			b.WriteByte('$')
			b.WriteString(strconv.Itoa(n))
		} else {
			b.WriteByte(q[i])
		}
	}
	return b.String()
}

// schemaStmts returns the table DDL for the active dialect. Statements are run
// one-by-one (Postgres' extended protocol rejects multi-statement Exec).
//
//   - users    — every registered account (email + license).
//   - sessions — cookie-backed login persistence.
//   - payments — billing state per account; populated later when Stripe is
//     wired in (status active|canceled|past_due|trialing). "Registered users"
//     = users; "paid users" = payments WHERE status='active'.
func schemaStmts() []string {
	idCol := "id INTEGER PRIMARY KEY AUTOINCREMENT"
	if dbDialect == "postgres" {
		idCol = "id BIGSERIAL PRIMARY KEY"
	}
	return []string{
		`CREATE TABLE IF NOT EXISTS meta (
            key   TEXT PRIMARY KEY,
            value TEXT
        )`,
		`CREATE TABLE IF NOT EXISTS users (
            ` + idCol + `,
            email         TEXT UNIQUE NOT NULL,
            pass_hash     TEXT NOT NULL,
            org_name      TEXT NOT NULL,
            license_token TEXT NOT NULL,
            created       TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS sessions (
            token   TEXT PRIMARY KEY,
            email   TEXT NOT NULL,
            created TEXT NOT NULL
        )`,
		`CREATE TABLE IF NOT EXISTS payments (
            ` + idCol + `,
            email           TEXT NOT NULL,
            plan            TEXT NOT NULL,
            status          TEXT NOT NULL,
            provider        TEXT,
            customer_id     TEXT,
            subscription_id TEXT,
            amount_cents    INTEGER,
            currency        TEXT,
            created         TEXT NOT NULL,
            updated         TEXT NOT NULL
        )`,
		`CREATE INDEX IF NOT EXISTS idx_payments_email ON payments (email)`,
		`CREATE TABLE IF NOT EXISTS revoked_jti (
            jti     TEXT PRIMARY KEY,
            revoked TEXT NOT NULL
        )`,
	}
}

// initStore opens the portal database. When DATABASE_URL is set it connects to
// the external Postgres/Supabase instance; otherwise it falls back to a local
// SQLite file at sqlitePath. The schema is identical (bar id type) either way,
// so user/session/payment data is portable between the two.
func initStore(sqlitePath string) error {
	var conn *sql.DB
	var err error

	if url := strings.TrimSpace(os.Getenv("DATABASE_URL")); url != "" {
		dbDialect = "postgres"
		if conn, err = sql.Open("pgx", url); err != nil {
			return err
		}
		conn.SetMaxOpenConns(10)
		conn.SetMaxIdleConns(2)
		conn.SetConnMaxLifetime(30 * time.Minute)
	} else {
		dbDialect = "sqlite"
		dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)", sqlitePath)
		if conn, err = sql.Open("sqlite", dsn); err != nil {
			return err
		}
		conn.SetMaxOpenConns(1)
	}

	if err = conn.Ping(); err != nil {
		return fmt.Errorf("%s connect: %w", dbDialect, err)
	}
	for _, stmt := range schemaStmts() {
		if _, err = conn.Exec(stmt); err != nil {
			return fmt.Errorf("schema (%s): %w", dbDialect, err)
		}
	}
	pdb = conn
	return loadOrCreateKeypair()
}

// ---- Billing / payments (wired to Stripe webhooks later) ----

// recordPayment upserts the billing state for an account (one active row per
// subscription). Called from the future payment webhook handler.
func recordPayment(email, plan, status, provider, customerID, subID string, amountCents int, currency string) error {
	now := time.Now().Format(time.RFC3339)
	email = strings.ToLower(strings.TrimSpace(email))
	// Update the latest open row for this subscription if present, else insert.
	res, err := pdb.Exec(rb(`UPDATE payments SET plan=?, status=?, provider=?, customer_id=?,
        amount_cents=?, currency=?, updated=? WHERE subscription_id=? AND subscription_id<>''`),
		plan, status, provider, customerID, amountCents, currency, now, subID)
	if err == nil {
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
	}
	_, err = pdb.Exec(rb(`INSERT INTO payments
        (email, plan, status, provider, customer_id, subscription_id, amount_cents, currency, created, updated)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		email, plan, status, provider, customerID, subID, amountCents, currency, now, now)
	return err
}

// userPlan reports the active paid plan for an email, or "" if not a paying
// customer. Lets the app/portal distinguish registered vs paid users.
func userPlan(email string) string {
	var plan string
	err := pdb.QueryRow(rb(`SELECT plan FROM payments WHERE email=? AND status='active'
        ORDER BY updated DESC LIMIT 1`), strings.ToLower(strings.TrimSpace(email))).Scan(&plan)
	if err != nil {
		return ""
	}
	return plan
}

func loadOrCreateKeypair() error {
	var seedB64 string
	err := pdb.QueryRow(`SELECT value FROM meta WHERE key = 'ed25519_seed'`).Scan(&seedB64)
	if err == sql.ErrNoRows {
		_, priv, gerr := ed25519.GenerateKey(rand.Reader)
		if gerr != nil {
			return gerr
		}
		seed := priv.Seed()
		if _, err = pdb.Exec(rb(`INSERT INTO meta (key, value) VALUES ('ed25519_seed', ?)`),
			base64.StdEncoding.EncodeToString(seed)); err != nil {
			return err
		}
		signingPriv = priv
		signingPub = priv.Public().(ed25519.PublicKey)
		return nil
	} else if err != nil {
		return err
	}
	seed, err := base64.StdEncoding.DecodeString(seedB64)
	if err != nil {
		return err
	}
	signingPriv = ed25519.NewKeyFromSeed(seed)
	signingPub = signingPriv.Public().(ed25519.PublicKey)
	return nil
}

// ---- Password hashing (PBKDF2-SHA256, same scheme as the desktop core) ----

const (
	pwIters     = 600_000 // OWASP 2023 PBKDF2-HMAC-SHA256 baseline (new hashes)
	pwLegacy    = 100_000 // hashes written before the format carried its iter count
	pwKeyLen    = 32
)

// hashPassword formats as "pbkdf2:<iter>:<salt>:<dk>" so raising the cost later
// never invalidates existing hashes (they verify at their stored iteration count).
func hashPassword(pw string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	dk, err := pbkdf2.Key(sha256.New, pw, salt, pwIters, pwKeyLen)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("pbkdf2:%d:%s:%s", pwIters,
		base64.StdEncoding.EncodeToString(salt), base64.StdEncoding.EncodeToString(dk)), nil
}

func verifyPassword(pw, stored string) bool {
	parts := strings.Split(strings.TrimPrefix(stored, "pbkdf2:"), ":")
	iters, saltB64, dkB64 := pwLegacy, "", ""
	switch len(parts) {
	case 2: // legacy "salt:dk" (100k, no iter field)
		saltB64, dkB64 = parts[0], parts[1]
	case 3: // "iter:salt:dk"
		n, err := strconv.Atoi(parts[0])
		if err != nil {
			return false
		}
		iters, saltB64, dkB64 = n, parts[1], parts[2]
	default:
		return false
	}
	salt, e1 := base64.StdEncoding.DecodeString(saltB64)
	dk, e2 := base64.StdEncoding.DecodeString(dkB64)
	if e1 != nil || e2 != nil {
		return false
	}
	cand, err := pbkdf2.Key(sha256.New, pw, salt, iters, pwKeyLen)
	if err != nil {
		return false
	}
	return hmac.Equal(cand, dk)
}

// ---- License token: base64url(payload).base64url(ed25519 signature) ----

type LicensePayload struct {
	Org    string `json:"org"`
	Email  string `json:"email"`
	Plan   string `json:"plan"`
	Issued int64  `json:"issued"`
	Exp    int64  `json:"exp"`
	JTI    string `json:"jti"`
}

func mintLicense(org, email, plan string) (string, error) {
	jti := make([]byte, 12)
	if _, err := rand.Read(jti); err != nil {
		return "", err
	}
	payload := LicensePayload{
		Org:    org,
		Email:  email,
		Plan:   plan,
		Issued: time.Now().Unix(),
		Exp:    time.Now().AddDate(1, 0, 0).Unix(), // 1-year license
		JTI:    base64.RawURLEncoding.EncodeToString(jti),
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sig := ed25519.Sign(signingPriv, body)
	return base64.RawURLEncoding.EncodeToString(body) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

// jtiOf extracts the token's unique id (jti) from its signed payload — the id an
// engine denylists to revoke it. Returns "" if the token can't be parsed.
func jtiOf(token string) string {
	parts := strings.SplitN(strings.TrimSpace(token), ".", 2)
	if len(parts) < 1 {
		return ""
	}
	body, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ""
	}
	var p LicensePayload
	if json.Unmarshal(body, &p) != nil {
		return ""
	}
	return p.JTI
}

// revokedJTIs returns every revoked token id — hubs poll this to reject leaked
// tokens org-wide.
func revokedJTIs() ([]string, error) {
	rows, err := pdb.Query(`SELECT jti FROM revoked_jti`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var j string
		if err := rows.Scan(&j); err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

// revokeAndReissue denylists the user's current token and issues a fresh one
// (new jti), so a leaked key can be killed without re-keying the whole org.
func revokeAndReissue(email string) (string, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var org, oldTok string
	if err := pdb.QueryRow(rb(`SELECT org_name, license_token FROM users WHERE email = ?`), email).Scan(&org, &oldTok); err != nil {
		return "", err
	}
	if jti := jtiOf(oldTok); jti != "" {
		if _, err := pdb.Exec(rb(`INSERT INTO revoked_jti (jti, revoked) VALUES (?, ?)
            ON CONFLICT(jti) DO NOTHING`), jti, time.Now().Format(time.RFC3339)); err != nil {
			return "", err
		}
	}
	newTok, err := mintLicense(org, email, "pro")
	if err != nil {
		return "", err
	}
	if _, err := pdb.Exec(rb(`UPDATE users SET license_token = ? WHERE email = ?`), newTok, email); err != nil {
		return "", err
	}
	return newTok, nil
}

// ---- Session operations (cookie-backed login persistence) ----

// createSession mints a random session token bound to an email and stores it.
func createSession(email string) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	tok := base64.RawURLEncoding.EncodeToString(b)
	if _, err := pdb.Exec(rb(`INSERT INTO sessions (token, email, created) VALUES (?, ?, ?)`),
		tok, strings.ToLower(email), time.Now().Format(time.RFC3339)); err != nil {
		return "", err
	}
	return tok, nil
}

// sessionUser resolves a session token to its user (or an error if unknown).
const sessionMaxAgeDays = 30 // matches the cookie MaxAge; sessions expire server-side too

// sweepSessions deletes sessions older than the max age so the table doesn't
// grow forever and a leaked-but-old token can't be replayed indefinitely.
func sweepSessions() {
	cutoff := time.Now().AddDate(0, 0, -sessionMaxAgeDays).Format(time.RFC3339)
	_, _ = pdb.Exec(rb(`DELETE FROM sessions WHERE created < ?`), cutoff)
}

func sessionUser(token string) (*User, error) {
	if token == "" {
		return nil, fmt.Errorf("no session")
	}
	// Server-side expiry: an old session token is rejected even if the client
	// still presents the cookie (which it can replay past MaxAge).
	cutoff := time.Now().AddDate(0, 0, -sessionMaxAgeDays).Format(time.RFC3339)
	var email string
	if err := pdb.QueryRow(rb(`SELECT email FROM sessions WHERE token = ? AND created > ?`), token, cutoff).Scan(&email); err != nil {
		return nil, err
	}
	var org, lic string
	if err := pdb.QueryRow(rb(`SELECT org_name, license_token FROM users WHERE email = ?`), email).Scan(&org, &lic); err != nil {
		return nil, err
	}
	return &User{Email: email, OrgName: org, LicenseToken: lic}, nil
}

func deleteSession(token string) {
	if token != "" {
		_, _ = pdb.Exec(rb(`DELETE FROM sessions WHERE token = ?`), token)
	}
}

// ---- User operations ----

type User struct {
	Email        string
	OrgName      string
	LicenseToken string
}

func createUser(email, password, org string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	org = strings.TrimSpace(org)
	if email == "" || password == "" || org == "" {
		return nil, fmt.Errorf("email, password and organization are required")
	}
	if len(password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	var exists int
	if err := pdb.QueryRow(rb(`SELECT COUNT(*) FROM users WHERE email = ?`), email).Scan(&exists); err != nil {
		return nil, fmt.Errorf("could not check for an existing account: %w", err)
	}
	if exists > 0 {
		return nil, fmt.Errorf("an account with this email already exists")
	}

	ph, err := hashPassword(password)
	if err != nil {
		return nil, err
	}
	token, err := mintLicense(org, email, "pro")
	if err != nil {
		return nil, err
	}
	if _, err = pdb.Exec(rb(`INSERT INTO users (email, pass_hash, org_name, license_token, created)
        VALUES (?, ?, ?, ?, ?)`), email, ph, org, token, time.Now().Format(time.RFC3339)); err != nil {
		return nil, err
	}
	return &User{Email: email, OrgName: org, LicenseToken: token}, nil
}

// findOrCreateOAuthUser logs in (or provisions) a user authenticated via an
// external identity provider. OAuth users get a random unusable password hash.
func findOrCreateOAuthUser(email, org string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return nil, fmt.Errorf("identity provider did not return an email")
	}

	var existingOrg, token string
	err := pdb.QueryRow(rb(`SELECT org_name, license_token FROM users WHERE email = ?`), email).
		Scan(&existingOrg, &token)
	if err == nil {
		return &User{Email: email, OrgName: existingOrg, LicenseToken: token}, nil
	} else if err != sql.ErrNoRows {
		return nil, err
	}

	if org == "" {
		org = orgFromEmail(email)
	}
	rnd := make([]byte, 24)
	if _, e := rand.Read(rnd); e != nil {
		return nil, e
	}
	ph, err := hashPassword("oauth:" + base64.StdEncoding.EncodeToString(rnd))
	if err != nil {
		return nil, err
	}
	token, err = mintLicense(org, email, "pro")
	if err != nil {
		return nil, err
	}
	if _, err = pdb.Exec(rb(`INSERT INTO users (email, pass_hash, org_name, license_token, created)
        VALUES (?, ?, ?, ?, ?)`), email, ph, org, token, time.Now().Format(time.RFC3339)); err != nil {
		return nil, err
	}
	return &User{Email: email, OrgName: org, LicenseToken: token}, nil
}

// orgFromEmail derives a reasonable organization name from an email domain.
func orgFromEmail(email string) string {
	at := strings.LastIndex(email, "@")
	if at < 0 || at+1 >= len(email) {
		return "My Organization"
	}
	domain := email[at+1:]
	label := domain
	if dot := strings.Index(domain, "."); dot > 0 {
		label = domain[:dot]
	}
	if label == "" {
		return "My Organization"
	}
	return strings.ToUpper(label[:1]) + label[1:]
}

func authUser(email, password string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var ph, org, token string
	err := pdb.QueryRow(rb(`SELECT pass_hash, org_name, license_token FROM users WHERE email = ?`), email).
		Scan(&ph, &org, &token)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("invalid email or password")
	} else if err != nil {
		return nil, err
	}
	if !verifyPassword(password, ph) {
		return nil, fmt.Errorf("invalid email or password")
	}
	return &User{Email: email, OrgName: org, LicenseToken: token}, nil
}
