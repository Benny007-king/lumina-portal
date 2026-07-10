package main

import (
	"html/template"
	"net/http"
)

/*
   ======================================================================
   DOCUMENTATION  (/docs)
   ----------------------------------------------------------------------
   Moved off the marketing hero: the full supported-vendor catalogue plus
   getting-started / security-coverage reference, in a clean glass-doc
   layout consistent with the landing page.
   ======================================================================
*/

type vendorCat struct {
	Name    string
	Vendors []string
}

// vendorCatalog is the public, grouped vendor list (shown on /docs).
var vendorCatalog = []vendorCat{
	{"Firewalls & Security", []string{"Check Point", "Fortinet", "Palo Alto", "Sophos", "SonicWall", "pfSense / OPNsense", "F5 BIG-IP"}},
	{"Load Balancers / ADC", []string{"Citrix NetScaler / ADC"}},
	{"Switches & Routers", []string{"Cisco", "Arista", "Juniper", "HP / Aruba", "Huawei", "MikroTik", "Extreme", "Ubiquiti / UniFi", "Bezeq Home"}},
	{"Wireless", []string{"Ruckus SmartZone", "UniFi", "Aruba"}},
	{"Servers & Hypervisors", []string{"VMware ESXi", "Nutanix AHV", "Proxmox VE", "Linux / Unix", "Windows Server", "Dell iDRAC"}},
	{"VDI & Apps", []string{"VMware Horizon", "Citrix Virtual Apps/Desktops"}},
	{"Storage (NAS)", []string{"Synology DSM", "QNAP QTS", "TrueNAS"}},
	{"Endpoints & Mobile", []string{"Windows Workstation", "iPhone / Android (BYOD)"}},
	{"Management", []string{"SNMP v2c (read-only)"}},
}

// securityChecks documents what the Security AI evaluates.
var securityChecks = []string{
	"Default credentials (banner-detected, e.g. NetScaler RPC)",
	"Weak SNMP community (public/private) — config & live",
	"Deprecated TLS / weak ciphers (SSLv3, TLS 1.0, RC4, 3DES)",
	"Cleartext management (Telnet 23, FTP 21, HTTP-only admin)",
	"Exposed management plane on a user/endpoint segment",
	"Flat lateral-movement paths (endpoint → server, no firewall)",
	"Missing HA / single point of failure on critical devices",
	"Wireless reaching a server segment without isolation",
	"Unmanaged / shadow assets (discovered, never authenticated)",
	"Unpatched Windows (latest update > 60 days old)",
}

func docsHandler(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"Vendors": vendorCatalog,
		"Checks":  securityChecks,
		"Guides":  docLibrary,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = docsTmpl.Execute(w, data)
}

var docsTmpl = template.Must(template.New("docs").Parse(withAuthNav(docsHTML)))

