/* Hevy Signal muscle catalog v2. Data attribution: https://wger.de (per-item CC licenses).
 * ExerciseDB optional free non-commercial API: https://oss.exercisedb.dev/docs (no persistent caching).
 * Name-match confidence indicates matching quality, NOT evidence strength or hypertrophy effect. */
(function(root,factory){
  const api=factory();
  if(typeof module!=='undefined'&&module.exports)module.exports=api;
  root.HevyMuscles=api;
})(typeof globalThis!=='undefined'?globalThis:this,function(){
  'use strict';
  const WGER='https://wger.de/api/v2/exerciseinfo/';
  const EDB='https://oss.exercisedb.dev/api/v1/exercises';
  const OVERRIDE_KEY='hevy-signal-muscle-profiles-v2';
  const OLD_KEY='hevy-signal-muscle-overrides-v2';
  const CACHE_KEY='hevy-signal-wger-v2';
  let wger=[],edb=[],status='Local classification only',onlineError='';
  const mayStore=typeof localStorage!=='undefined';
  const read=(key,fallback)=>{if(!mayStore)return fallback;try{return JSON.parse(localStorage.getItem(key))??fallback}catch{return fallback}};
  const write=(key,val)=>{if(mayStore)try{localStorage.setItem(key,JSON.stringify(val))}catch(e){/* Quota/private mode */}};
  const clean=s=>String(s||'').toLowerCase().replace(/\b(rdls?)\b/g,'romanian deadlift').replace(/\b(db)\b/g,'dumbbell').replace(/\b(bb)\b/g,'barbell').replace(/\b(1 leg|one leg|single leg)\b/g,'singleleg').replace(/\b(hamstring curls?)\b/g,'leg curl').replace(/\b(hip abductions?)\b/g,'abduction').replace(/\b(hip adductions?)\b/g,'adduction').replace(/[^a-z0-9]+/g,' ').replace(/\b(extensions|raises|presses|squats|rows|curls|lunges|flyes|flies)\b/g,s=>({extensions:'extension',raises:'raise',presses:'press',squats:'squat',rows:'row',curls:'curl',lunges:'lunge',flyes:'fly',flies:'fly'})[s]).replace(/\s+/g,' ').trim();
  const generic=/^(with|and|the|exercise|standing|machine|lever|barbell|dumbbell|cable|bodyweight|weighted|plate|loaded|smith|seated|lying|band|resistance)$/;
  const tags=s=>clean(s).split(' ').filter(t=>t&&!generic.test(t));
  const families=[
    [/\b(extension|reverse nordic)\b/,/leg|knee|quadricep/],
    [/\b(leg curl|nordic curl)\b/,/hamstring|leg/],
    [/\b(abduction)\b/,null],
    [/\b(adduction)\b/,null],
    [/\b(hip thrust|glute bridge|glute drive)\b/,null],
    [/\b(romanian deadlift|stiff leg|good morning)\b/,null],
    [/\b(squat)\b/,null],
    [/\b(lunge|split squat|bulgarian)\b/,null],
    [/\b(row)\b/,null],
    [/\b(pulldown|pull up|chin up)\b/,null],
    [/\b(lateral raise|side raise)\b/,null],
    [/\b(overhead press|shoulder press)\b/,null],
    [/\b(bench press|chest press|chest fly)\b/,null]
  ];
  const family=s=>{const n=clean(s);for(let i=0;i<families.length;i++){const [r,required]=families[i];if(r.test(n)&&(!required||required.test(n)))return i;}return -1;};
  function score(a,b){
    const x=clean(a),y=clean(b);if(!x||!y)return 0;
    if(x===y)return 1;
    const f1=family(x),f2=family(y);if(f1>=0&&f2>=0&&f1!==f2)return 0;
    const tx=new Set(tags(x)),ty=new Set(tags(y));if(!tx.size||!ty.size)return 0;
    const shared=[...tx].filter(t=>ty.has(t)).length;
    if(!shared)return 0;
    const overlap=shared/Math.max(tx.size,ty.size),dice=2*shared/(tx.size+ty.size);
    const sameVariant=/\b(singleleg|incline|decline|sumo|close grip|wide grip)\b/g;
    const ax=x.match(sameVariant)||[],by=y.match(sameVariant)||[];
    const variantMismatch=ax.some(t=>!by.includes(t))||by.some(t=>!ax.includes(t));
    return Math.max(0,(overlap*.65+dice*.35) - (variantMismatch?.13:0));
  }
  const names=x=>Array.isArray(x)?x.map(v=>typeof v==='string'?v:v?.name_en||v?.name||'').filter(Boolean):[];
  const unique=a=>[...new Set(a.filter(Boolean))];
  const normMuscle=s=>{
    const v=String(s||'').trim().replace(/\s+/g,' ');
    const n=v.toLowerCase();
    if(n.startsWith('quads —')||n.startsWith('quads -'))return v;
    if(/gluteus medius|gluteus minimus/.test(n))return 'Glute med/min';
    if(/gluteus maximus/.test(n))return 'Glute max';
    if(/vastus (medialis|lateralis|intermedius)/.test(n))return 'Quads — vasti';
    if(/rectus femoris/.test(n))return 'Quads — rectus femoris';
    if(/quadriceps|quads/.test(n))return 'Quads';
    if(/biceps femoris|semitendinosus|semimembranosus|hamstrings/.test(n))return 'Hamstrings';
    if(/latissimus/.test(n))return 'Lats';
    if(/rhomboid|middle trapezius/.test(n))return 'Mid traps/rhomboids';
    if(/deltoid.*(middle|lateral)|lateral deltoid/.test(n))return 'Side delts';
    if(/deltoid.*(rear|posterior)/.test(n))return 'Rear delts';
    if(/deltoid.*(front|anterior)/.test(n))return 'Front delts';
    if(/pectoralis/.test(n))return 'Chest';
    if(/gastrocnemius|soleus/.test(n))return 'Calves';
    if(/triceps brachii/.test(n))return 'Triceps';
    if(/biceps brachii/.test(n))return 'Biceps';
    if(/erector spinae/.test(n))return 'Spinal erectors';
    return v;
  };
  const pack=(primary,secondary,stabilizers,source,confidence,matched='',note='')=>({
    primary:unique(primary.map(normMuscle)),secondary:unique(secondary.map(normMuscle)).filter(n=>!primary.map(normMuscle).includes(n)),
    stabilizers:unique(stabilizers.map(normMuscle)).filter(n=>!primary.concat(secondary).map(normMuscle).includes(n)),
    source,confidence,matched,note
  });
  // Transparent offline fallback: biomechanical approximations, not database-verified.
  const localRules=[
    [/abduct|clamshell|lateral band walk|monster walk/i,[['Glute med/min'],['Glute max'],[]]],
    [/hip thrust|glute bridge|glute drive|frog pump/i,[['Glute max'],['Hamstrings'],[]]],
    [/kickback|donkey kick|hip extension/i,[['Glute max'],['Hamstrings'],[]]],
    [/leg extension|reverse nordic/i,[['Quads — vasti','Quads — rectus femoris'],[],[]]],
    [/leg curl|nordic curl/i,[['Hamstrings'],[],[]]],
    [/romanian deadlift|\brdl\b|stiff.leg|good morning/i,[['Hamstrings','Glute max'],[],['Spinal erectors']]],
    [/sumo deadlift/i,[['Glute max','Adductors','Quads'],['Hamstrings'],['Spinal erectors']]],
    [/deadlift/i,[['Glute max','Spinal erectors'],['Hamstrings','Quads'],[]]],
    [/hip adduct|adductor machine|copenhagen/i,[['Adductors'],[],[]]],
    [/bulgarian|split squat|lunge|step.?up/i,[['Quads','Glute max'],['Adductors'],[]]],
    [/hack squat|pendulum squat|leg press|squat/i,[['Quads','Glute max'],['Adductors'],[]]],
    [/calf raise|plantar flex/i,[['Calves'],[],[]]],
    [/face pull|reverse fly|reverse pec|rear delt/i,[['Rear delts','Mid traps/rhomboids'],['External rotators'],[]]],
    [/lateral raise|side raise|cable y.?raise/i,[['Side delts'],['Upper traps'],[]]],
    [/overhead press|shoulder press|military press/i,[['Front delts'],['Triceps','Side delts'],[]]],
    [/chest.supported row|high row|wide.grip row|upper.back row/i,[['Mid traps/rhomboids','Rear delts'],['Lats','Biceps'],[]]],
    [/row/i,[['Lats','Mid traps/rhomboids'],['Rear delts','Biceps'],[]]],
    [/lat pulldown|pull.up|chin.up|pullover/i,[['Lats'],['Biceps'],[]]],
    [/incline (bench|press|chest)/i,[['Upper chest'],['Front delts','Triceps'],[]]],
    [/bench|chest press|push.up|pec deck|chest fly|cable fly/i,[['Chest'],['Front delts','Triceps'],[]]],
    [/tricep|skull.crusher|pushdown|overhead extension/i,[['Triceps'],[],[]]],
    [/bicep|hammer curl|preacher curl|incline curl|concentration curl/i,[['Biceps'],['Brachialis/brachioradialis'],[]]],
    [/crunch|sit.up|leg raise|ab wheel|plank/i,[['Abs'],['Hip flexors'],[]]]
  ];
  // ExRx's pages are linked only when manually confirmed. No scraping or redistribution.
  const validated=[
    {rule:/^barbell deadlift$/i,primary:['Spinal erectors'],secondary:['Glute max','Adductor magnus','Quads','Hamstrings','Soleus'],stabilizers:['Mid traps/rhomboids','Upper traps','Abs'],url:'https://exrx.net/WeightExercises/ErectorSpinae/BBDeadlift'},
    {rule:/^(sit.up|sit up)$/i,primary:['Abs'],secondary:['Iliopsoas','Tensor fasciae latae','Quads — rectus femoris','Sartorius','Obliques'],stabilizers:['Tibialis anterior'],url:'https://exrx.net/WeightExercises/RectusAbdominis/BWSitUp'}
  ];
  function fallback(ex){
    const v=validated.find(e=>e.rule.test(ex.trim()));
    if(v)return pack(v.primary,v.secondary,v.stabilizers,{name:'ExRx',url:v.url},'curated',ex,'Linked to a specific reviewed ExRx movement; your execution may differ.');
    for(const [re,parts] of localRules)if(re.test(ex))return pack(...parts,{name:'Local biomechanics rule',url:''},'estimated',ex,'Review this estimate, especially for custom movement variations.');
    return pack([],[],[],{name:'Unmapped',url:''},'unmapped','', 'No sufficiently reliable automatic classification.');
  }
  function overrides(){return read(OVERRIDE_KEY,{});}
  function setOverride(ex,profile){
    if(!ex||!profile)throw new Error('Exercise and mapping are required');
    const primary=unique((profile.primary||[]).map(String).map(s=>s.trim())),secondary=unique((profile.secondary||[]).map(String).map(s=>s.trim()));
    const stabilizers=unique((profile.stabilizers||[]).map(String).map(s=>s.trim()));
    if(!primary.length)throw new Error('Add at least one primary muscle');
    const all=pack(primary,secondary,stabilizers,{name:'Your correction',url:''},'manual',ex,'User-reviewed muscle mapping');
    const obj=overrides();obj[ex]=all;write(OVERRIDE_KEY,obj);return all;
  }
  function clearOverride(ex){const obj=overrides();delete obj[ex];write(OVERRIDE_KEY,obj);}
  function profile(ex){
    const custom=overrides()[ex];if(custom)return custom;
    const legacy=read(OLD_KEY,{})[ex];if(Array.isArray(legacy)){
      const first=legacy.filter(a=>Array.isArray(a)&&Number(a[1])>=.75).map(a=>a[0]);
      const second=legacy.filter(a=>Array.isArray(a)&&Number(a[1])<.75).map(a=>a[0]);
      if(first.length)return pack(first,second,[],{name:'Legacy correction',url:''},'manual',ex,'Imported from older muscle weights. Review if needed.');
    }
    const options=[];
    for(const item of wger){let top=0,matched='';for(const title of item.titles){const n=score(ex,title);if(n>top){top=n;matched=title;}}
      if(top>=.81)options.push({top,matched,item,provider:0});}
    for(const item of edb){const top=score(ex,item.name);if(top>=.88)options.push({top,matched:item.name,item,provider:1});}
    options.sort((a,b)=>b.top-a.top || a.provider-b.provider);
    if(options.length){
      const best=options[0],runner=options[1];
      // Avoid guessing if two similarly named but anatomically different records compete.
      if(!runner||best.top-runner.top>=.06 || JSON.stringify(best.item.primary)===JSON.stringify(runner.item.primary)){
        const source=best.provider===0?{name:'wger',url:'https://wger.de/api/v2/exerciseinfo/'+best.item.id+'/'}:{name:'ExerciseDB (free API)',url:'https://oss.exercisedb.dev/docs'};
        return pack(best.item.primary,best.item.secondary,[],source,best.top>=.97?'high':best.top>=.88?'medium':'review',best.matched,'Database exercise-name match; stabilizers were not supplied.');
      }
    }
    const f=fallback(ex);if(options.length)f.note+=' Multiple close database matches; manual review recommended.';return f;
  }
  function count(rows,{secondaryCredit=0}={}){
    const output={};for(const row of rows){const p=profile(row.exercise);
      for(const name of p.primary){const x=output[name]||(output[name]={primary:0,secondary:0,stabilizer:0,effective:0});x.primary+=1;}
      for(const name of p.secondary){const x=output[name]||(output[name]={primary:0,secondary:0,stabilizer:0,effective:0});x.secondary+=1;}
      for(const name of p.stabilizers){const x=output[name]||(output[name]={primary:0,secondary:0,stabilizer:0,effective:0});x.stabilizer+=1;}
    }
    for(const item of Object.values(output))item.effective=item.primary+secondaryCredit*item.secondary;
    return output;
  }
  const timeoutFetch=async (url,ms=12000)=>{const ctrl=new AbortController(),id=setTimeout(()=>ctrl.abort(),ms);try{const r=await fetch(url,{signal:ctrl.signal,headers:{Accept:'application/json'}});if(!r.ok)throw Error('HTTP '+r.status);return await r.json()}finally{clearTimeout(id)}};
  function convertWger(x){
    const english=(x.translations||[]).filter(t=>t.language===2 && t.name);
    if(!english.length)return null;
    const primary=names(x.muscles),secondary=names(x.muscles_secondary);
    if(!primary.length && !secondary.length)return null;
    return {id:x.id,titles:unique(english.map(t=>t.name)),primary,secondary};
  }
  function convertEDB(x){
    const primary=names(x.targetMuscles||(x.target?[x.target]:[])),secondary=names(x.secondaryMuscles);
    if(!x.name||!primary.length)return null;
    return {id:x.exerciseId||x.id,name:x.name,primary,secondary};
  }
  async function sync({useExerciseDB=true,force=false}={}){
    onlineError='';
    if(!useExerciseDB)edb=[];
    let record=read(CACHE_KEY,null);
    if(!force&&record&&Date.now()-record.when<7*86400000&&Array.isArray(record.rows)){
      wger=record.rows;status='wger cached catalog';}
    else{
      try{
        const items=[];let url=WGER+'?language=2&limit=100&offset=0';let page=0;
        while(url&&page<25){const data=await timeoutFetch(url);if(!data||!Array.isArray(data.results))throw Error('Unexpected wger API response');items.push(...data.results.map(convertWger).filter(Boolean));page++;
          url=data.next&&page<25?new URL(data.next,WGER).href:null;
          if(url&&!url.startsWith('https://wger.de/'))throw Error('Unexpected pagination host');
        }
        if(!items.length)throw Error('No English exercises returned');
        wger=items;write(CACHE_KEY,{when:Date.now(),rows:items});
        status='wger exercise catalog loaded';onlineError='';
      }catch(e){onlineError=String(e.message||e);status=wger.length?'wger cached catalog (offline)':'Offline local rules (wger unavailable)';}
    }
    if(useExerciseDB){
      try{
        // Only hold in memory; ExerciseDB terms do not allow permanent local storage.
        const data=await timeoutFetch(EDB,15000);
        const rows=Array.isArray(data)?data:Array.isArray(data.data)?data.data:Array.isArray(data.results)?data.results:Array.isArray(data.exercises)?data.exercises:[];
        edb=rows.map(convertEDB).filter(Boolean);
        if(edb.length)status+=' + ExerciseDB supplemental matches';
      }catch(e){onlineError+=(onlineError?'; ':'')+'ExerciseDB: '+String(e.message||e);}
    }
    return {status,wgerCount:wger.length,exerciseDBCount:edb.length,error:onlineError};
  }
  function diagnostics(){return {status,wgerCount:wger.length,exerciseDBCount:edb.length,error:onlineError};}
  return {profile,count,sync,score,clean,setOverride,clearOverride,diagnostics,convertWger,convertEDB,sourceNotes:'wger data are per-item CC licensed. ExerciseDB free API is non-commercial, attribution required, no persistent caching. Secondary counts are exposure, not proven fractions of hypertrophy.'};
});