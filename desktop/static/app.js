const $=s=>document.querySelector(s), $$=s=>[...document.querySelectorAll(s)]; let META=[],BRIEF='';
const esc=s=>String(s??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));
async function api(url,opts={}){try{const r=await fetch(url,{headers:{'Content-Type':'application/json',...(opts.headers||{})},...opts});const text=await r.text();let j={};try{j=text?JSON.parse(text):{}}catch(_){throw Error(`Local app returned an unreadable response (${r.status}). Reopen Hevy Signal.`)}if(!r.ok)throw Error(j.error||r.statusText||`HTTP ${r.status}`);return j}catch(err){if(err instanceof TypeError&&/fetch/i.test(err.message))throw Error('Lost connection to Hevy Signal. Restart Hevy Signal and use the new tab it opens.');throw err}}

async function refreshProgramFolders(){
 const select=$('#programFolderSelect'),note=$('#programFolderMessage');
 if(!select||!note)return;
 try{
  const d=await api('/api/program-folders');
  const canChoose=d.programSource==='hevy-api'&&d.hevyConfigured;
  const folders=Array.isArray(d.folders)?d.folders:[];
  select.replaceChildren();
  const create=(id,label,disabled=false)=>{
   const option=document.createElement('option');
   option.value=id;option.textContent=label;option.disabled=disabled;
   select.appendChild(option);
  };
  create('','All saved routines');
  for(const f of folders)create(String(f.id),String(f.title));
  if(d.activeFolderId&&!folders.some(f=>String(f.id)===String(d.activeFolderId)))
   create(String(d.activeFolderId),'Unavailable folder — select another');
  select.value=String(d.activeFolderId||'');
  select.disabled=!canChoose;
  if(d.programSource==='csv-inferred'){
   note.textContent='CSV mode cannot read saved Hevy folders. Connect your Hevy Developer API and sync to choose Cut or Hypertrophy.';
  }else if(!d.hevyConfigured){
   note.textContent='Connect Hevy in Settings, then Sync Hevy to load program folders.';
  }else if(!folders.length){
   note.textContent='No folders loaded. Click Sync Hevy to retrieve saved Cut and Hypertrophy folders.';
  }else{
   note.textContent=(d.activeFolderName||'All saved routines')+' · '+d.currentRoutines+' active routines. Other folders and all workout history remain saved.';
  }
 }catch(e){note.textContent='Could not load folder selection: '+e.message;select.disabled=true;}
}

async function loadConfig(){const c=await api('/api/config');$('#hevyState').textContent=c.hevyConfigured?'Hevy key saved locally on this computer.':'Not connected yet.';return c}
function openSettings(){$('#settingsDialog').showModal()} $('#settingsBtn').onclick=openSettings;$('#connectBtn').onclick=openSettings;$('#settingsClose').onclick=()=>$('#settingsDialog').close();
$('#settingsForm').onsubmit=async e=>{e.preventDefault();const msg=$('#settingsMessage'),hk=$('#hevyKey').value.trim();if(hk.startsWith('sk-')){msg.textContent='That looks like an OpenAI key. Hevy Signal only needs your Hevy Developer API key.';return}msg.textContent='Saving and testing…';try{if(hk)await api('/api/config',{method:'POST',body:JSON.stringify({hevy_api_key:hk})});$('#hevyKey').value='';const t=await api('/api/hevy-test',{method:'POST'});msg.textContent=`Hevy connected — ${t.workoutCount} workouts found.`;await loadConfig();await load()}catch(err){msg.textContent=err.message}};

