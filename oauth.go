package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

/*
   ======================================================================
   OAUTH 2.0 (Authorization Code) — Google & GitHub
   ----------------------------------------------------------------------
   Credentials come from environment variables; the flow is fully wired
   and works as soon as they are provided:
     GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET
     GITHUB_CLIENT_ID / GITHUB_CLIENT_SECRET
     PORTAL_BASE_URL  (default http://localhost:8090)

   After a successful callback the user is provisioned and their license
   key is stashed under a one-time "claim" id; the browser is redirected
   to /?claim=<id> and the SPA fetches the key via /api/claim.
   ======================================================================
*/

type oauthProvider struct {
	name         string
	clientID     string
	clientSecret string
	authURL      string
	tokenURL     string
	scope        string
}

func googleProvider() oauthProvider {
	return oauthProvider{
		name:         "google",
		clientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		clientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		authURL:      "https://accounts.google.com/o/oauth2/v2/auth",
		tokenURL:     "https://oauth2.googleapis.com/token",
		scope:        "openid email profile",
	}
}

func githubProvider() oauthProvider {
	return oauthProvider{
		name:         "github",
		clientID:     os.Getenv("GITHUB_CLIENT_ID"),
		clientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
		authURL:      "https://github.com/login/oauth/authorize",
		tokenURL:     "https://github.com/login/oauth/access_token",
		scope:        "read:user user:email",
	}
}

func (p oauthProvider) configured() bool { return p.clientID != "" && p.clientSecret != "" }

func portalBaseURL() string {
	if v := os.Getenv("PORTAL_BASE_URL"); v != "" {
		return strings.TrimRight(v, "/")
	}
	return "http://localhost:8090"
}

// --- short-lived CSRF state + one-time license claim stores ---

type expiring struct {
	value   string
	expires time.Time
}

var (
	oauthMu     sync.Mutex
	stateStore  = map[string]expiring{}   // state -> provider name
	claimStore  = map[string]claimRecord{} // claim id -> license payload
)

type claimRecord struct {
	user    User
	expires time.Time
}

// randID returns a fresh random token (used as the OAuth CSRF state and as the
// post-login claim ID). A failed read from the OS CSPRNG must never fall
// through to a partially-random/predictable value — panic (net/http recovers
// per-request and 500s) rather than mint a guessable state token.
func randID() string {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		panic("randID: crypto/rand unavailable: " + err.Error())
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func putState(provider string) string {
	id := randID()
	oauthMu.Lock()
	stateStore[id] = expiring{provider, time.Now().Add(10 * time.Minute)}
	oauthMu.Unlock()
	return id
}

func takeState(id string) (string, bool) {
	oauthMu.Lock()
	defer oauthMu.Unlock()
	s, ok := stateStore[id]
	delete(stateStore, id)
	if !ok || time.Now().After(s.expires) {
		return "", false
	}
	return s.value, true
}

func putClaim(u User) string {
	id := randID()
	oauthMu.Lock()
	claimStore[id] = claimRecord{u, time.Now().Add(5 * time.Minute)}
	oauthMu.Unlock()
	return id
}

func takeClaim(id string) (User, bool) {
	oauthMu.Lock()
	defer oauthMu.Unlock()
	c, ok := claimStore[id]
	delete(claimStore, id)
	if !ok || time.Now().After(c.expires) {
		return User{}, false
	}
	return c.user, true
}

// --- handlers ---

func oauthConfigHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	writeJSON(w, 200, map[string]bool{
		"google": googleProvider().configured(),
		"github": githubProvider().configured(),
	})
}

func startOAuth(p oauthProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !p.configured() {
			http.Redirect(w, r, "/?oauth_error="+url.QueryEscape(p.name+" sign-in is not configured on this server"), http.StatusFound)
			return
		}
		state := putState(p.name)
		q := url.Values{}
		q.Set("client_id", p.clientID)
		q.Set("redirect_uri", portalBaseURL()+"/auth/"+p.name+"/callback")
		q.Set("response_type", "code")
		q.Set("scope", p.scope)
		q.Set("state", state)
		if p.name == "google" {
			q.Set("access_type", "online")
			q.Set("prompt", "select_account")
		}
		http.Redirect(w, r, p.authURL+"?"+q.Encode(), http.StatusFound)
	}
}

