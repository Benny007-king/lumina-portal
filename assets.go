package main

import (
	"embed"
	"net/http"
	"strings"
	"time"
)

/*
   ======================================================================
   EMBEDDED DOCUMENTATION ASSETS  (served under /docs/...)
   ----------------------------------------------------------------------
   The portal ships as a single static binary (distroless, no filesystem),
   so the PDF/markdown guides that live in the repo's docs/ are mirrored
   into portal/assets/ and embedded at build time. They are served with
   long-lived cache headers and a content-disposition that lets the
   browser preview the PDF inline (and download from the viewer).
   ======================================================================
*/

//go:embed assets/*
var docAssets embed.FS

// docAsset describes a downloadable document surfaced on the /docs page.
type docAsset struct {
	Title string // human label
	Desc  string // one-line description
	File  string // file name inside assets/
	Path  string // public URL path
	Kind  string // badge text (PDF / MD)
	Type  string // Content-Type
	Size  string // human size for the UI
}

// docLibrary is the public list of guides shown on /docs and wired to routes.
var docLibrary = []docAsset{
	{
		Title: "Security & Firewall Hardening Guide",
		Desc:  "Recommended firewall ports, TLS settings, credential hygiene and deployment precautions for running Lumina NetOS safely on-prem.",
		File:  "Lumina-Security-Hardening.pdf",
		Path:  "/docs/security-hardening.pdf",
		Kind:  "PDF",
		Type:  "application/pdf",
		Size:  "7 KB",
	},
	{
		Title: "Productization & Deployment Notes",
		Desc:  "Packaging, appliance/Docker deployment, licensing and go-to-market notes for operators.",
		File:  "PRODUCTIZATION.md",
		Path:  "/docs/productization.md",
		Kind:  "MD",
		Type:  "text/markdown; charset=utf-8",
		Size:  "5 KB",
	},
	{
		Title: "External User Database (Supabase / Postgres)",
		Desc:  "Point the portal at a managed Postgres/Supabase database to manage registered and paying users from a hosted dashboard. One env var, no code changes.",
		File:  "SUPABASE-SETUP.md",
		Path:  "/docs/supabase-setup.md",
		Kind:  "MD",
		Type:  "text/markdown; charset=utf-8",
		Size:  "3 KB",
	},
	{
		Title: "Organization Sync — Design",
		Desc:  "How members on one license share discovered assets and admin settings (LDAP, OTP, idle timeout): the hub/client model, license-token auth, data model, API and conflict policy.",
		File:  "ORG-SYNC.md",
		Path:  "/docs/org-sync.md",
		Kind:  "MD",
		Type:  "text/markdown; charset=utf-8",
		Size:  "5 KB",
	},
}

// docAssetHandler serves a single embedded document by its public path.
func docAssetHandler(w http.ResponseWriter, r *http.Request) {
	for _, d := range docLibrary {
		if r.URL.Path == d.Path {
			b, err := docAssets.ReadFile("assets/" + d.File)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", d.Type)
			w.Header().Set("Cache-Control", "public, max-age=86400")
			// PDFs preview inline; markdown is shown as text.
			if strings.HasSuffix(d.File, ".pdf") {
				w.Header().Set("Content-Disposition", `inline; filename="`+d.File+`"`)
			}
			http.ServeContent(w, r, d.File, time.Time{}, strings.NewReader(string(b)))
			return
		}
	}
	http.NotFound(w, r)
}
