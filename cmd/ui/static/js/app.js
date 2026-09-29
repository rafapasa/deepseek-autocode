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
  const sel=document.getElementById('projectSelector');
  if(sel){
    sel.addEventListener('change', async (e)=>{
      state.currentProject=e.target.value;
      saveCurrentProject();
      state.currentChatId=null;
      const info=document.getElementById('currentProjectInfo');
      if(info) info.innerText=state.currentProject?`Projeto atual: ${state.currentProject}`:'Todos';
      if(state.currentProject){ await createNewChat(); await renderIssuesForCurrent(); }
    });
  }
  const pl=document.getElementById('projectList');
  if(pl){
    pl.addEventListener('click', (e)=>{
      const header=e.target.closest('.project-header');
      if(header){ toggleExplorer(header.dataset.project); return; }
      const editP=e.target.closest('[data-edit-projeto]'); if(editP){ modals.editProjeto(editP.dataset.editProjeto); return; }
      const upP=e.target.closest('[data-upload-projeto]'); if(upP){ modals.uploadProjeto(upP.dataset.uploadProjeto); return; }
      const edit=e.target.closest('[data-edit]'); if(edit){ const [proj,file]=edit.dataset.edit.split('|'); modals.editIssue(proj,file); return; }
      const rep=e.target.closest('[data-replace]'); if(rep){ const [proj,file]=rep.dataset.replace.split('|'); modals.uploadReplaceIssue(proj,file); return; }
    });
  }
  const il=document.getElementById('issuesList');
  if(il){
    il.addEventListener('click', (e)=>{
      const run=e.target.closest('[data-run]'); if(run){ const [proj,file]=run.dataset.run.split('|'); runIssue(proj,file); }
    });
  }
  const bs=document.getElementById('btnSend');
  if(bs) bs.addEventListener('click', sendMessage);
  const ci=document.getElementById('chatInput');
  if(ci) ci.addEventListener('keydown', (e)=>{ if(e.key==='Enter'&&!e.shiftKey){ e.preventDefault(); sendMessage(); } });
  const ti=document.getElementById('tab-issues');
  if(ti) ti.addEventListener('click', ()=>switchTab('issues'));
  const tc=document.getElementById('tab-chat');
  if(tc) tc.addEventListener('click', ()=>switchTab('chat'));
  const bni=document.getElementById('btnNewIssue');
  if(bni) bni.addEventListener('click', ()=>modals.openCreateModal());
  const bui=document.getElementById('btnUploadIssue');
  if(bui) bui.addEventListener('click', ()=>modals.openUploadModal());
  const eB=document.querySelector('[data-action="editBase"]');
  if(eB) eB.addEventListener('click', ()=>modals.editBase());
  const uB=document.querySelector('[data-action="uploadBase"]');
  if(uB) uB.addEventListener('click', ()=>modals.uploadBase());
  const bse=document.getElementById('btnSaveEdit');
  if(bse) bse.addEventListener('click', ()=>modals.saveEdit());
  const bsur=document.getElementById('btnSubmitUploadReplace');
  if(bsur) bsur.addEventListener('click', ()=>modals.submitUploadReplace());
  const bsc=document.getElementById('btnSubmitCreate');
  if(bsc) bsc.addEventListener('click', ()=>modals.submitCreate());
  const bsu=document.getElementById('btnSubmitUpload');
  if(bsu) bsu.addEventListener('click', ()=>modals.submitUpload());
  document.querySelectorAll('[data-close]').forEach(b=>b.addEventListener('click', ()=>{
    const el=document.getElementById(b.dataset.close);
    if(el) el.classList.remove('open');
  }));
  window.addEventListener('explorer:reload', loadExplorer);
  window.addEventListener('projects:reload', loadProjectSelector);
}

function switchTab(tab){
  document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
  const el=document.getElementById('tab-'+tab);
  if(el) el.classList.add('active');
  const cp=document.getElementById('chatPanel');
  const dp=document.getElementById('diffPanel');
  const ip=document.getElementById('issuesPanel');
  if(cp) cp.style.display=tab==='chat'?'flex':'none';
  if(dp) dp.style.display=tab==='chat'?'flex':'none';
  if(ip) ip.style.display=tab==='issues'?'flex':'none';
}

init();
