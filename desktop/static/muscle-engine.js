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
  const CACHE_KEY='hevy-signal-wger-v3';
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
    if(/(?:deltoid.*(?:middle|lateral)|(?:middle|lateral) deltoid)/.test(n))return 'Side delts';
    if(/(?:deltoid.*(?:rear|posterior)|(?:rear|posterior) deltoid)/.test(n))return 'Rear delts';
    if(/(?:deltoid.*(?:front|anterior)|(?:front|anterior) deltoid)/.test(n))return 'Front delts';
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
  // Offline movement-family coverage for common Hevy names and custom variants.
  // These are anatomical heuristics, not database-verified classifications.
  // Order is important: narrow variations precede general compound patterns.
  const localRules=[
    [/^archer pull$/i,[['Rear delts'],['Mid traps/rhomboids','External rotators'],[]]],
    [/^serratus press\b/i,[['Serratus anterior'],['Chest'],[]]],
    [/^scapular (pull.?down|depression)|^straight.arm scapular (pull|depress)/i,[['Lower traps','Lats'],['Pectoralis minor'],[]]],
    [/^suitcase front carry/i,[['Obliques','Abs'],['Forearms','Front delts'],[]]],
    [/^suitcase (carry|walk|hold)/i,[['Obliques','Abs'],['Forearms','Quadratus lumborum'],[]]],
    [/^(walk|walking|treadmill walk|brisk walk)(ing)?$/i,[['Cardio / walking'],[],[]]],

    [/hip adduct|adduction|adductor machine|copenhagen plank|copenhagen adduct/i,[['Adductors'],[],[]]],
    [/hip abduct|abduction|abductor machine|clamshell|clam shell|lateral band walk|monster walk|side lying hip raise|side lying leg raise/i,[['Glute med/min'],['Glute max'],[]]],
    [/hip thrust|glute bridge|glute drive|frog pump|kas bridge/i,[['Glute max'],['Hamstrings'],[]]],
    [/glute kickback|cable kickback|donkey kick|quadruped hip extension|hip extension|hip ext\b/i,[['Glute max'],['Hamstrings'],[]]],
    [/cable pull through|pull through|reverse hyperextension|reverse hyper/i,[['Glute max'],['Hamstrings'],['Spinal erectors']]],
    [/45.degree (back |hip )?extension|back extension|hyperextension|hip hyperextension/i,[['Spinal erectors','Glute max'],['Hamstrings'],[]]],
    [/leg extension|reverse nordic|sissy squat|spanish squat/i,[['Quads — vasti','Quads — rectus femoris'],[],[]]],
    [/leg curl|hamstring curl|nordic curl|glute ham raise|\bghr\b|razor curl/i,[['Hamstrings'],[],[]]],
    [/romanian deadlift|\brdls?\b|stiff.leg|straight.leg deadlift|good morning/i,[['Hamstrings','Glute max'],[],['Spinal erectors']]],
    [/sumo deadlift/i,[['Glute max','Adductors','Quads'],['Hamstrings'],['Spinal erectors']]],
    [/deadlift|rack pull|block pull|trap bar lift/i,[['Glute max','Spinal erectors'],['Hamstrings','Quads'],[]]],
    [/leg press|hack squat|pendulum squat|belt squat|goblet squat|front squat|back squat|smith squat|squat|wall sit/i,[['Quads','Glute max'],['Adductors'],[]]],
    [/bulgarian|split squat|lunge|step.?up|step.?down/i,[['Quads','Glute max'],['Adductors'],[]]],
    [/calf press|calf raise|soleus raise|seated calf|standing calf|donkey calf|toe press/i,[['Calves'],[],[]]],
    [/tibialis|toe raise|dorsiflexion/i,[['Tibialis anterior'],[],[]]],
    [/face pull|reverse fly|reverse pec|rear delt|reverse butterfly/i,[['Rear delts','Mid traps/rhomboids'],['External rotators'],[]]],
    [/lateral raise|side raise|cable y.?raise|lean.?away raise|scaption/i,[['Side delts'],['Upper traps'],[]]],
    [/front raise|front plate raise/i,[['Front delts'],[],[]]],
    [/external rotation|rotator cuff|cuban press/i,[['External rotators'],['Rear delts'],[]]],
    [/shrug|farmer.?s? (walk|carry|hold)/i,[['Upper traps'],['Forearms'],[]]],
    [/upright row|high pull/i,[['Side delts','Upper traps'],['Biceps'],[]]],
    [/arnold press|overhead press|shoulder press|military press|landmine press|push press/i,[['Front delts'],['Triceps','Side delts'],[]]],
    [/chest.supported row|high row|wide.grip row|upper.back row|rear delt row|face.away row/i,[['Mid traps/rhomboids','Rear delts'],['Lats','Biceps'],[]]],
    [/t.?bar row|inverted row|horizontal row|seal row|bent.?over row|seated row|one.arm row|single.arm row|machine row|cable row|dumbbell row|barbell row|\brow\b/i,[['Lats','Mid traps/rhomboids'],['Rear delts','Biceps'],[]]],
    [/lat pulldown|pull.?down|pull.?up|chin.?up|pull.?over|straight.arm pulldown|lat prayer|scapular pull/i,[['Lats'],['Biceps'],[]]],
    [/close.grip bench|close.grip press|jm press|tricep dip|triceps dip|bench dip/i,[['Triceps'],['Chest','Front delts'],[]]],
    [/chest dip|weighted dip|parallel bar dip|\bdips?\b/i,[['Chest','Triceps'],['Front delts'],[]]],
    [/incline (dumbbell |barbell |smith |machine )?(bench|press|chest)|incline bench|incline dumbbell (bench|press|fly)/i,[['Upper chest'],['Front delts','Triceps'],[]]],
    [/bench|chest press|push.?up|pec.?deck|chest fly|cable fly|dumbbell fly|butterfly|cable cross.?over|chest crossover/i,[['Chest'],['Front delts','Triceps'],[]]],
    [/tricep|triceps|skull.?crusher|push.?down|overhead extension|rope extension|french press|kickback \((dumbbell|cable)\)/i,[['Triceps'],[],[]]],
    [/wrist extension|wrist curl|reverse curl|forearm curl|grip trainer|wrist roller/i,[['Forearms'],['Biceps'],[]]],
    [/hammer curl|preacher curl|incline curl|concentration curl|spider curl|zottman curl|bayesian curl|bicep curl|biceps curl|cable curl|dumbbell curl|barbell curl|ez.bar curl|\bcurl\b/i,[['Biceps'],['Brachialis/brachioradialis'],[]]],
    [/pallof|anti.?rotation|woodchop|wood chop|russian twist|side plank|oblique crunch/i,[['Obliques'],['Abs'],[]]],
    [/crunch|sit.?up|leg raise|knee raise|hanging knee|ab wheel|plank|dead bug|hollow hold|hollow body|mountain climber|v.?up|toes.to.bar|abdominal/i,[['Abs'],['Hip flexors'],[]]],
    [/back raise|superman|bird dog/i,[['Spinal erectors'],['Glute max'],[]]],
    [/hip flexion|psoas march|hip flexor raise/i,[['Hip flexors'],[],[]]],
    [/neck flexion|neck extension|neck harness/i,[['Neck'],[],[]]]
  ];
  // ExRx's pages are linked only when manually confirmed. No scraping or redistribution.
  const validated=[
    {rule:/^barbell deadlift$/i,primary:['Spinal erectors'],secondary:['Glute max','Adductor magnus','Quads','Hamstrings','Soleus'],stabilizers:['Mid traps/rhomboids','Upper traps','Levator scapulae','Abs'],url:'https://exrx.net/WeightExercises/ErectorSpinae/BBDeadlift'},
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
  // Database-only lookup: intentionally bypass local rules, manual overrides,
  // and previously imported muscle maps. Use for authoritative bulk remapping.
  // This is exercise-name confidence, not validation of individual technique.
  const equipmentMarker=s=>{
    const x=clean(s);
    const found=['dumbbell','barbell','kettlebell','cable','smith','machine','band','bodyweight']
      .filter(t=>new RegExp('\\b'+t+'\\b').test(x));
    return found;
  };
  function databaseProfile(ex){
    const wanted=clean(ex);
    if(!wanted||!wger.length)return null;
    const candidates=[];
    for(const item of wger) {
      for(const name of item.titles) {
        const candidate=clean(name);
        const eq=equipmentMarker(ex),other=equipmentMarker(name);
        // Explicitly contradictory equipment variants cannot be silently
        // treated as the same exercise, even when the remaining words match.
        if(eq.length&&other.length&&!eq.some(x=>other.includes(x)))continue;
        // Prevent side-to-side, single-leg, incline/decline variant collapse.
        const modifiers=['singleleg','incline','decline','sumo','close grip','wide grip'];
        if(modifiers.some(m=>wanted.includes(m)!==candidate.includes(m)))continue;
        const similarity=score(ex,name);
        // 0.97+ allows e.g. Leg Extensions vs Leg Extension (Machine),
        // but not a broadly similar row or different exercise.
        if(similarity<.97)continue;
        if(!Array.isArray(item.primary)||!item.primary.length)continue;
        candidates.push({item,name,similarity});
      }
    }
    candidates.sort((a,b)=>b.similarity-a.similarity);
    if(!candidates.length)return null;
    const best=candidates[0];
    const ties=candidates.filter(c=>c.similarity>=best.similarity-.01);
    if(ties.some(x=>JSON.stringify(x.item.primary.slice().sort())!==JSON.stringify(best.item.primary.slice().sort())))return null;
    const p=pack(best.item.primary,best.item.secondary,[],{name:'wger',url:'https://wger.de/api/v2/exerciseinfo/'+best.item.id+'/'},'high',best.name,
      'Exact or strongly normalized wger name match. Variations in exercise setup may change emphasis.');
    // Enforce canonical quad grouping rather than counting subhead plus whole.
    const normParts=parts=>unique(parts.map(x=>/^(quads|quads [—-]|rectus femoris|vastus|quadriceps)/i.test(x)?'Quads':x));
    p.primary=normParts(p.primary);p.secondary=normParts(p.secondary).filter(x=>!p.primary.includes(x));
    return p;
  }
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
      const conflict=!!runner&&best.top-runner.top<.06&&JSON.stringify(best.item.primary)!==JSON.stringify(runner.item.primary);
      // Exact database matches win; close non-exact, conflicting options require review.
      if(best.top>=.97||!conflict){
        const source=best.provider===0?{name:'wger',url:'https://wger.de/api/v2/exerciseinfo/'+best.item.id+'/'}:{name:'ExerciseDB (free API)',url:'https://oss.exercisedb.dev/docs'};
        const note=conflict?'Other similarly named reference records classify different primary muscles; review this mapping.':'Database exercise-name match; stabilizers were not supplied.';
        return pack(best.item.primary,best.item.secondary,[],source,conflict?'review':best.top>=.97?'high':best.top>=.88?'medium':'review',best.matched,note);
      }
    }
    const f=fallback(ex);if(options.length)f.note+=' Multiple close database matches; manual review recommended.';return f;
  }
  function count(rows,{secondaryCredit=0}={}){
    const output={};for(const row of rows){const p=profile(row.exercise);
      if(p.primary.length===1 && p.primary[0]==='Cardio / walking')continue;
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
        const items=[];let url=WGER+'?language=2&limit=100&offset=0';let page=0,truncated=false;
        while(url&&page<30){const data=await timeoutFetch(url);if(!data||!Array.isArray(data.results))throw Error('Unexpected wger API response');items.push(...data.results.map(convertWger).filter(Boolean));page++;
          if(data.next && page>=30){truncated=true;url=null;}
          else url=data.next?new URL(data.next,WGER).href:null;
          if(url&&!url.startsWith('https://wger.de/'))throw Error('Unexpected pagination host');
        }
        if(!items.length)throw Error('No English exercises returned');
        wger=items;write(CACHE_KEY,{when:Date.now(),rows:items});
        status=truncated?'wger catalog partially loaded (page limit)':'wger exercise catalog loaded';onlineError='';
      }catch(e){onlineError=String(e.message||e);status=wger.length?'wger cached catalog (offline)':'Offline local rules (wger unavailable)';}
    }
    if(useExerciseDB){
      const seen=new Set(),items=[];
      try{
        const pageSize=100,maxPages=18;
        for(let page=0,offset=0;page<maxPages;page++){
          // Fetch public catalog pages; do not send personal workout/exercise names.
          const endpoint=EDB+'?limit='+pageSize+'&offset='+offset;
          const data=await timeoutFetch(endpoint,15000);
          const rows=Array.isArray(data)?data:Array.isArray(data.data)?data.data:
            Array.isArray(data.results)?data.results:Array.isArray(data.exercises)?data.exercises:[];
          if(!rows.length)break;
          let added=0;
          for(const item of rows.map(convertEDB).filter(Boolean)){
            const id=String(item.id||item.name);
            if(!seen.has(id)){seen.add(id);items.push(item);added++;}
          }
          if(!added)break;
          offset+=rows.length;
          const total=Number(data.total??data.count??data.pagination?.total??NaN);
          if(Number.isFinite(total)&&offset>=total)break;
          // If the server explicitly says there is no next page, stop.
          if(data.next===null || data.pagination?.hasNext===false)break;
          // No reliable end-of-catalog metadata: continue until the final empty page.
        }
        edb=items;
        if(edb.length)status+=' + ExerciseDB '+edb.length+' supplemental records';
      }catch(e){
        if(items.length){edb=items;status+=' + ExerciseDB partial catalog ('+items.length+' records)';}
        onlineError+=(onlineError?'; ':'')+'ExerciseDB: '+String(e.message||e);
      }
    }
    return {status,wgerCount:wger.length,exerciseDBCount:edb.length,error:onlineError};
  }
  function diagnostics(){return {status,wgerCount:wger.length,exerciseDBCount:edb.length,error:onlineError};}
  return {profile,databaseProfile,count,sync,score,clean,setOverride,clearOverride,diagnostics,convertWger,convertEDB,sourceNotes:'wger data are per-item CC licensed. ExerciseDB free API is non-commercial, attribution required, no persistent caching. Secondary counts are exposure, not proven fractions of hypertrophy.'};
});