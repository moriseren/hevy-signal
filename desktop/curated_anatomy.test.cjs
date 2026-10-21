const assert=require('node:assert/strict');
const C=require('./static/curated-anatomy.js');
const expected={
 'Cable Abduction':['Glute med/min'],
 'Chest Press (Machine)':['Chest'],
 'Farmers Walk':['Upper traps','Forearms'],
 'Hip Adduction (Machine)':['Adductors'],
 'Hip Thrust (Machine)':['Glute max'],
 'Lateral Raise (Cable)':['Side delts'],
 'Leg Extension (Machine)':['Quads'],
 'Leg Press (Machine)':['Quads','Glute max'],
 'Rear Delt Reverse Fly (Cable)':['Rear delts'],
 'Romanian Deadlift (Dumbbell) B-Stance':['Hamstrings','Glute max'],
 'Scapular Pulldown':['Lower traps','Lats'],
 'Seated Leg Curl (Machine)':['Hamstrings'],
 'Seated Row (Machine)':['Lats','Mid traps/rhomboids'],
 'Serratus Press':['Serratus anterior'],
 'Shoulder Press (Dumbbell)':['Front delts','Side delts'],
 'Single Leg Press Horizontal (Machine)':['Quads','Glute max'],
 'Standing Cable Glute Kickbacks':['Glute max'],
 'Suitcase Front Carry (Dumbbell)':['Abs','Obliques'],
 'Suitcase Carry (Dumbbell)':['Obliques','Forearms'],
 'Treadmill':['Cardio / walking']
};
for(const [name,parts] of Object.entries(expected)){
 const p=C.lookup(name);
 assert.ok(p,'Unmapped known Hevy name: '+name);
 assert.deepEqual(p.primary,parts,name);
 assert.equal(p.confidence,'curated');
 assert.match(p.source.name,/crosswalk/i,'Must NOT falsely label wger');
 assert.ok(p.note.includes('not a verified wger'));
}
for(const name of ['Hip Abduction (Machine)','Hip Adduction (Machine)']){
 const p=C.lookup(name);
 assert.ok(p);
 if(/Adduction/i.test(name)){assert.deepEqual(p.primary,['Adductors']);assert.ok(!p.primary.includes('Glute max'))}
}
assert.equal(C.lookup('Treadmill').mode,'cardio');
for(const name of ['Farmers Walk','Suitcase Carry (Dumbbell)','Suitcase Front Carry (Dumbbell)'])
 assert.equal(C.lookup(name).mode,'carry',name);
for(const name of ['Mystery Exercise 1234','Seated Leg Press Cable Custom'])
 assert.equal(C.lookup(name),null,'Do not make up anatomy from partial overlap');
assert.ok(!C.lookup('Leg Press (Machine)').primary.includes('Hamstrings'));
assert.ok(!C.lookup('Leg Press (Machine)').primary.includes('Calves'));
assert.ok(!C.lookup('Shoulder Press (Dumbbell)').secondary.includes('Biceps'));
console.log('PASS '+Object.keys(expected).length+' Hevy exercise names, source fidelity, and carry/cardio classification');
