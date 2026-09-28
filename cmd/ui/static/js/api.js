export async function apiGet(path){ const r=await fetch(path); if(!r.ok) throw new Error(await r.text()); return r.json(); }
export async function apiPost(path, body, isForm=false){
  const opts={method:'POST'};
  if(isForm){ opts.body=body; } else { opts.headers={'Content-Type':'application/json'}; opts.body=JSON.stringify(body); }
  const r=await fetch(path, opts);
  if(!r.ok) throw new Error(await r.text());
  const text=await r.text();
  try{ return JSON.parse(text); }catch{ return {ok:true}; }
}
export async function apiGetFile(project, file){
  const r=await fetch(`/api/file/${encodeURIComponent(project)}/${encodeURIComponent(file)}`);
  if(!r.ok) throw new Error(await r.text());
  return r.json();
}