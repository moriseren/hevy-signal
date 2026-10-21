/* Desktop v3.3 automatic mapping assistant. Uses local rules first; wger only
   for remaining unknowns. User-created mappings are NEVER replaced. */
(()=>{
  const sleep=ms=>new Promise(r=>setTimeout(r,ms));
  const missing=m=>!Array.isArray(m.major_muscles)||m.major_muscles.length===0||
    (m.major_muscles.length===1&&String(m.major_muscles[0]).toLowerCase()==='unmapped');
  let active=false,button,status;
  const request=async(path,opts)=>{
    const r=await fetch(path,opts);
    if(!r.ok)throw new Error('App returned '+r.status);
    return r.json();
  };
  async function applyMapping(m){
    const p=window.HevyMuscles.profile(m.exercise_title||'');
    if(!p.primary.length||p.confidence==='unmapped')return false;
    const updated={...m,major_muscles:p.primary,supporting_muscles:p.secondary,auto_tagged:true};
    if(!updated.movement||updated.movement==='Unmapped')updated.movement='Automatically classified (review if custom)';
    await request('/api/metadata',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(updated)});
    return p;
  }
  async function repair(){
    if(active)return;
    active=true;if(button)button.disabled=true;
    let changed=0,unmatched=0,catalog='';
    try{
      // Rules require no external network. Remote catalog fetch does not send a workout CSV,
      // exercise names, or the local Hevy API key.
      const list=await request('/api/metadata');
      for(const m of Array.isArray(list)?list:[]){
        if(!missing(m))continue;
        const result=await applyMapping(m);
        if(result)changed++; else unmatched++;
      }
      if(unmatched){
        status.textContent='Checking optional anatomy reference catalog…';
        const diag=await window.HevyMuscles.sync({useExerciseDB:false});
        catalog=diag.wgerCount?' · wger '+diag.wgerCount+' reference entries':' · offline rules only';
        const next=await request('/api/metadata');
        for(const m of Array.isArray(next)?next:[]){
          if(!missing(m))continue;
          const result=await applyMapping(m);
          if(result){changed++;unmatched--;}
        }
      }
      status.textContent=changed+' previously unmapped exercise(s) classified. '+unmatched+' require review.'+catalog;
      if(changed&&typeof window.load==='function')await window.load();
    }catch(e){status.textContent='Automatic mapping unavailable: '+(e?.message||String(e));}
    finally{active=false;if(button)button.disabled=false;}
  }
  window.addEventListener('load',async()=>{
    const editor=document.querySelector('#editor');
    if(!editor)return;
    const controls=document.createElement('div');
    controls.style.cssText='display:flex;flex-wrap:wrap;align-items:center;gap:10px;padding:12px 0';
    button=document.createElement('button');
    button.type='button';button.textContent='Auto-map unresolved exercises';
    status=document.createElement('span');
    status.style.cssText='font-size:12px;color:#bac5d2';
    status.textContent='Only unknown mappings are updated. Your custom edits are preserved.';
    controls.append(button,status);
    editor.parentNode.insertBefore(controls,editor);
    button.addEventListener('click',repair);
    await sleep(1100);
    repair();
  });
})();
