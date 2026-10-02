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
   Releases map 1:1 to git tags (v<version>). The source repo is private, so
   installers are published as GitHub Releases on the PUBLIC downloads repo
   (installers + notes only, no source); DownloadURL points at that release.
   Versions before 1.27.2 were never published as installers and link to the
   downloads repo's release list instead.
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

// downloadsRepo is the PUBLIC repo that hosts the installers (the source repo
// is private, so its release pages 404 for customers).
const downloadsRepo = "https://github.com/Benny007-king/lumina-netos-releases"

// installerURL is a direct download of the newest Windows installer: every
// release uploads it under this stable asset name.
const installerURL = downloadsRepo + "/releases/latest/download/LuminaNetOS-Setup-x64.exe"

// releases — newest first. Add a new entry (and push a matching git tag) per release.
var releases = []Release{
	{"1.28.1", "2026-10-02", "stable",
		[]string{
			"Fixes the installed Windows app failing at the login screen with \"The Core Engine returned an unreadable response\": the installed app now talks to its local engine (it was sending requests to its own page), and the engine accepts the installed app's requests",
			"The login screen no longer shows \"Demo Environment\" and hides the LDAP sign-in on a production install when the app is opened before its engine finished starting",
			"The login screen shows the real version instead of v1.0.0",
		},
		downloadsRepo + "/releases/tag/v1.28.1", "v1.28.1"},
	{"1.28.0", "2026-10-02", "stable",
		[]string{
			"The map connects each device to its real default gateway instead of guessing: the scanner reads the routing table of every device it can log into (SSH, WinRM, SNMP) and of the computer running the scan, and links the device to whoever owns that gateway address",
			"Devices it can't log into follow the gateway the readable devices on the same segment use; a gateway router that exposes no ports (and so was missed by the sweep) is still added because routing tables prove it exists",
			"Link details show where each connection came from: read from the device, shared by the segment, or inferred when no routing data exists",
		},
		downloadsRepo + "/releases/tag/v1.28.0", "v1.28.0"},
	{"1.27.2", "2026-10-02", "stable",
		[]string{
			"The installer is now downloadable: after signing in, \"Download Lumina NetOS\" fetches the Windows installer directly (it used to open an empty or 404 GitHub page)",
			"Correct map for segments behind a firewall: hosts on a firewall's LAN connect to the firewall itself, even when one of them happens to own the .1 address (they used to be drawn chained one behind another), and the old wrong links are cleared on the next scan",
		},
		downloadsRepo + "/releases/tag/v1.27.2", "v1.27.2"},
	{"1.27.1", "2026-10-02", "stable",
		[]string{
			"The desktop app now uses the same local AI as the appliance when both run on one machine — the model is published on this computer's loopback address only (never on the network, Ollama has no login)",
			"Clearer AI remediation text: fixes that are instructions no longer read \"upgrade to Apply …\", and grouped findings carry each device's IP so the AI names the right one",
		},
		downloadsRepo + "/releases", "v1.27.1"},
	{"1.27.0", "2026-09-30", "stable",
		[]string{
			"Local, offline Security AI: \"Explain my posture\" and the AI chat run on a model inside your own appliance (Ollama, llama3.2) — network data never leaves your site. Enable it with docker compose --profile ai up -d",
			"Much better AI answers: the model now gets each device's identity, whether it was actually authenticated, and every serious finding grouped by device — so its remediation plan names the right device and the concrete fix instead of generic advice, and it is instructed never to invent devices, versions or CVEs",
			"The AI chat shows a \"thinking\" indicator while the local model works, and local AI answers no longer time out on CPU-only machines",
		},
		downloadsRepo + "/releases", "v1.27.0"},
	{"1.26.1", "2026-09-30", "stable",
		[]string{
			"Security AI findings are now accurate: pfSense/OPNsense and other firewall brands count as firewalls (no false \"No perimeter firewall\" or flat paths through them); NetScaler build numbers are read correctly (no false Citrix Bleed on a patched build); unverified CVEs are no longer pinned on devices the scanner couldn't identify; servers with RDP aren't treated as user endpoints; repeated findings are merged per segment instead of per device",
			"RDP / SSH from the browser (Docker appliance): RDP downloads a .rdp file that opens Remote Desktop on your computer, SSH copies the ssh command — the server can't open windows on your PC from its container",
			"NetScaler HA pairs cloned from one template (same serial number) are no longer merged into one device",
		},
		downloadsRepo + "/releases", "v1.26.1"},
	{"1.26.0", "2026-09-30", "stable",
		[]string{
			"NetScaler HA pairs stay two devices side by side: the two nodes share floating SNIP/VIP addresses, which made the scanner merge them into one — the secondary then reappeared as a nameless \"HOST-…\" placeholder far from its partner. A rescan now keeps both named NetScalers and draws them as a pair",
			"No more false HA links: unnamed devices of the same vendor (e.g. a NetScaler with an ADM agent) were paired as HA because their placeholder names looked alike; wrong HA links stored by earlier scans are removed on the next scan",
			"Phones using a private (randomized) Wi-Fi address — the default on iPhone and Android — are now recognised as mobile devices instead of unknown hosts",
			"New iPhone and tower-server icons, and zoom in / zoom out buttons in the corner of the map",
		},
		downloadsRepo + "/releases", "v1.26.0"},
	{"1.25.3", "2026-09-30", "stable",
		[]string{
			"Scan results now appear when the scan actually finishes: the map used to stop waiting after a few seconds (discovered assets are only merged at the end of the whole crawl, which can take minutes), so it showed an empty map. The engine now reports whether a scan is running, the map waits for it, and a scan still in progress is picked up again after a refresh or when you return to the map",
			"Only one scan runs at a time — starting another while one is in progress follows the running scan instead",
			"Refreshing the page no longer signs you out: the session is kept for the browser tab (and cleared when the tab closes)",
		},
		downloadsRepo + "/releases", "v1.25.3"},
	{"1.25.2", "2026-09-30", "stable",
		[]string{
			"When your session ends (it expired, or the engine restarted), the app now returns you to the sign-in screen with a clear message — previously every screen silently showed empty data, which looked like lost credentials and a broken scan",
			"The licensing portal and the org hub now work behind connection poolers such as Supabase's: a second instance starting alongside a running one (e.g. during a deploy) no longer fails with \"prepared statement … already exists\"",
		},
		downloadsRepo + "/releases", "v1.25.2"},
	{"1.25.1", "2026-09-30", "stable",
		[]string{
			"Read-only (Viewer) accounts no longer see controls they can't use: scan, clear, delete, re-classify, connect, device dump, mark-as-solved, clear audit log and credential editing are hidden, and the credentials page shows a read-only notice",
			"Fixed: saving a vendor credential with the secret left blank (e.g. only changing the username) wiped the stored password while the card still showed \"Set\" — a blank secret now keeps the stored one, as the form promises",
		},
		downloadsRepo + "/releases", "v1.25.1"},
	{"1.25.0", "2026-09-30", "stable",
		[]string{
			"Role permissions are now enforced by the engine, not just the screen: Viewers are read-only, Operators can scan, connect and manage assets but can't change system settings, and only administrators can change LDAP, session timeout, organization sync, switch modes or apply updates",
			"Security hardening from a full QA review: only the local administrator can change the local admin password (and must confirm the current one); the factory-password lockdown now covers every screen; password and two-factor attempts are limited per user even under bursts of parallel requests; signing out now ends the session on the server; exported PDF/CSV reports safely escape device-supplied text; a portal error can no longer wipe the list of revoked license keys",
			"Licensing portal: Google/GitHub sign-in now completes (it used to stop on the home page) and is bound to the browser that started it; the session cookie is always Secure behind HTTPS",
			"Fixes: Security AI no longer lists the same server exposure twice (which inflated the critical count); a rare engine-wide freeze when opening SSH/RDP during a scan; overlapping scan-progress refreshes; organization-sync timing races; failed actions now show the real reason instead of silently doing nothing",
			"The desktop app now uses the hosted licensing portal by default, so sign-up, activation and update checks work out of the box",
		},
		downloadsRepo + "/releases", "v1.25.0"},
	{"1.24.2", "2026-07-11", "stable",
		[]string{
			"The post-update \"What's New\" dialog now shows only what changed in the latest update instead of the entire release history, and it resizes to the screen with its own scrollbar so a long changelog never runs off-screen",
		},
		downloadsRepo + "/releases", "v1.24.2"},
	{"1.24.1", "2026-07-11", "stable",
		[]string{
			"The licensing portal now honours the PORT environment variable and binds all interfaces, so it deploys cleanly to Fly.io / Render / Railway / Cloud Run (no more port-mismatch health-check timeouts)",
		},
		downloadsRepo + "/releases", "v1.24.1"},
	{"1.24.0", "2026-07-10", "stable",
		[]string{
			"The desktop \"Upgrade Now\" button and update checks can now point at a published, always-on website instead of a locally-running portal — set PUBLIC_PORTAL_URL on the appliance to your hosted portal URL",
			"Fixes the releases/website not reflecting the newest version: the portal image is now rebuilt on every release so this page always shows the latest",
		},
		downloadsRepo + "/releases", "v1.24.0"},
	{"1.23.1", "2026-07-04", "stable",
		[]string{
			"Internal cleanup (over-engineering pass): removed a redundant field from the organization-sync pull response; no behaviour change",
		},
		downloadsRepo + "/releases", "v1.23.1"},
	{"1.23.0", "2026-07-04", "stable",
		[]string{
			"The credential master-key protector can now be rotated with no downtime — set the new secret plus LUMINA_MASTER_KEY_OLD, start once to re-seal, then drop the old one",
			"Licensing-portal sessions now expire server-side after 30 days and are swept periodically, so an old session cookie can't be replayed indefinitely",
		},
		downloadsRepo + "/releases", "v1.23.0"},
	{"1.22.0", "2026-07-04", "stable",
		[]string{
			"Organization sync now scales to large estates: members pull only the assets that changed since their last sync (with a periodic full reconcile) instead of the whole map every 30 seconds, and the hub writes each batch in a single transaction",
			"The hub also garbage-collects assets that have left the estate, so the shared organization map no longer grows forever",
		},
		downloadsRepo + "/releases", "v1.22.0"},
	{"1.21.0", "2026-07-04", "stable",
		[]string{
			"License keys can now be revoked — if a key leaks, click \"Revoke & Reissue Key\" on your account page to kill the old one and get a fresh key, without re-keying the whole organization",
			"Engines/hubs poll the portal's revocation list and reject any organization-sync request that uses a revoked key (within ~15 minutes); the last-known list is cached so the check keeps working offline",
		},
		downloadsRepo + "/releases", "v1.21.0"},
	{"1.20.0", "2026-07-04", "stable",
		[]string{
			"Organization settings sync (LDAP + MFA) can now be gated by a shared admin token: set the same LUMINA_ORG_ADMIN_TOKEN on the hub and every member and only holders of that token can push or pull the org's directory config and MFA enrollments",
			"This stops a member who only has the product licence key from rewriting org-wide LDAP or planting an MFA enrollment. The asset map still syncs on the licence alone; leaving the token unset keeps the previous behaviour (with a startup warning)",
		},
		downloadsRepo + "/releases", "v1.20.0"},
	{"1.19.0", "2026-07-04", "stable",
		[]string{
			"MFA/TOTP secrets are now encrypted at rest — on each device and on the organization hub — with the same key that already protects stored device credentials, so a stolen database no longer exposes live authenticator seeds",
			"Backward compatible: existing 2FA secrets keep working and are re-sealed transparently; the raw seed only exists in memory and over the encrypted, certificate-pinned org-sync channel",
		},
		downloadsRepo + "/releases", "v1.19.0"},
	{"1.18.0", "2026-07-04", "stable",
		[]string{
			"Security: password hashing raised to 600,000 PBKDF2-SHA256 iterations (OWASP 2023) using the standard library, with a self-describing hash format so existing passwords keep working and can be re-costed later without a reset",
			"Security: enabling two-factor now verifies a live authenticator code before it activates (you can no longer lock yourself out); org-sync reconfiguration is restricted to administrators so a low-privilege session can't force a hub-certificate re-pin; Google sign-in now requires a Google-verified email",
			"Robustness + cleanup: the SSH interactive shell reader no longer shares one buffer across concurrent reads; the login rate-limit table is swept so it can't grow unbounded on a long-running appliance; the manual-update screen only makes an http(s) download link clickable; removed several dead functions",
		},
		downloadsRepo + "/releases", "v1.18.0"},
	{"1.17.1", "2026-06-28", "stable",
		[]string{
			"Fixed: \"Mark as solved\" in Security AI could resolve every CVE finding on a device when only one was actually fixed (the two findings shared the same category+node key) — findings are now identified individually",
			"Fixed: the licensing portal could report a successful registration/login while the session actually failed to save, leaving the user signed out with no explanation",
			"Hardening: TOTP codes are compared in constant time; SSH template probing checks for a broken pipe instead of risking a crash; the pre-auth session store is swept periodically instead of growing unbounded; the scan-progress poll stops cleanly if you navigate away mid-scan",
		},
		downloadsRepo + "/releases", "v1.17.1"},
	{"1.17.0", "2026-06-28", "stable",
		[]string{
			"Install your own TLS certificate the appliance way (like a NetScaler OVF): it still ships self-signed by design, and now you can point TLS_CERT_FILE / TLS_KEY_FILE at a cert mounted read-only anywhere (or drop cert.pem/key.pem into the data volume) and restart — the engine logs a reminder while still self-signed",
			"The Windows desktop installer is now wired for optional Authenticode code-signing (fill in the cert thumbprint or a cloud-HSM sign command); see docs/CODE-SIGNING.md for the two-certificate distinction and step-by-step",
		},
		downloadsRepo + "/releases", "v1.17.0"},
	{"1.16.0", "2026-06-28", "stable",
		[]string{
			"The appliance can now run a built-in, fully-offline AI: `docker compose --profile ai up` adds a local Ollama model server (pre-wired to the engine) so the Security AI answers in natural language with no cloud calls",
			"It's opt-in because the model is a ~2 GB download; without the profile the engine is unchanged and falls back to the deterministic rules engine. Air-gapped sites can pre-seed the model volume",
		},
		downloadsRepo + "/releases", "v1.16.0"},
	{"1.15.1", "2026-06-28", "stable",
		[]string{
			"Security: a session that's still on the factory password is now restricted server-side to the change-password screen only — it no longer relies on the browser to enforce the forced change",
			"Security: the login lockout no longer trusts the X-Forwarded-For header by default (which an attacker could spoof to dodge the lockout); set TRUST_PROXY=1 only when the appliance sits behind a known reverse proxy",
		},
		downloadsRepo + "/releases", "v1.15.1"},
	{"1.15.0", "2026-06-28", "stable",
		[]string{
			"The organization hub can now run on Postgres / Supabase instead of the built-in SQLite — set DATABASE_URL on the appliance and many desktops can fan in their scans concurrently, removing SQLite's single-writer ceiling for large estates",
			"The desktop app is unchanged (local SQLite); only the shared hub benefits. The org tables were already partitioned per organization, so it's a drop-in switch with no data migration",
		},
		downloadsRepo + "/releases", "v1.15.0"},
	{"1.14.0", "2026-06-28", "stable",
		[]string{
			"Security: org sync now pins the hub's TLS certificate on first contact (trust-on-first-use) instead of trusting any certificate — an on-path attacker can no longer intercept the licence token used to authenticate sync",
			"Security: the credential master key can now be sealed with an external protector (LUMINA_MASTER_KEY or MASTER_KEY_FILE) so a stolen lumina.db no longer reveals stored credentials; without it the key stays in the DB (with a warning) for backward compatibility",
			"Re-saving the org hub URL re-pins the certificate — the escape hatch after a legitimate hub cert rotation",
		},
		downloadsRepo + "/releases", "v1.14.0"},
	{"1.13.3", "2026-06-14", "stable",
		[]string{
			"Fix: \"Clear assets\" now actually clears in org-sync mode — it also wipes the org store (local + hub) so the map no longer flickers and comes back",
		},
		downloadsRepo + "/releases", "v1.13.3"},
	{"1.13.2", "2026-06-14", "stable",
		[]string{
			"Re-classifying an asset now also sets its protocol + credentials (e.g. 'Server' → RDP + Windows account), not just the icon",
			"Security AI: 'Mark as solved' on any finding turns it green until the next scan re-checks it",
			"New smartphone icon for phones/mobiles",
		},
		downloadsRepo + "/releases", "v1.13.2"},
	{"1.13.1", "2026-06-14", "stable",
		[]string{
			"Fix: a wrong OTP no longer locks you out — the code can be retried (up to 5 times) without restarting the app",
			"Fix: SSH/RDP to a NetScaler now opens with its credential (nsroot); the HA secondary is marked a NetScaler from the primary's 'show ha node'",
			"New: a 'Set type…' picker on each asset to classify it by hand (no re-scan), and it sticks across scans",
		},
		downloadsRepo + "/releases", "v1.13.1"},
	{"1.13.0", "2026-06-14", "stable",
		[]string{
			"Network printers/MFPs (any make) are detected (ports 9100/631/515) and get their own lime printer icon on the map",
			"Topology layout: ring radius adapts to crowded segments and HA peers are placed snug side-by-side",
			"Discovery ignores multicast/broadcast ARP noise so junk like 230.x never becomes a phantom node",
		},
		downloadsRepo + "/releases", "v1.13.0"},
	{"1.12.3", "2026-06-13", "stable",
		[]string{
			"HA secondary is now authenticated with the primary's credentials (learned from 'show ha node') instead of showing as an ARP-only ghost",
			"SSH/RDP 'Connect' now opens as the credential the host was discovered with (e.g. the NetScaler's nsroot), not your local account",
		},
		downloadsRepo + "/releases", "v1.12.3"},
	{"1.12.2", "2026-06-13", "stable",
		[]string{
			"Org sync now pushes on a timer (not only after a scan), so existing assets converge without re-scanning",
			"New sync indicator in Settings — Synced ✓ / last time / the exact error if a member's license was issued by a different portal",
		},
		downloadsRepo + "/releases", "v1.12.2"},
	{"1.12.1", "2026-06-13", "stable",
		[]string{
			"Fix: org sync now works against the appliance's self-signed HTTPS (the desktop trusts its own org hub)",
			"Fix: discovery no longer drops segments learned through a firewall's other leg (ARP-behind hosts are never auto-pruned)",
			"Fix: HA sync links are drawn as a curved arc so they route around a node sitting between the pair",
		},
		downloadsRepo + "/releases", "v1.12.1"},
	{"1.12.0", "2026-06-12", "stable",
		[]string{
			"Org sync complete: members pull the merged asset map (not just push), so every desktop shows the whole org's network",
			"Org-wide settings sync — LDAP, idle timeout, and OTP/MFA are shared across the org (one TOTP secret everywhere, no more browser-vs-app mismatch)",
			"New Organization Sync design doc on /docs",
		},
		downloadsRepo + "/releases", "v1.12.0"},
	{"1.11.0", "2026-06-12", "stable",
		[]string{
			"Organization asset sync (stage 1): point every member at a shared appliance hub and scans push discovered assets to the org so everyone sees one merged map",
			"Authenticated by your license key (Ed25519) and partitioned per organization; configure the hub URL in Settings (blank = standalone)",
		},
		downloadsRepo + "/releases", "v1.11.0"},
	{"1.10.0", "2026-06-12", "stable",
		[]string{
			"Idle auto-logout (default 15 min, configurable in Settings, 0 = off) for the desktop app and the browser UI",
			"Scans now auto-prune hosts that stop responding (3 missed scans on a swept subnet) so the map self-heals",
			"Logged-in visitors see an account avatar (email initial) on the website instead of Sign in / Sign up",
		},
		downloadsRepo + "/releases", "v1.10.0"},
	{"1.9.4", "2026-06-12", "stable",
		[]string{
			"New \"Clear assets\" button wipes the discovered map (keeps credentials/LDAP/license) to remove phantom/stale hosts left by an earlier scan",
			"Topology map warns when running in appliance mode that layer-2 discovery (Wi-Fi/phones, NetScaler VIP folding) needs the desktop app",
		},
		downloadsRepo + "/releases", "v1.9.4"},
	{"1.9.3", "2026-06-12", "stable",
		[]string{
			"Scan: the headless server/Docker appliance no longer lists its own container IP as a discovered asset",
			"Clarified that full layer-2 (ARP/MAC, Wi-Fi, VIP folding) discovery needs to run from a host on the LAN",
		},
		downloadsRepo + "/releases", "v1.9.3"},
	{"1.9.2", "2026-06-11", "stable",
		[]string{
			"Fix: license activation in the Dockerized server now reaches the portal by service name (PORTAL_URL=http://portal:8090) instead of 127.0.0.1",
			"compose wires PORTAL_URL + depends_on so the all-in-Docker web UI activates out of the box",
		},
		downloadsRepo + "/releases", "v1.9.2"},
	{"1.9.1", "2026-06-11", "stable",
		[]string{
			"Fix: production license activation no longer fails with \"connection refused [::1]:8090\" against a Dockerized portal (IPv4 fallback)",
			"Update checks use the same IPv4 fallback; default portal URL is now 127.0.0.1",
		},
		downloadsRepo + "/releases", "v1.9.1"},
	{"1.9.0", "2026-06-11", "stable",
		[]string{
			"Portal can now use an external Postgres / Supabase database (set DATABASE_URL) to manage registered &amp; paying users from a hosted dashboard",
			"New payments table + billing scaffolding, ready to wire a payment provider (Stripe) when you start selling",
			"Falls back to local SQLite when no DATABASE_URL is set; new Supabase setup guide under /docs",
		},
		downloadsRepo + "/releases", "v1.9.0"},
	{"1.8.4", "2026-06-11", "stable",
		[]string{
			"Docs site now hosts the Security &amp; Firewall Hardening guide (PDF) and deployment notes under /docs",
			"Guides are embedded in the portal binary and served with a new Guides &amp; downloads section",
		},
		downloadsRepo + "/releases", "v1.8.4"},
	{"1.8.3", "2026-06-11", "stable",
		[]string{
			"Fix: appliance licensing portal now starts in Docker (writable data dir + DATA_DIR support)",
			"Portal stores its database under /data so registrations &amp; sessions persist across restarts",
		},
		downloadsRepo + "/releases", "v1.8.3"},
	{"1.8.2", "2026-06-10", "stable",
		[]string{
			"Fix: \"Upgrade Now\" now reliably opens the registration portal (popup-blocker)",
			"Portal keeps you signed in with a session cookie — your license &amp; download persist",
		},
		downloadsRepo + "/releases", "v1.8.2"},
	{"1.8.1", "2026-06-10", "stable",
		[]string{
			"Docker compose now publishes the HTTPS UI on host port 443 (self-signed cert out of the box)",
			"Documented host-networking option for full LAN (ARP) discovery on the appliance",
		},
		downloadsRepo + "/releases", "v1.8.1"},
	{"1.8.0", "2026-06-10", "stable",
		[]string{
			"\"Explain my posture\" — one-click AI summary &amp; prioritized remediation plan (local model, offline)",
			"Falls back to a deterministic summary when no local AI is available",
		},
		downloadsRepo + "/releases", "v1.8.0"},
	{"1.7.0", "2026-06-10", "stable",
		[]string{
			"Built-in local AI (Ollama) — offline natural-language answers grounded in your topology &amp; findings",
			"HTTPS with an auto-generated self-signed cert (admin-replaceable) for the server/appliance",
			"Security &amp; firewall hardening guide (PDF)",
		},
		downloadsRepo + "/releases", "v1.7.0"},
	{"1.6.1", "2026-06-08", "stable",
		[]string{
			"LLDP/CDP layer-2 links drawn as bold emerald 'L2' edges on the map",
			"Downloads now require registration; release nav matches the home page",
		},
		downloadsRepo + "/releases", "v1.6.1"},
	{"1.6.0", "2026-06-08", "stable",
		[]string{
			"SNMPv3 (NoAuthNoPriv) with automatic v2c fallback",
			"Security AI ↔ CVE database (incl. CISA KEV) → upgrade recommendations",
			"Real SNMP throughput (Gbps) on gateways; v3/v2c node badge",
			"In-app version notifications: auto / manual update + What's New",
		},
		downloadsRepo + "/releases", "v1.6.0"},
	{"1.5.0", "2026-06-08", "stable",
		[]string{"Security posture score + history trend", "PDF/CSV export", "Per-segment risk drill-down"},
		downloadsRepo + "/releases", "v1.5.0"},
	{"1.4.0", "2026-06-03", "stable",
		[]string{"30+ vendor drivers with serial/uptime/build", "Network-learned Security AI + 0-100 score", "Multi-homed + HA + ARP-behind-gateway"},
		downloadsRepo + "/releases", "v1.4.0"},
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
	_ = releasesTmpl.Execute(w, map[string]any{"Releases": releases, "Repo": downloadsRepo})
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
