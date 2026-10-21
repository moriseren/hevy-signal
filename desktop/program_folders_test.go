package main

import (
 "bytes"
 "encoding/json"
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "strings"
 "testing"
 "time"
)

func TestParseHevyFoldersNumericAndStringIDs(t *testing.T){
 raw:=map[string]any{"routine_folders":[]any{
  map[string]any{"id":json.Number("42"),"title":"Cut"},
  map[string]any{"id":"hypertrophy-7","title":"Hypertrophy"},
  map[string]any{"id":"hypertrophy-7","title":"Duplicate"},
 }}
 folders:=parseHevyFolders(raw)
 if len(folders)!=2||folders[0].ID!="42"||folders[0].Title!="Cut"||
 folders[1].ID!="hypertrophy-7"||folders[1].Title!="Hypertrophy" {
  t.Fatalf("incorrect folder parsing: %+v",folders)
 }
}

func folderTestApp(t *testing.T)*App{
 t.Helper()
 stamp:=time.Now().Format(time.RFC3339)
 store:=Store{
  ProgramSource:"hevy-api",
  WorkoutIDs:map[string]string{"cut-workout":"Lower","growth-workout":"Lower"},
  Meta:map[string]Meta{"hip-thrust":{TemplateID:"hip-thrust",ExerciseTitle:"Hip Thrust (Barbell)",
   MajorMuscles:[]string{"Glute max"},Role:"primary",MaxReps:12,TargetSets:2}},
  RoutineFolders:[]RoutineFolder{{ID:"a",Title:"Cut"},{ID:"b",Title:"Hypertrophy"}},
  Routines:[]RoutineSummary{
   {ID:"cut-routine",Title:"Cut Lower",FolderID:"a",Exercises:[]RoutineExercise{{TemplateID:"hip-thrust",ExerciseTitle:"Hip Thrust (Barbell)",SetCount:2}}},
   {ID:"growth-routine",Title:"Hypertrophy Lower",FolderID:"b",Exercises:[]RoutineExercise{{TemplateID:"hip-thrust",ExerciseTitle:"Hip Thrust (Barbell)",SetCount:5}}},
  },
  CurrentRoutineExercises:map[string]bool{"hip-thrust":true},
  Sets:[]SetRow{
   {WorkoutID:"cut-workout",WorkoutTitle:"Cut Lower",PerformedAt:stamp,
    RoutineID:"cut-routine",TemplateID:"hip-thrust",ExerciseTitle:"Hip Thrust (Barbell)",WeightKg:100,HasWeight:true,Reps:12,HasReps:true},
   {WorkoutID:"growth-workout",WorkoutTitle:"Hypertrophy Lower",PerformedAt:stamp,
    RoutineID:"growth-routine",TemplateID:"hip-thrust",ExerciseTitle:"Hip Thrust (Barbell)",WeightKg:180,HasWeight:true,Reps:10,HasReps:true},
   {WorkoutID:"unlinked",WorkoutTitle:"Unlinked",PerformedAt:stamp,
    TemplateID:"hip-thrust",ExerciseTitle:"Hip Thrust (Barbell)",WeightKg:220,HasWeight:true,Reps:5,HasReps:true},
  },
 }
 return &App{dir:t.TempDir(),cfg:Config{HevyAPIKey:"test-key"},store:store}
}

func requestFolder(t *testing.T,a *App,id string) (int,map[string]any) {
 t.Helper()
 data,_:=json.Marshal(map[string]any{"folder_id":id})
 req:=httptest.NewRequest("POST","/api/select-program-folder",bytes.NewReader(data))
 res:=httptest.NewRecorder()
 a.handler().ServeHTTP(res,req)
 var body map[string]any
 if err:=json.Unmarshal(res.Body.Bytes(),&body);err!=nil{t.Fatal(err,res.Body.String())}
 return res.Code,body
}

func TestSelectingProgramFolderDoesNotMergeCutAndHypertrophy(t *testing.T) {
 a:=folderTestApp(t)
 code,r:=requestFolder(t,a,"a")
 if code!=200 {t.Fatalf("select cut: %d %+v",code,r)}
 if a.store.ActiveFolderID!="a"||len(a.selectedRoutinesLocked())!=1 {t.Fatal("cut folder not isolated")}
 if a.selectedRoutinesLocked()[0].Title!="Cut Lower" {t.Fatal("wrong routine displayed")}
 sessions:=a.sessionsFor("hip-thrust",6)
 if len(sessions)!=1||sessions[0][0].WeightKg!=100 {t.Fatalf("mixed other programs' strength history: %+v",sessions)}
 sets,_:=a.currentRoutineStimulus(14)
 if len(sets)!=1||sets[0]["sets"]!=float64(1) {t.Fatalf("wrong selected-folder muscle volume: %+v",sets)}
 brief:=a.chatGPTBrief()
 if !strings.Contains(brief,"Active Hevy program:** Cut")||strings.Contains(brief,"### Hypertrophy Lower") {t.Fatal("handoff includes wrong folder")}
 persisted,err:=os.ReadFile(filepath.Join(a.dir,"data.json"))
 if err!=nil{t.Fatal(err)}
 var roundtrip Store
 if err:=json.Unmarshal(persisted,&roundtrip);err!=nil{t.Fatal(err)}
 if roundtrip.ActiveFolderID!="a"||len(roundtrip.Routines)!=2||len(roundtrip.Sets)!=3 {t.Fatal("switch failed to persist full history")}
 code,_=requestFolder(t,a,"b")
 if code!=200||a.selectedRoutinesLocked()[0].Title!="Hypertrophy Lower" {t.Fatal("hypertrophy folder not selected")}
 sessions=a.sessionsFor("hip-thrust",6)
 if len(sessions)!=1||sessions[0][0].WeightKg!=180 {t.Fatalf("growth strength data mixed with cut: %+v",sessions)}
 code,_=requestFolder(t,a,"")
 if code!=200||len(a.selectedRoutinesLocked())!=2 {t.Fatal("failed to restore all routines")}
 if len(a.sessionsFor("hip-thrust",6))!=3 {t.Fatal("switching lost saved workouts")}
}

func TestFolderSelectionRejectsUnknownOrCSV(t *testing.T){
 a:=folderTestApp(t)
 code,_:=requestFolder(t,a,"does-not-exist")
 if code!=404||a.store.ActiveFolderID!="" {t.Fatal("unknown folder accepted")}
 a.store.ProgramSource="csv-inferred"
 code,_=requestFolder(t,a,"a")
 if code!=409||a.store.ActiveFolderID!="" {t.Fatal("CSV folder selection must be blocked")}
}

func TestFolderAPIListsCorrectFolders(t *testing.T){
 a:=folderTestApp(t)
 req:=httptest.NewRequest(http.MethodGet,"/api/program-folders",nil)
 res:=httptest.NewRecorder()
 a.handler().ServeHTTP(res,req)
 if res.Code!=200 {t.Fatal(res.Code,res.Body.String())}
 var body struct {
  Folders []RoutineFolder `json:"folders"`
  CurrentRoutines int `json:"currentRoutines"`
 }
 if err:=json.Unmarshal(res.Body.Bytes(),&body);err!=nil{t.Fatal(err)}
 if len(body.Folders)!=2||body.CurrentRoutines!=2 {t.Fatalf("bad API listing: %+v",body)}
}
