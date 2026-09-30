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

   After a successful callback the user is provisioned, the session cookie
   is set directly on the callback response, and the browser lands on
   /signup, which shows the license from the session (/api/me).
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

// --- short-lived CSRF state store ---

type expiring struct {
	value   string
	expires time.Time
}

var (
	oauthMu    sync.Mutex
	stateStore = map[string]expiring{} // state -> provider name
)

// randID returns a fresh random token (used as the OAuth CSRF state). A failed
// read from the OS CSPRNG must never fall
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

// sweepOAuth drops abandoned OAuth states, which are otherwise only removed
// when redeemed.
func sweepOAuth() {
	now := time.Now()
	oauthMu.Lock()
	defer oauthMu.Unlock()
	for k, s := range stateStore {
		if now.After(s.expires) {
			delete(stateStore, k)
		}
	}
}

// stateCookie binds the OAuth state to the browser that started the flow, so a
// callback URL minted in someone else's browser can't log this one in (login CSRF).
const stateCookie = "lumina_oauth_state"

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
			http.Redirect(w, r, "/signup?oauth_error="+url.QueryEscape(p.name+" sign-in is not configured on this server"), http.StatusFound)
			return
		}
		// Start on the canonical host: the state cookie is host-only and the
		// provider returns to portalBaseURL(), so a flow begun on another name
		// (127.0.0.1 vs localhost, onrender.com vs a custom domain) would never
		// see its cookie.
		if base, err := url.Parse(portalBaseURL()); err == nil && base.Host != "" && !strings.EqualFold(base.Host, requestHost(r)) {
			http.Redirect(w, r, portalBaseURL()+r.URL.Path, http.StatusFound)
			return
		}
		state := putState(p.name)
		http.SetCookie(w, &http.Cookie{
			Name: stateCookie, Value: state, Path: "/auth/",
			HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: isHTTPS(r),
			MaxAge: 600,
		})
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
			http.Redirect(w, r, "/signup?oauth_error="+url.QueryEscape(errMsg), http.StatusFound)
			return
		}
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		c, cerr := r.Cookie(stateCookie)
		http.SetCookie(w, &http.Cookie{Name: stateCookie, Value: "", Path: "/auth/", HttpOnly: true, MaxAge: -1})
		provider, ok := takeState(state)
		if !ok || provider != p.name || code == "" || cerr != nil || c.Value != state {
			http.Redirect(w, r, "/signup?oauth_error="+url.QueryEscape("invalid or expired OAuth state"), http.StatusFound)
			return
		}

		token, err := p.exchangeCode(code)
		if err != nil {
			http.Redirect(w, r, "/signup?oauth_error="+url.QueryEscape("token exchange failed: "+err.Error()), http.StatusFound)
			return
		}
		email, err := p.fetchEmail(token)
		if err != nil || email == "" {
			http.Redirect(w, r, "/signup?oauth_error="+url.QueryEscape("could not read your email from "+p.name), http.StatusFound)
			return
		}

		user, err := findOrCreateOAuthUser(email, "")
		if err != nil {
			http.Redirect(w, r, "/signup?oauth_error="+url.QueryEscape(err.Error()), http.StatusFound)
			return
		}
		// Sign in right here, in the browser that started (and proved, via the
		// state cookie) this flow. A bearer "claim" link in between could be sent
		// to someone else and log THEM into this account.
		if err := setSession(w, r, user.Email); err != nil {
			log.Printf("session creation failed after oauth email=%q: %v", user.Email, err)
			http.Redirect(w, r, "/signup?oauth_error="+url.QueryEscape("sign-in failed; please try again"), http.StatusFound)
			return
		}
		http.Redirect(w, r, "/signup", http.StatusFound)
	}
}

// requestHost is the host the browser used (a proxy may pass it on separately).
func requestHost(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		return h
	}
	return r.Host
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
