package main

import (
	"encoding/json"
	"html/template"
	"net/http"
)

/*
   ======================================================================
   RELEASE / VERSION MANAGEMENT (the "website" side)
   ----------------------------------------------------------------------
   The portal is the source of truth for "what's the latest version".
   The desktop engine polls /api/latest and compares to its own build.
   Releases map 1:1 to git tags (v<version>); DownloadURL points at the
   GitHub release page for that tag.
   ======================================================================
*/

type Release struct {
	Version     string   `json:"version"`
	Date        string   `json:"date"`
	Channel     string   `json:"channel"`
	Notes       []string `json:"notes"`
	DownloadURL string   `json:"downloadUrl"`
	GitTag      string   `json:"gitTag"`
}

const repoURL = "https://github.com/Benny007-king/-Lumina-NetOS"

// releases — newest first. Add a new entry (and push a matching git tag) per release.
var releases = []Release{
	{"1.24.2", "2026-07-11", "stable",
		[]string{
			"The post-update \"What's New\" dialog now shows only what changed in the latest update instead of the entire release history, and it resizes to the screen with its own scrollbar so a long changelog never runs off-screen",
		},
		repoURL + "/releases/tag/v1.24.2", "v1.24.2"},
	{"1.24.1", "2026-07-11", "stable",
		[]string{
			"The licensing portal now honours the PORT environment variable and binds all interfaces, so it deploys cleanly to Fly.io / Render / Railway / Cloud Run (no more port-mismatch health-check timeouts)",
		},
		repoURL + "/releases/tag/v1.24.1", "v1.24.1"},
	{"1.24.0", "2026-07-10", "stable",
		[]string{
			"The desktop \"Upgrade Now\" button and update checks can now point at a published, always-on website instead of a locally-running portal — set PUBLIC_PORTAL_URL on the appliance to your hosted portal URL",
			"Fixes the releases/website not reflecting the newest version: the portal image is now rebuilt on every release so this page always shows the latest",
		},
		repoURL + "/releases/tag/v1.24.0", "v1.24.0"},
	{"1.23.1", "2026-07-04", "stable",
		[]string{
			"Internal cleanup (over-engineering pass): removed a redundant field from the organization-sync pull response; no behaviour change",
		},
		repoURL + "/releases/tag/v1.23.1", "v1.23.1"},
	{"1.23.0", "2026-07-04", "stable",
		[]string{
			"The credential master-key protector can now be rotated with no downtime — set the new secret plus LUMINA_MASTER_KEY_OLD, start once to re-seal, then drop the old one",
			"Licensing-portal sessions now expire server-side after 30 days and are swept periodically, so an old session cookie can't be replayed indefinitely",
		},
		repoURL + "/releases/tag/v1.23.0", "v1.23.0"},
	{"1.22.0", "2026-07-04", "stable",
		[]string{
			"Organization sync now scales to large estates: members pull only the assets that changed since their last sync (with a periodic full reconcile) instead of the whole map every 30 seconds, and the hub writes each batch in a single transaction",
			"The hub also garbage-collects assets that have left the estate, so the shared organization map no longer grows forever",
		},
		repoURL + "/releases/tag/v1.22.0", "v1.22.0"},
	{"1.21.0", "2026-07-04", "stable",
		[]string{
			"License keys can now be revoked — if a key leaks, click \"Revoke & Reissue Key\" on your account page to kill the old one and get a fresh key, without re-keying the whole organization",
			"Engines/hubs poll the portal's revocation list and reject any organization-sync request that uses a revoked key (within ~15 minutes); the last-known list is cached so the check keeps working offline",
		},
		repoURL + "/releases/tag/v1.21.0", "v1.21.0"},
	{"1.20.0", "2026-07-04", "stable",
		[]string{
			"Organization settings sync (LDAP + MFA) can now be gated by a shared admin token: set the same LUMINA_ORG_ADMIN_TOKEN on the hub and every member and only holders of that token can push or pull the org's directory config and MFA enrollments",
			"This stops a member who only has the product licence key from rewriting org-wide LDAP or planting an MFA enrollment. The asset map still syncs on the licence alone; leaving the token unset keeps the previous behaviour (with a startup warning)",
		},
		repoURL + "/releases/tag/v1.20.0", "v1.20.0"},
	{"1.19.0", "2026-07-04", "stable",
		[]string{
			"MFA/TOTP secrets are now encrypted at rest — on each device and on the organization hub — with the same key that already protects stored device credentials, so a stolen database no longer exposes live authenticator seeds",
			"Backward compatible: existing 2FA secrets keep working and are re-sealed transparently; the raw seed only exists in memory and over the encrypted, certificate-pinned org-sync channel",
		},
		repoURL + "/releases/tag/v1.19.0", "v1.19.0"},
	{"1.18.0", "2026-07-04", "stable",
		[]string{
			"Security: password hashing raised to 600,000 PBKDF2-SHA256 iterations (OWASP 2023) using the standard library, with a self-describing hash format so existing passwords keep working and can be re-costed later without a reset",
			"Security: enabling two-factor now verifies a live authenticator code before it activates (you can no longer lock yourself out); org-sync reconfiguration is restricted to administrators so a low-privilege session can't force a hub-certificate re-pin; Google sign-in now requires a Google-verified email",
			"Robustness + cleanup: the SSH interactive shell reader no longer shares one buffer across concurrent reads; the login rate-limit table is swept so it can't grow unbounded on a long-running appliance; the manual-update screen only makes an http(s) download link clickable; removed several dead functions",
		},
		repoURL + "/releases/tag/v1.18.0", "v1.18.0"},
	{"1.17.1", "2026-06-28", "stable",
		[]string{
			"Fixed: \"Mark as solved\" in Security AI could resolve every CVE finding on a device when only one was actually fixed (the two findings shared the same category+node key) — findings are now identified individually",
			"Fixed: the licensing portal could report a successful registration/login while the session actually failed to save, leaving the user signed out with no explanation",
			"Hardening: TOTP codes are compared in constant time; SSH template probing checks for a broken pipe instead of risking a crash; the pre-auth session store is swept periodically instead of growing unbounded; the scan-progress poll stops cleanly if you navigate away mid-scan",
		},
		repoURL + "/releases/tag/v1.17.1", "v1.17.1"},
	{"1.17.0", "2026-06-28", "stable",
		[]string{
			"Install your own TLS certificate the appliance way (like a NetScaler OVF): it still ships self-signed by design, and now you can point TLS_CERT_FILE / TLS_KEY_FILE at a cert mounted read-only anywhere (or drop cert.pem/key.pem into the data volume) and restart — the engine logs a reminder while still self-signed",
			"The Windows desktop installer is now wired for optional Authenticode code-signing (fill in the cert thumbprint or a cloud-HSM sign command); see docs/CODE-SIGNING.md for the two-certificate distinction and step-by-step",
		},
		repoURL + "/releases/tag/v1.17.0", "v1.17.0"},
	{"1.16.0", "2026-06-28", "stable",
		[]string{
			"The appliance can now run a built-in, fully-offline AI: `docker compose --profile ai up` adds a local Ollama model server (pre-wired to the engine) so the Security AI answers in natural language with no cloud calls",
			"It's opt-in because the model is a ~2 GB download; without the profile the engine is unchanged and falls back to the deterministic rules engine. Air-gapped sites can pre-seed the model volume",
		},
		repoURL + "/releases/tag/v1.16.0", "v1.16.0"},
	{"1.15.1", "2026-06-28", "stable",
		[]string{
			"Security: a session that's still on the factory password is now restricted server-side to the change-password screen only — it no longer relies on the browser to enforce the forced change",
			"Security: the login lockout no longer trusts the X-Forwarded-For header by default (which an attacker could spoof to dodge the lockout); set TRUST_PROXY=1 only when the appliance sits behind a known reverse proxy",
		},
		repoURL + "/releases/tag/v1.15.1", "v1.15.1"},
	{"1.15.0", "2026-06-28", "stable",
		[]string{
			"The organization hub can now run on Postgres / Supabase instead of the built-in SQLite — set DATABASE_URL on the appliance and many desktops can fan in their scans concurrently, removing SQLite's single-writer ceiling for large estates",
			"The desktop app is unchanged (local SQLite); only the shared hub benefits. The org tables were already partitioned per organization, so it's a drop-in switch with no data migration",
		},
		repoURL + "/releases/tag/v1.15.0", "v1.15.0"},
	{"1.14.0", "2026-06-28", "stable",
		[]string{
			"Security: org sync now pins the hub's TLS certificate on first contact (trust-on-first-use) instead of trusting any certificate — an on-path attacker can no longer intercept the licence token used to authenticate sync",
			"Security: the credential master key can now be sealed with an external protector (LUMINA_MASTER_KEY or MASTER_KEY_FILE) so a stolen lumina.db no longer reveals stored credentials; without it the key stays in the DB (with a warning) for backward compatibility",
			"Re-saving the org hub URL re-pins the certificate — the escape hatch after a legitimate hub cert rotation",
		},
		repoURL + "/releases/tag/v1.14.0", "v1.14.0"},
	{"1.13.3", "2026-06-14", "stable",
		[]string{
			"Fix: \"Clear assets\" now actually clears in org-sync mode — it also wipes the org store (local + hub) so the map no longer flickers and comes back",
		},
		repoURL + "/releases/tag/v1.13.3", "v1.13.3"},
	{"1.13.2", "2026-06-14", "stable",
		[]string{
			"Re-classifying an asset now also sets its protocol + credentials (e.g. 'Server' → RDP + Windows account), not just the icon",
			"Security AI: 'Mark as solved' on any finding turns it green until the next scan re-checks it",
			"New smartphone icon for phones/mobiles",
		},
		repoURL + "/releases/tag/v1.13.2", "v1.13.2"},
	{"1.13.1", "2026-06-14", "stable",
		[]string{
			"Fix: a wrong OTP no longer locks you out — the code can be retried (up to 5 times) without restarting the app",
			"Fix: SSH/RDP to a NetScaler now opens with its credential (nsroot); the HA secondary is marked a NetScaler from the primary's 'show ha node'",
			"New: a 'Set type…' picker on each asset to classify it by hand (no re-scan), and it sticks across scans",
		},
		repoURL + "/releases/tag/v1.13.1", "v1.13.1"},
	{"1.13.0", "2026-06-14", "stable",
		[]string{
			"Network printers/MFPs (any make) are detected (ports 9100/631/515) and get their own lime printer icon on the map",
			"Topology layout: ring radius adapts to crowded segments and HA peers are placed snug side-by-side",
			"Discovery ignores multicast/broadcast ARP noise so junk like 230.x never becomes a phantom node",
		},
		repoURL + "/releases/tag/v1.13.0", "v1.13.0"},
	{"1.12.3", "2026-06-13", "stable",
		[]string{
			"HA secondary is now authenticated with the primary's credentials (learned from 'show ha node') instead of showing as an ARP-only ghost",
			"SSH/RDP 'Connect' now opens as the credential the host was discovered with (e.g. the NetScaler's nsroot), not your local account",
		},
		repoURL + "/releases/tag/v1.12.3", "v1.12.3"},
	{"1.12.2", "2026-06-13", "stable",
		[]string{
			"Org sync now pushes on a timer (not only after a scan), so existing assets converge without re-scanning",
			"New sync indicator in Settings — Synced ✓ / last time / the exact error if a member's license was issued by a different portal",
		},
		repoURL + "/releases/tag/v1.12.2", "v1.12.2"},
	{"1.12.1", "2026-06-13", "stable",
		[]string{
			"Fix: org sync now works against the appliance's self-signed HTTPS (the desktop trusts its own org hub)",
			"Fix: discovery no longer drops segments learned through a firewall's other leg (ARP-behind hosts are never auto-pruned)",
			"Fix: HA sync links are drawn as a curved arc so they route around a node sitting between the pair",
		},
		repoURL + "/releases/tag/v1.12.1", "v1.12.1"},
	{"1.12.0", "2026-06-12", "stable",
		[]string{
			"Org sync complete: members pull the merged asset map (not just push), so every desktop shows the whole org's network",
			"Org-wide settings sync — LDAP, idle timeout, and OTP/MFA are shared across the org (one TOTP secret everywhere, no more browser-vs-app mismatch)",
			"New Organization Sync design doc on /docs",
		},
		repoURL + "/releases/tag/v1.12.0", "v1.12.0"},
	{"1.11.0", "2026-06-12", "stable",
		[]string{
			"Organization asset sync (stage 1): point every member at a shared appliance hub and scans push discovered assets to the org so everyone sees one merged map",
			"Authenticated by your license key (Ed25519) and partitioned per organization; configure the hub URL in Settings (blank = standalone)",
		},
		repoURL + "/releases/tag/v1.11.0", "v1.11.0"},
	{"1.10.0", "2026-06-12", "stable",
		[]string{
			"Idle auto-logout (default 15 min, configurable in Settings, 0 = off) for the desktop app and the browser UI",
			"Scans now auto-prune hosts that stop responding (3 missed scans on a swept subnet) so the map self-heals",
			"Logged-in visitors see an account avatar (email initial) on the website instead of Sign in / Sign up",
		},
		repoURL + "/releases/tag/v1.10.0", "v1.10.0"},
	{"1.9.4", "2026-06-12", "stable",
		[]string{
			"New \"Clear assets\" button wipes the discovered map (keeps credentials/LDAP/license) to remove phantom/stale hosts left by an earlier scan",
			"Topology map warns when running in appliance mode that layer-2 discovery (Wi-Fi/phones, NetScaler VIP folding) needs the desktop app",
		},
		repoURL + "/releases/tag/v1.9.4", "v1.9.4"},
	{"1.9.3", "2026-06-12", "stable",
		[]string{
			"Scan: the headless server/Docker appliance no longer lists its own container IP as a discovered asset",
			"Clarified that full layer-2 (ARP/MAC, Wi-Fi, VIP folding) discovery needs to run from a host on the LAN",
		},
		repoURL + "/releases/tag/v1.9.3", "v1.9.3"},
	{"1.9.2", "2026-06-11", "stable",
		[]string{
			"Fix: license activation in the Dockerized server now reaches the portal by service name (PORTAL_URL=http://portal:8090) instead of 127.0.0.1",
			"compose wires PORTAL_URL + depends_on so the all-in-Docker web UI activates out of the box",
		},
		repoURL + "/releases/tag/v1.9.2", "v1.9.2"},
	{"1.9.1", "2026-06-11", "stable",
		[]string{
			"Fix: production license activation no longer fails with \"connection refused [::1]:8090\" against a Dockerized portal (IPv4 fallback)",
			"Update checks use the same IPv4 fallback; default portal URL is now 127.0.0.1",
		},
		repoURL + "/releases/tag/v1.9.1", "v1.9.1"},
	{"1.9.0", "2026-06-11", "stable",
		[]string{
			"Portal can now use an external Postgres / Supabase database (set DATABASE_URL) to manage registered &amp; paying users from a hosted dashboard",
			"New payments table + billing scaffolding, ready to wire a payment provider (Stripe) when you start selling",
			"Falls back to local SQLite when no DATABASE_URL is set; new Supabase setup guide under /docs",
		},
		repoURL + "/releases/tag/v1.9.0", "v1.9.0"},
	{"1.8.4", "2026-06-11", "stable",
		[]string{
			"Docs site now hosts the Security &amp; Firewall Hardening guide (PDF) and deployment notes under /docs",
			"Guides are embedded in the portal binary and served with a new Guides &amp; downloads section",
		},
		repoURL + "/releases/tag/v1.8.4", "v1.8.4"},
	{"1.8.3", "2026-06-11", "stable",
		[]string{
			"Fix: appliance licensing portal now starts in Docker (writable data dir + DATA_DIR support)",
			"Portal stores its database under /data so registrations &amp; sessions persist across restarts",
		},
		repoURL + "/releases/tag/v1.8.3", "v1.8.3"},
	{"1.8.2", "2026-06-10", "stable",
		[]string{
			"Fix: \"Upgrade Now\" now reliably opens the registration portal (popup-blocker)",
			"Portal keeps you signed in with a session cookie — your license &amp; download persist",
		},
		repoURL + "/releases/tag/v1.8.2", "v1.8.2"},
	{"1.8.1", "2026-06-10", "stable",
		[]string{
			"Docker compose now publishes the HTTPS UI on host port 443 (self-signed cert out of the box)",
			"Documented host-networking option for full LAN (ARP) discovery on the appliance",
		},
		repoURL + "/releases/tag/v1.8.1", "v1.8.1"},
	{"1.8.0", "2026-06-10", "stable",
		[]string{
			"\"Explain my posture\" — one-click AI summary &amp; prioritized remediation plan (local model, offline)",
			"Falls back to a deterministic summary when no local AI is available",
		},
		repoURL + "/releases/tag/v1.8.0", "v1.8.0"},
	{"1.7.0", "2026-06-10", "stable",
		[]string{
			"Built-in local AI (Ollama) — offline natural-language answers grounded in your topology &amp; findings",
			"HTTPS with an auto-generated self-signed cert (admin-replaceable) for the server/appliance",
			"Security &amp; firewall hardening guide (PDF)",
		},
		repoURL + "/releases/tag/v1.7.0", "v1.7.0"},
	{"1.6.1", "2026-06-08", "stable",
		[]string{
			"LLDP/CDP layer-2 links drawn as bold emerald 'L2' edges on the map",
			"Downloads now require registration; release nav matches the home page",
		},
		repoURL + "/releases/tag/v1.6.1", "v1.6.1"},
	{"1.6.0", "2026-06-08", "stable",
		[]string{
			"SNMPv3 (NoAuthNoPriv) with automatic v2c fallback",
			"Security AI ↔ CVE database (incl. CISA KEV) → upgrade recommendations",
			"Real SNMP throughput (Gbps) on gateways; v3/v2c node badge",
			"In-app version notifications: auto / manual update + What's New",
		},
		repoURL + "/releases/tag/v1.6.0", "v1.6.0"},
	{"1.5.0", "2026-06-08", "stable",
		[]string{"Security posture score + history trend", "PDF/CSV export", "Per-segment risk drill-down"},
		repoURL + "/releases/tag/v1.5.0", "v1.5.0"},
	{"1.4.0", "2026-06-03", "stable",
		[]string{"30+ vendor drivers with serial/uptime/build", "Network-learned Security AI + 0-100 score", "Multi-homed + HA + ARP-behind-gateway"},
		repoURL + "/releases/tag/v1.4.0", "v1.4.0"},
}

func latestHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	if len(releases) == 0 {
		writeJSON(w, 404, map[string]string{"error": "no releases"})
		return
	}
	writeJSON(w, 200, releases[0])
}

func releasesAPIHandler(w http.ResponseWriter, r *http.Request) {
	cors(w)
	if r.Method == http.MethodOptions {
		return
	}
	writeJSON(w, 200, releases)
}

func releasesPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = releasesTmpl.Execute(w, map[string]any{"Releases": releases, "Repo": repoURL})
}

var releasesTmpl = template.Must(template.New("rel").Funcs(template.FuncMap{
	"json": func(v any) (template.JS, error) { b, e := json.Marshal(v); return template.JS(b), e },
}).Parse(withAuthNav(releasesHTML)))

const releasesHTML = `<!doctype html><html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Lumina NetOS — Releases</title>
<link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Sora:wght@600;700;800&family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
<style>
:root{--bg:#070512;--ink:#f2effb;--mut:#9b93c4;--vio:#a855f7;--mag:#d946ef;--cy:#22d3ee;--glass:rgba(168,148,230,.07);--brd:rgba(168,148,230,.16)}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--ink);font-family:Inter,system-ui,sans-serif;line-height:1.6}
a{color:inherit;text-decoration:none}h1,h2,h3{font-family:Sora,sans-serif;letter-spacing:-.02em;margin:0}
.wrap{max-width:880px;margin:0 auto;padding:0 24px}
nav{position:sticky;top:0;z-index:20;display:flex;justify-content:space-between;align-items:center;padding:16px 36px;border-bottom:1px solid var(--brd);background:rgba(7,5,18,.65);backdrop-filter:blur(14px)}
.logo{font-family:Sora;font-weight:800;letter-spacing:.5px}.logo b{color:var(--mag)}
.menu{display:flex;gap:24px}
.menu a{position:relative;color:#e7e2f7;font-size:12.5px;font-weight:700;letter-spacing:.12em;opacity:.85;padding-bottom:4px}
.menu a:hover{opacity:1}
.menu a::after{content:"";position:absolute;left:50%;right:50%;bottom:0;height:2px;border-radius:2px;background:linear-gradient(90deg,var(--vio),var(--cy));transition:left .28s ease,right .28s ease}
.menu a:hover::after,.menu a:focus-visible::after{left:0;right:0}
@media(max-width:760px){.menu{display:none}}
.authtoggle{display:flex;gap:8px;align-items:center}
.btn-primary{padding:9px 16px;border-radius:12px;font-weight:700;font-size:13px;background:linear-gradient(120deg,var(--vio),var(--mag));color:#fff;box-shadow:0 8px 30px -8px rgba(217,70,239,.6);transition:transform .2s}
.btn-primary:hover{transform:translateY(-2px)}
.btn-ghost{padding:9px 16px;border-radius:12px;font-weight:700;font-size:13px;color:#e7e2f7;border:1px solid var(--brd);background:var(--glass);transition:transform .2s,border-color .2s}
.btn-ghost:hover{border-color:var(--cy);transform:translateY(-2px)}
@media(prefers-reduced-motion:reduce){.menu a::after{transition:none}}
.dochero{padding:72px 0 28px;background:radial-gradient(120% 120% at 50% -10%,#2a1150 0%,#1a0a36 40%,#0a0518 75%,#070512 100%)}
.eyebrow{color:var(--cy);font-weight:700;font-size:13px;letter-spacing:.14em;text-transform:uppercase}
h1{font-size:clamp(34px,6vw,50px);margin:8px 0;background:linear-gradient(120deg,#fff,#e9b8ff 60%,#22d3ee);-webkit-background-clip:text;background-clip:text;color:transparent}
.lead{color:var(--mut);max-width:600px}
.rel{display:flex;gap:22px;padding:26px 0;border-top:1px solid var(--brd)}
.rel .meta{min-width:150px}
.ver{font-family:Sora;font-weight:800;font-size:22px}
.tagrow{display:flex;gap:8px;align-items:center;margin-top:6px;flex-wrap:wrap}
.badge{font-size:10px;font-weight:800;text-transform:uppercase;letter-spacing:.08em;padding:3px 9px;border-radius:999px;border:1px solid var(--brd);color:var(--cy)}
.latest{background:linear-gradient(120deg,var(--vio),var(--mag));color:#fff;border-color:transparent}
.date{font-size:12px;color:var(--mut);font-family:ui-monospace,monospace;margin-top:6px}
.notes{list-style:none;padding:0;margin:0}
.notes li{padding:5px 0 5px 22px;position:relative;color:var(--mut);font-size:14px}
.notes li::before{content:"";position:absolute;left:0;top:12px;width:7px;height:7px;border-radius:50%;background:var(--cy)}
.dl{display:inline-block;margin-top:12px;font-size:13px;font-weight:700;color:var(--cy)}
footer{padding:40px 0;border-top:1px solid var(--brd);color:var(--mut);font-size:13px;text-align:center}
@media(max-width:640px){.rel{flex-direction:column;gap:8px}}
</style></head><body>
<nav>
  <div class="logo">LUMINA <b>·</b> NETOS</div>
  <div class="menu"><a href="/">HOME</a><a href="/#features">FEATURES</a><a href="/#pricing">PRICING</a><a href="/#roadmap">ROADMAP</a><a href="/docs">DOCS</a></div>
  <div class="authtoggle"><a class="btn-ghost" href="/signup">Sign In</a><a class="btn-primary" href="/signup">Sign Up</a></div>
</nav>
<div class="dochero"><div class="wrap"><span class="eyebrow">Releases</span>
  <h1>Version history</h1>
  <p class="lead">Every release maps to a git tag. The desktop app checks here for updates and shows what's new.</p></div></div>
<div class="wrap">
{{range $i, $r := .Releases}}
  <div class="rel">
    <div class="meta">
      <div class="ver">v{{$r.Version}}</div>
      <div class="tagrow">{{if eq $i 0}}<span class="badge latest">Latest</span>{{end}}<span class="badge">{{$r.Channel}}</span></div>
      <div class="date">{{$r.Date}}</div>
    </div>
    <div>
      <ul class="notes">{{range $r.Notes}}<li>{{.}}</li>{{end}}</ul>
      <a class="dl" href="/signup?next=download&amp;v={{$r.Version}}">Sign in to download v{{$r.Version}} →</a>
    </div>
  </div>
{{end}}
</div>
<footer class="wrap">© 2026 Lumina NetOS · <a href="/">Home</a> · <a href="{{.Repo}}" target="_blank" rel="noopener">GitHub</a></footer>
</body></html>`
