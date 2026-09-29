import { state } from './state.js';

export async function renderIssuesForCurrent(){
  const logEl=document.getElementById('runLog');
  const issuesListEl=document.getElementById('issuesList');
  const btn=document.getElementById('btnRunSelected');
  const nameSpan=document.getElementById('selectedIssueName');
  
  if(!state.currentProject){
    if(logEl) logEl.innerHTML='<div style="color:#64748B;text-align:center;margin-top:40px">Selecione projeto no combo superior</div>';
    if(issuesListEl) issuesListEl.style.display='none';
    if(btn) btn.style.display='none';
    if(nameSpan) nameSpan.style.display='none';
    return;
  }
  
  if(state.currentIssue && state.currentIssue.project === state.currentProject){
    if(btn){
      btn.style.display='inline-flex';
      btn.dataset.project = state.currentIssue.project;
      btn.dataset.file = state.currentIssue.file;
    }
    if(nameSpan){
      nameSpan.textContent = state.currentIssue.file;
      nameSpan.style.display='inline-block';
    }
  } else {
    if(btn) btn.style.display='none';
    if(nameSpan) nameSpan.style.display='none';
  }
  
  if(issuesListEl) issuesListEl.style.display='none';
  if(logEl && !logEl.innerText.trim()){
    logEl.innerHTML='<div style="color:#64748B;text-align:center;margin-top:40px">📋 Selecione uma issue no explorer à esquerda<br><span style="font-size:11px">e clique em Executar ▶</span></div>';
  }
}

export async function runIssue(project, issue){
  if(!project || !issue){
    if(state.currentIssue){
      project = state.currentIssue.project;
      issue = state.currentIssue.file;
    } else {
      alert('Selecione uma issue no explorer primeiro');
      return;
    }
  }
  
  if(!confirm(`Rodar ${issue}?`)) return;
  
  const logEl=document.getElementById('runLog');
  if(!logEl) return;
  
  const tabIssues=document.getElementById('tab-issues');
  if(tabIssues && !tabIssues.classList.contains('active')){
    tabIssues.click();
  }
  
  logEl.style.display='block';
  logEl.innerHTML=`<div style="color:#16A34A;font-weight:600">▶ Iniciando ${issue}...</div><div style="color:#94A3B8;font-size:11px;margin-bottom:8px">Projeto: ${project} • ${new Date().toLocaleTimeString()}</div><div id="logContent" style="font-family:monospace;white-space:pre-wrap;line-height:1.6"></div>`;
  const contentEl = document.getElementById('logContent');
  logEl.scrollTop=0;
  
  try{
    const res=await fetch('/api/run',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({project,issue})});
    if(!res.ok){ 
      const err=await res.text();
      contentEl.innerHTML+=`<div style="color:#ef4444">❌ Erro ao iniciar: ${err}</div>`;
      return; 
    }
    const {id}=await res.json();
    
    let finished = false;
    const es=new EventSource('/api/stream/'+id);
    
    es.onmessage=e=>{
      try{
        const data=JSON.parse(e.data);
        if(data.type==='line'){
          // Usa textContent pra preservar quebras mas escapar HTML
          const line = document.createElement('div');
          line.textContent = data.text;
          contentEl.appendChild(line);
        } else if(data.type==='done'){
          finished = true;
          const resultDiv = document.createElement('div');
          resultDiv.style.marginTop='12px';
          resultDiv.style.padding='8px 12px';
          resultDiv.style.borderRadius='6px';
          if(data.success){
            resultDiv.style.background='rgba(22,163,74,0.15)';
            resultDiv.style.color='#16A34A';
            resultDiv.style.border='1px solid #16A34A';
            resultDiv.innerHTML='✅ Concluído com sucesso';
          } else {
            resultDiv.style.background='rgba(239,68,68,0.15)';
            resultDiv.style.color='#ef4444';
            resultDiv.style.border='1px solid #ef4444';
            resultDiv.innerHTML='❌ Falhou - veja log acima';
          }
          contentEl.appendChild(resultDiv);
          es.close();
          window.dispatchEvent(new Event('explorer:reload'));
        }
        logEl.scrollTop=logEl.scrollHeight;
      }catch(err){
        console.error('Erro parse SSE:', err, e.data);
      }
    };
    
    es.onerror=()=>{
      if(!finished){
        // Só mostra erro se não terminou normalmente
        // EventSource fecha com error também quando o servidor fecha stream no done, então ignora se já recebeu done
        const isNormalClose = contentEl.innerHTML.includes('Concluído') || contentEl.innerHTML.includes('Falhou');
        if(!isNormalClose){
          contentEl.innerHTML+=`<div style="color:#f59e0b;margin-top:8px">⚠️ Conexão com log encerrada. Se o processo ainda estiver rodando, recarregue a página.</div>`;
        }
      }
      es.close();
    };
  }catch(err){
    contentEl.innerHTML+=`<div style="color:#ef4444">❌ Erro: ${err.message}</div>`;
  }
}

window.runIssue = runIssue;
