/* Hevy Signal: reviewed anatomical movement-family crosswalk.
 * These are biomechanics-informed classifications, NOT wger matches.
 * Exact movement variants are matched conservatively; technique still matters.
 */
(function(root,factory){
 const api=factory();
 if(typeof module!=='undefined'&&module.exports)module.exports=api;
 root.HevyCurated=api;
})(typeof globalThis!=='undefined'?globalThis:this,function(){
 'use strict';
 const simplify=t=>String(t||'').toLowerCase()
   .replace(/\b(1 leg|one leg|single leg)\b/g,'singleleg')
   .replace(/\b(hip abductions?)\b/g,'abduction')
   .replace(/\b(hip adductions?)\b/g,'adduction')
   .replace(/\b(rdls?)\b/g,'romanian deadlift')
   .replace(/[^a-z0-9]+/g,' ').replace(/\s+/g,' ').trim();
 const entries=[
  [/^(?:chest press|machine chest press)(?: machine)?$/,['Chest'],['Triceps','Front delts'],'Conventional horizontal chest press'],
  [/^(?:shoulder press|dumbbell shoulder press)(?: dumbbell)?$/,['Front delts','Side delts'],['Triceps'],'Overhead press; the triceps extend the elbow'],
  [/^(?:hip adduction|adduction|machine hip adduction)(?: machine)?$/,['Adductors'],[],'Hip adduction, not glute abduction'],
  [/^(?:cable abduction|hip abduction|machine hip abduction|abduction)(?: cable|machine)?$/,['Glute med/min'],['Glute max'],'Hip abduction; hip and torso position can change emphasis'],
  [/^(?:hip thrust|machine hip thrust)(?: machine)?$/,['Glute max'],['Hamstrings'],'Hip extension; glute maximus is prime mover'],
  [/^(?:lateral raise|cable lateral raise)(?: cable)?$/,['Side delts'],['Upper traps'],'Shoulder abduction'],
  [/^(?:leg extension|machine leg extension)(?: machine)?$/,['Quads'],[],'Knee extension, rectus femoris and vasti counted within quads'],
  [/^(?:leg press|machine leg press)(?: machine)?$/,['Quads','Glute max'],['Adductors'],'Knee and hip extension; foot placement and depth influence relative emphasis'],
  [/^(?:singleleg press horizontal|horizontal singleleg press|single leg press horizontal)(?: machine)?$/,['Quads','Glute max'],['Adductors'],'Unilateral horizontal leg press; stance and depth matter'],
  [/^(?:rear delt reverse fly|rear delt fly|reverse fly)(?: cable|machine)?$/,['Rear delts'],['Mid traps/rhomboids'],'Horizontal shoulder abduction and scapular retraction'],
  [/^(?:romanian deadlift|dumbbell romanian deadlift)(?: dumbbell)? b stance$/,['Hamstrings','Glute max'],[],'Staggered-stance hip hinge; erectors act as stabilizers'],
  [/^(?:scapular pulldown|scapular pull down|scapular depression)$/,['Lower traps','Lats'],[],'Scapular depression; check precise shoulder and elbow action'],
  [/^(?:seated leg curl|machine seated leg curl)(?: machine)?$/,['Hamstrings'],[],'Knee flexion'],
  [/^(?:seated row|machine seated row)(?: machine)?$/,['Lats','Mid traps/rhomboids'],['Rear delts','Biceps'],'Row path, grip and elbow angle vary mid-back vs lat emphasis'],
  [/^serratus press(?: cable|machine|dumbbell)?$/,['Serratus anterior'],[],'Scapular protraction; confirm specific technique'],
  [/^(?:standing cable glute kickbacks?|glute kickbacks? cable|cable glute kickbacks?)$/,['Glute max'],['Hamstrings'],'Hip extension with a cable; degree of knee flexion matters'],
  [/^(?:farmers? walk|farmers? carry|farmers? hold)$/,['Upper traps','Forearms'],['Abs'],'Loaded carry', 'carry'],
  [/^suitcase carry(?: dumbbell)?$/,['Obliques','Forearms'],['Quadratus lumborum'],'Unilateral loaded carry', 'carry'],
  [/^suitcase front carry(?: dumbbell)?$/,['Abs','Obliques'],['Forearms','Front delts'],'Front-loaded carry', 'carry'],
  [/^(?:treadmill|treadmill walk|treadmill running|walking|walk)$/,['Cardio / walking'],[],'Conditioning by time, pace and distance', 'cardio']
 ];
 function lookup(name){
  const simplified=simplify(name);
  const e=entries.find(x=>x[0].test(simplified));
  if(!e)return null;
  return {primary:[...e[1]],secondary:[...e[2]],matched:name,
    confidence:'curated',source:{name:'Reviewed biomechanics crosswalk',url:'https://exrx.net/Lists/Directory'},
    mode:e[4]||'resistance',
    note:'Biomechanics-informed movement family, not a verified wger item. '+e[3]};
 }
 return {lookup,listed:entries.length,simplify};
});