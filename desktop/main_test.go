package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func testApp(t *testing.T) *App {
	t.Helper()
	dir := t.TempDir()
	return &App{
		dir: dir,
		cfg: Config{},
		store: Store{
			WorkoutIDs:              map[string]string{},
			Meta:                    map[string]Meta{},
			Routines:                []RoutineSummary{},
			CurrentRoutineExercises: map[string]bool{},
		},
	}
}

func addRoutineExercise(a *App, tid, title string, m Meta) {
	m.TemplateID = tid
	m.ExerciseTitle = title
	a.store.Meta[tid] = normalizeMeta(m)
	a.store.CurrentRoutineExercises[tid] = true
	a.store.Routines = []RoutineSummary{{ID: "r1", Title: "Current", Exercises: []RoutineExercise{{TemplateID: tid, ExerciseTitle: title, SetCount: 3, RepTargets: "10"}}}}
}

func TestRecForEmptyAndUnweightedHistoryNeverPanics(t *testing.T) {
	a := testApp(t)
	m := Meta{TemplateID: "abd", ExerciseTitle: "Hip Abduction", MajorMuscles: []string{"Glute med/min"}, MaxReps: 20, MinReps: 10, TargetSets: 3, IncrementKg: 2.5, Role: "accessory", Stimulus: "shortened"}
	addRoutineExercise(a, "abd", "Hip Abduction", m)

	if got := a.recFor(m); got.Status != "insufficient" {
		t.Fatalf("empty history status=%q", got.Status)
	}

	// Two sessions with reps but deliberately no weight. This used to be a crash class.
	a.store.Sets = []SetRow{
		{WorkoutID: "w2", WorkoutTitle: "Lower", PerformedAt: "2026-08-15T18:00:00Z", TemplateID: "abd", ExerciseTitle: "Hip Abduction", Reps: 20, HasReps: true},
		{WorkoutID: "w1", WorkoutTitle: "Lower", PerformedAt: "2026-08-10T18:00:00Z", TemplateID: "abd", ExerciseTitle: "Hip Abduction", Reps: 18, HasReps: true},
	}
	got := a.recFor(m)
	if got.Status == "progress" {
		t.Fatalf("unweighted history should not auto-progress: %+v", got)
	}
}

func TestRecForProgression(t *testing.T) {
	a := testApp(t)
	m := Meta{TemplateID: "ht", ExerciseTitle: "Hip Thrust", MajorMuscles: []string{"Glute max"}, MaxReps: 10, MinReps: 6, TargetSets: 3, IncrementKg: 5, Role: "primary", Stimulus: "shortened"}
	addRoutineExercise(a, "ht", "Hip Thrust", m)
	for i := 0; i < 3; i++ {
		a.store.Sets = append(a.store.Sets, SetRow{WorkoutID: "new", WorkoutTitle: "Lower", PerformedAt: "2026-08-15T18:00:00Z", TemplateID: "ht", ExerciseTitle: "Hip Thrust", WeightKg: 100, HasWeight: true, Reps: 10, HasReps: true})
	}
	for i := 0; i < 3; i++ {
		a.store.Sets = append(a.store.Sets, SetRow{WorkoutID: "old", WorkoutTitle: "Lower", PerformedAt: "2026-08-10T18:00:00Z", TemplateID: "ht", ExerciseTitle: "Hip Thrust", WeightKg: 100, HasWeight: true, Reps: 9, HasReps: true})
	}
	got := a.recFor(m)
	if got.Status != "progress" {
		t.Fatalf("expected progress, got %+v", got)
	}
}

