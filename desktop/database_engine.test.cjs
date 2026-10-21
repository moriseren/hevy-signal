const assert=require('node:assert/strict');
global.localStorage={
  value:{"hevy-signal-muscle-profiles-v2":JSON.stringify({
    "Shoulder Press (Dumbbell)":{primary:["Biceps"],secondary:[],source:{name:"Your correction"},confidence:"manual"}
  })},
  getItem(k){return this.value[k]??null},
  setItem(k,v){this.value[k]=v},
};
const examples=[
 {id:10,translations:[{language:2,name:"Shoulder Press Dumbbell"}],
  muscles:[{name_en:"Anterior deltoid"}],muscles_secondary:[{name_en:"Triceps brachii"}]},
 {id:11,translations:[{language:2,name:"Shoulder Press Barbell"}],
  muscles:[{name_en:"Anterior deltoid"}],muscles_secondary:[{name_en:"Triceps brachii"}]},
 {id:12,translations:[{language:2,name:"Leg Extensions"}],
  muscles:[{name_en:"Rectus femoris"},{name_en:"Vastus lateralis"}],muscles_secondary:[]},
 {id:13,translations:[{language:2,name:"Leg Curl"}],
  muscles:[{name_en:"Biceps femoris"}],muscles_secondary:[]}
];
global.fetch=async()=>({ok:true,json:async()=>({results:examples,next:null})});
const T=require('./static/muscle-engine.js');
(async()=>{
 const d=await T.sync({useExerciseDB:false,force:true});
 assert.equal(d.wgerCount,4);
 const shoulder=T.databaseProfile("Shoulder Press (Dumbbell)");
 assert.ok(shoulder,"dumbbell press must match wger");
 assert.equal(shoulder.source.name,"wger");
 assert.deepEqual(shoulder.primary,["Front delts"]);
 assert.deepEqual(shoulder.secondary,["Triceps"]);
 assert.notDeepEqual(T.profile("Shoulder Press (Dumbbell)").primary,shoulder.primary,"manual override must NOT control database-only lookup");
 assert.equal(T.databaseProfile("Shoulder Press (Machine)"),null,"machine must not silently become dumbbell");
 const quad=T.databaseProfile("Leg Extension (Machine)");
 assert.ok(quad,"leg extension normalized matching missing");
 assert.deepEqual(quad.primary,["Quads"],"rectus femoris and vastus belong to one quads group");
 assert.equal(T.databaseProfile("Leg Curl").primary[0],"Hamstrings");
 assert.equal(T.databaseProfile("Not A Real Exercise XYZ"),null);
 console.log("PASS database-only mapping, ignoring manual maps, variation safety and quad canonicalization");
})().catch(e=>{console.error(e);process.exitCode=1});
