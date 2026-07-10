package main

// portalHTML is the single-page licensing UI served at "/".
const portalHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<meta name="viewport" content="width=device-width, initial-scale=1.0" />
<title>Lumina NetOS - Activate Production</title>
<style>
  * { box-sizing: border-box; margin: 0; padding: 0; font-family: ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, sans-serif; }
  body { min-height: 100vh; background: radial-gradient(1200px 600px at 50% -10%, #1e1b4b 0%, #020617 55%); color: #e2e8f0; display: flex; align-items: center; justify-content: center; padding: 24px; }
  .card { width: 100%; max-width: 460px; background: rgba(15,23,42,.8); border: 1px solid #1e293b; border-radius: 20px; padding: 32px; box-shadow: 0 30px 80px rgba(0,0,0,.5); backdrop-filter: blur(10px); }
  .brand { display:flex; align-items:center; gap:12px; justify-content:center; margin-bottom: 6px; }
  .logo { width: 40px; height: 40px; border-radius: 12px; background: linear-gradient(135deg,#6366f1,#8b5cf6); display:flex; align-items:center; justify-content:center; font-weight:800; color:white; box-shadow:0 8px 24px rgba(99,102,241,.4); }
  h1 { font-size: 15px; letter-spacing: .18em; text-transform: uppercase; text-align:center; }
  .sub { text-align:center; color:#64748b; font-size: 11px; font-family: ui-monospace, monospace; margin: 4px 0 22px; text-transform: uppercase; letter-spacing:.1em; }
  .tabs { display:grid; grid-template-columns: 1fr 1fr; gap:6px; background:#020617; border:1px solid #1e293b; border-radius:12px; padding:5px; margin-bottom: 20px; }
  .tab { padding:9px; border:none; background:transparent; color:#94a3b8; font-weight:700; font-size:12px; text-transform:uppercase; letter-spacing:.08em; border-radius:8px; cursor:pointer; transition:.15s; }
  .tab.active { background:#4f46e5; color:white; box-shadow:0 4px 12px rgba(79,70,229,.4); }
  label { display:block; font-size:10px; font-weight:700; text-transform:uppercase; letter-spacing:.1em; color:#94a3b8; margin: 14px 0 6px; }
  input { width:100%; background:#020617; border:1px solid #1e293b; border-radius:10px; padding:11px 13px; color:#e2e8f0; font-size:14px; outline:none; transition:.15s; }
  input:focus { border-color:#6366f1; box-shadow:0 0 0 3px rgba(99,102,241,.15); }
  .btn { width:100%; margin-top:20px; padding:12px; border:none; border-radius:10px; background:linear-gradient(135deg,#4f46e5,#7c3aed); color:white; font-weight:700; font-size:14px; cursor:pointer; box-shadow:0 8px 20px rgba(79,70,229,.35); transition:.15s; }
  .btn:hover { filter:brightness(1.1); }
  .btn:disabled { opacity:.6; cursor:not-allowed; }
  .oauth { display:grid; grid-template-columns:1fr 1fr; gap:10px; margin-top:16px; }
  .oauth button { padding:11px 12px; border-radius:10px; font-size:13px; font-weight:600; cursor:pointer; display:flex; align-items:center; justify-content:center; gap:9px; transition:.15s; border:1px solid transparent; }
  .oauth button svg { width:18px; height:18px; }
  .btn-google { background:#ffffff; color:#1f1f1f; border-color:#dadce0 !important; }
  .btn-google:hover { background:#f7f8f8; box-shadow:0 1px 3px rgba(60,64,67,.3); }
  .btn-github { background:#24292f; color:#ffffff; border-color:#24292f !important; }
  .btn-github:hover { background:#32383f; }
  .divider { display:flex; align-items:center; gap:12px; margin:18px 0 4px; color:#475569; font-size:10px; text-transform:uppercase; letter-spacing:.1em; }
  .divider::before,.divider::after { content:""; flex:1; height:1px; background:#1e293b; }
  .msg { margin-top:14px; padding:11px 13px; border-radius:10px; font-size:12px; line-height:1.5; }
  .msg.err { background:rgba(244,63,94,.1); border:1px solid rgba(244,63,94,.25); color:#fb7185; }
  .keybox { margin-top:8px; }
  .keyval { width:100%; background:#020617; border:1px dashed #4f46e5; border-radius:10px; padding:12px; color:#a5b4fc; font-family:ui-monospace,monospace; font-size:11px; word-break:break-all; line-height:1.6; }
  .copybtn { margin-top:10px; width:100%; padding:10px; border:1px solid #4f46e5; background:rgba(79,70,229,.12); color:#c7d2fe; border-radius:10px; font-weight:700; font-size:12px; cursor:pointer; }
  .hint { color:#64748b; font-size:11px; line-height:1.6; margin-top:14px; }
  .success-icon { width:54px;height:54px;border-radius:50%;background:rgba(16,185,129,.12);border:1px solid rgba(16,185,129,.3);display:flex;align-items:center;justify-content:center;margin:0 auto 14px;font-size:26px; }
</style>
</head>
<body>
<div class="card">
  <div class="brand"><div class="logo">L</div></div>
  <h1>Lumina NetOS</h1>
  <div class="sub">Production License Activation</div>

  <div id="auth-view">
    <div class="tabs">
      <button class="tab active" id="tab-register" onclick="switchTab('register')">Create Account</button>
      <button class="tab" id="tab-login" onclick="switchTab('login')">Sign In</button>
    </div>

    <div id="org-field">
      <label>Organization</label>
      <input id="org" placeholder="Acme Corp" autocomplete="organization" />
    </div>
    <label>Email</label>
    <input id="email" type="email" placeholder="you@company.com" autocomplete="email" />
    <label>Password</label>
    <input id="password" type="password" placeholder="••••••••" autocomplete="current-password" />

    <button class="btn" id="submit" onclick="submitAuth()">Create Account & Generate Key</button>

    <div class="divider">or continue with</div>
    <div class="oauth">
      <button class="btn-google" id="btn-google" onclick="oauth('google')">
        <svg viewBox="0 0 48 48" xmlns="http://www.w3.org/2000/svg"><path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z"/><path fill="#4285F4" d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z"/><path fill="#FBBC05" d="M10.53 28.59c-.48-1.45-.76-2.99-.76-4.59s.27-3.14.76-4.59l-7.98-6.19C.92 16.46 0 20.12 0 24c0 3.88.92 7.54 2.56 10.78l7.97-6.19z"/><path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z"/></svg>
        Google
      </button>
      <button class="btn-github" id="btn-github" onclick="oauth('github')">
        <svg viewBox="0 0 16 16" fill="currentColor" xmlns="http://www.w3.org/2000/svg"><path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.01 8.01 0 0 0 16 8c0-4.42-3.58-8-8-8z"/></svg>
        GitHub
      </button>
    </div>

    <div id="msg"></div>
  </div>

  <div id="key-view" style="display:none;">
    <div class="success-icon">✓</div>
    <h1 style="margin-bottom:4px;">Account Ready</h1>
    <div class="sub" id="org-label"></div>
    <label>Your Production License Key</label>
    <div class="keybox">
      <div class="keyval" id="license"></div>
      <button class="copybtn" onclick="copyKey()">Copy License Key</button>
    </div>
    <p class="hint">
      Open Lumina NetOS, switch to <b>Production</b>, and on the login screen paste this key
      together with your admin password to activate. Keep it safe, it identifies your organization.
    </p>
    <a class="btn" href="%%DOWNLOAD_URL%%" target="_blank" rel="noopener" style="display:block;text-align:center;text-decoration:none;background:linear-gradient(120deg,#6366f1,#06b6d4);margin-bottom:10px;">⬇ Download Lumina NetOS</a>
    <button class="btn" onclick="revokeKey()" style="background:#7f1d1d;margin-bottom:10px;">Revoke &amp; Reissue Key</button>
    <button class="btn" onclick="signOut()" style="background:#1e293b;">Sign out</button>
  </div>
</div>

<script>
  var mode = 'register';
  function switchTab(m){
    mode = m;
    document.getElementById('tab-register').classList.toggle('active', m==='register');
    document.getElementById('tab-login').classList.toggle('active', m==='login');
    document.getElementById('org-field').style.display = m==='register' ? 'block' : 'none';
    document.getElementById('submit').textContent = m==='register' ? 'Create Account & Generate Key' : 'Sign In & Retrieve Key';
    document.getElementById('msg').innerHTML = '';
  }
  function showErr(t){ document.getElementById('msg').innerHTML = '<div class="msg err">'+t+'</div>'; }

  var oauthEnabled = { google:false, github:false };
  function oauth(p){
    if(oauthEnabled[p]){ window.location.href = '/auth/' + p; }
    else { showErr(p.charAt(0).toUpperCase()+p.slice(1)+' sign-in is not configured on this server yet. Use email & password, or set the provider client ID/secret.'); }
  }

  // Discover which OAuth providers are live.
  fetch('/api/oauth-config').then(function(r){return r.json();}).then(function(d){
    oauthEnabled = d;
    if(!d.google) document.getElementById('btn-google').style.opacity = '.6';
    if(!d.github) document.getElementById('btn-github').style.opacity = '.6';
  }).catch(function(){});

  // Handle return from an OAuth round-trip.
  (function(){
    var params = new URLSearchParams(window.location.search);
    var err = params.get('oauth_error');
    if(err){ showErr(err); history.replaceState({}, '', '/'); }
    var claim = params.get('claim');
    if(claim){
      fetch('/api/claim?id=' + encodeURIComponent(claim)).then(function(r){return r.json();}).then(function(d){
        history.replaceState({}, '', '/');
        if(d.licenseKey){
          document.getElementById('license').textContent = d.licenseKey;
          document.getElementById('org-label').textContent = d.org + ' · ' + d.email;
          document.getElementById('auth-view').style.display='none';
          document.getElementById('key-view').style.display='block';
        } else { showErr(d.error || 'Could not retrieve your license key.'); }
      }).catch(function(){ showErr('Could not retrieve your license key.'); });
    }
  })();

  // Already signed in? (cookie session) → show the key view straight away.
  function showKey(d){
    document.getElementById('license').textContent = d.licenseKey;
    document.getElementById('org-label').textContent = d.org + ' · ' + d.email;
    document.getElementById('auth-view').style.display='none';
    document.getElementById('key-view').style.display='block';
  }
  fetch('/api/me').then(function(r){ return r.ok ? r.json() : null; }).then(function(d){
    if(d && d.licenseKey){ showKey(d); }
  }).catch(function(){});

  function signOut(){
    fetch('/api/logout', {method:'POST'}).then(function(){
      document.getElementById('key-view').style.display='none';
      document.getElementById('auth-view').style.display='block';
      document.getElementById('password').value='';
    });
  }

  function revokeKey(){
    if(!confirm('Revoke your current License Key and issue a new one?\n\nThe old key stops working for organization sync within ~15 minutes. Every install using it must be re-activated with the new key.')) return;
    fetch('/api/revoke-key', {method:'POST'}).then(function(r){ return r.ok ? r.json() : null; }).then(function(d){
      if(d && d.licenseKey){
        document.getElementById('license').textContent = d.licenseKey;
        alert('A new License Key was issued. Re-activate your installs with it.');
      } else {
        alert('Could not revoke the key. Please try again.');
      }
    });
  }

  async function submitAuth(){
    var email = document.getElementById('email').value.trim();
    var password = document.getElementById('password').value;
    var org = document.getElementById('org').value.trim();
    var btn = document.getElementById('submit');
    document.getElementById('msg').innerHTML = '';
    if(!email || !password || (mode==='register' && !org)){ showErr('Please fill in all fields.'); return; }
    btn.disabled = true;
    try {
      var path = mode==='register' ? '/api/register' : '/api/login';
      var body = mode==='register' ? {email:email,password:password,org:org} : {email:email,password:password};
      var res = await fetch(path, {method:'POST', headers:{'Content-Type':'application/json'}, body:JSON.stringify(body)});
      var data = await res.json();
      if(!res.ok){ showErr(data.error || 'Something went wrong.'); btn.disabled=false; return; }
      document.getElementById('license').textContent = data.licenseKey;
      document.getElementById('org-label').textContent = data.org + ' · ' + data.email;
      document.getElementById('auth-view').style.display='none';
      document.getElementById('key-view').style.display='block';
    } catch(e){ showErr('Cannot reach the licensing server.'); }
    btn.disabled = false;
  }
  function copyKey(){
    var k = document.getElementById('license').textContent;
    navigator.clipboard.writeText(k).then(function(){
      var b = event.target; var t=b.textContent; b.textContent='Copied ✓'; setTimeout(function(){b.textContent=t;},1500);
    });
  }
  function resetView(){
    document.getElementById('key-view').style.display='none';
    document.getElementById('auth-view').style.display='block';
    document.getElementById('password').value='';
  }
</script>
</body>
</html>`