func callbackOAuth(p oauthProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if errMsg := r.URL.Query().Get("error"); errMsg != "" {
			http.Redirect(w, r, "/?oauth_error="+url.QueryEscape(errMsg), http.StatusFound)
			return
		}
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		provider, ok := takeState(state)
		if !ok || provider != p.name || code == "" {
			http.Redirect(w, r, "/?oauth_error="+url.QueryEscape("invalid or expired OAuth state"), http.StatusFound)
			return
		}

		token, err := p.exchangeCode(code)
		if err != nil {
			http.Redirect(w, r, "/?oauth_error="+url.QueryEscape("token exchange failed: "+err.Error()), http.StatusFound)
			return
		}
		email, err := p.fetchEmail(token)
		if err != nil || email == "" {
			http.Redirect(w, r, "/?oauth_error="+url.QueryEscape("could not read your email from "+p.name), http.StatusFound)
			return
		}

		user, err := findOrCreateOAuthUser(email, "")
		if err != nil {
			http.Redirect(w, r, "/?oauth_error="+url.QueryEscape(err.Error()), http.StatusFound)
			return
		}
		claim := putClaim(*user)
		http.Redirect(w, r, "/?claim="+url.QueryEscape(claim), http.StatusFound)
	}
}

func claimHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	u, ok := takeClaim(r.URL.Query().Get("id"))
	if !ok {
		writeJSON(w, 404, map[string]string{"error": "claim expired or not found"})
		return
	}
	if err := setSession(w, r, u.Email); err != nil {
		log.Printf("session creation failed after oauth claim email=%q: %v", u.Email, err)
		writeJSON(w, 500, map[string]string{"error": "sign-in failed; please try again"})
		return
	}
	writeJSON(w, 200, map[string]string{"org": u.OrgName, "email": u.Email, "licenseKey": u.LicenseToken})
}

// --- provider HTTP exchanges ---

func (p oauthProvider) exchangeCode(code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", p.clientID)
	form.Set("client_secret", p.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", portalBaseURL()+"/auth/"+p.name+"/callback")
	form.Set("grant_type", "authorization_code")

	req, _ := http.NewRequest("POST", p.tokenURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("provider status %d", resp.StatusCode)
	}
	var out struct {
		AccessToken string `json:"access_token"`
	}
	if err = json.Unmarshal(body, &out); err != nil || out.AccessToken == "" {
		return "", fmt.Errorf("no access token returned")
	}
	return out.AccessToken, nil
}

func (p oauthProvider) fetchEmail(token string) (string, error) {
	switch p.name {
	case "google":
		return googleVerifiedEmail(token)
	case "github":
		return githubPrimaryEmail(token)
	}
	return "", fmt.Errorf("unknown provider")
}

// googleVerifiedEmail returns the Google account email ONLY if Google reports it
// as verified — otherwise an attacker could sign in / provision an account for an
// email they don't control. (email_verified may be a bool or the string "true".)
func googleVerifiedEmail(token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://openidconnect.googleapis.com/v1/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var m map[string]any
	if err = json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return "", err
	}
	email, _ := m["email"].(string)
	if email == "" {
		return "", fmt.Errorf("google email not present")
	}
	verified := false
	switch v := m["email_verified"].(type) {
	case bool:
		verified = v
	case string:
		verified = v == "true"
	}
	if !verified {
		return "", fmt.Errorf("google email is not verified")
	}
	return email, nil
}

func githubPrimaryEmail(token string) (string, error) {
	req, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err = json.NewDecoder(resp.Body).Decode(&emails); err != nil {
		return "", err
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email, nil
		}
	}
	for _, e := range emails {
		if e.Verified {
			return e.Email, nil
		}
	}
	return "", fmt.Errorf("no verified email")
}
