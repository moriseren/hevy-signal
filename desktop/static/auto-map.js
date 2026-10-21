/* Hevy Signal v3.4: refresh only the machine-assigned anatomy of exercises.
 * Manual classification is immutable to this module. Server verifies each request. */
(()=>{
  let active=false, button=null,status=null;
  const unknown=m=>!Array.isArray(m.major_muscles)||m.major_muscles.length===0||
    (m.major_muscles.length===1&&String(m.major_muscles[0]).trim().toLowerCase()==='unmapped');
  const automatic=m=>m.auto_tagged===true ||
    (unknown(m)&&(m.movement==='Unmapped'||!m.movement)) ||
    String(m.movement||'').startsWith('Automatically classified');
  async function request(url,opts={}){
    const r=await fetch(url,{...opts,headers:{'Content-Type':'application/json',...(opts.headers||{})}});
    const data=await r.json();
    if(!r.ok)throw Error(data.error||'Local app error '+r.status);
    return data;
  }
  function report(s){if(status)status.textContent=s}
  async function remap({withReferences=true}={}){
    if(active)return;
    active=true;
    if(button)button.disabled=true;
    try{
      report('Rechecking automatic mappings without changing manual edits…');
      const local=await request('/api/auto-remap',{method:'POST',body:'{"suggestions":[]}'});
      let total=local.changed||0, referenceMapped=0, last=local, note='';
      if(withReferences){
        report('Checking exact matches in the public exercise reference catalog…');
        try{
          const diag=await window.HevyMuscles.sync({useExerciseDB:false});
          if(diag.wgerCount>0){
            const metas=await request('/api/metadata');
            const suggestions=[];
            for(const m of Array.isArray(metas)?metas:[]){
              if(!automatic(m)||!m.template_id||!m.exercise_title)continue;
              const p=window.HevyMuscles.profile(m.exercise_title);
              // A shared family name is not enough: protect custom technique variants.
              if(p.source?.name!=='wger'||p.confidence!=='high'||!Array.isArray(p.primary)||!p.primary.length)continue;
              if(p.matched?.trim().toLowerCase()!==m.exercise_title.trim().toLowerCase())continue;
              suggestions.push({
                template_id:m.template_id,exercise_title:m.exercise_title,matched:p.matched,
                primary:p.primary,secondary:p.secondary,confidence:'high',source:'wger'
              });
            }
            if(suggestions.length){
              const res=await request('/api/auto-remap',{method:'POST',body:JSON.stringify({suggestions})});
              total+=res.changed||0;
              referenceMapped=res.referenceMapped||0;
              last=res;
            }
            note=' · '+diag.wgerCount+' catalog records checked';
          }else note=' · catalog unavailable; offline rules used';
        }catch(e){note=' · catalog unavailable ('+String(e.message||e)+')';}
      }
      report(total+' automatic classification(s) updated; '+last.manualPreserved+
        ' manual mapping(s) protected; '+last.unmapped+' still unmapped'+
        (referenceMapped?' · '+referenceMapped+' exact reference updates':'')+note+'.');
      if(typeof window.load==='function')await window.load();
    }catch(e){
      report('Unable to remap: '+String(e.message||e));
    }finally{active=false;if(button)button.disabled=false}
  }
  window.addEventListener('load',()=>{
    const editor=document.querySelector('#editor');
    if(!editor)return;
    const wrap=document.createElement('div');
    wrap.style.cssText='display:flex;flex-wrap:wrap;align-items:center;gap:10px;padding:12px 0';
    button=document.createElement('button');
    button.type='button';
    button.textContent='Re-map all automatic exercises';
    status=document.createElement('span');
    status.setAttribute('role','status');
    status.style.cssText='font-size:12px;color:#bac5d2;max-width:520px';
    status.textContent='Keeps your manually entered muscles and all programming settings.';
    wrap.append(button,status);
    editor.parentNode.insertBefore(wrap,editor);
    button.addEventListener('click',()=>remap({withReferences:true}));
    // Local rule updates also happen in the Go backend at startup; no online
    // request is needed until the user explicitly asks to check the catalog.
  });
})();