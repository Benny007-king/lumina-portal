package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// OAuth results must land on /signup — the only page that consumes ?claim= /
// ?oauth_error=. Redirecting to / (the landing page) silently broke sign-in.
func TestOAuthRedirectsLandOnSignup(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "")
	t.Setenv("GOOGLE_CLIENT_SECRET", "")
	rec := httptest.NewRecorder()
	startOAuth(googleProvider())(rec, httptest.NewRequest("GET", "/auth/google", nil))
	if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/signup?oauth_error=") {
		t.Fatalf("unconfigured provider redirect = %q, want /signup?oauth_error=...", loc)
	}
}

func TestOAuthStartSetsStateCookie(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret")
	t.Setenv("PORTAL_BASE_URL", "http://example.com") // httptest's default request host
	rec := httptest.NewRecorder()
	startOAuth(googleProvider())(rec, httptest.NewRequest("GET", "/auth/google", nil))
	var state string
	for _, c := range rec.Result().Cookies() {
		if c.Name == stateCookie {
			state = c.Value
		}
	}
	if state == "" || !strings.Contains(rec.Header().Get("Location"), "state="+state) {
		t.Fatalf("state cookie %q not matching the authorize URL %q", state, rec.Header().Get("Location"))
	}
}

// The state cookie is host-only, so a flow started on a non-canonical host must
// first bounce to PORTAL_BASE_URL's host — otherwise the callback (which the
// provider sends to PORTAL_BASE_URL) never sees the cookie.
func TestOAuthStartBouncesToCanonicalHost(t *testing.T) {
	t.Setenv("GOOGLE_CLIENT_ID", "id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret")
	t.Setenv("PORTAL_BASE_URL", "https://portal.example.org")
	req := httptest.NewRequest("GET", "http://lumina-portal.onrender.com/auth/google", nil)
	rec := httptest.NewRecorder()
	startOAuth(googleProvider())(rec, req)
	if loc := rec.Header().Get("Location"); loc != "https://portal.example.org/auth/google" {
		t.Fatalf("redirect = %q, want the canonical host", loc)
	}
	if len(rec.Result().Cookies()) != 0 {
		t.Fatal("no state cookie should be set on the non-canonical host")
	}
}

// A callback whose state wasn't started in THIS browser (no/mismatched cookie)
// must be rejected before any token exchange — this blocks OAuth login CSRF.
func TestOAuthCallbackRejectsStateFromAnotherBrowser(t *testing.T) {
	for name, cookie := range map[string]string{"no cookie": "", "mismatched cookie": "someone-elses-state"} {
		t.Run(name, func(t *testing.T) {
			state := putState("google")
			req := httptest.NewRequest("GET", "/auth/google/callback?code=abc&state="+state, nil)
			if cookie != "" {
				req.AddCookie(&http.Cookie{Name: stateCookie, Value: cookie})
			}
			rec := httptest.NewRecorder()
			callbackOAuth(googleProvider())(rec, req)
			if loc := rec.Header().Get("Location"); !strings.HasPrefix(loc, "/signup?oauth_error=") {
				t.Fatalf("redirect = %q, want /signup?oauth_error=...", loc)
			}
			if _, ok := takeState(state); ok {
				t.Fatal("state should be consumed even on rejection")
			}
		})
	}
}

// Behind a TLS-terminating proxy (Render/Fly) r.TLS is nil; the session cookie
// must still be Secure.
func TestIsHTTPSBehindProxy(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	if isHTTPS(r) {
		t.Fatal("plain HTTP request reported as HTTPS")
	}
	r.Header.Set("X-Forwarded-Proto", "https")
	if !isHTTPS(r) {
		t.Fatal("X-Forwarded-Proto: https not honoured")
	}
}

func TestSweepOAuthDropsExpiredOnly(t *testing.T) {
	oauthMu.Lock()
	stateStore["old"] = expiring{"google", time.Now().Add(-time.Minute)}
	stateStore["new"] = expiring{"google", time.Now().Add(time.Minute)}
	oauthMu.Unlock()
	sweepOAuth()
	oauthMu.Lock()
	defer oauthMu.Unlock()
	_, oldState := stateStore["old"]
	_, newState := stateStore["new"]
	if oldState || !newState {
		t.Fatalf("sweep wrong: old state kept=%v, fresh state kept=%v", oldState, newState)
	}
	delete(stateStore, "new")
}
