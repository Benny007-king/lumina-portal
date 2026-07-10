package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

/*
   ======================================================================
   LUMINA NETOS — LICENSING PORTAL
   ----------------------------------------------------------------------
   A standalone web app the desktop "Upgrade Now" button links to.
   Users register/log in, receive a unique Ed25519-signed license key
   per organization, and paste it into the desktop for the first
   production login (alongside the admin password).
   ======================================================================
*/

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

const sessionCookie = "lumina_session"

// setSession issues a session for the user and drops an HttpOnly cookie so the
// login persists across visits (Secure when served over TLS). Callers MUST
// check the error: previously a failure here was swallowed and the caller
// still responded 200 with the license key, so the user saw "success" but had
// no session cookie and /api/me immediately 401'd.
func setSession(w http.ResponseWriter, r *http.Request, email string) error {
	tok, err := createSession(email)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: tok, Path: "/",
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		MaxAge:   60 * 60 * 24 * 30, // 30 days
	})
	return nil
}

func clearSession(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		deleteSession(c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", HttpOnly: true, MaxAge: -1})
}

// meHandler returns the logged-in user (from the cookie) so the page can show the
// signed-in state + license/download without re-entering credentials.
func meHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "not signed in"})
		return
	}
	u, err := sessionUser(c.Value)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "session expired"})
		return
	}
	writeJSON(w, 200, map[string]string{"org": u.OrgName, "email": u.Email, "licenseKey": u.LicenseToken})
}

func logoutHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	clearSession(w, r)
	writeJSON(w, 200, map[string]bool{"ok": true})
}

// revokedHandler serves the denylist of revoked token ids. Engines (hubs) poll
// it and reject any org-sync token whose jti is on the list. The ids are opaque
// random values, not secrets, so the list is public (like a CRL).
func revokedHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	list, err := revokedJTIs()
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "failed to list revocations"})
		return
	}
	writeJSON(w, 200, map[string][]string{"revoked": list})
}

// revokeKeyHandler revokes the signed-in user's CURRENT license key and issues a
// fresh one — so a leaked key can be killed without re-keying the whole org.
func revokeKeyHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, 405, map[string]string{"error": "use POST"})
		return
	}
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "not signed in"})
		return
	}
	u, err := sessionUser(c.Value)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": "session expired"})
		return
	}
	newTok, err := revokeAndReissue(u.Email)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "failed to reissue key"})
		return
	}
	log.Printf("revoked + reissued license key for %s", u.Email)
	writeJSON(w, 200, map[string]string{"licenseKey": newTok})
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct{ Email, Password, Org string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	u, err := createUser(req.Email, req.Password, req.Org)
	if err != nil {
		writeJSON(w, 400, map[string]string{"error": err.Error()})
		return
	}
	log.Printf("registered org=%q email=%q", u.OrgName, u.Email)
	if err := setSession(w, r, u.Email); err != nil {
		log.Printf("session creation failed after register email=%q: %v", u.Email, err)
		writeJSON(w, 500, map[string]string{"error": "account created but sign-in failed; please try logging in"})
		return
	}
	writeJSON(w, 200, map[string]string{"org": u.OrgName, "email": u.Email, "licenseKey": u.LicenseToken})
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	var req struct{ Email, Password string }
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid request"})
		return
	}
	u, err := authUser(req.Email, req.Password)
	if err != nil {
		writeJSON(w, 401, map[string]string{"error": err.Error()})
		return
	}
	if err := setSession(w, r, u.Email); err != nil {
		log.Printf("session creation failed after login email=%q: %v", u.Email, err)
		writeJSON(w, 500, map[string]string{"error": "sign-in failed; please try again"})
		return
	}
	writeJSON(w, 200, map[string]string{"org": u.OrgName, "email": u.Email, "licenseKey": u.LicenseToken})
}

// pubkeyHandler exposes the Ed25519 public key so the desktop can verify
// license signatures offline after a one-time fetch.
func pubkeyHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	writeJSON(w, 200, map[string]string{
		"pubkey": base64.StdEncoding.EncodeToString(signingPub),
		"alg":    "ed25519",
	})
}

// signupHandler serves the register/login app (the former home page), now at
// /signup. The app download link is only revealed AFTER registration/login (in
// the success view), so the public site can't be downloaded from without signing
// up. The real URL is injected here (env-configurable).
func signupHandler(w http.ResponseWriter, r *http.Request) {
	dl := os.Getenv("DEMO_DOWNLOAD_URL")
	if dl == "" {
		dl = repoURL + "/releases/latest"
	}
	html := strings.ReplaceAll(portalHTML, "%%DOWNLOAD_URL%%", dl)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

func main() {
	// Data dir: DATA_DIR (Docker sets /data) takes precedence, else %APPDATA%
	// on Windows. filepath.Join keeps paths OS-correct so the same binary runs
	// on Windows and in a Linux container (mirrors the core engine's logic).
	dbPath := "portal.db"
	base := os.Getenv("DATA_DIR")
	if base == "" {
		base = os.Getenv("APPDATA")
	}
	if base != "" {
		dir := filepath.Join(base, "LuminaNetOS")
		_ = os.MkdirAll(dir, 0700)
		dbPath = filepath.Join(dir, "portal.db")
	}
	if err := initStore(dbPath); err != nil {
		log.Fatalf("store init failed: %v", err)
	}

	// Expire stale sessions on boot, then hourly.
	sweepSessions()
	go func() {
		t := time.NewTicker(time.Hour)
		defer t.Stop()
		for range t.C {
			sweepSessions()
		}
	}()

	http.HandleFunc("/", landingHandler)        // marketing landing page
	http.HandleFunc("/docs", docsHandler)       // vendors + security coverage docs
	for _, d := range docLibrary {              // downloadable guides (PDF/MD)
		http.HandleFunc(d.Path, docAssetHandler)
	}
	http.HandleFunc("/releases", releasesPageHandler) // version history page
	http.HandleFunc("/api/latest", latestHandler)     // newest release (app polls this)
	http.HandleFunc("/api/releases", releasesAPIHandler)
	http.HandleFunc("/signup", signupHandler)   // register / login app
	http.HandleFunc("/api/register", registerHandler)
	http.HandleFunc("/api/login", loginHandler)
	http.HandleFunc("/api/me", meHandler)         // who am I (from cookie)
	http.HandleFunc("/api/logout", logoutHandler) // clear session cookie
	http.HandleFunc("/api/pubkey", pubkeyHandler)
	http.HandleFunc("/api/revoked", revokedHandler)     // denylist of revoked token ids (engines poll this)
	http.HandleFunc("/api/revoke-key", revokeKeyHandler) // revoke my current key + reissue a fresh one

	// OAuth (Google / GitHub) — active only when client credentials are set.
	http.HandleFunc("/api/oauth-config", oauthConfigHandler)
	http.HandleFunc("/api/claim", claimHandler)
	http.HandleFunc("/auth/google", startOAuth(googleProvider()))
	http.HandleFunc("/auth/google/callback", callbackOAuth(googleProvider()))
	http.HandleFunc("/auth/github", startOAuth(githubProvider()))
	http.HandleFunc("/auth/github/callback", callbackOAuth(githubProvider()))

	addr := "127.0.0.1:8090"
	if v := os.Getenv("PORTAL_ADDR"); v != "" {
		addr = v
	}
	log.Printf("Lumina licensing portal listening on http://%s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("portal server failed: %v", err)
	}
}
