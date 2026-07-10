package main

import (
	"html/template"
	"net/http"
	"os"
)

/*
   ======================================================================
   MARKETING LANDING PAGE — cinematic dark-purple hero
   ----------------------------------------------------------------------
   Hero inspired by an immersive "wordmark + central focal object" layout,
   but the focal object is an animated 3D NETWORK GRAPH (canvas, vanilla
   JS) — on-brand for a network tool. Below the hero: feature bento,
   pricing, roadmap, blog. The full vendor catalogue lives at /docs.
   Self-contained HTML/CSS/JS from a Go template; reduced-motion aware.
   ======================================================================
*/

type blogPost struct {
	Date, Tag, Title, Excerpt string
}

type roadmapItem struct {
	Version, When, Title, Status string
	Points                       []string
}

var blogPosts = []blogPost{
	{"2026-06-03", "Security", "Network-learned Security AI + 0-100 score",
		"Findings now derive live from the discovered graph — default credentials, weak SNMP/TLS, flat segments, missing HA, unpatched Windows — and roll into one posture score."},
	{"2026-06-03", "Discovery", "Multi-homed devices, SNMP & far-side hosts",
		"Firewalls and load balancers are modelled as one device across every leg; hosts behind a NAT/SNIP are found via the gateway's ARP table — even with no route from the scanner."},
	{"2026-06-03", "Vendors", "30+ vendor drivers with deep facts",
		"Each driver pulls hostname, OS/build, serial, uptime and security posture over SSH/WinRM/SNMP, with its own colour-coded topology icon."},
}

var roadmap = []roadmapItem{
	{"v1.4", "Shipped", "Vendor depth & Security AI", "done",
		[]string{"30+ vendor drivers, serial/uptime/build", "Network-learned findings + 0-100 score", "Multi-homed + HA + ARP-behind-gateway"}},
	{"v1.5", "Shipped", "Scoring trends & reporting", "done",
		[]string{"Security-score history graph", "PDF/CSV posture export", "Per-segment risk drill-down"}},
	{"v1.6", "Shipping now", "SNMPv3, CVE advice & topology depth", "done",
		[]string{"SNMPv3 NoAuthNoPriv (v2c fallback)", "CVE-matched upgrade recommendations", "SNMP throughput (Gbps) + LLDP/CDP links"}},
	{"v2.0", "Next", "Appliance & SaaS", "next",
		[]string{"Docker server mode (live)", "FreeBSD A/B-ZFS appliance", "Continuous scheduled discovery + SSO"}},
}

func landingHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	data := map[string]any{
		"Posts":       blogPosts,
		"Roadmap":     roadmap,
		"CheckoutURL": envOr("PAYMENT_URL", "/signup?plan=pro"),
		"DemoURL":     envOr("DEMO_DOWNLOAD_URL", "https://github.com/lumina-netos/releases/latest"),
		"LinkedInURL": envOr("LINKEDIN_URL", "https://www.linkedin.com/company/lumina-netos"),
		"YouTubeURL":  envOr("YOUTUBE_URL", "https://www.youtube.com/@lumina-netos"),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = landingTmpl.Execute(w, data)
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

var landingTmpl = template.Must(template.New("landing").Parse(withAuthNav(landingHTML)))

const landingHTML = `<!doctype html><html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Lumina NetOS — Autonomous Network Discovery & Security AI</title>
<link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Sora:wght@600;700;800&family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
<style>
:root{
  --bg:#070512; --ink:#f2effb; --mut:#9b93c4;
  --vio:#a855f7; --mag:#d946ef; --ind:#6366f1; --cy:#22d3ee;
  --glass:rgba(168,148,230,.07); --brd:rgba(168,148,230,.16); --r:18px; --maxw:1140px;
}
*{box-sizing:border-box}
html{scroll-behavior:smooth}
body{margin:0;background:var(--bg);color:var(--ink);font-family:Inter,system-ui,Segoe UI,sans-serif;line-height:1.6;overflow-x:hidden}
a{color:inherit;text-decoration:none}
h1,h2,h3,h4{font-family:Sora,Inter,sans-serif;letter-spacing:-.02em;margin:0}
.wrap{max-width:var(--maxw);margin:0 auto;padding:0 24px}
:focus-visible{outline:2px solid var(--cy);outline-offset:3px;border-radius:8px}
.glass{background:var(--glass);border:1px solid var(--brd);border-radius:var(--r);backdrop-filter:blur(14px);-webkit-backdrop-filter:blur(14px)}
.btn{display:inline-flex;align-items:center;gap:8px;padding:12px 22px;border-radius:14px;font-weight:600;font-size:14px;cursor:pointer;border:1px solid transparent;transition:transform .2s,box-shadow .2s,background .2s;min-height:44px}
.btn:active{transform:scale(.97)}
.btn-primary{background:linear-gradient(120deg,var(--vio),var(--mag));color:#fff;box-shadow:0 8px 30px -8px rgba(217,70,239,.6)}
.btn-primary:hover{box-shadow:0 14px 44px -8px rgba(217,70,239,.8);transform:translateY(-2px)}
.btn-ghost{border-color:var(--brd);color:var(--ink);background:var(--glass)}
.btn-ghost:hover{border-color:var(--cy);transform:translateY(-2px)}

/* ---------- CINEMATIC HERO ---------- */
.cine{position:relative;min-height:100svh;display:flex;flex-direction:column;
  background:radial-gradient(120% 90% at 50% 35%,#2a1150 0%,#1a0a36 38%,#0a0518 72%,#070512 100%);overflow:hidden}
#net3d{position:absolute;inset:0;width:100%;height:100%;z-index:1}
.cine-nav{position:relative;z-index:5;display:flex;align-items:center;justify-content:space-between;padding:22px 36px}
.cine-nav .logo{font-family:Sora;font-weight:800;font-size:17px;letter-spacing:.5px}
.cine-nav .logo b{color:var(--mag)}
.menu{display:flex;gap:26px}
.menu a{position:relative;color:#e7e2f7;font-size:12.5px;font-weight:700;letter-spacing:.12em;opacity:.85;padding-bottom:4px}
.menu a:hover{opacity:1;color:#fff}
/* underline grows from the centre out to both sides on hover */
.menu a::after{content:"";position:absolute;left:50%;right:50%;bottom:0;height:2px;border-radius:2px;
  background:linear-gradient(90deg,var(--vio),var(--cy));transition:left .28s ease,right .28s ease}
.menu a:hover::after,.menu a:focus-visible::after{left:0;right:0}
@media(prefers-reduced-motion:reduce){.menu a::after{transition:none}}
@media(max-width:760px){.menu{display:none}}
.authtoggle{display:flex;background:rgba(20,10,40,.5);border:1px solid var(--brd);border-radius:999px;padding:5px;backdrop-filter:blur(10px)}
.authtoggle a{padding:9px 20px;border-radius:999px;font-size:13px;font-weight:700;color:#e7e2f7}
.authtoggle a.on{background:linear-gradient(120deg,var(--vio),var(--mag));color:#fff;box-shadow:0 6px 20px -6px rgba(217,70,239,.7)}

.stage{position:relative;z-index:4;flex:1;display:grid;place-items:center;text-align:center;padding:0 24px}
.wordmark{font-family:Sora;font-weight:800;font-size:clamp(64px,17vw,200px);line-height:.9;letter-spacing:.04em;
  color:transparent;-webkit-text-stroke:0;
  background:linear-gradient(180deg,rgba(255,255,255,.96),rgba(232,214,255,.62));-webkit-background-clip:text;background-clip:text;
  text-shadow:0 0 60px rgba(217,70,239,.25);pointer-events:none;user-select:none;mix-blend-mode:screen}
.tagline-mid{position:absolute;bottom:34px;left:0;right:0;text-align:center;color:#d8cff0;z-index:4}

.side{position:absolute;z-index:5;top:50%;transform:translateY(-50%);display:flex;flex-direction:column;align-items:center;gap:14px;color:#cbbff0}
.side.left{left:30px;writing-mode:vertical-rl}
.side.left .yr{font-size:12px;letter-spacing:.3em;font-weight:700}
.side.left .ln{width:1px;height:70px;background:linear-gradient(var(--mag),transparent)}
.side.right{right:30px}
.side.right a{width:34px;height:34px;display:grid;place-items:center;border-radius:10px;background:var(--glass);border:1px solid var(--brd)}
.side.right svg{width:16px;height:16px;fill:#e7e2f7}
.side.right a:hover{border-color:var(--mag)}
@media(max-width:760px){.side{display:none}}

.cine-foot{position:relative;z-index:5;display:flex;align-items:flex-end;justify-content:space-between;gap:20px;padding:0 36px 40px}
.about .kick{display:inline-flex;align-items:center;gap:8px;font-size:11px;font-weight:800;letter-spacing:.2em;color:var(--cy);text-transform:uppercase}
.about .kick::before{content:"";width:7px;height:7px;border-radius:50%;background:var(--cy);box-shadow:0 0 10px var(--cy)}
.about h2{font-size:clamp(22px,3vw,30px);font-weight:700;margin-top:10px;max-width:380px;
  background:linear-gradient(120deg,#fff,#e9b8ff);-webkit-background-clip:text;background-clip:text;color:transparent}
.watch{display:inline-flex;align-items:center;gap:12px;padding:13px 20px;border-radius:14px;font-weight:700;font-size:14px;
  background:rgba(20,10,40,.55);border:1px solid var(--brd);backdrop-filter:blur(10px)}
.watch .play{width:30px;height:30px;border-radius:9px;display:grid;place-items:center;background:linear-gradient(120deg,var(--vio),var(--mag))}
.watch .play svg{width:13px;height:13px;fill:#fff}
.watch:hover{border-color:var(--mag);transform:translateY(-2px);transition:.2s}
@media(max-width:760px){.cine-foot{flex-direction:column;align-items:flex-start}}

/* ---------- SECTIONS BELOW ---------- */
section.blk{padding:80px 0;position:relative}
.eyebrow{color:var(--cy);font-weight:700;font-size:13px;letter-spacing:.14em;text-transform:uppercase}
.h2{font-size:clamp(28px,4vw,38px);font-weight:700;margin:8px 0 10px}
.sub{color:var(--mut);max-width:620px;margin:0 0 34px}
.bento{display:grid;grid-template-columns:repeat(6,1fr);gap:16px}
.bento .card{padding:24px;border-radius:var(--r);transition:transform .25s,border-color .25s}
.bento .card:hover{transform:translateY(-4px);border-color:rgba(217,70,239,.4)}
.bento .card h3{font-size:18px;margin-bottom:8px}.bento .card p{color:var(--mut);font-size:14px;margin:0}
.c4{grid-column:span 4}.c3{grid-column:span 3}.c2{grid-column:span 2}
.ic{width:42px;height:42px;border-radius:12px;display:grid;place-items:center;margin-bottom:14px;background:linear-gradient(135deg,rgba(168,85,247,.28),rgba(34,211,238,.22));border:1px solid var(--brd)}
.ic svg{width:22px;height:22px;stroke:var(--cy)}
@media(max-width:860px){.bento{grid-template-columns:repeat(2,1fr)}.c4,.c3,.c2{grid-column:span 2}}

.prices{display:grid;grid-template-columns:repeat(3,1fr);gap:18px}
@media(max-width:860px){.prices{grid-template-columns:1fr}}
.price{padding:28px;border-radius:var(--r);position:relative}
.price.feat{border-color:rgba(217,70,239,.6);box-shadow:0 0 0 1px var(--mag),0 20px 60px -20px rgba(217,70,239,.5)}
.price .tier{font-size:13px;font-weight:700;color:var(--cy);text-transform:uppercase;letter-spacing:.1em}
.price .amt{font-family:Sora;font-size:40px;font-weight:800;margin:10px 0}.price .amt small{font-size:14px;color:var(--mut);font-weight:600}
.price ul{list-style:none;padding:0;margin:16px 0 22px}
.price li{padding:8px 0;color:var(--mut);font-size:14px;border-bottom:1px solid var(--brd);display:flex;gap:9px;align-items:center}
.price li svg{width:16px;height:16px;stroke:var(--cy);flex:0 0 auto}
.feat-tag{position:absolute;top:-12px;left:28px;background:linear-gradient(120deg,var(--vio),var(--mag));color:#fff;font-size:11px;font-weight:700;padding:4px 12px;border-radius:999px}

.tl{position:relative;padding-left:28px}
.tl::before{content:"";position:absolute;left:7px;top:6px;bottom:6px;width:2px;background:linear-gradient(var(--vio),var(--cy),transparent)}
.tl .node{position:relative;padding:0 0 26px 22px}
.tl .node::before{content:"";position:absolute;left:-25px;top:5px;width:14px;height:14px;border-radius:50%;background:var(--bg);border:2px solid var(--cy);box-shadow:0 0 12px var(--cy)}
.tl .node.done::before{background:var(--cy)}
.tl .ver{font-family:Sora;font-weight:800;font-size:16px}
.tl .when{font-size:11px;font-weight:700;color:var(--cy);text-transform:uppercase;letter-spacing:.08em;margin-left:8px}
.tl .node h4{margin:2px 0 8px;font-size:15px}.tl .node ul{margin:0;padding-left:18px;color:var(--mut);font-size:13.5px}.tl .node li{margin:3px 0}

.post{display:flex;gap:18px;padding:18px 20px;border-radius:14px;margin-bottom:12px;transition:border-color .2s}
.post:hover{border-color:rgba(34,211,238,.35)}
.post .date{color:var(--mut);font-size:12px;font-family:ui-monospace,monospace;min-width:92px;flex:0 0 auto}
.post .tag{display:inline-block;font-size:11px;font-weight:700;color:var(--mag);text-transform:uppercase;letter-spacing:.06em}
.post h4{margin:3px 0 6px;font-size:16px}.post p{margin:0;color:var(--mut);font-size:14px}
footer{padding:48px 0;border-top:1px solid var(--brd);color:var(--mut);font-size:13px;text-align:center}footer a{color:var(--cy)}

.reveal{opacity:0;transform:translateY(24px);transition:opacity .6s,transform .6s}
.reveal.in{opacity:1;transform:none}
@media(prefers-reduced-motion:reduce){.reveal{opacity:1;transform:none;transition:none}html{scroll-behavior:auto}}
</style></head><body>

<!-- ============ CINEMATIC HERO ============ -->
<div class="cine">
  <canvas id="net3d" aria-hidden="true"></canvas>

  <div class="cine-nav">
    <div class="logo">LUMINA <b>·</b> NETOS</div>
    <div class="menu">
      <a href="#features">FEATURES</a>
      <a href="#pricing">PRICING</a>
      <a href="#roadmap">ROADMAP</a>
      <a href="/releases">RELEASES</a>
      <a href="/docs">DOCS</a>
    </div>
    <div class="authtoggle">
      <a href="/signup" class="on">Sign Up</a>
      <a href="/signup">Sign In</a>
    </div>
  </div>

  <div class="stage"><h1 class="wordmark">LUMINA</h1></div>

  <div class="side left"><span class="yr">2026</span><span class="ln"></span></div>
  <div class="side right">
    <a href="{{.LinkedInURL}}" target="_blank" rel="noopener" aria-label="LinkedIn"><svg viewBox="0 0 24 24"><path d="M20.45 20.45h-3.56v-5.57c0-1.33-.02-3.04-1.85-3.04-1.86 0-2.14 1.45-2.14 2.94v5.67H9.34V9h3.42v1.56h.05c.48-.9 1.64-1.85 3.37-1.85 3.6 0 4.27 2.37 4.27 5.46v6.28zM5.34 7.43a2.07 2.07 0 1 1 0-4.14 2.07 2.07 0 0 1 0 4.14zM7.12 20.45H3.55V9h3.57v11.45zM22.22 0H1.77C.79 0 0 .77 0 1.73v20.54C0 23.23.79 24 1.77 24h20.45c.98 0 1.78-.77 1.78-1.73V1.73C24 .77 23.2 0 22.22 0z"/></svg></a>
    <a href="{{.YouTubeURL}}" target="_blank" rel="noopener" aria-label="YouTube"><svg viewBox="0 0 24 24"><path d="M23 7.5a3 3 0 0 0-2.1-2.13C19.05 5 12 5 12 5s-7.05 0-8.9.37A3 3 0 0 0 1 7.5 31.3 31.3 0 0 0 .62 12 31.3 31.3 0 0 0 1 16.5a3 3 0 0 0 2.1 2.13C4.95 19 12 19 12 19s7.05 0 8.9-.37A3 3 0 0 0 23 16.5 31.3 31.3 0 0 0 23.38 12 31.3 31.3 0 0 0 23 7.5zM9.75 15.5v-7l6 3.5z"/></svg></a>
  </div>

  <div class="cine-foot">
    <div class="about">
      <span class="kick">Platform</span>
      <h2>See every device.<br>Score every risk.</h2>
    </div>
    <a class="watch" href="{{.DemoURL}}">
      <span>Watch a tutorial</span>
      <span class="play"><svg viewBox="0 0 24 24"><path d="M8 5v14l11-7z"/></svg></span>
    </a>
  </div>
</div>

<!-- ============ FEATURES ============ -->
<section id="features" class="blk wrap">
  <div class="reveal"><span class="eyebrow">Built for real infrastructure</span>
  <h2 class="h2">It talks to your actual gear — and learns from it</h2>
  <p class="sub">Not a mock. Every finding is derived live from what the engine discovers. <a href="/docs" style="color:var(--cy)">See all 30+ vendors →</a></p></div>
  <div class="bento">
    <div class="card glass c4 reveal"><div class="ic"><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><circle cx="12" cy="12" r="3"/><path d="M12 2v4M12 18v4M2 12h4M18 12h4"/></svg></div><h3>30+ vendor drivers</h3><p>Cisco, Fortinet, Palo Alto, NetScaler, Nutanix, VMware ESXi/Horizon, Citrix, Proxmox, Synology, QNAP, MikroTik, Ruckus &amp; more — each with model, serial, OS build &amp; uptime.</p></div>
    <div class="card glass c2 reveal"><div class="ic"><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg></div><h3>Security AI</h3><p>Default creds, weak SNMP/TLS, exposed mgmt, flat segments, missing HA, unpatched Windows.</p></div>
    <div class="card glass c2 reveal"><div class="ic"><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M3 12h4l3 8 4-16 3 8h4"/></svg></div><h3>0-100 posture score</h3><p>A grade that adapts as you remediate and re-scan.</p></div>
    <div class="card glass c4 reveal"><div class="ic"><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/><path d="M10 6h6a2 2 0 0 1 2 2v6"/></svg></div><h3>Multi-homed &amp; HA aware</h3><p>One device across all its legs, hosts discovered behind firewalls via ARP, and animated HA-sync links between redundant pairs.</p></div>
    <div class="card glass c3 reveal"><div class="ic"><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><rect x="3" y="4" width="18" height="16" rx="2"/><path d="M3 9h18"/></svg></div><h3>Live topology map</h3><p>Colour-coded auto-layout, per-link latency, path tracing, one-click RDP / SSH.</p></div>
    <div class="card glass c3 reveal"><div class="ic"><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M12 1v6M12 17v6M4.2 4.2l4.3 4.3M15.5 15.5l4.3 4.3M1 12h6M17 12h6"/></svg></div><h3>LDAP &amp; MFA</h3><p>Real LDAP bind auth, per-user TOTP, role-based access.</p></div>
  </div>
</section>

<!-- ============ PRICING ============ -->
<section id="pricing" class="blk wrap">
  <div class="reveal"><span class="eyebrow">Pricing</span><h2 class="h2">Start free. Upgrade for production.</h2></div>
  <div class="prices">
    <div class="price glass reveal"><div class="tier">Community</div><div class="amt">$0</div>
      <ul><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>Demo-mode topology</li><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>Security AI preview</li><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>Single workstation</li></ul>
      <a class="btn btn-ghost" style="width:100%;justify-content:center" href="/signup?next=download">Download demo</a></div>
    <div class="price glass feat reveal"><span class="feat-tag">Most popular</span><div class="tier">Pro</div><div class="amt">$49 <small>/mo</small></div>
      <ul><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>Production scanning</li><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>All vendor drivers</li><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>LDAP + MFA</li><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>Posture score</li></ul>
      <a class="btn btn-primary" style="width:100%;justify-content:center" href="{{.CheckoutURL}}">Buy Pro</a></div>
    <div class="price glass reveal"><div class="tier">Enterprise</div><div class="amt">Custom</div>
      <ul><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>Unlimited assets</li><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>On-prem / air-gapped</li><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>SSO &amp; audit export</li><li><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>Priority SLA</li></ul>
      <a class="btn btn-ghost" style="width:100%;justify-content:center" href="/signup?plan=enterprise">Contact sales</a></div>
  </div>
</section>

<!-- ============ ROADMAP ============ -->
<section id="roadmap" class="blk wrap">
  <div class="reveal"><span class="eyebrow">What's next</span><h2 class="h2">Roadmap &amp; future versions</h2>
  <p class="sub">Pro &amp; Enterprise get new feature releases on their channel.</p></div>
  <div class="tl">{{range .Roadmap}}
    <div class="node reveal {{if eq .Status "done"}}done{{end}}"><span class="ver">{{.Version}}</span><span class="when">{{.When}}</span>
      <h4>{{.Title}}</h4><ul>{{range .Points}}<li>{{.}}</li>{{end}}</ul></div>
  {{end}}</div>
</section>

<!-- ============ BLOG ============ -->
<section id="blog" class="blk wrap">
  <div class="reveal"><span class="eyebrow">Product updates</span><h2 class="h2">From the changelog</h2></div>
  {{range .Posts}}<div class="post glass reveal"><div class="date">{{.Date}}</div><div><span class="tag">{{.Tag}}</span><h4>{{.Title}}</h4><p>{{.Excerpt}}</p></div></div>{{end}}
</section>

<footer class="wrap">© 2026 Lumina NetOS · <a href="/docs">Docs</a> · <a href="/signup">Sign in</a> · <a href="/signup?next=download">Download demo</a></footer>

<script>
// ---- 3D NETWORK GRAPH (canvas, vanilla) ----
(function(){
  var c=document.getElementById('net3d'); if(!c) return;
  var ctx=c.getContext('2d'), W,H,DPR=Math.min(devicePixelRatio||1,2);
  var reduce=matchMedia('(prefers-reduced-motion:reduce)').matches;
  var COLORS=['#a855f7','#d946ef','#22d3ee','#8b5cf6','#f0abfc'];
  var N=44, R=1, nodes=[], edges=[];
  function rand(a,b){return a+Math.random()*(b-a);}
  for(var i=0;i<N;i++){ // points in a rough sphere shell
    var u=Math.random()*2-1, t=Math.random()*Math.PI*2, s=Math.sqrt(1-u*u);
    var rr=rand(.55,1);
    nodes.push({x:s*Math.cos(t)*rr, y:u*rr, z:s*Math.sin(t)*rr,
      col:COLORS[i%COLORS.length], sz:rand(1.4,3.2)});
  }
  for(var a=0;a<N;a++)for(var b=a+1;b<N;b++){
    var dx=nodes[a].x-nodes[b].x,dy=nodes[a].y-nodes[b].y,dz=nodes[a].z-nodes[b].z;
    if(dx*dx+dy*dy+dz*dz < 0.34) edges.push([a,b]);
  }
  function resize(){ W=c.clientWidth;H=c.clientHeight;c.width=W*DPR;c.height=H*DPR;ctx.setTransform(DPR,0,0,DPR,0,0); }
  window.addEventListener('resize',resize); resize();

  var ang=0;
  function proj(p){ // rotate yaw + slight pitch, perspective project
    var ca=Math.cos(ang),sa=Math.sin(ang);
    var x=p.x*ca - p.z*sa, z=p.x*sa + p.z*ca, y=p.y;
    var pitch=0.38, cp=Math.cos(pitch),sp=Math.sin(pitch);
    var y2=y*cp - z*sp, z2=y*sp + z*cp;
    var scale=Math.min(W,H)* (reduce?0.30:0.34);
    var f=2.4/(2.4+z2);
    return {sx:W/2 + x*scale*f, sy:H/2 + y2*scale*f, f:f, z:z2};
  }
  function frame(){
    ctx.clearRect(0,0,W,H);
    var pts=nodes.map(proj);
    // edges
    for(var e=0;e<edges.length;e++){
      var p=pts[edges[e][0]], q=pts[edges[e][1]];
      var g=ctx.createLinearGradient(p.sx,p.sy,q.sx,q.sy);
      g.addColorStop(0,'rgba(168,85,247,'+(0.10+0.18*p.f)+')');
      g.addColorStop(1,'rgba(34,211,238,'+(0.10+0.18*q.f)+')');
      ctx.strokeStyle=g; ctx.lineWidth=0.6*Math.max(p.f,q.f);
      ctx.beginPath();ctx.moveTo(p.sx,p.sy);ctx.lineTo(q.sx,q.sy);ctx.stroke();
    }
    // nodes (glow)
    for(var i=0;i<N;i++){
      var p=pts[i], r=nodes[i].sz*p.f;
      var rg=ctx.createRadialGradient(p.sx,p.sy,0,p.sx,p.sy,r*5);
      rg.addColorStop(0,nodes[i].col); rg.addColorStop(.4,nodes[i].col+'88'); rg.addColorStop(1,'transparent');
      ctx.fillStyle=rg; ctx.beginPath();ctx.arc(p.sx,p.sy,r*5,0,7);ctx.fill();
      ctx.fillStyle='#fff'; ctx.globalAlpha=.85*p.f; ctx.beginPath();ctx.arc(p.sx,p.sy,r,0,7);ctx.fill();ctx.globalAlpha=1;
    }
    if(!reduce){ ang+=0.0024; requestAnimationFrame(frame); }
  }
  frame();
})();

// ---- scroll reveal (stagger) ----
(function(){
  var io=new IntersectionObserver(function(es){es.forEach(function(e){if(e.isIntersecting){var el=e.target;
    setTimeout(function(){el.classList.add('in')},(el.dataset.delay||0)*1);io.unobserve(el);}});},
    {threshold:.12,rootMargin:'0px 0px -8% 0px'});
  document.querySelectorAll('.reveal').forEach(function(el,i){el.dataset.delay=(i%6)*40;io.observe(el);});
})();
</script>
</body></html>`
