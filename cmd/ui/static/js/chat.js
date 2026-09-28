import { state } from './state.js';

export function addMessage(role, content, returnEl=false){
  const c=document.getElementById('chatMessages');
  const e=c.querySelector('.empty-state'); if(e) e.remove();
  const div=document.createElement('div');
  div.className='message '+role;
  div.innerHTML=role==='user'?escapeHtml(content):formatContent(content);
  c.appendChild(div); c.scrollTop=c.scrollHeight;
  if(returnEl) return div;
}
export function formatContent(t){
  return escapeHtml(t).replace(/```([\s\S]*?)```/g,(m,c)=>`<div style="background:#0F172A;color:#E2E8F0;padding:10px;border-radius:8px;margin:6px 0;font-family:monospace;font-size:12px;overflow:auto;border-left:3px solid #16A34A">${escapeHtml(c)}</div>`).replace(/\n/g,'<br>');
}
export function escapeHtml(s){ return (s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }

export async function createNewChat(){
  const chatMessages=document.getElementById('chatMessages');
  if(!state.currentProject){ chatMessages.innerHTML='<div class="empty-state">Selecione projeto no combo</div>'; return; }
  const res=await fetch('/api/chat/new',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({project:state.currentProject})});
  const data=await res.json(); state.currentChatId=data.id;
  chatMessages.innerHTML=`<div class="message system">Chat ${data.id.slice(0,8)} - ${state.currentProject} ✅</div>`;
}

export async function sendMessage(){
  const input=document.getElementById('chatInput');
  const text=input.value.trim();
  if(!text||!state.currentChatId){ alert('Selecione projeto no combo'); return; }
  input.value=''; addMessage('user', text);
  const assistantDiv=addMessage('assistant','',true); let full='';
  const res=await fetch(`/api/chat/${state.currentChatId}/message`,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({content:text})});
  const reader=res.body.getReader(); const decoder=new TextDecoder(); let buffer='';
  while(true){
    const {done,value}=await reader.read(); if(done) break;
    buffer+=decoder.decode(value,{stream:true});
    const lines=buffer.split('\n\n'); buffer=lines.pop();
    for(const line of lines){
      if(!line.startsWith('data: ')) continue;
      try{
        const data=JSON.parse(line.slice(6));
        if(data.type==='delta'){ full+=data.content; assistantDiv.innerHTML=formatContent(full); }
      }catch{}
    }
  }
}