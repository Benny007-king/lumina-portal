package main

import "strings"

/*
   ======================================================================
   AUTH-AWARE NAV (avatar)  — shared across landing / docs / releases
   ----------------------------------------------------------------------
   Every marketing page has a `.authtoggle` element with "Sign In / Sign
   Up". This script checks the session cookie via /api/me and, when the
   visitor is logged in, replaces those buttons with an avatar circle
   showing the first letter of their email + a dropdown (account / sign
   out). Because the session cookie is HttpOnly + 30 days, the avatar
   persists across reloads and pages — the visible proof of persistence.
   ======================================================================
*/

// withAuthNav injects the shared avatar script just before </body>.
func withAuthNav(html string) string {
	return strings.Replace(html, "</body>", authNavScript+"</body>", 1)
}

const authNavScript = `<script>
(function(){
  fetch('/api/me',{credentials:'include'}).then(function(r){return r.ok?r.json():null;}).then(function(u){
    if(!u||!u.email) return;
    var box=document.querySelector('.authtoggle'); if(!box) return;
    var ini=(u.email.charAt(0)||'?').toUpperCase();
    box.style.background='transparent'; box.style.border='none'; box.style.padding='0';
    box.innerHTML='<div id="lumAvatar" title="'+u.email+'" style="width:40px;height:40px;border-radius:50%;display:grid;place-items:center;font-weight:800;font-size:16px;color:#fff;cursor:pointer;user-select:none;background:linear-gradient(120deg,#a855f7,#d946ef);box-shadow:0 6px 20px -6px rgba(217,70,239,.7)">'+ini+'</div>';
    var menu=document.createElement('div');
    menu.style.cssText='position:fixed;right:24px;top:70px;background:rgba(20,10,40,.97);border:1px solid rgba(168,148,230,.25);border-radius:14px;padding:12px 14px;font-size:13px;color:#e7e2f7;display:none;z-index:60;backdrop-filter:blur(12px);min-width:200px;box-shadow:0 20px 50px -20px rgba(0,0,0,.8)';
    menu.innerHTML='<div style="opacity:.65;font-size:11px;margin-bottom:10px;word-break:break-all">Signed in as<br><b style="color:#fff">'+u.email+'</b></div><a href="/signup" style="display:block;color:#22d3ee;font-weight:700;padding:6px 0;text-decoration:none">My license &amp; download</a><a href="#" id="lumSignout" style="display:block;color:#f87171;font-weight:700;padding:6px 0;text-decoration:none">Sign out</a>';
    document.body.appendChild(menu);
    document.getElementById('lumAvatar').addEventListener('click',function(e){e.stopPropagation();menu.style.display=(menu.style.display==='none'?'block':'none');});
    document.addEventListener('click',function(){menu.style.display='none';});
    document.getElementById('lumSignout').addEventListener('click',function(e){e.preventDefault();fetch('/api/logout',{method:'POST',credentials:'include'}).then(function(){location.reload();});});
  }).catch(function(){});
})();
</script>`
