import { apiGet } from './api.js';
import { state } from './state.js';

export async function renderIssuesForCurrent(){
  const el=document.getElementById('issuesList');
  if(!state.currentProject){ el.innerHTML='<div class="empty-state">Selecione projeto no combo superior</div>'; return; }
  el.innerHTML='Carregando...';
  const data=await apiGet('/api/issues');
  const p=(data.projects||[]).find(x=>x.name===state.currentProject);
  if(!p){ el.innerHTML='Sem issues'; return; }
  el.innerHTML=(p.issues||[]).map(f=>`<div style="padding:10px;background:#fff;border:1px solid var(--border);border-radius:8px;display:flex;justify-content:space-between;margin-bottom:6px"><span>📄 ${f}</span><button class="btn ghost" data-run="${state.currentProject}|${f}">▶ Run</button></div>`).join('')||'Nenhuma';
}

export async function runIssue(project, issue){
  if(!confirm(`Rodar ${issue}?`)) return;
  const res=await fetch('/api/run',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({project,issue})});
  if(!res.ok){ alert(await res.text()); return; }
  const {id}=await res.json();
  const logEl=document.getElementById('runLog'); logEl.style.display='block'; logEl.innerText='Iniciando...\n';
  const es=new EventSource('/api/stream/'+id);
  es.onmessage=e=>{
    const data=JSON.parse(e.data);
    if(data.type==='line') logEl.innerText+=data.text+'\n';
    else if(data.type==='done'){ logEl.innerText+=(data.success?'✅ Concluído':'❌ Falhou')+'\n'; es.close(); }
    logEl.scrollTop=logEl.scrollHeight;
  };
}