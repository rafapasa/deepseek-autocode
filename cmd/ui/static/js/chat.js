import { state } from './state.js';

function escapeHtml(s){ return (s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
function formatContent(t){
  if(!t) return '<span class="muted">...</span>';
  return escapeHtml(t)
    .replace(/```([\s\S]*?)```/g,(m,c)=>`<div style="background:#0F172A;color:#E2E8F0;padding:10px;border-radius:8px;margin:6px 0;font-family:monospace;font-size:12px;overflow:auto;border-left:3px solid #16A34A;white-space:pre-wrap">${escapeHtml(c.trim())}</div>`)
    .replace(/\n/g,'<br>');
}

export function addMessage(role, content, returnEl=false){
  const c=document.getElementById('chatMessages');
  if(!c) return null;
  const e=c.querySelector('.empty-state'); if(e) e.remove();
  const div=document.createElement('div');
  div.className='message '+role;
  div.innerHTML=role==='user'?escapeHtml(content):formatContent(content);
  c.appendChild(div); c.scrollTop=c.scrollHeight;
  if(returnEl) return div;
  return null;
}

export async function createNewChat(){
  const chatMessages=document.getElementById('chatMessages');
  const sel=document.getElementById('projectSelector');
  if(sel && sel.value && !state.currentProject) state.currentProject=sel.value;
  if(!state.currentProject){
    if(chatMessages) chatMessages.innerHTML='<div class="empty-state">Selecione projeto no combo superior</div>';
    return null;
  }
  try{
    const res=await fetch('/api/chat/new',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({project:state.currentProject})});
    if(!res.ok){
      const err=await res.text();
      if(chatMessages) chatMessages.innerHTML=`<div class="message system" style="background:#fef2f2;color:#b91c1c">Erro criar chat: ${escapeHtml(err)}</div>`;
      return null;
    }
    const data=await res.json();
    if(!data.id) return null;
    state.currentChatId=data.id;
    if(chatMessages) chatMessages.innerHTML=`<div class="message system">Chat ${data.id.slice(0,8)} - ${state.currentProject} ✅ Pronto</div>`;
    return data.id;
  }catch(e){
    if(chatMessages) chatMessages.innerHTML=`<div class="message system" style="background:#fef2f2">Erro: ${escapeHtml(e.message)}</div>`;
    return null;
  }
}

export async function sendMessage(){
  const input=document.getElementById('chatInput');
  const sel=document.getElementById('projectSelector');
  const text=input?input.value.trim():'';
  if(sel && sel.value && state.currentProject!==sel.value){
    state.currentProject=sel.value;
    localStorage.setItem('etools_current_project', state.currentProject);
    state.currentChatId=null;
  }
  if(!text) return;
  if(!state.currentProject){ alert('Selecione projeto no combo superior'); return; }
  if(!state.currentChatId){
    const newId=await createNewChat();
    if(!newId){ alert('Falha ao criar chat'); return; }
  }
  if(input) input.value='';
  addMessage('user', text);
  const assistantDiv=addMessage('assistant','',true);
  let full='';
  try{
    const res=await fetch(`/api/chat/${state.currentChatId}/message`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:text})});
    if(!res.ok){
      const errText=await res.text();
      if(res.status===404){
        state.currentChatId=null;
        const nid=await createNewChat();
        if(nid) return sendMessageWithText(text, assistantDiv);
      }
      if(assistantDiv) assistantDiv.innerHTML=`<span style="color:#b91c1c">❌ Erro ${res.status}: ${escapeHtml(errText)}</span>`;
      return;
    }
    const reader=res.body.getReader(); const decoder=new TextDecoder(); let buffer='';
    while(true){
      const {done,value}=await reader.read(); if(done) break;
      buffer+=decoder.decode(value,{stream:true});
      const lines=buffer.split('\n\n'); buffer=lines.pop()||'';
      for(const line of lines){
        if(!line.startsWith('data: ')) continue;
        try{
          const data=JSON.parse(line.slice(6));
          if(data.type==='delta'){ full+=data.content||''; if(assistantDiv) assistantDiv.innerHTML=formatContent(full); }
          else if(data.type==='tool'){ addMessage('system', `🔧 ${data.name}`); }
          else if(data.type==='error'){ if(assistantDiv) assistantDiv.innerHTML+=`<br><span style="color:#b91c1c">❌ ${escapeHtml(data.error)}</span>`; }
        }catch{}
      }
    }
  }catch(e){ if(assistantDiv) assistantDiv.innerHTML=`<span style="color:#b91c1c">❌ ${e.message}</span>`; }
}

async function sendMessageWithText(text, assistantDiv){
  if(!state.currentChatId) return;
  let full='';
  try{
    const res=await fetch(`/api/chat/${state.currentChatId}/message`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:text})});
    if(!res.ok){ if(assistantDiv) assistantDiv.innerHTML=`<span style="color:#b91c1c">❌ ${await res.text()}</span>`; return; }
    const reader=res.body.getReader(); const decoder=new TextDecoder(); let buffer='';
    while(true){
      const {done,value}=await reader.read(); if(done) break;
      buffer+=decoder.decode(value,{stream:true});
      const lines=buffer.split('\n\n'); buffer=lines.pop()||'';
      for(const line of lines){
        if(!line.startsWith('data: ')) continue;
        try{
          const data=JSON.parse(line.slice(6));
          if(data.type==='delta'){ full+=data.content||''; if(assistantDiv) assistantDiv.innerHTML=formatContent(full); }
        }catch{}
      }
    }
  }catch(e){ if(assistantDiv) assistantDiv.innerHTML=`<span style="color:#b91c1c">❌ ${e.message}</span>`; }
}

export { escapeHtml, formatContent };
