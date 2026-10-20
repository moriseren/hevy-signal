
const assert=require('node:assert/strict');
const T=require('../docs/muscle-engine.js');
assert.equal(T.profile('Cable Hip Abduction').primary[0],'Glute med/min');
assert.equal(T.profile('Seated Leg Curl').primary[0],'Hamstrings');
assert.ok(T.profile('Bulgarian Split Squat').primary.includes('Glute max'));
assert.equal(T.profile('Mystery Exercise').confidence,'unmapped');
assert.equal(T.score('Leg Extension (Machine)','Leg Extensions'),1);
assert.equal(T.score('Leg Curl','Leg Extension'),0);
const ex=T.convertWger({id:1,translations:[{language:2,name:'Cable Row'},{language:1,name:'Rudern'}],muscles:[{name_en:'Latissimus dorsi'}],muscles_secondary:[{name_en:'Biceps brachii'}]});
assert.deepEqual(ex.primary,['Latissimus dorsi']);
assert.deepEqual(ex.secondary,['Biceps brachii']);
const ed=T.convertEDB({name:'barbell bench press',targetMuscles:['pectorals'],secondaryMuscles:['triceps']});
assert.equal(ed.secondary[0],'triceps');
const counted=T.count([{exercise:'Seated Leg Curl'},{exercise:'Seated Leg Curl'}]);
assert.equal(counted.Hamstrings.primary,2);
console.log('Hevy Muscle Engine unit tests passed');
