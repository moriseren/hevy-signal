/* Hevy Signal v3.7. Database-first muscle mapping; includes previous manual maps.
 * Hevy workout history and API keys never leave the local application.
 * wger public exercise catalog is downloaded; exercise names matched locally.
 */
(()=>{
  let active=false,button,status;
  async function localAPI(path,opts={}){
    const r=await fetch(path,{
      ...opts,headers:{'Content-Type':'application/json',...(opts.headers||{})}
    });
    const data=await r.json();
    if(!r.ok)throw Error(data.error||'Signal request failed: '+r.status);
    return data;
  }
  function report(msg){if(status)status.textContent=msg}
  async function remapAll(){
    if(active)return;
    active=true;if(button)button.disabled=true;
    try{
      report('Downloading wger exercise database; manual and automatic labels will ALL be checked…');
      const result=await window.HevyMuscles.sync({useExerciseDB:false});
      const ready=result.wgerCount>0 && !/partial|unavailable|offline/i.test(result.status) && !result.error;
      if(!ready)throw Error('Could not load a complete wger reference catalog: '+(result.error||result.status)+'. Existing classifications were not changed.');
      const all=await localAPI('/api/all-exercises');
      const exercises=Array.isArray(all)?all:[];
      if(!exercises.length){
        report('No exercises to remap. Sync Hevy or import your workouts first.');
        return;
      }
      const suggestions=[];
      let notMatched=0;
      for(const ex of exercises){
        if(!ex.template_id || !ex.exercise_title)continue;
        const best=window.HevyMuscles.databaseProfile(ex.exercise_title);
        if(!best || best.source.name!=='wger' || best.confidence!=='high'){
          notMatched++;continue;
        }
        suggestions.push({
          template_id:ex.template_id,
          exercise_title:ex.exercise_title,
          matched:best.matched,
          source:'wger',
          confidence:'high',
          primary:best.primary,
          secondary:best.secondary
        });
      }
      if(!suggestions.length){
        throw Error('No exercises matched wger reliably. Previous classifications were preserved for safety. Check exercise names and catalog coverage.');
      }
      report('Applying '+suggestions.length+' database matches, including replacements for manual maps…');
      const res=await localAPI('/api/database-remap',{
        method:'POST',body:JSON.stringify({catalog_loaded:true,suggestions})
      });
      report(res.matched+' / '+res.checked+' exercises matched to wger; '+
        res.needsReview+' need verification and are excluded from muscle-volume totals. '+
        res.changed+' classifications updated. No manual mapping was exempted.');
      if(typeof window.load==='function')await window.load();
    }catch(e){
      report('Database remapping incomplete: '+String(e.message||e));
    }finally{
      active=false;if(button)button.disabled=false;
    }
  }
  window.HevyMusclesRemapAll=remapAll;
  window.addEventListener('load',()=>{
    const editor=document.querySelector('#editor');
    if(!editor)return;
    const wrap=document.createElement('div');
    wrap.style.cssText='display:flex;flex-wrap:wrap;align-items:center;gap:10px;padding:12px 0';
    button=document.createElement('button');
    button.type='button';
    button.textContent='Rebuild ALL muscle mappings from wger';
    status=document.createElement('span');
    status.setAttribute('role','status');
    status.style.cssText='font-size:12px;color:#bac5d2;max-width:610px';
    status.textContent='Includes manual maps. Unmatched exercises stay flagged and do not contribute to modeled muscle totals.';
    wrap.append(button,status);
    editor.parentNode.insertBefore(wrap,editor);
    button.addEventListener('click',remapAll);
    // Initial import from old versions; after Hevy sync, app.js requests it again.
    setTimeout(remapAll,1500);
  });
})();
