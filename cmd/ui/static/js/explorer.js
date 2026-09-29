import { apiGet } from './api.js';
import { saveExpanded, state } from './state.js';

export async function loadProjectSelector(){
  try{
    const projects = await apiGet('/api/projects');
    const sel=document.getElementById('projectSelector');
    if(!sel) return;
    sel.innerHTML='<option value="">-- selecione projeto --</option>'+projects.map(p=>`<option value="${p}">${p}</option>`).join('');
    if(state.currentProject && projects.includes(state.currentProject)){
      sel.value = state.currentProject;
    } else if(sel.value){
      state.currentProject = sel.value;
    }
  }catch(e){ console.error('[eTools] Erro loadProjectSelector:', e); }
}

export async function loadExplorer(){
  try{
    const data = await apiGet('/api/issues');
    const list=document.getElementById('projectList');
    if(!list) return;
    list.innerHTML='';
    (data.projects||[]).forEach(p=>{
      const isOpen=state.expandedProjects.has(p.name);
      const count = (p.issues||[]).length;
      const div=document.createElement('div');
      div.className='project '+(isOpen?'open':'');
      div.dataset.projectName = p.name;
      
      // Header com cores eTools
      const header = document.createElement('div');
      header.className = 'project-header';
      header.dataset.project = p.name;
      header.innerHTML = `
        <span class="project-arrow">▶</span>
        <span>📁</span>
        <span class="project-name" title="${p.name}">${p.name}</span>
        <span class="project-count">${count}</span>
      `;
      
      // File tree
      const tree = document.createElement('div');
      tree.className = 'file-tree';
      
      // projeto.json item
      const projetoItem = document.createElement('div');
      projetoItem.className = 'file-item projeto-file';
      projetoItem.dataset.projetoFile = p.name;
      projetoItem.title = 'projeto.json - ' + p.name;
      projetoItem.innerHTML = `
        <span class="file-label">
          <span class="file-icon">📄</span>
          <span class="file-name">projeto.json</span>
          ${p.projeto_exists ? '<span class="file-check">✓</span>' : ''}
        </span>
        <span class="file-actions">
          <button class="icon-btn" data-edit-projeto="${p.name}" title="Editar">✎</button>
          <button class="icon-btn" data-upload-projeto="${p.name}" title="Upload">⤴</button>
        </span>
      `;
      tree.appendChild(projetoItem);
      
      // Issues items
      (p.issues||[]).forEach(f=>{
        const isSelected = state.currentIssue && state.currentIssue.project === p.name && state.currentIssue.file === f;
        const item = document.createElement('div');
        item.className = 'file-item' + (isSelected ? ' selected' : '');
        item.dataset.issueFile = f;
        item.dataset.issueProject = p.name;
        item.title = f;
        item.innerHTML = `
          <span class="file-label">
            <span class="file-icon">📄</span>
            <span class="file-name">${f}</span>
          </span>
          <span class="file-actions">
            <button class="icon-btn" data-edit="${p.name}|${f}" title="Editar">✎</button>
            <button class="icon-btn" data-replace="${p.name}|${f}" title="Substituir">⤴</button>
          </span>
        `;
        tree.appendChild(item);
      });
      
      div.appendChild(header);
      div.appendChild(tree);
      list.appendChild(div);
    });
  }catch(e){ console.error('[eTools] Erro loadExplorer:', e); }
}

export function toggleExplorer(name){
  if(state.expandedProjects.has(name)) state.expandedProjects.delete(name); else state.expandedProjects.add(name);
  saveExpanded();
  loadExplorer();
}

export function selectIssue(project, file){
  state.currentIssue = { project, file };
  localStorage.setItem('etools_current_issue', JSON.stringify(state.currentIssue));
  // Atualiza visual
  document.querySelectorAll('.file-item').forEach(el=>el.classList.remove('selected'));
  const sel = document.querySelector(`[data-issue-file="${file}"][data-issue-project="${project}"]`);
  if(sel) sel.classList.add('selected');
  
  // Mostra botão executar no centro
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
  
  // Dispara evento pra quem quiser ouvir
  window.dispatchEvent(new CustomEvent('issue:selected', { detail: { project, file } }));
}
