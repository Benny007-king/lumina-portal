package main

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatedDownloads(t *testing.T) {
	if err := initStore(filepath.Join(t.TempDir(), "portal.db")); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { pdb.Close() }) // before TempDir removal (cleanups run last-in first-out)
	t.Setenv("GITHUB_RELEASES_TOKEN", "") // public fallback URL, no network

	// Website download without a session → sent to sign in, no file link.
	w := httptest.NewRecorder()
	downloadHandler(w, httptest.NewRequest(http.MethodGet, "/download/installer", nil))
	if w.Code != http.StatusFound || !strings.HasPrefix(w.Header().Get("Location"), "/signup") {
		t.Fatalf("anonymous download: %d → %q", w.Code, w.Header().Get("Location"))
	}

	// Engine update package: refused without / with a bad license.
	for _, key := range []string{"", "garbage", "eyJ4IjoxfQ.AAAA"} {
		r := httptest.NewRequest(http.MethodGet, "/api/update/package", nil)
		r.Header.Set("X-License-Key", key)
		w = httptest.NewRecorder()
		updatePackageHandler(w, r)
		if w.Code != http.StatusForbidden {
			t.Errorf("license %q: got %d, want 403", key, w.Code)
		}
	}

	// A license this portal issued → redirect to the package of the latest release.
	tok, err := mintLicense("acme", "ops@acme.example", "pro")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest(http.MethodGet, "/api/update/package", nil)
	r.Header.Set("X-License-Key", tok)
	w = httptest.NewRecorder()
	updatePackageHandler(w, r)
	want := "/releases/download/v" + releases[0].Version + "/" + offlineAssetName(releases[0].Version)
	if w.Code != http.StatusFound || !strings.HasSuffix(w.Header().Get("Location"), want) {
		t.Fatalf("licensed package: %d → %q (want …%s)", w.Code, w.Header().Get("Location"), want)
	}

	// Tampered license (payload changed, signature kept) → refused.
	parts := strings.SplitN(tok, ".", 2)
	r = httptest.NewRequest(http.MethodGet, "/api/update/package", nil)
	r.Header.Set("X-License-Key", "eyJvcmciOiJldmlsIn0."+parts[1])
	w = httptest.NewRecorder()
	updatePackageHandler(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("tampered license: got %d, want 403", w.Code)
	}

	// Path traversal / odd versions are rejected.
	r = httptest.NewRequest(http.MethodGet, "/api/update/package?version=../../x", nil)
	r.Header.Set("X-License-Key", tok)
	w = httptest.NewRecorder()
	updatePackageHandler(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("bad version: got %d, want 400", w.Code)
	}
}
