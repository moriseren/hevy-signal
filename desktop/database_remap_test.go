package main
import(
 "bytes"
 "encoding/json"
 "net/http/httptest"
 "os"
 "path/filepath"
 "testing"
 "strings"
)
type dbSuggestion struct {
 TemplateID string `json:"template_id"`
 ExerciseTitle string `json:"exercise_title"`
 Matched string `json:"matched"`
 Source string `json:"source"`
 Confidence string `json:"confidence"`
 Primary []string `json:"primary"`
 Secondary []string `json:"secondary"`
}
func remapReq(t *testing.T,a *App,loaded bool,s []dbSuggestion)(int,map[string]any){
 t.Helper()
 request,_:=json.Marshal(map[string]any{"catalog_loaded":loaded,"suggestions":s})
 r:=httptest.NewRequest("POST","/api/database-remap",bytes.NewReader(request))
 w:=httptest.NewRecorder()
 a.handler().ServeHTTP(w,r)
 result:=map[string]any{}
 if err:=json.Unmarshal(w.Body.Bytes(),&result);err!=nil{t.Fatal(err)}
 return w.Code,result
}
func TestFullDatabaseRemapOverwritesManualAndPreservesProgramming(t *testing.T){
 a:=folderTestApp(t)
 a.store.Meta["manual"]=Meta{TemplateID:"manual",ExerciseTitle:"Shoulder Press (Dumbbell)",
  MajorMuscles:[]string{"Biceps"},SupportingMuscles:[]string{"Biceps"},AutoTagged:false,
  Notes:"MY PROGRAMMED 3x12",Role:"primary",MinReps:8,MaxReps:12,TargetSets:3,IncrementKg:1.25}
 a.store.Meta["quads"]=Meta{TemplateID:"quads",ExerciseTitle:"Leg Extension (Machine)",
  MajorMuscles:[]string{"Quads","Rectus femoris"},AutoTagged:false,Notes:"leave notes"}
 a.store.Meta["unmatched"]=Meta{TemplateID:"unmatched",ExerciseTitle:"Special Custom Lift",
  MajorMuscles:[]string{"Rear delts"},SupportingMuscles:[]string{"Biceps"},AutoTagged:false}
 // This test also covers exercises from inactive program folders; it is NOT
 // restricted to CurrentRoutineExercises.
 if err:=a.saveStore();err!=nil{t.Fatal(err)}
 proposals:=[]dbSuggestion{
  {TemplateID:"manual",ExerciseTitle:"Shoulder Press (Dumbbell)",Matched:"Shoulder Press Dumbbell",
   Source:"wger",Confidence:"high",Primary:[]string{"Front delts"},Secondary:[]string{"Triceps"}},
  {TemplateID:"quads",ExerciseTitle:"Leg Extension (Machine)",Matched:"Leg Extension",
   Source:"wger",Confidence:"high",Primary:[]string{"Quads","Quads — rectus femoris"},Secondary:[]string{"Quads — vasti"}},
 }
 code,res:=remapReq(t,a,true,proposals)
 if code!=200{t.Fatalf("failed remap: %d %+v",code,res)}
 if res["matched"]!=float64(2)||res["allPreviousMappingsRechecked"]!=true {t.Fatalf("didn't cover all saved exercises: %+v",res)}
 m:=a.store.Meta["manual"]
 if m.MajorMuscles[0]!="Front delts"||strings.Contains(strings.Join(m.SupportingMuscles,","),"Biceps")||
 m.AnatomyConfidence!="high"||!strings.HasPrefix(m.AnatomySource,"wger")||m.AnatomyNeedsReview {t.Fatalf("manual map not replaced by wger: %+v",m)}
 if m.Notes!="MY PROGRAMMED 3x12"||m.MaxReps!=12||m.IncrementKg!=1.25 {t.Fatalf("unrelated settings changed: %+v",m)}
 quad:=a.store.Meta["quads"]
 if len(quad.MajorMuscles)!=1||quad.MajorMuscles[0]!="Quads"||len(quad.SupportingMuscles)>0 {t.Fatalf("quadriceps duplicate groups: %+v",quad)}
 if !a.store.Meta["unmatched"].AnatomyNeedsReview||a.store.Meta["unmatched"].MajorMuscles[0]!="Unmapped" {t.Fatal("unmatched old manual mapping misrepresented as verified")}
 if !a.store.Meta["hip-thrust"].AnatomyNeedsReview{t.Fatal("excluded inactive program exercise from full remap")}
 raw,err:=os.ReadFile(filepath.Join(a.dir,"data-before-v3.7-database-remap.json"))
 if err!=nil{t.Fatal("backup missing",err)}
 if !bytes.Contains(raw,[]byte("MY PROGRAMMED 3x12")){t.Fatal("backup did not preserve previous manual map")}
}
func TestDatabaseUnavailableKeepsExistingMappings(t *testing.T){
 a:=folderTestApp(t)
 if err:=a.saveStore();err!=nil{t.Fatal(err)}
 prior:=a.store.Meta["hip-thrust"]
 code,_:=remapReq(t,a,false,nil)
 if code!=409||a.store.Meta["hip-thrust"].AnatomyNeedsReview!=prior.AnatomyNeedsReview {t.Fatal("database outage replaced existing map")}
 if _,err:=os.Stat(filepath.Join(a.dir,"data-before-v3.7-database-remap.json"));!os.IsNotExist(err){t.Fatal("backup/write made without catalog")}
}
func TestMissingReferenceDoesNotCountTowardHypertrophyVolume(t *testing.T){
 a:=folderTestApp(t)
 m:=a.store.Meta["hip-thrust"];m.AnatomyNeedsReview=true
 a.store.Meta["hip-thrust"]=m
 counts,_:=a.currentRoutineStimulus(14)
 if len(counts)!=0 {t.Fatalf("unverified anatomy included in volume: %+v",counts)}
}

func TestPublicWgerProxyRejectsBadPaginationBeforeOutboundRequests(t *testing.T){
 a:=folderTestApp(t)
 for _,query:=range []string{"?offset=-1","?offset=999999","?limit=0","?limit=500","?offset=not-number"}{
  req:=httptest.NewRequest("GET","/api/reference/wger/exerciseinfo"+query,nil)
  res:=httptest.NewRecorder()
  a.handler().ServeHTTP(res,req)
  if res.Code!=400{t.Fatalf("bad catalog page should not trigger network: %q -> %d",query,res.Code)}
 }
}

func TestWgerNormalizedMappingsSurviveStartupHeuristics(t *testing.T){
 a:=folderTestApp(t)
 a.store.Meta["wger-press"]=Meta{TemplateID:"wger-press",ExerciseTitle:"Shoulder Press (Dumbbell)",
  MajorMuscles:[]string{"Front delts"},SupportingMuscles:[]string{"Triceps"},
  AutoTagged:true,AnatomySource:"wger + biomechanical normalization",AnatomyConfidence:"high"}
 before:=a.store.Meta["wger-press"]
 a.repairAutomaticMappings()
 after:=a.store.Meta["wger-press"]
 if after.AnatomySource!=before.AnatomySource||
  strings.Join(after.MajorMuscles,",")!=strings.Join(before.MajorMuscles,",")||
  strings.Join(after.SupportingMuscles,",")!=strings.Join(before.SupportingMuscles,","){
  t.Fatalf("wger mapping was overwritten by legacy startup heuristic: %+v",after)
 }
}