func TestAnalysisConcurrentRequests(t *testing.T) {
	a := testApp(t)
	m := Meta{TemplateID: "ht", ExerciseTitle: "Hip Thrust", MajorMuscles: []string{"Glute max"}, MaxReps: 10, MinReps: 6, TargetSets: 3, IncrementKg: 5, Role: "primary", Stimulus: "shortened"}
	addRoutineExercise(a, "ht", "Hip Thrust", m)
	for n := 0; n < 20; n++ {
		wid := "w" + time.Now().Add(time.Duration(n)*time.Nanosecond).Format("150405.000000000")
		date := time.Now().Add(-time.Duration(n) * 24 * time.Hour).UTC().Format(time.RFC3339)
		a.store.Sets = append(a.store.Sets, SetRow{WorkoutID: wid, WorkoutTitle: "Lower", PerformedAt: date, TemplateID: "ht", ExerciseTitle: "Hip Thrust", WeightKg: 100, HasWeight: true, Reps: 8 + n%3, HasReps: true})
	}
	s := httptest.NewServer(a.handler())
	defer s.Close()
	var wg sync.WaitGroup
	errs := make(chan error, 200)
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := http.Get(s.URL + "/api/analysis")
			if err != nil {
				errs <- err
				return
			}
			defer r.Body.Close()
			if r.StatusCode != 200 {
				errs <- &statusError{r.StatusCode}
				return
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
}

type statusError struct{ code int }

func (e *statusError) Error() string { return http.StatusText(e.code) }

func TestImportExportResetLifecycle(t *testing.T) {
	a := testApp(t)
	m := Meta{TemplateID: "rdl", ExerciseTitle: "RDL", MajorMuscles: []string{"Hamstrings", "Glute max"}, SupportingMuscles: []string{"Erectors"}, MaxReps: 10, MinReps: 6, TargetSets: 3, IncrementKg: 2.5, Role: "primary", Stimulus: "lengthened"}
	addRoutineExercise(a, "rdl", "RDL", m)
	a.cfg.HevyAPIKey = "secret-hevy"
	a.store.WorkoutIDs["w1"] = "Lower"
	a.store.Sets = []SetRow{{WorkoutID: "w1", TemplateID: "rdl", ExerciseTitle: "RDL", PerformedAt: "2026-08-15T18:00:00Z", WeightKg: 50, HasWeight: true, Reps: 10, HasReps: true}}
	if err := a.saveStore(); err != nil {
		t.Fatal(err)
	}
	if err := a.saveConfig(); err != nil {
		t.Fatal(err)
	}

	// Export must not include secrets.
	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/export-metadata", nil)
	a.handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("export %d", rr.Code)
	}
	if bytes.Contains(rr.Body.Bytes(), []byte("secret-hevy")) {
		t.Fatal("mapping export leaked API key")
	}

	// Data reset retains credentials.
	body, _ := json.Marshal(map[string]any{"mode": "data"})
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/reset", bytes.NewReader(body))
	a.handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("data reset %d: %s", rr.Code, rr.Body.String())
	}
	if a.cfg.HevyAPIKey != "secret-hevy" {
		t.Fatal("data reset cleared credentials")
	}
	if len(a.store.Sets) != 0 || len(a.store.WorkoutIDs) != 0 {
		t.Fatal("data reset did not clear data")
	}

	// Full reset clears everything.
	body, _ = json.Marshal(map[string]any{"mode": "everything"})
	rr = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/reset", bytes.NewReader(body))
	a.handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("full reset %d: %s", rr.Code, rr.Body.String())
	}
	if a.cfg.HevyAPIKey != "" {
		t.Fatal("full reset did not clear credentials")
	}
	if _, err := os.Stat(filepath.Join(a.dir, "config.json")); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeMetaMultipleMajorContributors(t *testing.T) {
	m := normalizeMeta(Meta{MajorMuscles: []string{"Hamstrings", "Glute max", "Hamstrings"}, SupportingMuscles: []string{"Glute max", "Erectors"}})
	if len(m.MajorMuscles) != 2 || len(m.SupportingMuscles) != 1 || m.SupportingMuscles[0] != "Erectors" {
		t.Fatalf("bad normalization: %+v", m)
	}
}