const docsHTML = `<!doctype html><html lang="en"><head>
<meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Lumina NetOS — Documentation</title>
<link rel="preconnect" href="https://fonts.googleapis.com"><link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Sora:wght@600;700;800&family=Inter:wght@400;500;600&display=swap" rel="stylesheet">
<style>
:root{--bg:#070512;--ink:#f2effb;--mut:#9b93c4;--vio:#a855f7;--mag:#d946ef;--cy:#22d3ee;
  --glass:rgba(168,148,230,.07);--brd:rgba(168,148,230,.16);--r:18px;--maxw:1080px}
*{box-sizing:border-box}
html{scroll-behavior:smooth}
body{margin:0;background:var(--bg);color:var(--ink);font-family:Inter,system-ui,sans-serif;line-height:1.6;overflow-x:hidden}
a{color:inherit;text-decoration:none}
h1,h2,h3{font-family:Sora,sans-serif;letter-spacing:-.02em;margin:0}
.wrap{max-width:var(--maxw);margin:0 auto;padding:0 24px}
:focus-visible{outline:2px solid var(--cy);outline-offset:3px;border-radius:8px}
.glass{background:var(--glass);border:1px solid var(--brd);border-radius:var(--r);backdrop-filter:blur(14px);-webkit-backdrop-filter:blur(14px)}
.btn{display:inline-flex;align-items:center;gap:8px;padding:10px 18px;border-radius:12px;font-weight:700;font-size:13px;border:1px solid transparent;transition:transform .2s,box-shadow .2s}
.btn-primary{background:linear-gradient(120deg,var(--vio),var(--mag));color:#fff;box-shadow:0 8px 30px -8px rgba(217,70,239,.6)}
.btn-primary:hover{transform:translateY(-2px)}

/* nav — matches landing */
nav{position:sticky;top:0;z-index:20;display:flex;justify-content:space-between;align-items:center;padding:16px 36px;border-bottom:1px solid var(--brd);background:rgba(7,5,18,.65);backdrop-filter:blur(14px)}
.logo{font-family:Sora;font-weight:800;letter-spacing:.5px}.logo b{color:var(--mag)}
.menu{display:flex;gap:24px}
.menu a{position:relative;color:#e7e2f7;font-size:12.5px;font-weight:700;letter-spacing:.12em;opacity:.85;padding-bottom:4px}
.menu a:hover{opacity:1}
.menu a::after{content:"";position:absolute;left:50%;right:50%;bottom:0;height:2px;border-radius:2px;
  background:linear-gradient(90deg,var(--vio),var(--cy));transition:left .28s ease,right .28s ease}
.menu a:hover::after,.menu a:focus-visible::after{left:0;right:0}
@media(max-width:760px){.menu{display:none}}
.authtoggle{display:flex;gap:8px;align-items:center}
.btn-ghost{padding:9px 16px;border-radius:12px;font-weight:700;font-size:13px;color:#e7e2f7;border:1px solid var(--brd);background:var(--glass);transition:transform .2s,border-color .2s}
.btn-ghost:hover{border-color:var(--cy);transform:translateY(-2px)}
@media(prefers-reduced-motion:reduce){.menu a::after{transition:none}}

/* cinematic hero band */
.dochero{position:relative;padding:84px 0 48px;overflow:hidden;
  background:radial-gradient(120% 120% at 50% -10%,#2a1150 0%,#1a0a36 40%,#0a0518 75%,#070512 100%)}
.dochero::after{content:"";position:absolute;inset:0;pointer-events:none;
  background:radial-gradient(600px 300px at 80% 0,rgba(34,211,238,.12),transparent 70%)}
.eyebrow{color:var(--cy);font-weight:700;font-size:13px;letter-spacing:.14em;text-transform:uppercase}
.dochero h1{font-size:clamp(34px,6vw,54px);margin:10px 0;line-height:1.05;
  background:linear-gradient(120deg,#fff,#e9b8ff 60%,#22d3ee);-webkit-background-clip:text;background-clip:text;color:transparent}
.lead{color:var(--mut);max-width:660px;font-size:16px}
.docnav{display:flex;gap:8px;flex-wrap:wrap;margin-top:24px}
.docnav a{padding:8px 14px;border-radius:999px;font-size:13px;font-weight:600;color:var(--mut)}
.docnav a:hover{color:var(--ink);border-color:var(--cy)}

section{padding:54px 0;position:relative}
.h2{font-size:clamp(24px,3.4vw,32px);font-weight:700;margin-bottom:6px}
.sub{color:var(--mut);margin:0 0 26px;font-size:14px}
.cat{margin-bottom:26px}
.cat h3{font-size:13px;color:var(--cy);margin-bottom:12px;font-family:Inter;font-weight:800;text-transform:uppercase;letter-spacing:.1em}
.vgrid{display:grid;grid-template-columns:repeat(auto-fill,minmax(160px,1fr));gap:12px}
.vcard{display:flex;align-items:center;gap:11px;padding:12px 14px;border-radius:14px;transition:transform .2s,border-color .2s}
.vcard:hover{transform:translateY(-3px);border-color:rgba(217,70,239,.4)}
.badge{width:32px;height:32px;border-radius:10px;display:grid;place-items:center;font-family:Sora;font-weight:800;font-size:12px;color:#fff;background:linear-gradient(135deg,var(--vio),var(--mag) 55%,var(--cy));flex:0 0 auto}
.vn{font-size:13px;font-weight:600}
.checks{list-style:none;padding:0;margin:0;display:grid;grid-template-columns:repeat(auto-fill,minmax(290px,1fr));gap:12px}
.checks li{display:flex;gap:10px;align-items:flex-start;padding:14px 16px;border-radius:14px;font-size:13.5px;color:var(--mut)}
.checks li svg{width:16px;height:16px;stroke:var(--cy);flex:0 0 auto;margin-top:3px}
.steps{counter-reset:s;list-style:none;padding:0;display:grid;gap:12px}
.steps li{counter-increment:s;position:relative;padding:16px 16px 16px 56px;border-radius:14px;color:var(--mut)}
.steps li::before{content:counter(s);position:absolute;left:14px;top:14px;width:30px;height:30px;border-radius:9px;display:grid;place-items:center;font-family:Sora;font-weight:800;font-size:13px;color:#fff;background:linear-gradient(135deg,var(--vio),var(--cy))}
.steps b{color:var(--ink)}
.guides{display:grid;grid-template-columns:repeat(auto-fill,minmax(320px,1fr));gap:14px}
.guide{display:flex;align-items:center;gap:14px;padding:18px 18px;border-radius:16px;transition:transform .2s,border-color .2s}
.guide:hover{transform:translateY(-3px);border-color:rgba(34,211,238,.45)}
.gkind{flex:0 0 auto;width:46px;height:46px;border-radius:12px;display:grid;place-items:center;font-family:Sora;font-weight:800;font-size:13px;color:#fff;background:linear-gradient(135deg,var(--vio),var(--mag) 55%,var(--cy))}
.gbody{display:flex;flex-direction:column;gap:3px;min-width:0}
.gt{font-size:14.5px;font-weight:700;color:var(--ink)}
.gd{font-size:12.5px;color:var(--mut);line-height:1.45}
.gsize{font-size:11px;color:var(--cy);font-family:ui-monospace,monospace;margin-top:2px}
.gdl{width:20px;height:20px;stroke:var(--cy);flex:0 0 auto;margin-left:auto;opacity:.7;transition:opacity .2s,transform .2s}
.guide:hover .gdl{opacity:1;transform:translateY(2px)}
footer{padding:44px 0;border-top:1px solid var(--brd);color:var(--mut);font-size:13px;text-align:center}footer a{color:var(--cy)}
.reveal{opacity:0;transform:translateY(22px);transition:opacity .6s,transform .6s}
.reveal.in{opacity:1;transform:none}
@media(prefers-reduced-motion:reduce){.reveal{opacity:1;transform:none;transition:none}html{scroll-behavior:auto}}
</style></head><body>

<nav>
  <div class="logo">LUMINA <b>·</b> NETOS</div>
  <div class="menu"><a href="/">HOME</a><a href="/#features">FEATURES</a><a href="/#pricing">PRICING</a><a href="/#roadmap">ROADMAP</a></div>
  <div class="authtoggle">
    <a class="btn-ghost" href="/signup">Sign In</a>
    <a class="btn btn-primary" href="/signup">Sign Up</a>
  </div>
</nav>

<div class="dochero">
  <div class="wrap">
    <span class="eyebrow">Documentation</span>
    <h1>Supported vendors<br>&amp; security coverage</h1>
    <p class="lead">Lumina NetOS ships 30+ vendor drivers. Each pulls inventory — hostname, model, serial, OS build, uptime — and security posture over its native protocol: SSH, WinRM or SNMP.</p>
    <div class="docnav">
      <a class="glass" href="#start">Getting started</a>
      <a class="glass" href="#vendors">Vendors</a>
      <a class="glass" href="#coverage">Security coverage</a>
      <a class="glass" href="#guides">Guides &amp; downloads</a>
    </div>
  </div>
</div>

<section id="start" class="wrap">
  <h2 class="h2 reveal">Getting started</h2>
  <p class="sub reveal">Three steps from install to a scored map.</p>
  <ol class="steps">
    <li class="glass reveal"><b>Add credentials.</b> In Settings → Credentials, set one secret per vendor tag (and an SNMP community).</li>
    <li class="glass reveal"><b>Run a scan.</b> Enter a seed IP (your gateway or a known host). The engine sweeps the /24, fingerprints vendors, and spreads via ARP / neighbor tables.</li>
    <li class="glass reveal"><b>Review the score.</b> The Security AI analyzes the live graph and produces a 0-100 posture score with prioritized findings.</li>
  </ol>
</section>

<section id="vendors" class="wrap">
  <h2 class="h2 reveal">Supported vendors</h2>
  <p class="sub reveal">Grouped by role. Coverage grows every release.</p>
  {{range .Vendors}}
  <div class="cat reveal"><h3>{{.Name}}</h3>
    <div class="vgrid">{{range .Vendors}}
      <div class="vcard glass"><span class="badge">{{slice . 0 1}}</span><span class="vn">{{.}}</span></div>
    {{end}}</div>
  </div>{{end}}
</section>

<section id="coverage" class="wrap">
  <h2 class="h2 reveal">Security AI coverage</h2>
  <p class="sub reveal">What the analyzer evaluates on every scan — all derived live from the discovered graph.</p>
  <ul class="checks">{{range .Checks}}
    <li class="glass reveal"><svg fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M20 6L9 17l-5-5"/></svg>{{.}}</li>
  {{end}}</ul>
</section>

<section id="guides" class="wrap">
  <h2 class="h2 reveal">Guides &amp; downloads</h2>
  <p class="sub reveal">Operator documentation — security hardening, firewall ports and deployment notes. Open in the browser or save a copy.</p>
  <div class="guides">{{range .Guides}}
    <a class="guide glass reveal" href="{{.Path}}" target="_blank" rel="noopener">
      <span class="gkind">{{.Kind}}</span>
      <span class="gbody"><span class="gt">{{.Title}}</span><span class="gd">{{.Desc}}</span><span class="gsize">{{.Size}}</span></span>
      <svg class="gdl" fill="none" stroke-width="2" viewBox="0 0 24 24"><path d="M12 3v12m0 0l-4-4m4 4l4-4M5 21h14"/></svg>
    </a>
  {{end}}</div>
</section>

<footer class="wrap">© 2026 Lumina NetOS · <a href="/">Home</a> · <a href="/signup">Sign in</a></footer>

<script>
(function(){
  var io=new IntersectionObserver(function(es){es.forEach(function(e){if(e.isIntersecting){var el=e.target;
    setTimeout(function(){el.classList.add('in')},(el.dataset.delay||0)*1);io.unobserve(el);}});},
    {threshold:.1,rootMargin:'0px 0px -6% 0px'});
  document.querySelectorAll('.reveal').forEach(function(el,i){el.dataset.delay=(i%5)*40;io.observe(el);});
})();
</script>
</body></html>`
