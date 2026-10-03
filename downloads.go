package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

/*
   ======================================================================
   GATED DOWNLOADS
   ----------------------------------------------------------------------
   Installers and update packages live as GitHub Releases on the
   lumina-netos-releases repo. That repo is meant to be PRIVATE: the files
   are only handed out through the portal —
     - /download/installer, /download/offline  → a signed-in customer;
     - /api/update/package                     → an engine presenting a
                                                  valid, unrevoked license.
   The portal uses a read-only token (GITHUB_RELEASES_TOKEN, fine-grained,
   Contents: read on that one repo) to ask GitHub for a short-lived download
   link and redirects to it, so the files never stream through this server
   and the token never leaves it. Without the token it falls back to the
   public download URL (only works while the repo is still public).
   ======================================================================
*/

const releasesRepoPath = "Benny007-king/lumina-netos-releases"

// Asset names every release uploads (see CLAUDE.md → Installers).
const (
	assetInstaller = "LuminaNetOS-Setup-x64.exe"
	assetMSI       = "LuminaNetOS-x64.msi"
)

// offlineAssetName is the signed update package for a version.
func offlineAssetName(version string) string { return "LuminaNetOS-" + version + "-x64.lupdate" }

// validVersion accepts only dotted numbers ("1.31.0").
func validVersion(v string) bool {
	if v == "" || len(v) > 20 {
		return false
	}
	for _, c := range v {
		if c != '.' && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// resolveVersion maps "" / "latest" to the newest release this portal lists.
func resolveVersion(v string) (string, bool) {
	if v == "" || v == "latest" {
		if len(releases) == 0 {
			return "", false
		}
		return releases[0].Version, true
	}
	return v, validVersion(v)
}

var assetCache = struct {
	sync.Mutex
	urls map[string]struct {
		api     string
		fetched time.Time
	}
}{}

// releaseAssetURL returns a URL the client can download the asset from.
func releaseAssetURL(version, name string) (string, error) {
	token := strings.TrimSpace(os.Getenv("GITHUB_RELEASES_TOKEN"))
	if token == "" {
		return fmt.Sprintf("https://github.com/%s/releases/download/v%s/%s", releasesRepoPath, version, name), nil
	}
	apiURL, err := assetAPIURL(token, version, name)
	if err != nil {
		return "", err
	}
	// Ask for the binary; GitHub answers with a redirect to a link that is
	// valid for a few minutes and carries no credentials of ours.
	req, _ := http.NewRequest(http.MethodGet, apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/octet-stream")
	client := &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	resp.Body.Close()
	if loc := resp.Header.Get("Location"); resp.StatusCode/100 == 3 && loc != "" {
		return loc, nil
	}
	return "", fmt.Errorf("file host answered %d", resp.StatusCode)
}

// assetAPIURL finds an asset's API URL in the release for version (cached).
func assetAPIURL(token, version, name string) (string, error) {
	key := version + "/" + name
	assetCache.Lock()
	if assetCache.urls == nil {
		assetCache.urls = map[string]struct {
			api     string
			fetched time.Time
		}{}
	}
	if c, ok := assetCache.urls[key]; ok && time.Since(c.fetched) < 10*time.Minute {
		assetCache.Unlock()
		return c.api, nil
	}
	assetCache.Unlock()

	req, _ := http.NewRequest(http.MethodGet, fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/v%s", releasesRepoPath, version), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := (&http.Client{Timeout: 20 * time.Second}).Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return "", errors.New("no such release")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file host answered %d", resp.StatusCode)
	}
	var rel struct {
		Assets []struct {
			Name string `json:"name"`
			URL  string `json:"url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	for _, a := range rel.Assets {
		if a.Name == name {
			assetCache.Lock()
			assetCache.urls[key] = struct {
				api     string
				fetched time.Time
			}{a.URL, time.Now()}
			assetCache.Unlock()
			return a.URL, nil
		}
	}
	return "", errors.New("this release has no " + name)
}

// verifyCustomerLicense accepts a license token this portal signed that has
// not expired and was not revoked. (Paid-plan checks go here once billing is
// wired: today every registered account gets a license.)
func verifyCustomerLicense(token string) (*LicensePayload, error) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return nil, errors.New("malformed license")
	}
	body, err1 := base64.RawURLEncoding.DecodeString(parts[0])
	sig, err2 := base64.RawURLEncoding.DecodeString(parts[1])
	if err1 != nil || err2 != nil || !ed25519.Verify(signingPub, body, sig) {
		return nil, errors.New("license not issued by this portal")
	}
	var p LicensePayload
	if err := json.Unmarshal(body, &p); err != nil {
		return nil, errors.New("malformed license")
	}
	if p.Exp != 0 && time.Now().Unix() > p.Exp {
		return nil, errors.New("license expired")
	}
	if revoked, err := revokedJTIs(); err == nil {
		for _, j := range revoked {
			if j == p.JTI {
				return nil, errors.New("license revoked")
			}
		}
	}
	return &p, nil
}

// signedInUser returns the portal account of the request's session, or nil.
func signedInUser(r *http.Request) *User {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return nil
	}
	u, err := sessionUser(c.Value)
	if err != nil {
		return nil
	}
	return u
}

// downloadHandler serves /download/installer, /download/msi and
// /download/offline to signed-in customers.
func downloadHandler(w http.ResponseWriter, r *http.Request) {
	if signedInUser(r) == nil {
		http.Redirect(w, r, "/signup?next=download", http.StatusFound)
		return
	}
	version, ok := resolveVersion(r.URL.Query().Get("version"))
	if !ok {
		http.Error(w, "Unknown version", http.StatusBadRequest)
		return
	}
	var name string
	switch strings.TrimPrefix(r.URL.Path, "/download/") {
	case "installer":
		name = assetInstaller
	case "msi":
		name = assetMSI
	case "offline":
		name = offlineAssetName(version)
	default:
		http.NotFound(w, r)
		return
	}
	url, err := releaseAssetURL(version, name)
	if err != nil {
		http.Error(w, "Download unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}

// updatePackageHandler hands an engine the signed update package for a version
// (default: latest) when it presents a valid license (X-License-Key).
func updatePackageHandler(w http.ResponseWriter, r *http.Request) {
	if _, err := verifyCustomerLicense(r.Header.Get("X-License-Key")); err != nil {
		http.Error(w, "Updates require a valid license: "+err.Error(), http.StatusForbidden)
		return
	}
	version, ok := resolveVersion(r.URL.Query().Get("version"))
	if !ok {
		http.Error(w, "Unknown version", http.StatusBadRequest)
		return
	}
	url, err := releaseAssetURL(version, offlineAssetName(version))
	if err != nil {
		http.Error(w, "Update unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, url, http.StatusFound)
}
