import { apiGet, apiGetFile, apiPost } from './api.js';
import { state } from './state.js';

function showAlert(id, msg, ok=false){ const el=document.getElementById(id); el.className=ok?'alert ok':'alert err'; el.textContent=msg; el.style.display='block'; }

export async function editBase(){ const data=await apiGet('/api/base'); openEdit('base.json', data.path, data.content, 'base'); }
export function uploadBase(){ openUploadReplace('base.json','base','base'); }
export async function editProjeto(project){
  try{ const data=await apiGet(`/api/projeto/${project}`); openEdit('projeto.json', data.path, data.content, 'projeto', project); }
  catch{ openEdit('projeto.json','/issues/'+project+'/projeto.json','{}','projeto',project); }
}
export function uploadProjeto(project){ openUploadReplace('projeto.json',project,'projeto'); }
export async function editIssue(project,file){
  const data=await apiGetFile(project, file);
  openEdit(file, data.path, data.content, 'issue', project, file);
}
export function uploadReplaceIssue(project,file){ openUploadReplace(file,project,'issue',file); }

export function openEdit(title,path,content,type,project,file){
  state.editContext={type,project,file,path};
  document.getElementById('editTitle').innerText='Editar: '+title;
  document.getElementById('editPath').innerText=path;
  document.getElementById('editContent').value=content;
  document.getElementById('editAlert').className='alert';
  document.getElementById('editModal').classList.add('open');
}
export function closeEditModal(){ document.getElementById('editModal').classList.remove('open'); }
export async function saveEdit(){
  const content=document.getElementById('editContent').value;
  let url='';
  if(state.editContext.type==='base') url='/api/base';
  else if(state.editContext.type==='projeto') url=`/api/projeto/${state.editContext.project}`;
  else if(state.editContext.type==='issue') url=`/api/file/${state.editContext.project}/${encodeURIComponent(state.editContext.file)}`;
  else { closeEditModal(); return; }
  try{
    await apiPost(url,{content});
    showAlert('editAlert','✅ Salvo',true);
    setTimeout(()=>{ closeEditModal(); window.dispatchEvent(new CustomEvent('explorer:reload')); },500);
  }catch(e){ showAlert('editAlert', e.message, false); }
}

export function openUploadReplace(title,project,type,file){
  state.uploadReplaceContext={type,project,file};
  document.getElementById('uploadReplaceTitle').innerText='Upload replace: '+title;
  document.getElementById('uploadReplacePath').innerText=type==='base'?'/issues/base.json':'/issues/'+project+'/'+(file||'projeto.json');
  document.getElementById('uploadReplaceFile').value='';
  document.getElementById('uploadReplaceAlert').className='alert';
  document.getElementById('uploadReplaceModal').classList.add('open');
}
export function closeUploadReplaceModal(){ document.getElementById('uploadReplaceModal').classList.remove('open'); }
export async function submitUploadReplace(){
  const f=document.getElementById('uploadReplaceFile').files[0];
  if(!f){ showAlert('uploadReplaceAlert','Selecione arquivo'); return; }
  const fd=new FormData(); fd.append('file',f);
  let url='';
  if(state.uploadReplaceContext.type==='base') url='/api/base/upload';
  else if(state.uploadReplaceContext.type==='projeto') url=`/api/projeto/${state.uploadReplaceContext.project}/upload`;
  else url=`/api/file/${state.uploadReplaceContext.project}/${encodeURIComponent(state.uploadReplaceContext.file)}/upload`;
  const res=await fetch(url,{method:'POST',body:fd});
  const alertEl=document.getElementById('uploadReplaceAlert');
  if(res.ok){ alertEl.className='alert ok'; alertEl.textContent='✅ Upload ok'; setTimeout(()=>{ closeUploadReplaceModal(); window.dispatchEvent(new CustomEvent('explorer:reload')); },600); }
  else{ alertEl.className='alert err'; alertEl.textContent=await res.text(); }
}

export function openCreateModal(){ document.getElementById('crProject').value=state.currentProject||''; document.getElementById('createModal').classList.add('open'); }
export function closeCreateModal(){ document.getElementById('createModal').classList.remove('open'); }
export async function submitCreate(){
  const project=document.getElementById('crProject').value.trim();
  const filename=document.getElementById('crFilename').value.trim();
  const demanda=document.getElementById('crDemanda').value.trim();
  if(!project||!filename||!demanda){ showAlert('crAlert','Preencha'); return; }
  try{
    await apiPost('/api/issues/create',{project,filename,demanda,raiz:'.',arquivos:['dummy'],tarefas:[{id:'1',arquivo:'dummy',tipo:'edit',descricao:'dummy'}]});
    showAlert('crAlert','Criada',true);
    setTimeout(()=>{ closeCreateModal(); window.dispatchEvent(new CustomEvent('explorer:reload')); window.dispatchEvent(new CustomEvent('projects:reload')); },600);
  }catch(e){ showAlert('crAlert', e.message); }
}

export function openUploadModal(){ document.getElementById('upProject').value=state.currentProject||''; document.getElementById('uploadModal').classList.add('open'); }
export function closeUploadModal(){ document.getElementById('uploadModal').classList.remove('open'); }
export async function submitUpload(){
  const project=document.getElementById('upProject').value.trim();
  const file=document.getElementById('upFile').files[0];
  if(!project||!file){ showAlert('upAlert','Obrigatório'); return; }
  const fd=new FormData(); fd.append('project',project); fd.append('file',file);
  const res=await fetch('/api/issues/upload',{method:'POST',body:fd});
  const alertEl=document.getElementById('upAlert');
  if(res.ok){ alertEl.className='alert ok'; alertEl.textContent='Upload ok'; setTimeout(()=>{ closeUploadModal(); window.dispatchEvent(new CustomEvent('explorer:reload')); },600); }
  else{ alertEl.className='alert err'; alertEl.textContent=await res.text(); }
}