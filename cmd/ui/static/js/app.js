import { createNewChat, deleteChat, exportChatResumo, loadChatList, openChat, renameChat, resizeChatInput, sendMessage } from './chat.js';
import { loadExplorer, loadProjectSelector, toggleExplorer, selectIssue, openItemMenu, closeItemMenu } from './explorer.js';
import { renderIssuesForCurrent, runIssue } from './issues.js';
import * as modals from './modals.js';
import { saveCurrentProject, state } from './state.js';

async function init(){
  await loadProjectSelector();
  await loadExplorer();
  await renderIssuesForCurrent();
  await loadChatList();
  if(state.currentChatId){
    await openChat(state.currentChatId);
  }
  bindEvents();
  switchTab(state.activeTab === 'issues' ? 'issues' : 'chat');
}

function bindEvents(){
  const sel = document.getElementById('projectSelector');
  if(sel){
    sel.addEventListener('change', async (e) => {
      state.currentProject = e.target.value || '';
      saveCurrentProject();
      state.currentChatId = null;
      localStorage.removeItem('etools_current_chat');
      closeItemMenu();
      await loadExplorer();
      await loadChatList();
      const msgs = document.getElementById('chatMessages');
      if(msgs){
        msgs.innerHTML = state.currentProject
          ? '<div class="empty-state">Selecione um chat à esquerda ou clique em + novo</div>'
          : '<div class="empty-state">Selecione um projeto no combo para usar o chat</div>';
      }
      await renderIssuesForCurrent();
    });
  }

  function onExplorerClick(e){
    const kebab = e.target.closest('.kebab-btn');
    if(kebab){
      e.preventDefault();
      e.stopPropagation();
      try{
        const data = JSON.parse(decodeURIComponent(kebab.dataset.menu || '%7B%7D'));
        openItemMenu(kebab, data);
      }catch(err){
        console.error(err);
      }
      return;
    }

    const header = e.target.closest('.project-header');
    if(header){ toggleExplorer(header.dataset.project); return; }

    const fileItem = e.target.closest('.file-item');
    if(fileItem && fileItem.dataset.issueFile){
      selectIssue(fileItem.dataset.issueProject, fileItem.dataset.issueFile);
      const tabIssues = document.getElementById('tab-issues');
      if(tabIssues) tabIssues.click();
    }
  }

  const pl = document.getElementById('projectList');
  if(pl) pl.addEventListener('click', onExplorerClick);
  const fl = document.getElementById('filesList');
  if(fl) fl.addEventListener('click', onExplorerClick);

  const menu = document.getElementById('itemMenu');
  if(menu){
    menu.addEventListener('click', (e) => {
      const btn = e.target.closest('[data-act]');
      if(!btn) return;
      const act = btn.dataset.act;
      const kind = menu.dataset.kind;
      const project = menu.dataset.project;
      const file = menu.dataset.file;
      const chatId = menu.dataset.chatId;
      closeItemMenu();
      if(kind === 'issue'){
        if(act === 'run') runIssue(project, file);
        else if(act === 'edit') modals.editIssue(project, file);
        else if(act === 'replace') modals.uploadReplaceIssue(project, file);
      } else if(kind === 'projeto'){
        if(act === 'edit') modals.editProjeto(project);
        else if(act === 'replace') modals.uploadProjeto(project);
      } else if(kind === 'base'){
        if(act === 'edit') modals.editBase();
        else if(act === 'replace') modals.uploadBase();
      } else if(kind === 'chat'){
        if(act === 'rename') renameChat(chatId);
        else if(act === 'export') exportChatResumo(chatId);
        else if(act === 'delete') deleteChat(chatId);
      }
    });
  }

  document.addEventListener('click', (e) => {
    if(e.target.closest('.kebab-btn') || e.target.closest('#itemMenu')) return;
    closeItemMenu();
  });
  window.addEventListener('scroll', closeItemMenu, true);
  window.addEventListener('resize', closeItemMenu);

  const baseCard = document.getElementById('baseCard');
  if(baseCard){
    baseCard.addEventListener('click', (e) => {
      const kebab = e.target.closest('.kebab-btn');
      if(!kebab) return;
      e.preventDefault();
      e.stopPropagation();
      openItemMenu(kebab, { kind: 'base' });
    });
  }

  const il = document.getElementById('issuesList');
  if(il){
    il.addEventListener('click', (e) => {
      const run = e.target.closest('[data-run]');
      if(run){ const [proj, file] = run.dataset.run.split('|'); runIssue(proj, file); }
    });
  }

  const btnRunSelected = document.getElementById('btnRunSelected');
  if(btnRunSelected){
    btnRunSelected.addEventListener('click', () => {
      const project = btnRunSelected.dataset.project || (state.currentIssue && state.currentIssue.project);
      const file = btnRunSelected.dataset.file || (state.currentIssue && state.currentIssue.file);
      if(project && file) runIssue(project, file);
    });
  }

  const bs = document.getElementById('btnSend');
  if(bs) bs.addEventListener('click', sendMessage);
  const ci = document.getElementById('chatInput');
  if(ci){
    resizeChatInput();
    ci.addEventListener('input', resizeChatInput);
    ci.addEventListener('keydown', (e) => {
      if(e.key === 'Enter' && !e.shiftKey){ e.preventDefault(); sendMessage(); }
    });
  }
  const ti = document.getElementById('tab-issues');
  if(ti) ti.addEventListener('click', () => switchTab('issues'));
  const tc = document.getElementById('tab-chat');
  if(tc) tc.addEventListener('click', () => switchTab('chat'));
  const bnc = document.getElementById('btnNewChat');
  if(bnc) bnc.addEventListener('click', () => createNewChat());
  const chatList = document.getElementById('chatList');
  if(chatList){
    chatList.addEventListener('click', (e) => {
      const kebab = e.target.closest('.kebab-btn');
      if(kebab){
        e.preventDefault();
        e.stopPropagation();
        try{
          const data = JSON.parse(decodeURIComponent(kebab.dataset.menu || '%7B%7D'));
          openItemMenu(kebab, data);
        }catch(err){ console.error(err); }
        return;
      }
      const item = e.target.closest('[data-chat-id]');
      if(item) openChat(item.dataset.chatId);
    });
  }
  const bni = document.getElementById('btnNewIssue');
  if(bni) bni.addEventListener('click', () => modals.openCreateModal());
  const bui = document.getElementById('btnUploadIssue');
  if(bui) bui.addEventListener('click', () => modals.openUploadModal());
  const bse = document.getElementById('btnSaveEdit');
  if(bse) bse.addEventListener('click', () => modals.saveEdit());
  const bsur = document.getElementById('btnSubmitUploadReplace');
  if(bsur) bsur.addEventListener('click', () => modals.submitUploadReplace());
  const bsc = document.getElementById('btnSubmitCreate');
  if(bsc) bsc.addEventListener('click', () => modals.submitCreate());
  const bsu = document.getElementById('btnSubmitUpload');
  if(bsu) bsu.addEventListener('click', () => modals.submitUpload());
  document.querySelectorAll('[data-close]').forEach(b => b.addEventListener('click', () => {
    const el = document.getElementById(b.dataset.close);
    if(el) el.classList.remove('open');
  }));
  window.addEventListener('explorer:reload', loadExplorer);
  window.addEventListener('projects:reload', loadProjectSelector);
}

function switchTab(tab){
  state.activeTab = tab;
  localStorage.setItem('etools_active_tab', tab);
  document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
  const el = document.getElementById('tab-' + tab);
  if(el) el.classList.add('active');
  const cp = document.getElementById('chatPanel');
  const ip = document.getElementById('issuesPanel');
  const explorerPane = document.getElementById('explorerPane');
  const chatsPane = document.getElementById('chatsPane');
  const filesPanel = document.getElementById('filesPanel');
  if(cp) cp.style.display = tab === 'chat' ? 'flex' : 'none';
  if(ip) ip.style.display = tab === 'issues' ? 'flex' : 'none';
  if(explorerPane) explorerPane.style.display = tab === 'issues' ? 'block' : 'none';
  if(chatsPane) chatsPane.style.display = tab === 'chat' ? 'flex' : 'none';
  if(filesPanel) filesPanel.classList.toggle('visible', tab === 'chat');
  if(tab === 'issues') renderIssuesForCurrent();
  if(tab === 'chat') loadChatList();
}

init();