func TestEvidenceUsesConversationContextForFollowups(t *testing.T) {
	a := testApp(t)
	lean := Meta{TemplateID: "lean", ExerciseTitle: "Hip Abduction Lean Forward", MajorMuscles: []string{"Glute med/min"}, MaxReps: 20, MinReps: 10, TargetSets: 3, IncrementKg: 2.5, Role: "accessory", Stimulus: "shortened", Notes: "lean-forward execution variant"}
	recline := Meta{TemplateID: "recline", ExerciseTitle: "Hip Abduction Reclined", MajorMuscles: []string{"Glute med/min"}, MaxReps: 20, MinReps: 10, TargetSets: 3, IncrementKg: 2.5, Role: "accessory", Stimulus: "shortened", Notes: "reclined execution variant"}
	a.store.Meta["lean"] = normalizeMeta(lean)
	a.store.Meta["recline"] = normalizeMeta(recline)
	a.store.CurrentRoutineExercises["lean"] = true
	a.store.CurrentRoutineExercises["recline"] = true
	a.store.Routines = []RoutineSummary{{ID: "r1", Title: "Lower A", Exercises: []RoutineExercise{{TemplateID: "lean", ExerciseTitle: lean.ExerciseTitle}, {TemplateID: "recline", ExerciseTitle: recline.ExerciseTitle}}}}
	history := []ChatTurn{{Role: "user", Text: "How is my hip abduction lean forward progressing?"}, {Role: "assistant", Text: "Prior answer"}}
	ev := a.evidence("what about the reclined one?", history)
	focus, _ := ev["resolvedExerciseFocus"].([]string)
	found := false
	for _, x := range focus {
		if x == recline.ExerciseTitle {
			found = true
		}
	}
	if !found {
		t.Fatalf("follow-up did not retrieve reclined variant: %#v", focus)
	}
}

func TestEvidenceCanResolveRoutineName(t *testing.T) {
	a := testApp(t)
	m := Meta{TemplateID: "ht", ExerciseTitle: "Hip Thrust", MajorMuscles: []string{"Glute max"}, MaxReps: 10, MinReps: 6, TargetSets: 3, IncrementKg: 5, Role: "primary", Stimulus: "shortened"}
	a.store.Meta["ht"] = normalizeMeta(m)
	a.store.CurrentRoutineExercises["ht"] = true
	a.store.Routines = []RoutineSummary{{ID: "r1", Title: "Lower A", Exercises: []RoutineExercise{{TemplateID: "ht", ExerciseTitle: "Hip Thrust"}}}}
	ev := a.evidence("How does Lower A look?", nil)
	focus, _ := ev["resolvedExerciseFocus"].([]string)
	if len(focus) != 1 || focus[0] != "Hip Thrust" {
		t.Fatalf("routine name resolution failed: %#v", focus)
	}
}

func TestImportHevyCSVWithoutAPIKey(t *testing.T) {
	a := testApp(t)
	csvText := `title,start_time,end_time,description,exercise_title,superset_id,exercise_notes,set_index,set_type,weight_lbs,weight_kg,reps,distance_km,duration_seconds,rpe
Lower A,2026-08-15T18:00:00Z,2026-08-15T19:00:00Z,,Hip Thrust,,,0,normal,,100,10,,,8
Lower A,2026-08-15T18:00:00Z,2026-08-15T19:00:00Z,,Hip Thrust,,,1,normal,,100,10,,,8.5
Lower A,2026-08-15T18:00:00Z,2026-08-15T19:00:00Z,,Hip Thrust,,,2,warmup,,50,10,,,
Lower B,2026-08-13T18:00:00Z,2026-08-13T19:00:00Z,,Leg Extension,,,0,normal,110,,12,,,9
`
	workouts, sets, routines, err := a.importHevyCSV(strings.NewReader(csvText))
	if err != nil {
		t.Fatal(err)
	}
	if workouts != 2 || sets != 3 || routines != 2 {
		t.Fatalf("got workouts=%d sets=%d routines=%d", workouts, sets, routines)
	}
	if a.cfg.HevyAPIKey != "" {
		t.Fatal("CSV import unexpectedly required or set an API key")
	}
	if a.store.ProgramSource != "csv-inferred" {
		t.Fatalf("bad source: %q", a.store.ProgramSource)
	}
	if !a.store.CurrentRoutineExercises[csvExerciseID("Hip Thrust")] {
		t.Fatal("hip thrust missing from inferred program")
	}
	var converted float64
	for _, s := range a.store.Sets {
		if s.ExerciseTitle == "Leg Extension" {
			converted = s.WeightKg
		}
	}
	if converted < 49.8 || converted > 50.0 {
		t.Fatalf("lbs conversion wrong: %v", converted)
	}
}

func TestImportHevyCSVRejectsWrongFile(t *testing.T) {
	a := testApp(t)
	_, _, _, err := a.importHevyCSV(strings.NewReader("foo,bar\n1,2\n"))
	if err == nil {
		t.Fatal("expected non-Hevy CSV to be rejected")
	}
}
