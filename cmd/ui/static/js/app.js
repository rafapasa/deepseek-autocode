import { createNewChat, sendMessage } from './chat.js';
import { loadExplorer, loadProjectSelector, toggleExplorer } from './explorer.js';
import { renderIssuesForCurrent, runIssue } from './issues.js';
import * as modals from './modals.js';
import { saveCurrentProject, state } from './state.js';

async function init(){
  await loadProjectSelector();
  await loadExplorer();
  if(state.currentProject){ await createNewChat(); await renderIssuesForCurrent(); }
  bindEvents();
}

function bindEvents(){
  document.getElementById('projectSelector').addEventListener('change', async (e)=>{
    state.currentProject=e.target.value;
    saveCurrentProject();
    document.getElementById('currentProjectInfo').innerText=state.currentProject?`Projeto atual: ${state.currentProject}`:'Todos';
    await createNewChat();
    await renderIssuesForCurrent();
  });

  document.getElementById('projectList').addEventListener('click', (e)=>{
    const header=e.target.closest('.project-header');
    if(header){ toggleExplorer(header.dataset.project); return; }
    const editP=e.target.closest('[data-edit-projeto]'); if(editP){ modals.editProjeto(editP.dataset.editProjeto); return; }
    const upP=e.target.closest('[data-upload-projeto]'); if(upP){ modals.uploadProjeto(upP.dataset.uploadProjeto); return; }
    const edit=e.target.closest('[data-edit]'); if(edit){ const [proj,file]=edit.dataset.edit.split('|'); modals.editIssue(proj,file); return; }
    const rep=e.target.closest('[data-replace]'); if(rep){ const [proj,file]=rep.dataset.replace.split('|'); modals.uploadReplaceIssue(proj,file); return; }
  });

  document.getElementById('issuesList').addEventListener('click', (e)=>{
    const run=e.target.closest('[data-run]'); if(run){ const [proj,file]=run.dataset.run.split('|'); runIssue(proj,file); }
  });

  document.getElementById('btnSend').addEventListener('click', sendMessage);
  document.getElementById('chatInput').addEventListener('keydown', (e)=>{ if(e.key==='Enter'&&!e.shiftKey){ e.preventDefault(); sendMessage(); } });

  document.getElementById('tab-issues').addEventListener('click', ()=>switchTab('issues'));
  document.getElementById('tab-chat').addEventListener('click', ()=>switchTab('chat'));

  document.getElementById('btnNewIssue').addEventListener('click', ()=>modals.openCreateModal());
  document.getElementById('btnUploadIssue').addEventListener('click', ()=>modals.openUploadModal());
  document.querySelector('[data-action="editBase"]').addEventListener('click', ()=>modals.editBase());
  document.querySelector('[data-action="uploadBase"]').addEventListener('click', ()=>modals.uploadBase());

  document.getElementById('btnSaveEdit').addEventListener('click', ()=>modals.saveEdit());
  document.getElementById('btnSubmitUploadReplace').addEventListener('click', ()=>modals.submitUploadReplace());
  document.getElementById('btnSubmitCreate').addEventListener('click', ()=>modals.submitCreate());
  document.getElementById('btnSubmitUpload').addEventListener('click', ()=>modals.submitUpload());

  document.querySelectorAll('[data-close]').forEach(b=>b.addEventListener('click', ()=>{
    document.getElementById(b.dataset.close).classList.remove('open');
  }));

  window.addEventListener('explorer:reload', loadExplorer);
  window.addEventListener('projects:reload', loadProjectSelector);
}

function switchTab(tab){
  document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
  document.getElementById('tab-'+tab).classList.add('active');
  document.getElementById('chatPanel').style.display=tab==='chat'?'flex':'none';
  document.getElementById('diffPanel').style.display=tab==='chat'?'flex':'none';
  document.getElementById('issuesPanel').style.display=tab==='issues'?'flex':'none';
}

init();