package main

import (
 "reflect"
 "testing"
)

func TestRefreshAutoAndPreserveManual(t *testing.T) {
 a:=&App{store:Store{Meta:map[string]Meta{}}}
 manual:=Meta{TemplateID:"manual",ExerciseTitle:"Hip Thrust (Machine)",
  MajorMuscles:[]string{"Custom Glute Max"},SupportingMuscles:[]string{"My corrected stabilizer"},
  AutoTagged:false,Movement:"Hip extension",Notes:"My exact setup",MinReps:13,MaxReps:17,
  TargetSets:2,IncrementKg:1.25,Role:"primary",Stimulus:"lengthened"}
 auto:=Meta{TemplateID:"automatic",ExerciseTitle:"Lat Pulldown - Close Grip (Cable)",
  MajorMuscles:[]string{"Back"},SupportingMuscles:[]string{},AutoTagged:true,
  Notes:"do not change me",MinReps:6,MaxReps:12,TargetSets:4,IncrementKg:2.5,
  Role:"primary",Stimulus:"mixed"}
 legacy:=Meta{TemplateID:"legacy",ExerciseTitle:"Scapular Pulldown",
  MajorMuscles:[]string{"Lats"},SupportingMuscles:[]string{},
  AutoTagged:false,Movement:"Automatically classified (review if custom)",Notes:"hold at top"}
 unmapped:=Meta{TemplateID:"unmapped",ExerciseTitle:"Serratus Press",
  MajorMuscles:[]string{"Unmapped"},Movement:"Unmapped",Notes:"practice"}
 genuinelyUnknown:=Meta{TemplateID:"unknown",ExerciseTitle:"Uncommon Custom 938",
  MajorMuscles:[]string{"Unmapped"},Movement:"Unmapped"}
 a.store.Meta["manual"]=manual
 a.store.Meta["automatic"]=auto
 a.store.Meta["legacy"]=legacy
 a.store.Meta["unmapped"]=unmapped
 a.store.Meta["unknown"]=genuinelyUnknown

 changed:=a.repairAutomaticMappings()
 if changed!=3 {t.Fatalf("expected 3 updated mappings, got %d",changed)}
 gotManual:=a.store.Meta["manual"]
 if !reflect.DeepEqual(gotManual,manual) {t.Fatalf("manual assignment overwritten: %+v",gotManual)}
 gotAuto:=a.store.Meta["automatic"]
 if !reflect.DeepEqual(gotAuto.MajorMuscles,[]string{"Lats"}) ||
    gotAuto.MinReps!=6 || gotAuto.TargetSets!=4 || gotAuto.Notes!="do not change me" ||
    !gotAuto.AutoTagged {
   t.Fatalf("auto mapping / progression not updated properly: %+v",gotAuto)
 }
 if !a.store.Meta["legacy"].AutoTagged ||
    !reflect.DeepEqual(a.store.Meta["legacy"].MajorMuscles,[]string{"Lower traps","Lats"}) ||
    a.store.Meta["legacy"].Notes!="hold at top" {
   t.Fatalf("legacy v3.3 auto mapping was not recovered: %+v",a.store.Meta["legacy"])
 }
 if !reflect.DeepEqual(a.store.Meta["unmapped"].MajorMuscles,[]string{"Serratus anterior"}) {
   t.Fatalf("unmapped classification not repaired: %+v",a.store.Meta["unmapped"])
 }
 if !reflect.DeepEqual(a.store.Meta["unknown"],genuinelyUnknown) {
   t.Fatalf("unrecognized exercise was guessed: %+v",a.store.Meta["unknown"])
 }
 if again:=a.repairAutomaticMappings();again!=0 {
   t.Fatalf("expected idempotent re-map, got %d changed",again)
 }
}

func TestManualRulePrecedence(t *testing.T) {
 cases:=[]struct{title,major string}{
  {"Standing Cable Glute Kickbacks","Glute max"},
  {"Triceps Kickback (Dumbbell)","Triceps"},
  {"Hip Abduction (Machine)","Glute med/min"},
  {"Pec Deck Fly (Machine)","Chest"},
  {"Archer Pull","Rear delts"},
  {"Sideways B-Stance Cable RDL","Hamstrings"},
 }
 a:=&App{}
 for _,test:=range cases {
   got:=a.defaultMeta("example",test.title)
   found:=false
   for _,m:=range got.MajorMuscles {if m==test.major {found=true}}
   if !found {t.Errorf("%q mapped to %v, expected %q",test.title,got.MajorMuscles,test.major)}
 }
}
