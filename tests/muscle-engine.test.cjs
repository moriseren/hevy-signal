
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

const extraCases={
 'Archer Pull':'Rear delts',
 'Serratus Press':'Serratus anterior',
 'Scapular Pulldown':'Lower traps',
 'Suitcase Front Carry (Dumbbell)':'Obliques',
 'Farmers Walk':'Upper traps',
 'Walking':'Cardio / walking',
 'Lat Pulldown - Close Grip (Cable)':'Lats',
 'Sideways B-Stance Cable RDL':'Hamstrings',
 'Recline Hip Abduction (Machine)':'Glute med/min',
 'Romanian Deadlift (Dumbbell) B-Stance':'Hamstrings',
 'Standing Cable Glute Kickbacks':'Glute max',
 'Glute Kickback (Machine)':'Glute max',
 'Triceps Kickback (Dumbbell)':'Triceps',
 'Arnold Press (Dumbbell)':'Front delts',
 'Pec Deck Fly (Machine)':'Chest',
 'Upright Row (Barbell)':'Side delts',
 'Back Extension (Weighted)':'Spinal erectors',
 'Hip Adduction (Machine)':'Adductors',
 'Hip Abduction (Machine)':'Glute med/min',
 'Calf Press (Machine)':'Calves',
 'Cable Woodchop':'Obliques'
};
for(const [ex,muscle] of Object.entries(extraCases)) {
 assert.ok(T.profile(ex).primary.includes(muscle),ex+' should map to '+muscle);
}
assert.equal(T.count([{exercise:'Walking'}])['Cardio / walking'],undefined);
assert.equal(T.profile('Mystery movement 999').confidence,'unmapped');

console.log('Hevy Muscle Engine unit tests passed');