async function importCsvFile(file){
 if(!file)return;
 const msg=$('#message'); msg.textContent='Importing Hevy CSV locally…';
 try{
  const fd=new FormData(); fd.append('file',file);
  const r=await fetch('/api/import-hevy-csv',{method:'POST',body:fd}); const text=await r.text(); let j={}; try{j=JSON.parse(text)}catch(_){throw Error('Local app returned an unreadable CSV import response.')}; if(!r.ok)throw Error(j.error||`HTTP ${r.status}`);
  msg.textContent=`Imported ${j.workouts} workouts and ${j.sets} working sets. Current program inferred from ${j.routines} recent workout templates.`;
  onboardingDialog?.close(); $('#settingsDialog')?.close(); await loadConfig(); await load();
 }catch(e){msg.textContent='CSV import failed: '+e.message}
}
for(const id of ['importHevyCsv','bannerCsv','settingsCsv','onboardCsv']){const el=$('#'+id);if(el)el.onchange=async e=>{const f=e.target.files?.[0];await importCsvFile(f);e.target.value=''}}
function activate(name){$$('.tab').forEach(b=>b.classList.toggle('active',b.dataset.tab===name));$$('.panel').forEach(p=>p.classList.toggle('active',p.id===`panel-${name}`));if(name==='handoff')refreshBrief()}
$$('.tab').forEach(b=>b.onclick=()=>activate(b.dataset.tab));
async function load(){const d=await api('/api/analysis');const csvMode=d.programSource==='csv-inferred';$('#setupBanner').classList.toggle('show',!d.hevyConfigured&&!(d.counts.workouts>0));$('#programHeading').textContent=csvMode?'Current program — inferred from CSV':'Current Hevy routines';$('#sourceNotice').innerHTML=csvMode?'<div class="source-notice"><strong>CSV mode:</strong> Hevy workout exports do not contain saved routine definitions. Signal inferred these routine cards from the latest occurrence of recent workout names. Review Exercise setup before trusting progression targets.</div>':'';$('#stats').innerHTML=[[d.counts.routines||0,'current routines'],[d.counts.routineExercises||0,'program exercises'],[d.recs.filter(x=>x.status==='progress').length,'ready to progress'],[d.counts.unmapped,'unmapped program exercises']].map(x=>`<article><b>${x[0]}</b><span>${x[1]}</span></article>`).join('');$('#unmappedBadge').textContent=d.counts.unmapped?d.counts.unmapped:'';$('#routines').innerHTML=(d.routines||[]).map(r=>`<article class="routine-card"><div class="routine-title"><div><div class="eyebrow">HEVY ROUTINE</div><h3>${esc(r.title)}</h3></div><span class="pill">${(r.exercises||[]).length} movements</span></div><div class="routine-exercises">${(r.exercises||[]).map((x,i)=>`<div class="routine-ex"><span><b>${i+1}</b>${esc(x.exercise_title||'Exercise')}</span><span>${x.set_count||0} sets${x.rep_targets?` · ${esc(x.rep_targets)} reps`:''}</span></div>`).join('')||'<p class="muted">No exercises found.</p>'}</div></article>`).join('')||'<article class="routine-card"><h3>No current program yet</h3><p class="muted">Sync Hevy Pro or import a Hevy workout CSV.</p></article>';$('#recs').innerHTML=d.recs.map(r=>`<article class="rec" data-id="${esc(r.templateId)}"><div class="dot ${r.status}"></div><div class="grow"><div class="recrow"><h3>${esc(r.exercise)}</h3><span class="pill">${esc(r.muscle)}</span></div><strong>${esc(r.headline)}</strong><p>${esc(r.rationale)}</p><small>${Math.round(r.confidence*100)}% rule confidence · ${esc(r.stimulus)} · ${esc(r.role)}</small></div></article>`).join('')||'<p class="muted">Sync Hevy Pro or import a workout CSV to begin.</p>';$$('.rec').forEach(x=>x.onclick=()=>showHistory(x.dataset.id));$('#muscle').innerHTML=(d.muscle||[]).map(m=>`<div class="muscle"><span>${esc(m.muscle)}</span><b>${m.sets} effective sets</b></div>`).join('')||'<p class="muted">No recent data.</p>';META=await api('/api/metadata');renderEditor();await refreshProgramFolders()}
$('#sync').onclick=async()=>{const b=$('#sync');b.disabled=true;b.textContent='Checking Hevy…';$('#message').textContent='';try{const t=await api('/api/hevy-test',{method:'POST'});$('#message').textContent=`Hevy connected: ${t.workoutCount} workouts found. Importing…`;b.textContent='Syncing…';const j=await api('/api/sync',{method:'POST'});$('#message').textContent=`Done — synced ${j.synced} workouts and ${j.routines} current routines.`;await load()}catch(e){$('#message').textContent='Sync failed: '+e.message}finally{b.disabled=false;b.textContent='Sync Hevy'}};
function options(values,current){return values.map(v=>`<option ${v===current?'selected':''}>${v}</option>`).join('')}
function renderEditor(){const q=($('#filter').value||'').toLowerCase();const rows=META.filter(m=>m.exercise_title.toLowerCase().includes(q)||(m.major_muscles||[]).join(' ').toLowerCase().includes(q)||(m.supporting_muscles||[]).join(' ').toLowerCase().includes(q));$('#editor').innerHTML=rows.map(m=>{const unmapped=(m.major_muscles||[]).includes('Unmapped');return `<details class="editcard" ${unmapped?'open':''}><summary><div><strong>${esc(m.exercise_title)}</strong><span>${esc((m.major_muscles||['Unmapped']).join(' + '))} · ${esc(m.movement)} ${m.auto_tagged?'· auto-tagged':''}</span></div><span class="pill ${unmapped?'warn':''}">${esc(m.role)}</span></summary><form data-id="${esc(m.template_id)}"><div class="fields"><label>Major contributors<input name="major_muscles" value="${esc((m.major_muscles||[]).join(', '))}"></label><label>Supporting contributors<input name="supporting_muscles" value="${esc((m.supporting_muscles||[]).join(', '))}"></label><label>Movement<input name="movement" value="${esc(m.movement)}"></label><label>Stimulus<select name="stimulus">${options(['lengthened','shortened','mixed'],m.stimulus)}</select></label><label>Programming role<select name="role">${options(['primary','secondary','accessory','rehab'],m.role)}</select></label><label>Fatigue cost (1–5)<input name="fatigue_cost" type="number" min="1" max="5" value="${m.fatigue_cost}"></label><label>Min reps<input name="min_reps" type="number" min="1" value="${m.min_reps}"></label><label>Max reps<input name="max_reps" type="number" min="1" value="${m.max_reps}"></label><label>Sets to earn progression<input name="target_sets" type="number" min="1" value="${m.target_sets}"></label><label>Load jump, kg<input name="increment_kg" type="number" step="0.25" min="0" value="${m.increment_kg}"></label><label class="wide">Notes<input name="notes" value="${esc(m.notes||'')}"></label></div><button class="save">Save rules</button><span class="saved"></span></form></details>`}).join('')||'<p class="muted">No matching exercises.</p>';$$('#editor form').forEach(f=>f.onsubmit=saveMeta)} $('#filter').oninput=renderEditor;
async function saveMeta(e){e.preventDefault();const f=e.currentTarget,fd=new FormData(f),obj=Object.fromEntries(fd);obj.template_id=f.dataset.id;obj.major_muscles=obj.major_muscles.split(',').map(x=>x.trim()).filter(Boolean);obj.supporting_muscles=obj.supporting_muscles.split(',').map(x=>x.trim()).filter(Boolean);for(const k of ['fatigue_cost','min_reps','max_reps','target_sets','increment_kg'])obj[k]=Number(obj[k]);const s=f.querySelector('.saved');try{await api('/api/metadata',{method:'POST',body:JSON.stringify(obj)});s.textContent='Saved';setTimeout(()=>s.textContent='',1200);await load()}catch(err){s.textContent=err.message}}
async function showHistory(id){const d=await api('/api/history?template_id='+encodeURIComponent(id)),m=d.meta;$('#historyBody').innerHTML=`<div class="eyebrow">EXERCISE HISTORY</div><h2>${esc(m.exercise_title)}</h2><p>${esc((m.major_muscles||['Unmapped']).join(' + '))} · ${esc(m.stimulus)} · target ${m.min_reps}–${m.max_reps}</p><div class="historyrows">${d.sessions.map(s=>`<div><span>${esc((s.date||'').slice(0,10))}<small>${esc(s.workout)}</small></span><b>${s.bestSet[0]} kg × ${s.bestSet[1]}</b><span>${s.sets} sets · e1RM ${s.bestE1RM} kg${s.avgRpe?` · RPE ${s.avgRpe}`:''}</span></div>`).join('')}</div>`;$('#history').showModal()} $('#history .close').onclick=()=>$('#history').close();
async function refreshBrief(){const m=$('#handoffMessage');try{const j=await api('/api/chatgpt-brief');BRIEF=j.brief||'';$('#briefPreview').textContent=BRIEF||'Sync Hevy first.';return BRIEF}catch(e){m.textContent=e.message;throw e}}
$('#copyBrief').onclick=async()=>{const m=$('#handoffMessage');try{const brief=await refreshBrief();await navigator.clipboard.writeText(brief);m.textContent='Copied. Paste it into your ChatGPT training chat, then ask your question.'}catch(e){m.textContent='Could not copy automatically: '+e.message}};
$('#downloadBrief').onclick=()=>{window.location.href='/api/chatgpt-brief?download=1';$('#handoffMessage').textContent='Downloaded hevy-signal-chatgpt-brief.md.'}; $('#openChatGPT').onclick=()=>window.open('https://chatgpt.com/','_blank','noopener');
const privacyDialog=$('#privacyDialog'),onboardingDialog=$('#onboardingDialog');$('#privacyBtn').onclick=async()=>{privacyDialog.showModal();try{const a=await api('/api/about');$('#dataLocation').textContent='Local data folder: '+a.dataDir;$('#aboutInfo').textContent=`Hevy Signal ${a.version} · ${a.platform}/${a.arch} · manual updates`;document.body.dataset.platform=a.platform;}catch(e){$('#privacyMessage').textContent=e.message}};$('#privacyClose').onclick=()=>privacyDialog.close();$('#onboardStart').onclick=()=>{onboardingDialog.close();openSettings()};$('#onboardSkip').onclick=()=>onboardingDialog.close();async function maybeOnboard(){try{const c=await api('/api/config'),d=await api('/api/analysis');if(!c.hevyConfigured&&!(d.counts.workouts>0)&&!sessionStorage.getItem('hevySignalOnboarded')){sessionStorage.setItem('hevySignalOnboarded','1');onboardingDialog.showModal()}}catch(_){}}
$('#exportMappings').onclick=async()=>{try{const r=await fetch('/api/export-metadata');if(!r.ok)throw Error('Export failed');const blob=await r.blob(),u=URL.createObjectURL(blob),a=document.createElement('a');a.href=u;a.download='hevy-signal-exercise-mappings.json';a.click();URL.revokeObjectURL(u);$('#privacyMessage').textContent='Exercise mappings exported. API keys are not included.'}catch(e){$('#privacyMessage').textContent=e.message}};
$('#importMappings').onchange=async e=>{const f=e.target.files?.[0];if(!f)return;try{const obj=JSON.parse(await f.text()),j=await api('/api/import-metadata',{method:'POST',body:JSON.stringify(obj)});$('#privacyMessage').textContent=`Imported ${j.imported} matching exercise mappings.`;await load()}catch(err){$('#privacyMessage').textContent=err.message}finally{e.target.value=''}};
$('#resetData').onclick=async()=>{if(!confirm('Delete all synced Hevy workouts, routines, and mappings? Your Hevy key will be kept.'))return;await api('/api/reset',{method:'POST',body:JSON.stringify({mode:'data'})});$('#privacyMessage').textContent='Synced training data deleted.';await load()};$('#resetAll').onclick=async()=>{if(!confirm('Delete ALL local Hevy Signal data, including the saved Hevy key?'))return;await api('/api/reset',{method:'POST',body:JSON.stringify({mode:'everything'})});privacyDialog.close();await loadConfig();await load();onboardingDialog.showModal()};$('#quitApp').onclick=async()=>{if(!confirm('Quit Hevy Signal? You can reopen it whenever you want.'))return;try{await api('/api/quit',{method:'POST',body:'{}'});document.body.innerHTML='<main><section class=\"setup-banner show\"><div><h3>Hevy Signal quit.</h3><p>You can close this tab and reopen Hevy Signal anytime.</p></div></section></main>'}catch(_){document.body.innerHTML='<main><section class=\"setup-banner show\"><div><h3>Hevy Signal quit.</h3><p>You can close this tab.</p></div></section></main>'}};

$('#programFolderSelect').onchange=async e=>{
 const control=e.currentTarget,note=$('#programFolderMessage'),value=control.value;
 control.disabled=true;note.textContent='Switching active training program…';
 try{
  const d=await api('/api/select-program-folder',{method:'POST',body:JSON.stringify({folder_id:value})});
  note.textContent='Active program: '+d.activeFolderName;
  await load();
 }catch(err){
  note.textContent='Could not switch program: '+err.message;
  await refreshProgramFolders();
 }finally{control.disabled=false;}
};

loadConfig().then(async()=>{await load();await maybeOnboard()});
