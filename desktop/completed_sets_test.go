package main

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestCompletedSetsExcludePlaceholdersFromAllAnalysis(t *testing.T) {
	a := testApp(t)
	m := Meta{MajorMuscles: []string{"Glute max"}, SupportingMuscles: []string{"Hamstrings"}, Stimulus: "shortened", MaxReps: 10, MinReps: 6, TargetSets: 3, IncrementKg: 5}
	addRoutineExercise(a, "ht", "Hip Thrust", m)
	now := time.Now()
	// Include old cached rows, including a placeholder-only recent workout.
	a.store.Sets = []SetRow{
		{TemplateID: "ht", WorkoutID: "empty", PerformedAt: now.Format(time.RFC3339), WeightKg: 999, HasWeight: true},
		{TemplateID: "ht", WorkoutID: "valid", PerformedAt: now.Add(-time.Hour).Format(time.RFC3339), Reps: 10, HasReps: true, WeightKg: 100, HasWeight: true},
		{TemplateID: "ht", WorkoutID: "valid", PerformedAt: now.Add(-time.Hour).Format(time.RFC3339), Reps: 0, HasReps: true, WeightKg: 100, HasWeight: true, RPE: 10, HasRPE: true},
		{TemplateID: "ht", WorkoutID: "valid", PerformedAt: now.Add(-time.Hour).Format(time.RFC3339), Reps: -1, HasReps: true, WeightKg: 100, HasWeight: true},
		{TemplateID: "ht", WorkoutID: "valid", PerformedAt: now.Add(-time.Hour).Format(time.RFC3339), Reps: 10, HasReps: false},
		{TemplateID: "ht", WorkoutID: "valid", PerformedAt: now.Add(-time.Hour).Format(time.RFC3339), Reps: 1, HasReps: true}, // valid bodyweight/unweighted set
		{TemplateID: "ht", WorkoutID: "older", PerformedAt: now.Add(-24*time.Hour).Format(time.RFC3339), Reps: 10, HasReps: true, WeightKg: 100, HasWeight: true},
	}
	for _, volume := range []func(int) ([]map[string]any, map[string]map[string]float64){a.stimulus, a.currentRoutineStimulus} {
		sets, stim := volume(14)
		if len(sets) != 2 || sets[0]["muscle"] != "Glute max" || sets[0]["sets"] != float64(3) || sets[1]["sets"] != 1.5 || stim["Glute max"]["shortened"] != 3 {
			t.Fatalf("incorrect completed volume: %+v, %+v", sets, stim)
		}
	}
	sessions := a.sessionsFor("ht", 6)
	if len(sessions) != 2 || len(sessions[0]) != 2 || sessions[0][0].WorkoutID != "valid" {
		t.Fatalf("placeholder-only workout or blank sets in history: %+v", sessions)
	}
	s := summary(sessions[0])
	if s.Volume != 1000 || s.AvgRPE != nil || s.Sets != 2 {
		t.Fatalf("invalid set affected performance summary: %+v", s)
	}
	if rec := a.recFor(a.store.Meta["ht"]); rec.Status != "maintain" || len(rec.History) != 2 {
		t.Fatalf("placeholder affected progression: %+v", rec)
	}
	if len(a.store.Sets) != 7 || a.store.Routines[0].Exercises[0].SetCount != 3 {
		t.Fatal("analysis changed raw history or saved routine")
	}
	res := httptest.NewRecorder()
	a.handler().ServeHTTP(res, httptest.NewRequest("GET", "/api/analysis", nil))
	var result struct { Counts map[string]int `json:"counts"` }
	if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil || result.Counts["sets"] != 3 {
		t.Fatalf("dashboard count includes placeholders: %s (%v)", res.Body.String(), err)
	}
}

func TestHevyAPICompletedSetsExcludeWarmupsAndMissingReps(t *testing.T) {
	a := testApp(t)
	var workout map[string]any
	if err := json.Unmarshal([]byte(`{"id":"w","title":"Lower","start_time":"`+time.Now().Format(time.RFC3339)+`","exercises":[{"title":"Hip Thrust","exercise_template_id":"ht","sets":[{"type":"normal","weight_kg":100,"reps":10},{"type":"normal","weight_kg":100,"reps":0},{"type":"normal","weight_kg":100,"reps":null},{"type":"normal","weight_kg":100},{"type":"warmup","weight_kg":50,"reps":10},{"type":"drop_set","reps":4},{"type":"failure","reps":1}]}]}`), &workout); err != nil {
		t.Fatal(err)
	}
	a.ingestLocked(workout)
	groups := a.sessionsFor("ht", 6)
	if len(groups) != 1 || len(groups[0]) != 3 || completedRepSetCount(a.store.Sets) != 3 || len(a.store.Sets) != 6 {
		t.Fatalf("API counted invalid sets or lost valid work: %+v", a.store.Sets)
	}
}

func TestCSVCompletedSetsAndInferredRoutine(t *testing.T) {
	a := testApp(t)
	stamp := time.Now().Format(time.RFC3339)
	data := "title,start_time,exercise_title,set_index,set_type,weight_kg,reps,rpe\n" +
		"Lower,"+stamp+",Hip Thrust,0,normal,100,10,8\n" +
		"Lower,"+stamp+",Hip Thrust,1,normal,100,0,10\n" +
		"Lower,"+stamp+",Hip Thrust,2,normal,100,,\n" +
		"Lower,"+stamp+",Hip Thrust,3,warmup,50,10,\n" +
		"Lower,"+stamp+",Hip Thrust,4,normal,,1,\n" +
		"Lower,"+stamp+",Leg Extension,0,normal,60,,\n"
	workouts, sets, routines, err := a.importHevyCSV(strings.NewReader(data))
	if err != nil || workouts != 1 || sets != 2 || routines != 1 {
		t.Fatalf("bad CSV completed count: %d %d %d %v", workouts, sets, routines, err)
	}
	exercises := a.store.Routines[0].Exercises
	if len(exercises) != 1 || exercises[0].SetCount != 2 || exercises[0].RepTargets != "10/1" {
		t.Fatalf("inferred routine includes placeholders: %+v", exercises)
	}
	if len(a.store.Sets) != 5 {
		t.Fatal("raw non-warmup history was discarded")
	}
}
