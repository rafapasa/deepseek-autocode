import { apiGet } from './api.js';
import { saveExpanded, state } from './state.js';

export async function loadProjectSelector(){
  try{
    const projects = await apiGet('/api/projects');
    const sel = document.getElementById('projectSelector');
    if(!sel) return;
    sel.innerHTML = '<option value="">Todos</option>' +
      (projects || []).map(p => `<option value="${escapeHtml(p)}">${escapeHtml(p)}</option>`).join('');
    if(state.currentProject && projects.includes(state.currentProject)){
      sel.value = state.currentProject;
    } else if(!state.currentProject){
      sel.value = '';
    }
  }catch(e){ console.error('[eTools] Erro loadProjectSelector:', e); }
}

export async function loadExplorer(){
  try{
    const data = await apiGet('/api/issues');
    const list = document.getElementById('projectList');
    if(!list) return;
    list.innerHTML = '';

    let projects = data.projects || [];
    if(state.currentProject){
      projects = projects.filter(p => p.name === state.currentProject);
    }

    if(!projects.length){
      list.innerHTML = '<div class="explorer-empty">Nenhum projeto para exibir</div>';
      return;
    }

    projects.forEach(p => {
      const forceOpen = !!state.currentProject;
      const isOpen = forceOpen || state.expandedProjects.has(p.name);
      const count = (p.issues || []).length;

      const div = document.createElement('div');
      div.className = 'project' + (isOpen ? ' open' : '');
      div.dataset.projectName = p.name;

      const header = document.createElement('div');
      header.className = 'project-header';
      header.dataset.project = p.name;
      header.innerHTML = `
        <span class="project-arrow" aria-hidden="true"></span>
        <span class="folder-icon" aria-hidden="true"></span>
        <span class="project-name" title="${escapeHtml(p.name)}">${escapeHtml(p.name)}</span>
        <span class="project-count">${count}</span>
      `;

      const tree = document.createElement('div');
      tree.className = 'file-tree';

      tree.appendChild(buildFileItem({
        name: 'projeto.json',
        title: 'projeto.json — ' + p.name,
        extraClass: 'projeto-file',
        check: p.projeto_exists,
        attrs: { 'data-projeto-file': p.name },
        menu: { kind: 'projeto', project: p.name }
      }));

      (p.issues || []).forEach(f => {
        const isSelected = state.currentIssue &&
          state.currentIssue.project === p.name &&
          state.currentIssue.file === f;
        tree.appendChild(buildFileItem({
          name: f,
          title: f,
          extraClass: isSelected ? 'selected' : '',
          attrs: { 'data-issue-file': f, 'data-issue-project': p.name },
          menu: { kind: 'issue', project: p.name, file: f }
        }));
      });

      div.appendChild(header);
      div.appendChild(tree);
      list.appendChild(div);
    });
  }catch(e){ console.error('[eTools] Erro loadExplorer:', e); }
}

function buildFileItem({ name, title, extraClass = '', check = false, attrs = {}, menu }){
  const item = document.createElement('div');
  item.className = ('file-item ' + extraClass).trim();
  item.title = title || name;
  Object.entries(attrs).forEach(([k, v]) => item.setAttribute(k, v));

  const checkHtml = check ? '<span class="file-check" title="existe">●</span>' : '';
  const payload = encodeURIComponent(JSON.stringify(menu));
  item.innerHTML = `
    <span class="file-label">
      <span class="file-icon json-icon" aria-hidden="true">{ }</span>
      <span class="file-name">${escapeHtml(name)}</span>
      ${checkHtml}
    </span>
    <button type="button" class="kebab-btn" data-menu="${payload}" title="Ações" aria-label="Ações">
      <span></span><span></span><span></span>
    </button>
  `;
  return item;
}

export function toggleExplorer(name){
  if(state.currentProject){
    return;
  }
  if(state.expandedProjects.has(name)) state.expandedProjects.delete(name);
  else state.expandedProjects.add(name);
  saveExpanded();
  loadExplorer();
}

export function selectIssue(project, file){
  state.currentIssue = { project, file };
  localStorage.setItem('etools_current_issue', JSON.stringify(state.currentIssue));
  document.querySelectorAll('.file-item').forEach(el => el.classList.remove('selected'));
  const sel = document.querySelector(`[data-issue-file="${cssEscape(file)}"][data-issue-project="${cssEscape(project)}"]`);
  if(sel) sel.classList.add('selected');

  const btn = document.getElementById('btnRunSelected');
  const nameSpan = document.getElementById('selectedIssueName');
  if(btn){
    btn.style.display = 'inline-flex';
    btn.dataset.project = project;
    btn.dataset.file = file;
  }
  if(nameSpan){
    nameSpan.textContent = file;
    nameSpan.title = `${project}/${file}`;
    nameSpan.style.display = 'inline-block';
  }
  window.dispatchEvent(new CustomEvent('issue:selected', { detail: { project, file } }));
}

export function closeItemMenu(){
  const menu = document.getElementById('itemMenu');
  if(menu){
    menu.classList.remove('open');
    menu.hidden = true;
    menu.innerHTML = '';
  }
}

export function openItemMenu(anchor, menuData){
  const menu = document.getElementById('itemMenu');
  if(!menu || !menuData) return;
  const items = menuData.kind === 'issue'
    ? [
        { act: 'run', label: 'Executar', icon: 'play' },
        { act: 'edit', label: 'Editar', icon: 'edit' },
        { act: 'replace', label: 'Substituir arquivo', icon: 'up' }
      ]
    : [
        { act: 'edit', label: 'Editar', icon: 'edit' },
        { act: 'replace', label: 'Upload / substituir', icon: 'up' }
      ];

  menu.innerHTML = items.map(it =>
    `<button type="button" data-act="${it.act}">
       <span class="menu-ico ${it.icon}"></span>${it.label}
     </button>`
  ).join('');
  menu.dataset.kind = menuData.kind;
  menu.dataset.project = menuData.project || '';
  menu.dataset.file = menuData.file || '';

  const rect = anchor.getBoundingClientRect();
  menu.hidden = false;
  menu.classList.add('open');
  const mw = menu.offsetWidth || 180;
  const mh = menu.offsetHeight || 90;
  let left = rect.right - mw;
  let top = rect.bottom + 4;
  if(left < 8) left = 8;
  if(left + mw > window.innerWidth - 8) left = window.innerWidth - mw - 8;
  if(top + mh > window.innerHeight - 8) top = Math.max(8, rect.top - mh - 4);
  menu.style.left = left + 'px';
  menu.style.top = top + 'px';
}

function escapeHtml(s){
  return String(s ?? '').replace(/[&<>"']/g, c => ({
    '&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'
  }[c]));
}

function cssEscape(s){
  if(window.CSS && CSS.escape) return CSS.escape(s);
  return String(s).replace(/"/g, '\\"');
}
