package main

import (
	"strings"
	"testing"
)

// Ensures the cinematic landing template parses and renders with real data.
func TestLandingTemplateRenders(t *testing.T) {
	data := map[string]any{
		"Posts":       blogPosts,
		"Roadmap":     roadmap,
		"CheckoutURL": "/signup?plan=pro",
		"DemoURL":     "https://example.com/demo",
	}
	var sb strings.Builder
	if err := landingTmpl.Execute(&sb, data); err != nil {
		t.Fatalf("landing template execute failed: %v", err)
	}
	out := sb.String()
	for _, want := range []string{"LUMINA", "net3d", "Roadmap", "Buy Pro", "Watch a tutorial", "v2.0", "/docs"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered landing page missing %q", want)
		}
	}
	// Vendor catalogue must NOT be on the landing page anymore (moved to /docs).
	if strings.Contains(out, "Citrix Virtual Apps/Desktops") {
		t.Error("vendor catalogue should live on /docs, not the landing page")
	}
}

// Ensures the releases template + /api/latest data render correctly.
func TestReleasesTemplateRenders(t *testing.T) {
	var sb strings.Builder
	if err := releasesTmpl.Execute(&sb, map[string]any{"Releases": releases, "Repo": repoURL}); err != nil {
		t.Fatalf("releases template execute failed: %v", err)
	}
	out := sb.String()
	for _, want := range []string{"v1.6.0", "Latest", "Version history", "v1.4.0"} {
		if !strings.Contains(out, want) {
			t.Errorf("releases page missing %q", want)
		}
	}
	if len(releases) == 0 || releases[0].Version != "1.24.0" {
		t.Errorf("latest release should be 1.24.0, got %+v", releases)
	}
}

// Ensures the docs template parses and renders the vendor list + checks.
func TestDocsTemplateRenders(t *testing.T) {
	data := map[string]any{"Vendors": vendorCatalog, "Checks": securityChecks}
	var sb strings.Builder
	if err := docsTmpl.Execute(&sb, data); err != nil {
		t.Fatalf("docs template execute failed: %v", err)
	}
	out := sb.String()
	for _, want := range []string{"Cisco", "Nutanix", "Synology DSM", "Getting started", "Security AI coverage"} {
		if !strings.Contains(out, want) {
			t.Errorf("rendered docs page missing %q", want)
		}
	}
	if !strings.Contains(out, `class="badge">C<`) {
		t.Error("expected a vendor badge initial on /docs")
	}
}
