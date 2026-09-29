import { state } from './state.js';

let activePump = 0;

export async function renderIssuesForCurrent(){
  const logEl = document.getElementById('runLog');
  const issuesListEl = document.getElementById('issuesList');
  const btn = document.getElementById('btnRunSelected');
  const nameSpan = document.getElementById('selectedIssueName');
  const info = document.getElementById('currentProjectInfo');

  if(info){
    info.innerText = state.currentProject
      ? `Projeto: ${state.currentProject}`
      : 'Todos os projetos';
  }

  if(state.currentIssue && (!state.currentProject || state.currentIssue.project === state.currentProject)){
    if(btn){
      btn.style.display = 'inline-flex';
      btn.dataset.project = state.currentIssue.project;
      btn.dataset.file = state.currentIssue.file;
    }
    if(nameSpan){
      nameSpan.textContent = state.currentIssue.file;
      nameSpan.style.display = 'inline-block';
    }
  } else {
    if(btn) btn.style.display = 'none';
    if(nameSpan) nameSpan.style.display = 'none';
  }

  if(issuesListEl) issuesListEl.style.display = 'none';
  if(logEl && !logEl.dataset.activeRun && !logEl.innerText.trim()){
    logEl.innerHTML = '<div class="log-placeholder">Selecione uma issue no explorer e execute ▶</div>';
  }
}

function appendLogLine(container, text){
  const line = document.createElement('div');
  line.className = 'log-line';
  const t = String(text ?? '');
  if(/❌|erro|error|fail/i.test(t)) line.classList.add('err');
  else if(/⚠|warn/i.test(t)) line.classList.add('warn');
  else if(/✅|✔|sucesso|conclu/i.test(t)) line.classList.add('ok');
  else if(/▶|iniciando|\[ui\]|\[AutoCode\]|\[ds-ac\]|\[LLM\]/i.test(t)) line.classList.add('info');
  line.textContent = t;
  container.appendChild(line);
}

function renderDone(container, success){
  const resultDiv = document.createElement('div');
  resultDiv.className = 'log-result ' + (success ? 'ok' : 'err');
  resultDiv.textContent = success ? '✅ Concluído com sucesso' : '❌ Falhou — veja o log acima';
  container.appendChild(resultDiv);
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

  const logEl = document.getElementById('runLog');
  if(!logEl) return;

  const tabIssues = document.getElementById('tab-issues');
  if(tabIssues && !tabIssues.classList.contains('active')){
    tabIssues.click();
  }

  const pumpId = ++activePump;
  logEl.style.display = 'block';
  logEl.dataset.activeRun = '1';
  logEl.innerHTML =
    `<div class="log-head">▶ Iniciando ${issue}...</div>` +
    `<div class="log-meta">Projeto: ${project} • ${new Date().toLocaleTimeString()}</div>` +
    `<div id="logContent" class="log-content"></div>`;
  const contentEl = document.getElementById('logContent');
  logEl.scrollTop = 0;

  try{
    const res = await fetch('/api/run', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ project, issue })
    });
    if(!res.ok){
      appendLogLine(contentEl, '❌ Erro ao iniciar: ' + await res.text());
      delete logEl.dataset.activeRun;
      return;
    }
    const { id } = await res.json();
    appendLogLine(contentEl, '[ui] run ' + id);
    await pumpLogs(pumpId, id, logEl, contentEl);
  }catch(err){
    appendLogLine(contentEl, '❌ Erro: ' + err.message);
    delete logEl.dataset.activeRun;
  }
}

async function pumpLogs(pumpId, id, logEl, contentEl){
  let from = 0;
  let failures = 0;
  while(pumpId === activePump){
    try{
      const res = await fetch(`/api/run/${encodeURIComponent(id)}/logs?from=${from}&wait=20000`, {
        cache: 'no-store'
      });
      if(res.status === 404){
        failures += 1;
        if(failures >= 8){
          appendLogLine(contentEl, '❌ run não encontrado no servidor');
          break;
        }
        await sleep(400);
        continue;
      }
      if(!res.ok){
        failures += 1;
        appendLogLine(contentEl, '⚠ falha ao ler log HTTP ' + res.status);
        if(failures >= 10) break;
        await sleep(800);
        continue;
      }
      const data = await res.json();
      failures = 0;
      const lines = data.lines || [];
      for(const t of lines) appendLogLine(contentEl, t);
      if(typeof data.next === 'number') from = data.next;
      else from += lines.length;
      logEl.scrollTop = logEl.scrollHeight;
      if(data.finished){
        renderDone(contentEl, !!data.success);
        window.dispatchEvent(new Event('explorer:reload'));
        break;
      }
    }catch(err){
      failures += 1;
      if(failures >= 12){
        appendLogLine(contentEl, '❌ não foi possível ler o log: ' + err.message);
        break;
      }
      await sleep(Math.min(400 * failures, 2000));
    }
  }
  if(pumpId === activePump) delete logEl.dataset.activeRun;
}

function sleep(ms){ return new Promise(r => setTimeout(r, ms)); }

window.runIssue = runIssue;
