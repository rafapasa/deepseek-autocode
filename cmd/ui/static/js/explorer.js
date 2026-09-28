import { apiGet } from './api.js';
import { saveExpanded, state } from './state.js';

export async function loadProjectSelector(){
  const projects = await apiGet('/api/projects');
  const sel=document.getElementById('projectSelector');
  sel.innerHTML='<option value="">Todos projetos</option>'+projects.map(p=>`<option value="${p}" ${p===state.currentProject?'selected':''}>${p}</option>`).join('');
}

export async function loadExplorer(){
  const data = await apiGet('/api/issues');
  const list=document.getElementById('projectList');
  list.innerHTML='';
  (data.projects||[]).forEach(p=>{
    const isOpen=state.expandedProjects.has(p.name);
    const div=document.createElement('div');
    div.className='project '+(isOpen?'open':'');
    div.innerHTML=`
      <div class="project-header" data-project="${p.name}"><span>▶</span><span>📁 ${p.name}</span><span style="margin-left:auto;background:rgba(255,255,255,0.1);padding:2px 8px;border-radius:10px;font-size:11px">${(p.issues||[]).length}</span></div>
      <div class="file-tree">
        <div class="file-item" style="color:#7DD3FC"><span class="file-label">📄 projeto.json ${p.projeto_exists?'✓':''} <span class="file-actions"><button class="icon-btn" data-edit-projeto="${p.name}">✎</button><button class="icon-btn" data-upload-projeto="${p.name}">⤴</button></span></span></div>
        ${(p.issues||[]).map(f=>`<div class="file-item"><span class="file-label">📄 ${f} <span class="file-actions"><button class="icon-btn" data-edit="${p.name}|${f}">✎</button><button class="icon-btn" data-replace="${p.name}|${f}">⤴</button></span></span></div>`).join('')}
      </div>
    `;
    list.appendChild(div);
  });
}

export function toggleExplorer(name){
  if(state.expandedProjects.has(name)) state.expandedProjects.delete(name); else state.expandedProjects.add(name);
  saveExpanded();
  loadExplorer();
}