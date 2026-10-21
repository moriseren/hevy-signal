package main
import (
 "strings"
 "testing"
 "time"
)
func TestPressAutoCorrectAndManualWarning(t *testing.T){
 major,second:=sanitizeAutomaticPress("Chest Press (Machine)",[]string{"Chest"},[]string{"Biceps","Front delts"})
 if !hasMuscle(second,"Triceps")||containsBiceps(second)||!hasMuscle(major,"Chest"){t.Fatalf("chest press anatomy incorrect: %v + %v",major,second)}
 shoulder,sh2:=sanitizeAutomaticPress("Shoulder Press (Dumbbell)",[]string{"Front delts"},[]string{"Biceps"})
 if containsBiceps(sh2)||!hasMuscle(sh2,"Triceps")||!hasMuscle(shoulder,"Front delts"){t.Fatalf("shoulder press incorrect: %v + %v",shoulder,sh2)}
 a:=&App{store:Store{Meta:map[string]Meta{
  "manual":{TemplateID:"manual",ExerciseTitle:"Chest Press (Machine)",MajorMuscles:[]string{"Chest"},SupportingMuscles:[]string{"Biceps"},AutoTagged:false,Movement:"Push",Notes:"manual edit"},
  "auto":{TemplateID:"auto",ExerciseTitle:"Shoulder Press (Dumbbell)",MajorMuscles:[]string{"Front delts"},SupportingMuscles:[]string{"Biceps"},AutoTagged:true},
 }}}
 if changed:=a.repairAutomaticMappings();changed!=1{t.Fatalf("expected 1 auto correction got %d",changed)}
 if !hasMuscle(a.store.Meta["auto"].SupportingMuscles,"Triceps")||containsBiceps(a.store.Meta["auto"].SupportingMuscles){t.Fatal("automatic shoulder press not fixed")}
 if !containsBiceps(a.store.Meta["manual"].SupportingMuscles){t.Fatal("manual edit overwritten")}
 if rec:=a.recFor(a.store.Meta["manual"]);rec.Status!="review"{t.Fatalf("manual erroneous pressing muscle must request review: %+v",rec)}
}
func TestCanonicalQuadricepsNotAdditive(t *testing.T){
 mus:=map[string]float64{}
 stim:=map[string]map[string]float64{}
 m:=Meta{ExerciseTitle:"Leg Extension (Machine)",MajorMuscles:[]string{"Quads","Rectus femoris","Quads — vasti"},SupportingMuscles:[]string{"Rectus femoris","Vastus medialis"}}
 addMuscleExposure(m,mus,stim)
 if mus["Quads"]!=1||len(mus)!=1||stim["Quads"]["mixed"]!=1{t.Fatalf("rectus double counted: %+v %+v",mus,stim)}
 if label:=majorLabel(m);label!="Quads"{t.Fatalf("failed to canonicalize quad label %q",label)}
}
func TestWalkingAndCarryingNotRepetitionBased(t *testing.T){
 for _,title:=range []string{"Walking","Farmers Walk","Suitcase Carry (Dumbbell)","Suitcase Front Carry (Dumbbell)"}{
  m:=Meta{TemplateID:title,ExerciseTitle:title,MajorMuscles:[]string{"Abs"},MinReps:6,MaxReps:12,TargetSets:1,IncrementKg:5}
  a:=&App{store:Store{Meta:map[string]Meta{title:m},CurrentRoutineExercises:map[string]bool{title:true},Sets:[]SetRow{
    {TemplateID:title,WorkoutID:"1",WorkoutTitle:"Cardio",PerformedAt:time.Now().Format(time.RFC3339),HasReps:true,Reps:12,HasWeight:true,WeightKg:10},
    {TemplateID:title,WorkoutID:"2",WorkoutTitle:"Cardio",PerformedAt:time.Now().Add(-24*time.Hour).Format(time.RFC3339),HasReps:true,Reps:12,HasWeight:true,WeightKg:10},
  }}}
  rec:=a.recFor(m)
  if rec.Status!="tracking"||strings.Contains(strings.ToLower(rec.Rationale),"ready to progress"){t.Fatalf("%s got rep-based signal: %+v",title,rec)}
  mus,_:=a.currentRoutineStimulus(14)
  if len(mus)!=0{t.Fatalf("%s counted as hypertrophy sets: %v",title,mus)}
  brief:=a.chatGPTBrief()
  if strings.Contains(brief,"- Progression target:")||strings.Contains(brief,"; e1RM ")||strings.Contains(brief,"best set;"){t.Fatalf("%s still has rep target or e1RM in brief: %s",title,brief)}
 }
}
func TestIncomparableLoadVariantsAreNotProgressed(t *testing.T){
 m:=Meta{TemplateID:"abduction",ExerciseTitle:"Cable Abduction",MajorMuscles:[]string{"Glute med/min"},MaxReps:15,TargetSets:1,IncrementKg:2.5}
 now:=time.Now()
 a:=&App{store:Store{Sets:[]SetRow{
  {TemplateID:"abduction",WorkoutID:"recent",PerformedAt:now.Format(time.RFC3339),WeightKg:9.1,HasWeight:true,Reps:15,HasReps:true},
  {TemplateID:"abduction",WorkoutID:"older",PerformedAt:now.Add(-24*time.Hour).Format(time.RFC3339),WeightKg:40.8,HasWeight:true,Reps:15,HasReps:true},
 }}}
 if rec:=a.recFor(m);rec.Status!="review"||!strings.Contains(strings.ToLower(rec.Rationale),"variant"){t.Fatalf("expected setup review, got %+v",rec)}
 if !suspectLoadVariant(a.sessionsFor(m.TemplateID,12)){t.Fatal("missing historical load anomaly")}
 m.ExerciseTitle="Standing Cable Glute Kickbacks"
 if rec:=a.recFor(m);rec.Status!="review"{t.Fatalf("kickback variant not flagged: %+v",rec)}
}
