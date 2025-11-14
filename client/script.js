// API: /api/execute, /api/vars, /api/history
const output = document.getElementById('output');
const input = document.getElementById('input');
const form = document.getElementById('promptForm');

const btnVars = document.getElementById('btn-vars');
const btnHistory = document.getElementById('btn-history');
const btnClear = document.getElementById('btn-clear');

const cards = document.getElementById('cards');
const cardsInner = document.getElementById('cards-inner');
const closeCards = document.getElementById('close-cards');

let lineIndex = 0;

const AudioCtx = window.AudioContext || window.webkitAudioContext;
const audioCtx = AudioCtx ? new AudioCtx() : null;
function beep(type='success'){
  if(!audioCtx) return;
  const o = audioCtx.createOscillator();
  const g = audioCtx.createGain();
  o.type = 'sine';
  o.frequency.value = type==='success'?880:220;
  g.gain.value = 0.0001;
  o.connect(g);
  g.connect(audioCtx.destination);
  const now = audioCtx.currentTime;
  g.gain.setValueAtTime(0.0001, now);
  g.gain.exponentialRampToValueAtTime(0.06, now + 0.01);
  o.start(now);
  g.gain.exponentialRampToValueAtTime(0.0001, now + 0.18);
  o.stop(now + 0.2);
}

function esc(s){if(typeof s!=='string') s=JSON.stringify(s);return s.replace(/[&<>"']/g,(m)=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'})[m])}

function appendLine(text,{type='plain',typing=false}={}){
  lineIndex++;
  const wrapper=document.createElement('div');
  wrapper.className='line';
  wrapper.style.animationDelay=`${lineIndex*40}ms`;

  const pre=document.createElement('span');
  pre.className='line-pre';
  pre.style.color='var(--muted)';
  pre.style.marginRight='8px';
  pre.style.fontFamily='var(--mono)';
  pre.textContent='';

  const content=document.createElement('span');
  content.className=(type==='result')?'result':(type==='error'?'error':'');
  content.style.whiteSpace='pre-wrap';
  content.style.fontFamily='var(--mono)';
  content.innerHTML='';

  wrapper.appendChild(pre);
  wrapper.appendChild(content);
  output.appendChild(wrapper);
  output.scrollTop=output.scrollHeight;

  if(typing){
    const txt=typeof text==='string'?text:JSON.stringify(text,null,2);
    let i=0;
    const speed=10;
    const cursor=document.createElement('span');
    cursor.className='cursor';
    pre.appendChild(cursor);
    function step(){
      if(i<=txt.length){
        content.innerHTML=esc(txt.slice(0,i));
        output.scrollTop=output.scrollHeight;
        i++;
        setTimeout(step,speed);
      } else pre.removeChild(cursor);
    }
    step();
  } else {
    content.innerHTML=esc(typeof text==='string'?text:JSON.stringify(text,null,2));
  }
}

function clearOutput(){lineIndex=0;output.innerHTML='';}

async function sendCommand(cmdText){
  if(!cmdText||!cmdText.trim()) return;
  appendLine(`> ${cmdText}`,{typing:true});
  input.value='';
  try{
    const res=await fetch('/api/execute',{
      method:'POST',
      headers:{'Content-Type':'application/json'},
      body:JSON.stringify({input:cmdText})
    });
    const data=await res.json();
    if(data.error){ appendLine(data.error,{type:'error',typing:true}); beep('error'); }
    else{ appendLine(JSON.stringify(data.result,null,2),{type:'result',typing:true}); beep('success'); }
  } catch(e){ appendLine('Ошибка сети: '+String(e),{type:'error',typing:true}); beep('error'); }
}

form.addEventListener('submit',(ev)=>{ ev.preventDefault(); sendCommand(input.value); });

document.addEventListener('keydown',(ev)=>{
  if((ev.ctrlKey||ev.metaKey)&&ev.key.toLowerCase()==='k'){ ev.preventDefault(); input.focus(); input.select(); }
  if(ev.key==='Escape'){ hideCards(); }
});

btnVars.addEventListener('click',async ()=>{
  try{
    const r=await fetch('/api/vars');
    const vars=await r.json();
    appendLine('Переменные:',{type:'plain'});
    appendLine(JSON.stringify(vars,null,2),{type:'result',typing:true});
  } catch(e){ appendLine('Ошибка загрузки переменных: '+String(e),{type:'error',typing:true}); }
});

btnHistory.addEventListener('click',async ()=>{
  try{
    const r=await fetch('/api/history');
    const hist=await r.json();
    showCards(hist||[]);
  } catch(e){ appendLine('Ошибка загрузки истории: '+String(e),{type:'error',typing:true}); }
});

btnClear.addEventListener('click',clearOutput);

function showCards(items){
  cardsInner.innerHTML='';
  if(!Array.isArray(items)||items.length===0){
    const no=document.createElement('div'); no.className='card';
    no.innerHTML=`<div class="meta">История</div><div class="body">Пусто</div>`;
    cardsInner.appendChild(no);
  } else items.slice().reverse().forEach(cmd=>{
    const card=document.createElement('div'); card.className='card';
    const meta=document.createElement('div'); meta.className='meta';
    meta.textContent=`Команда • ${new Date().toLocaleString()}`;
    const body=document.createElement('div'); body.className='body'; body.textContent=String(cmd);
    card.appendChild(meta); card.appendChild(body);
    card.addEventListener('click',()=>{ input.value=cmd; input.focus(); card.animate([{transform:'scale(1)'},{transform:'scale(.995)'}],{duration:150,easing:'ease-out'}); });
    cardsInner.appendChild(card);
  });
  cards.classList.remove('hidden'); cards.setAttribute('aria-hidden','false');
}
function hideCards(){ cards.classList.add('hidden'); cards.setAttribute('aria-hidden','true'); }
document.getElementById('close-cards').addEventListener('click',hideCards);
document.addEventListener('click',(ev)=>{ if(!cards.classList.contains('hidden')){ const inside=cards.contains(ev.target)||ev.target===btnHistory; if(!inside) hideCards(); } });

clearOutput();
append
