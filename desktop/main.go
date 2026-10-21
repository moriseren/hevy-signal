package main

import (
	"bytes"
	"embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed static/*
var staticFS embed.FS

const baseURL = "https://api.hevyapp.com/v1"
const version = "3.6-folder-programs"

type Config struct {
	HevyAPIKey string `json:"hevy_api_key"`
}

type SetRow struct {
	WorkoutID     string  `json:"workout_id"`
	WorkoutTitle  string  `json:"workout_title"`
	PerformedAt   string  `json:"performed_at"`
	TemplateID    string  `json:"template_id"`
	RoutineID     string  `json:"routine_id,omitempty"`
	ExerciseTitle string  `json:"exercise_title"`
	SetIndex      int     `json:"set_index"`
	WeightKg      float64 `json:"weight_kg"`
	HasWeight     bool    `json:"has_weight"`
	Reps          int     `json:"reps"`
	HasReps       bool    `json:"has_reps"`
	RPE           float64 `json:"rpe"`
	HasRPE        bool    `json:"has_rpe"`
}

type Meta struct {
	TemplateID        string   `json:"template_id"`
	ExerciseTitle     string   `json:"exercise_title"`
	MajorMuscles      []string `json:"major_muscles"`
	SupportingMuscles []string `json:"supporting_muscles"`
	// Legacy v1.2 fields retained only so existing saved metadata migrates cleanly.
	PrimaryMuscle    string   `json:"primary_muscle,omitempty"`
	SecondaryMuscles []string `json:"secondary_muscles,omitempty"`
	Movement         string   `json:"movement"`
	Stimulus         string   `json:"stimulus"`
	FatigueCost      int      `json:"fatigue_cost"`
	Role             string   `json:"role"`
	MinReps          int      `json:"min_reps"`
	MaxReps          int      `json:"max_reps"`
	TargetSets       int      `json:"target_sets"`
	IncrementKg      float64  `json:"increment_kg"`
	Notes            string   `json:"notes"`
	AutoTagged       bool     `json:"auto_tagged"`
}

type RoutineExercise struct {
	TemplateID    string `json:"template_id"`
	ExerciseTitle string `json:"exercise_title"`
	SetCount      int    `json:"set_count"`
	RepTargets    string `json:"rep_targets"`
}

type RoutineFolder struct {
 ID string `json:"id"`
 Title string `json:"title"`
}

type RoutineSummary struct {
	ID        string            `json:"id"`
	Title     string            `json:"title"`
	FolderID  string            `json:"folder_id"`
	Exercises []RoutineExercise `json:"exercises"`
}

type Store struct {
	ProgramSource           string            `json:"program_source,omitempty"`
	WorkoutIDs              map[string]string `json:"workout_ids"`
	Sets                    []SetRow          `json:"sets"`
	Meta                    map[string]Meta   `json:"meta"`
	Routines                []RoutineSummary  `json:"routines"`
	CurrentRoutineExercises map[string]bool   `json:"current_routine_exercises"`
 ActiveFolderID string `json:"active_folder_id,omitempty"`
 RoutineFolders []RoutineFolder `json:"routine_folders,omitempty"`
}

type App struct {
	mu    sync.RWMutex
	cfg   Config
	store Store
	dir   string
}

type ChatTurn struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type MappingExport struct {
	FormatVersion int             `json:"format_version"`
	AppVersion    string          `json:"app_version"`
	ExportedAt    string          `json:"exported_at"`
	Metadata      map[string]Meta `json:"metadata"`
}

type preset struct {
	Pattern *regexp.Regexp
	Meta    Meta
}

var presets = []preset{
	{regexp.MustCompile(`(?i)archer pull`), Meta{MajorMuscles: []string{"Rear delts"}, SupportingMuscles: []string{"Mid traps/rhomboids", "External rotators"}, Movement: "Horizontal pull", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)serratus press`), Meta{MajorMuscles: []string{"Serratus anterior"}, SupportingMuscles: []string{"Chest"}, Movement: "Scapular protraction", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)scapular (pull.?down|depression)|straight.arm scapular`), Meta{MajorMuscles: []string{"Lower traps", "Lats"}, SupportingMuscles: []string{"Pectoralis minor"}, Movement: "Scapular depression", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)suitcase front carry`), Meta{MajorMuscles: []string{"Obliques", "Abs"}, SupportingMuscles: []string{"Forearms", "Front delts"}, Movement: "Loaded carry", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)suitcase (carry|walk|hold)`), Meta{MajorMuscles: []string{"Obliques", "Abs"}, SupportingMuscles: []string{"Forearms", "Quadratus lumborum"}, Movement: "Loaded carry", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)farmer.?s? (walk|carry|hold)`), Meta{MajorMuscles: []string{"Upper traps", "Forearms"}, SupportingMuscles: []string{"Abs"}, Movement: "Loaded carry", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)^(walking|walk|treadmill walk|brisk walk)(\s*\(.*\))?$`), Meta{MajorMuscles: []string{"Cardio / walking"}, SupportingMuscles: []string{}, Movement: "Cardio", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)triceps? kickback|skull.?crusher|triceps? pushdown|triceps? extension`), Meta{MajorMuscles: []string{"Triceps"}, SupportingMuscles: []string{}, Movement: "Elbow extension", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)glute kickback|cable kickback|donkey kick|hip extension`), Meta{MajorMuscles: []string{"Glute max"}, SupportingMuscles: []string{"Hamstrings"}, Movement: "Hip extension", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)abduct|clamshell|lateral band walk|monster walk`), Meta{MajorMuscles: []string{"Glute med/min"}, SupportingMuscles: []string{"Glute max"}, Movement: "Hip abduction", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)adduct|copenhagen`), Meta{MajorMuscles: []string{"Adductors"}, SupportingMuscles: []string{}, Movement: "Hip adduction", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)hip thrust|glute bridge|glute drive|frog pump`), Meta{MajorMuscles: []string{"Glute max"}, SupportingMuscles: []string{"Hamstrings"}, Movement: "Hip extension", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)leg extension|reverse nordic`), Meta{MajorMuscles: []string{"Quads"}, SupportingMuscles: []string{"Rectus femoris"}, Movement: "Knee extension", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)leg curl|hamstring curl|nordic curl|glute ham raise`), Meta{MajorMuscles: []string{"Hamstrings"}, SupportingMuscles: []string{}, Movement: "Knee flexion", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)romanian deadlift|\brdl\b|stiff.?leg|good morning|cable rdl`), Meta{MajorMuscles: []string{"Hamstrings", "Glute max"}, SupportingMuscles: []string{"Erectors"}, Movement: "Hip hinge", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)reverse hyper|cable pull through`), Meta{MajorMuscles: []string{"Glute max"}, SupportingMuscles: []string{"Hamstrings", "Erectors"}, Movement: "Hip extension", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)back extension|hyperextension|45.degree`), Meta{MajorMuscles: []string{"Glute max", "Spinal erectors"}, SupportingMuscles: []string{"Hamstrings"}, Movement: "Hip extension", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)sumo deadlift`), Meta{MajorMuscles: []string{"Glute max", "Adductors"}, SupportingMuscles: []string{"Quads", "Erectors"}, Movement: "Hip hinge", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)deadlift|rack pull|trap bar`), Meta{MajorMuscles: []string{"Glute max", "Erectors"}, SupportingMuscles: []string{"Hamstrings", "Quads"}, Movement: "Hip hinge", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)bulgarian|split squat|lunge|step.?up`), Meta{MajorMuscles: []string{"Quads", "Glute max"}, SupportingMuscles: []string{"Adductors"}, Movement: "Lunge", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)leg press|hack squat|pendulum|belt squat|front squat|back squat|goblet squat|smith squat|\bsquat\b`), Meta{MajorMuscles: []string{"Quads", "Glute max"}, SupportingMuscles: []string{"Adductors"}, Movement: "Squat/press", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)calf raise|calf press|soleus raise|toe press`), Meta{MajorMuscles: []string{"Calves"}, SupportingMuscles: []string{}, Movement: "Plantar flexion", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)face pull|reverse fly|reverse pec|rear delt`), Meta{MajorMuscles: []string{"Rear delts"}, SupportingMuscles: []string{"Mid traps/rhomboids"}, Movement: "Horizontal abduction", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)lateral raise|side raise|cable y.?raise|scaption`), Meta{MajorMuscles: []string{"Side delts"}, SupportingMuscles: []string{"Upper traps"}, Movement: "Shoulder abduction", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)front raise`), Meta{MajorMuscles: []string{"Front delts"}, SupportingMuscles: []string{}, Movement: "Shoulder flexion", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)shrug|high pull`), Meta{MajorMuscles: []string{"Upper traps"}, SupportingMuscles: []string{}, Movement: "Scapular elevation", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)upright row`), Meta{MajorMuscles: []string{"Side delts", "Upper traps"}, SupportingMuscles: []string{"Biceps"}, Movement: "Shoulder abduction", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)shoulder press|overhead press|arnold press|military press|landmine press`), Meta{MajorMuscles: []string{"Front delts"}, SupportingMuscles: []string{"Side delts", "Triceps"}, Movement: "Shoulder press", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)chest.supported row|high row|wide.grip row|upper.back row`), Meta{MajorMuscles: []string{"Mid traps/rhomboids", "Rear delts"}, SupportingMuscles: []string{"Lats", "Biceps"}, Movement: "Upper-back row", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)\brow\b`), Meta{MajorMuscles: []string{"Lats", "Mid traps/rhomboids"}, SupportingMuscles: []string{"Rear delts", "Biceps"}, Movement: "Horizontal pull", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)lat pulldown|pull.?down|pull.?up|chin.?up|pullover|lat prayer`), Meta{MajorMuscles: []string{"Lats"}, SupportingMuscles: []string{"Biceps"}, Movement: "Vertical pull", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)close.grip bench|jm press|tricep dip|bench dip`), Meta{MajorMuscles: []string{"Triceps"}, SupportingMuscles: []string{"Chest"}, Movement: "Elbow extension", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)chest dip|parallel bar dip|\bdips?\b`), Meta{MajorMuscles: []string{"Chest", "Triceps"}, SupportingMuscles: []string{"Front delts"}, Movement: "Push", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)incline (bench|chest|dumbbell|barbell|press)`), Meta{MajorMuscles: []string{"Upper chest"}, SupportingMuscles: []string{"Front delts", "Triceps"}, Movement: "Push", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)bench|chest press|push.?up|pec.?deck|chest fly|cable fly|dumbbell fly|butterfly`), Meta{MajorMuscles: []string{"Chest"}, SupportingMuscles: []string{"Front delts", "Triceps"}, Movement: "Push", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)wrist curl|wrist extension|reverse curl|wrist roller`), Meta{MajorMuscles: []string{"Forearms"}, SupportingMuscles: []string{}, Movement: "Forearm", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)hammer curl|preacher curl|bicep|barbell curl|dumbbell curl|cable curl|incline curl|concentration curl|\bcurl\b`), Meta{MajorMuscles: []string{"Biceps"}, SupportingMuscles: []string{"Brachialis/brachioradialis"}, Movement: "Elbow flexion", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)pallof|woodchop|wood chop|russian twist|side plank|oblique`), Meta{MajorMuscles: []string{"Obliques"}, SupportingMuscles: []string{"Abs"}, Movement: "Core", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)crunch|sit.?up|leg raise|knee raise|ab wheel|\bplank\b|dead bug|hollow hold|abdominal`), Meta{MajorMuscles: []string{"Abs"}, SupportingMuscles: []string{"Hip flexors"}, Movement: "Core", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)hip thrust|glute drive`), Meta{MajorMuscles: []string{"Glute max"}, SupportingMuscles: []string{"Hamstrings"}, Movement: "Hip extension", Stimulus: "shortened", FatigueCost: 2, Role: "primary", MinReps: 6, MaxReps: 12, TargetSets: 3, IncrementKg: 5}},
	{regexp.MustCompile(`(?i)back extension|hyperextension|45 degree hyper`), Meta{MajorMuscles: []string{"Glute max", "Hamstrings"}, SupportingMuscles: []string{"Erectors"}, Movement: "Hip extension", Stimulus: "lengthened", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)romanian deadlift|\brdl\b|stiff.?leg`), Meta{MajorMuscles: []string{"Hamstrings", "Glute max"}, SupportingMuscles: []string{"Erectors"}, Movement: "Hip hinge", Stimulus: "lengthened", FatigueCost: 3, Role: "primary", MinReps: 6, MaxReps: 10, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)leg curl|hamstring curl`), Meta{MajorMuscles: []string{"Hamstrings"}, Movement: "Knee flexion", Stimulus: "mixed", FatigueCost: 1, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)leg extension`), Meta{MajorMuscles: []string{"Quads"}, Movement: "Knee extension", Stimulus: "shortened", FatigueCost: 1, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)hip abduction|abductor`), Meta{MajorMuscles: []string{"Glute med/min"}, SupportingMuscles: []string{"Glute max"}, Movement: "Hip abduction", Stimulus: "shortened", FatigueCost: 1, Role: "accessory", MinReps: 10, MaxReps: 20, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)hip adduction|adductor`), Meta{MajorMuscles: []string{"Adductors"}, Movement: "Hip adduction", Stimulus: "mixed", FatigueCost: 1, Role: "accessory", MinReps: 10, MaxReps: 20, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)bulgarian|split squat|reverse lunge|walking lunge|smith lunge`), Meta{MajorMuscles: []string{"Glute max", "Quads"}, SupportingMuscles: []string{"Adductors"}, Movement: "Lunge", Stimulus: "lengthened", FatigueCost: 3, Role: "primary", MinReps: 6, MaxReps: 12, TargetSets: 3, IncrementKg: 2.5}},
	{regexp.MustCompile(`(?i)leg press|hack squat|pendulum`), Meta{MajorMuscles: []string{"Quads", "Glute max"}, SupportingMuscles: []string{"Adductors"}, Movement: "Squat/press", Stimulus: "lengthened", FatigueCost: 3, Role: "primary", MinReps: 6, MaxReps: 12, TargetSets: 3, IncrementKg: 5}},
	{regexp.MustCompile(`(?i)kickback`), Meta{MajorMuscles: []string{"Glute max"}, Movement: "Hip extension", Stimulus: "shortened", FatigueCost: 1, Role: "accessory", MinReps: 10, MaxReps: 20, TargetSets: 3, IncrementKg: 1.25}},
	{regexp.MustCompile(`(?i)lateral raise|side raise|y raise`), Meta{MajorMuscles: []string{"Side delts"}, Movement: "Shoulder abduction", Stimulus: "mixed", FatigueCost: 1, Role: "accessory", MinReps: 8, MaxReps: 20, TargetSets: 3, IncrementKg: 1}},
	{regexp.MustCompile(`(?i)face pull|reverse fly|rear delt`), Meta{MajorMuscles: []string{"Rear delts"}, SupportingMuscles: []string{"Upper back"}, Movement: "Horizontal abduction", Stimulus: "shortened", FatigueCost: 1, Role: "accessory", MinReps: 10, MaxReps: 20, TargetSets: 3, IncrementKg: 1}},
}

func cleanMuscles(xs []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, x := range xs {
		x = strings.TrimSpace(x)
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		out = append(out, x)
	}
	return out
}

func normalizeMeta(m Meta) Meta {
	if len(m.MajorMuscles) == 0 {
		if strings.TrimSpace(m.PrimaryMuscle) != "" {
			m.MajorMuscles = []string{strings.TrimSpace(m.PrimaryMuscle)}
		} else {
			m.MajorMuscles = []string{"Unmapped"}
		}
	}
	if len(m.SupportingMuscles) == 0 && len(m.SecondaryMuscles) > 0 {
		m.SupportingMuscles = append([]string{}, m.SecondaryMuscles...)
	}
	m.MajorMuscles = cleanMuscles(m.MajorMuscles)
	m.SupportingMuscles = cleanMuscles(m.SupportingMuscles)
	major := map[string]bool{}
	for _, x := range m.MajorMuscles {
		major[x] = true
	}
	filtered := m.SupportingMuscles[:0]
	for _, x := range m.SupportingMuscles {
		if !major[x] {
			filtered = append(filtered, x)
		}
	}
	m.SupportingMuscles = filtered
	// Clear legacy fields after migration so new saves use the new model only.
	m.PrimaryMuscle = ""
	m.SecondaryMuscles = nil
	return m
}

func majorLabel(m Meta) string {
	if len(m.MajorMuscles) == 0 {
		return "Unmapped"
	}
	seen := map[string]bool{}
	labels := []string{}
	for _, raw := range m.MajorMuscles {
		label := canonicalVolumeMuscle(raw)
		if label != "" && !seen[label] {
			seen[label] = true
			labels = append(labels, label)
		}
	}
	if len(labels) == 0 {
		return "Unmapped"
	}
	return strings.Join(labels, " + ")
}

func stateDirectory() string {
	if runtime.GOOS == "windows" {
		if base, err := os.UserConfigDir(); err == nil && strings.TrimSpace(base) != "" {
			return filepath.Join(base, "Hevy Signal")
		}
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".hevy-signal")
}

func newApp() *App {
	dir := stateDirectory()
	_ = os.MkdirAll(dir, 0700)
	a := &App{dir: dir, cfg: Config{}, store: Store{WorkoutIDs: map[string]string{}, Meta: map[string]Meta{}, CurrentRoutineExercises: map[string]bool{}}}
	a.load()
	if a.store.ProgramSource=="hevy-api" && a.store.ActiveFolderID!="" { a.rebuildCurrentRoutineExercisesLocked() }
	if err := a.backupBeforeAutoRepair(); err != nil {
		log.Printf("auto-mapping backup failed: %v", err)
	} else if a.repairAutomaticMappings() > 0 {
		_ = a.saveStore()
	}
	return a
}

func (a *App) load() {
	if b, err := os.ReadFile(filepath.Join(a.dir, "config.json")); err == nil {
		_ = json.Unmarshal(b, &a.cfg)
		// Rewrite legacy configs using the current schema. This preserves the Hevy key
		// while removing OpenAI fields saved by older Coach Chat beta builds.
		_ = a.saveConfig()
	}
	if b, err := os.ReadFile(filepath.Join(a.dir, "data.json")); err == nil {
		_ = json.Unmarshal(b, &a.store)
	}
	if a.store.WorkoutIDs == nil {
		a.store.WorkoutIDs = map[string]string{}
	}
	if a.store.Meta == nil {
		a.store.Meta = map[string]Meta{}
	}
	if a.store.CurrentRoutineExercises == nil {
		a.store.CurrentRoutineExercises = map[string]bool{}
	}
	// v1.2 migration: reclassify auto-tagged compound lower-body movements that
	// were previously labeled as secondary programming roles. Preserve any
	// metadata the user edited manually.
	changed := false
	for tid, m := range a.store.Meta {
		before, _ := json.Marshal(m)
		m = normalizeMeta(m)
		if m.AutoTagged && m.Role == "secondary" {
			for _, p := range presets {
				if p.Pattern.MatchString(m.ExerciseTitle) && p.Meta.Role == "primary" {
					m.Role = "primary"
					break
				}
			}
		}
		after, _ := json.Marshal(m)
		if string(before) != string(after) {
			changed = true
		}
		a.store.Meta[tid] = m
	}
	if changed {
		_ = a.saveStore()
	}
}

func atomicWrite(path string, v any, perm os.FileMode) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err = os.WriteFile(tmp, b, perm); err != nil {
		return err
	}
	_ = os.Chmod(tmp, perm)
	if runtime.GOOS == "windows" {
		// os.Rename cannot reliably replace an existing destination on Windows.
		// Remove the old small JSON state file first, then move the fully-written temp file in place.
		_ = os.Remove(path)
	}
	return os.Rename(tmp, path)
}
func (a *App) saveConfig() error {
	return atomicWrite(filepath.Join(a.dir, "config.json"), a.cfg, 0600)
}
func (a *App) saveStore() error { return atomicWrite(filepath.Join(a.dir, "data.json"), a.store, 0600) }

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 2<<20)).Decode(v)
}
func asString(v any) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	default:
		return fmt.Sprint(x)
	}
}
func asFloat(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case json.Number:
		f, e := x.Float64()
		return f, e == nil
	case string:
		f, e := strconv.ParseFloat(x, 64)
		return f, e == nil
	}
	return 0, false
}
func asInt(v any) (int, bool) { f, ok := asFloat(v); return int(f), ok }
func mget(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}

func (a *App) hevyGet(path string) (map[string]any, error) {
	a.mu.RLock()
	key := strings.TrimSpace(a.cfg.HevyAPIKey)
	a.mu.RUnlock()
	if key == "" {
		return nil, fmt.Errorf("Hevy is not connected yet. Open Settings and paste your Hevy API key")
	}
	req, _ := http.NewRequest("GET", baseURL+path, nil)
	req.Header.Set("api-key", key)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Hevy-Signal/"+version)
	c := &http.Client{Timeout: 35 * time.Second}
	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("Hevy API %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	var out map[string]any
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err = dec.Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

func (a *App) hevyTest() (int, error) {
	if _, err := a.hevyGet("/user/info"); err != nil {
		return 0, err
	}
	j, err := a.hevyGet("/workouts/count")
	if err != nil {
		return 0, err
	}
	if n, ok := asInt(mget(j, "workout_count", "workoutCount", "count")); ok {
		return n, nil
	}
	return 0, nil
}

func routineRepTargets(sets []any) string {
	parts := []string{}
	for _, sv := range sets {
		s, ok := sv.(map[string]any)
		if !ok {
			continue
		}
		st := strings.ToLower(asString(mget(s, "set_type", "type")))
		if st == "warmup" || st == "warm_up" {
			continue
		}
		if r, ok := asInt(mget(s, "reps", "rep_count", "target_reps")); ok && r > 0 {
			parts = append(parts, strconv.Itoa(r))
		}
	}
	if len(parts) == 0 {
		return ""
	}
	all := true
	for _, x := range parts[1:] {
		if x != parts[0] {
			all = false
			break
		}
	}
	if all {
		return parts[0]
	}
	return strings.Join(parts, "/")
}

// The complete saved routine library stays in Store.Routines. The active program
// is a view controlled by the user's explicitly selected Hevy folder.
func (a *App) selectedRoutinesLocked() []RoutineSummary {
 if a.store.ActiveFolderID=="" || a.store.ProgramSource=="csv-inferred" {return a.store.Routines}
 out:=[]RoutineSummary{}
 for _,r:=range a.store.Routines {if r.FolderID==a.store.ActiveFolderID{out=append(out,r)}}
 return out
}
func (a *App) selectedFolderTitleLocked() string {
 if a.store.ActiveFolderID=="" || a.store.ProgramSource=="csv-inferred" {return "All saved routines"}
 for _,f:=range a.store.RoutineFolders {if f.ID==a.store.ActiveFolderID {return f.Title}}
 return "Previously selected folder is unavailable (choose another)"
}
func (a *App) rebuildCurrentRoutineExercisesLocked() {
 active:=map[string]bool{}
 for _,r:=range a.selectedRoutinesLocked() {
  for _,ex:=range r.Exercises {if ex.TemplateID!="" {active[ex.TemplateID]=true}}
 }
 a.store.CurrentRoutineExercises=active
}
// When a specific folder is selected, the workout must reference a routine in
// that folder. Missing routine IDs are excluded, never silently misattributed.
func (a *App) isSelectedWorkoutSetLocked(row SetRow) bool {
 if a.store.ActiveFolderID=="" || a.store.ProgramSource=="csv-inferred" {return true}
 if row.RoutineID=="" {return false}
 for _,r:=range a.selectedRoutinesLocked() {if r.ID==row.RoutineID {return true}}
 return false
}
func parseHevyFolders(j map[string]any) []RoutineFolder {
 arr,_:=j["routine_folders"].([]any)
 if arr==nil {arr,_=j["data"].([]any)}
 out:=[]RoutineFolder{}
 seen:=map[string]bool{}
 for _,v:=range arr {
  row,ok:=v.(map[string]any);if !ok {continue}
  id:=strings.TrimSpace(asString(mget(row,"id","folder_id")))
  if id==""||seen[id] {continue}
  title:=strings.TrimSpace(asString(mget(row,"title","name")))
  if title=="" {title="Untitled folder"}
  seen[id]=true
  out=append(out,RoutineFolder{ID:id,Title:title})
 }
 return out
}
func (a *App) fetchRoutineFolders() ([]RoutineFolder,error) {
 folders:=[]RoutineFolder{}
 seen:=map[string]bool{}
 for page:=1;page<=100;page++ {
  q:=url.Values{};q.Set("page",strconv.Itoa(page));q.Set("pageSize","10")
  j,err:=a.hevyGet("/routine_folders?"+q.Encode())
  if err!=nil {return nil,err}
  pageFolders:=parseHevyFolders(j)
  for _,f:=range pageFolders {if !seen[f.ID] {seen[f.ID]=true;folders=append(folders,f)}}
  if count,ok:=asInt(mget(j,"page_count","pageCount"));ok {
   if page>=count {break}
  }else if len(pageFolders)<10 {break}
  if len(pageFolders)==0 {break}
 }
 return folders,nil
}

func (a *App) syncRoutines() (int, error) {
	routines := []RoutineSummary{}
	for page := 1; page <= 100; page++ {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("pageSize", "10")
		j, err := a.hevyGet("/routines?" + q.Encode())
		if err != nil {
			return 0, err
		}
		var arr []any
		if x, ok := j["routines"].([]any); ok {
			arr = x
		} else if x, ok := j["data"].([]any); ok {
			arr = x
		}
		if len(arr) == 0 {
			break
		}
		for _, rv := range arr {
			rm, ok := rv.(map[string]any)
			if !ok {
				continue
			}
			r := RoutineSummary{ID: asString(rm["id"]), Title: asString(mget(rm, "title", "name")), FolderID: asString(mget(rm, "folder_id", "folderId"))}
			if r.Title == "" {
				r.Title = "Routine"
			}
			exs, _ := rm["exercises"].([]any)
			for _, ev := range exs {
				ex, ok := ev.(map[string]any)
				if !ok {
					continue
				}
				tid := asString(mget(ex, "exercise_template_id", "exerciseTemplateId", "id"))
				et := asString(mget(ex, "title", "exercise_title", "name"))
				if tid == "" {
					continue
				}
				sets, _ := ex["sets"].([]any)
				count := 0
				for _, sv := range sets {
					if sm, ok := sv.(map[string]any); ok {
						st := strings.ToLower(asString(mget(sm, "set_type", "type")))
						if st != "warmup" && st != "warm_up" {
							count++
						}
					}
				}
				re := RoutineExercise{TemplateID: tid, ExerciseTitle: et, SetCount: count, RepTargets: routineRepTargets(sets)}
				r.Exercises = append(r.Exercises, re)
			}
			routines = append(routines, r)
		}
		if v, ok := asInt(mget(j, "page_count", "pageCount")); ok {
			if page >= v {
				break
			}
		} else if len(arr) < 10 {
			// Hevy has changed pagination metadata across API revisions.
			// If page_count is absent, a short page is the reliable end signal.
			break
		}
	}
	folders, err := a.fetchRoutineFolders()
	if err != nil {return 0,fmt.Errorf("Hevy routine folders could not be loaded: %w",err)}
	a.mu.Lock()
	for _, r := range routines {
		for _, ex := range r.Exercises {
			if old, ok := a.store.Meta[ex.TemplateID]; ok {
				if ex.ExerciseTitle != "" {
					old.ExerciseTitle = ex.ExerciseTitle
					a.store.Meta[ex.TemplateID] = old
				}
			} else {
				a.store.Meta[ex.TemplateID] = a.defaultMeta(ex.TemplateID, ex.ExerciseTitle)
			}
		}
	}
	a.store.Routines = routines
	a.store.RoutineFolders = folders
	a.store.ProgramSource = "hevy-api"
	a.rebuildCurrentRoutineExercisesLocked()
	activeCount:=len(a.selectedRoutinesLocked())
	a.mu.Unlock()
	return activeCount, nil
}

func (a *App) syncHevy() (int, int, error) {
	if _, err := a.hevyTest(); err != nil {
		return 0, 0, err
	}
	rn, rerr := a.syncRoutines()
	if rerr != nil {
		return 0, 0, rerr
	}
	all := []map[string]any{}
	for page := 1; page <= 500; page++ {
		q := url.Values{}
		q.Set("page", strconv.Itoa(page))
		q.Set("pageSize", "10")
		j, err := a.hevyGet("/workouts?" + q.Encode())
		if err != nil {
			return 0, 0, err
		}
		var arr []any
		if x, ok := j["workouts"].([]any); ok {
			arr = x
		} else if x, ok := j["data"].([]any); ok {
			arr = x
		}
		if len(arr) == 0 {
			break
		}
		for _, v := range arr {
			if m, ok := v.(map[string]any); ok {
				all = append(all, m)
			}
		}
		if v, ok := asInt(mget(j, "page_count", "pageCount")); ok {
			if page >= v {
				break
			}
		} else if len(arr) < 10 {
			// Hevy has changed pagination metadata across API revisions.
			// If page_count is absent, a short page is the reliable end signal.
			break
		}
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, w := range all {
		a.ingestLocked(w)
	}
	return len(all), rn, a.saveStore()
}

var csvHeaderReplacer = strings.NewReplacer(" ", "_", "-", "_", "(", "", ")", "", ".", "_")

func csvHeaderKey(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	s = csvHeaderReplacer.Replace(s)
	for strings.Contains(s, "__") {
		s = strings.ReplaceAll(s, "__", "_")
	}
	return strings.Trim(s, "_")
}

func csvValue(row map[string]string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(row[csvHeaderKey(k)]); v != "" {
			return v
		}
	}
	return ""
}

func csvFloat(row map[string]string, keys ...string) (float64, bool) {
	v := csvValue(row, keys...)
	if v == "" {
		return 0, false
	}
	v = strings.ReplaceAll(v, ",", "")
	f, err := strconv.ParseFloat(v, 64)
	return f, err == nil
}

func csvInt(row map[string]string, keys ...string) (int, bool) {
	f, ok := csvFloat(row, keys...)
	return int(f), ok
}

func csvExerciseID(title string) string {
	// Stable local identity for CSV mode. Keep execution variants separate when
	// Hevy exports them under distinct custom-exercise names.
	n := strings.ToLower(strings.TrimSpace(title))
	n = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(n, "-")
	n = strings.Trim(n, "-")
	if n == "" {
		n = "exercise"
	}
	return "csv:" + n
}

type csvWorkout struct {
	ID    string
	Title string
	Start string
	Rows  []SetRow
}

func (a *App) importHevyCSV(r io.Reader) (int, int, int, error) {
	cr := csv.NewReader(io.LimitReader(r, 40<<20))
	cr.FieldsPerRecord = -1
	cr.ReuseRecord = false
	headers, err := cr.Read()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("could not read CSV header: %w", err)
	}
	idx := map[int]string{}
	seen := map[string]bool{}
	for i, h := range headers {
		k := csvHeaderKey(h)
		idx[i] = k
		seen[k] = true
	}
	if !(seen["exercise_title"] || seen["exercise_name"]) {
		return 0, 0, 0, fmt.Errorf("this does not look like a Hevy workout export: missing exercise_title")
	}
	if !(seen["start_time"] || seen["date"]) {
		return 0, 0, 0, fmt.Errorf("this does not look like a Hevy workout export: missing start_time")
	}

	byWorkout := map[string]*csvWorkout{}
	order := []string{}
	rowCount := 0
	for {
		rec, e := cr.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return 0, 0, 0, fmt.Errorf("CSV row %d: %w", rowCount+2, e)
		}
		rowCount++
		m := map[string]string{}
		for i, v := range rec {
			if k, ok := idx[i]; ok {
				m[k] = strings.TrimSpace(v)
			}
		}
		et := csvValue(m, "exercise_title", "exercise_name")
		if et == "" {
			continue
		}
		wt := csvValue(m, "title", "workout_title", "workout_name")
		if wt == "" {
			wt = "Imported workout"
		}
		start := csvValue(m, "start_time", "date", "workout_start_time")
		if start == "" {
			continue
		}
		wk := wt + "\x00" + start
		cw := byWorkout[wk]
		if cw == nil {
			cw = &csvWorkout{ID: "csv-workout:" + strconv.Itoa(len(order)+1) + ":" + start, Title: wt, Start: start}
			byWorkout[wk] = cw
			order = append(order, wk)
		}
		st := strings.ToLower(csvValue(m, "set_type", "type"))
		if st == "warmup" || st == "warm_up" {
			continue
		}
		tid := csvExerciseID(et)
		sr := SetRow{WorkoutID: cw.ID, WorkoutTitle: wt, PerformedAt: start, TemplateID: tid, ExerciseTitle: et}
		if x, ok := csvInt(m, "set_index", "set_order", "set"); ok {
			sr.SetIndex = x
		}
		if x, ok := csvFloat(m, "weight_kg", "weightkg", "weight"); ok {
			sr.WeightKg = x
			sr.HasWeight = true
		} else if x, ok := csvFloat(m, "weight_lbs", "weight_lb", "weight_lbs."); ok {
			sr.WeightKg = x * 0.45359237
			sr.HasWeight = true
		}
		if x, ok := csvInt(m, "reps", "rep_count"); ok {
			sr.Reps = x
			sr.HasReps = true
		}
		if x, ok := csvFloat(m, "rpe"); ok {
			sr.RPE = x
			sr.HasRPE = true
		}
		cw.Rows = append(cw.Rows, sr)
	}
	if len(byWorkout) == 0 {
		return 0, 0, 0, fmt.Errorf("no workout sets were found in the CSV")
	}

	// Sort sessions newest-first when dates are parseable. This lets CSV mode infer
	// a current program from the latest occurrence of recent workout names.
	workouts := make([]*csvWorkout, 0, len(byWorkout))
	for _, k := range order {
		workouts = append(workouts, byWorkout[k])
	}
	sort.SliceStable(workouts, func(i, j int) bool {
		ti, oki := parseTime(workouts[i].Start)
		tj, okj := parseTime(workouts[j].Start)
		if oki && okj {
			return ti.After(tj)
		}
		return workouts[i].Start > workouts[j].Start
	})

	a.mu.Lock()
	defer a.mu.Unlock()
	oldMeta := a.store.Meta
	a.store.WorkoutIDs = map[string]string{}
	a.store.Sets = nil
	a.store.Routines = nil
	a.store.CurrentRoutineExercises = map[string]bool{}
	a.store.Meta = map[string]Meta{}
	a.store.ProgramSource = "csv-inferred"
	a.store.ActiveFolderID=""
	a.store.RoutineFolders=nil
	for _, cw := range workouts {
		a.store.WorkoutIDs[cw.ID] = cw.Title
		for _, sr := range cw.Rows {
			a.store.Sets = append(a.store.Sets, sr)
			if m, ok := oldMeta[sr.TemplateID]; ok {
				m.ExerciseTitle = sr.ExerciseTitle
				a.store.Meta[sr.TemplateID] = normalizeMeta(m)
			} else if _, ok := a.store.Meta[sr.TemplateID]; !ok {
				a.store.Meta[sr.TemplateID] = a.defaultMeta(sr.TemplateID, sr.ExerciseTitle)
			}
		}
	}

	// Infer routine cards from the latest occurrence of up to four distinct workout
	// names among the six most recent sessions. This is intentionally labeled as
	// inferred because Hevy's workout CSV does not carry saved routine definitions.
	seenTitles := map[string]bool{}
	for _, cw := range workouts {
		if len(seenTitles) >= 4 {
			break
		}
		key := strings.ToLower(strings.TrimSpace(cw.Title))
		if seenTitles[key] {
			continue
		}
		seenTitles[key] = true
		rs := RoutineSummary{ID: "csv-routine:" + strconv.Itoa(len(a.store.Routines)+1), Title: cw.Title + " (inferred)"}
		perEx := map[string][]SetRow{}
		exOrder := []string{}
		for _, sr := range cw.Rows {
			if _, ok := perEx[sr.TemplateID]; !ok {
				exOrder = append(exOrder, sr.TemplateID)
			}
			perEx[sr.TemplateID] = append(perEx[sr.TemplateID], sr)
		}
		for _, tid := range exOrder {
			rows := perEx[tid]
			if len(rows) == 0 {
				continue
			}
			reps := []string{}
			for _, sr := range rows {
				if sr.HasReps {
					reps = append(reps, strconv.Itoa(sr.Reps))
				}
			}
			rt := ""
			if len(reps) > 0 {
				all := true
				for _, x := range reps[1:] {
					if x != reps[0] {
						all = false
						break
					}
				}
				if all {
					rt = reps[0]
				} else {
					rt = strings.Join(reps, "/")
				}
			}
			rs.Exercises = append(rs.Exercises, RoutineExercise{TemplateID: tid, ExerciseTitle: rows[0].ExerciseTitle, SetCount: len(rows), RepTargets: rt})
			a.store.CurrentRoutineExercises[tid] = true
		}
		a.store.Routines = append(a.store.Routines, rs)
	}
	if err := a.saveStore(); err != nil {
		return 0, 0, 0, err
	}
	return len(workouts), len(a.store.Sets), len(a.store.Routines), nil
}

// isAutoEditable protects explicitly saved manual muscle classifications.
// v3.3's browser catalog accidentally marked machine-generated classifications
// as manual, recognizable by its distinctive Movement marker.
func isAutoEditable(m Meta) bool {
	unknown := len(m.MajorMuscles) == 0 ||
		(len(m.MajorMuscles) == 1 && strings.EqualFold(strings.TrimSpace(m.MajorMuscles[0]), "Unmapped"))
	defaultUnknown := unknown && (m.Movement == "" || m.Movement == "Unmapped")
	return m.AutoTagged || defaultUnknown ||
		strings.HasPrefix(m.Movement, "Automatically classified")
}

// repairAutomaticMappings re-evaluates ALL previously auto-tagged classifications,
// not just unmapped exercises. Never overwrites explicitly saved manual mapping
// or workout settings (reps, progression, training role, notes, stimulus).
func (a *App) repairAutomaticMappings() int {
	changed := 0
	for tid, old := range a.store.Meta {
		if !isAutoEditable(old) || strings.TrimSpace(old.ExerciseTitle) == "" {
			continue
		}
		updated := a.defaultMeta(tid, old.ExerciseTitle)
		updated.MajorMuscles, updated.SupportingMuscles = sanitizeAutomaticPress(old.ExerciseTitle, updated.MajorMuscles, updated.SupportingMuscles)
		if len(updated.MajorMuscles) == 0 || (len(updated.MajorMuscles) == 1 && updated.MajorMuscles[0] == "Unmapped") {
			continue
		}
		majorChanged := strings.Join(old.MajorMuscles, "\x1f") != strings.Join(updated.MajorMuscles, "\x1f")
		secondaryChanged := strings.Join(old.SupportingMuscles, "\x1f") != strings.Join(updated.SupportingMuscles, "\x1f")
		provenanceChanged := !old.AutoTagged
		movementChanged := old.Movement == "" || old.Movement == "Unmapped" || strings.HasPrefix(old.Movement, "Automatically classified")
		if !majorChanged && !secondaryChanged && !provenanceChanged && !movementChanged {
			continue
		}
		old.MajorMuscles = updated.MajorMuscles
		old.SupportingMuscles = updated.SupportingMuscles
		if movementChanged {
			old.Movement = updated.Movement
		}
		old.AutoTagged = true
		a.store.Meta[tid] = normalizeMeta(old)
		changed++
	}
	return changed
}

// Preserve the original local data.json before the first v3.4 repair, once.
// A separate backup is never included in exercise data exports.
func (a *App) backupBeforeAutoRepair() error {
	source := filepath.Join(a.dir, "data.json")
	target := filepath.Join(a.dir, "data-before-v3.4-remap.json")
	if _, err := os.Stat(target); err == nil {
		return nil
	}
	data, err := os.ReadFile(source)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0600)
}

func (a *App) defaultMeta(tid, title string) Meta {
	for _, p := range presets {
		if p.Pattern.MatchString(title) {
			m := p.Meta
			m.TemplateID = tid
			m.ExerciseTitle = title
			m.AutoTagged = true
			m.MajorMuscles, m.SupportingMuscles = sanitizeAutomaticPress(title, m.MajorMuscles, m.SupportingMuscles)
			return m
		}
	}
	return Meta{TemplateID: tid, ExerciseTitle: title, MajorMuscles: []string{"Unmapped"}, Movement: "Unmapped", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 6, MaxReps: 12, TargetSets: 3, IncrementKg: 2.5}
}

func (a *App) ingestLocked(w map[string]any) {
	wid := asString(w["id"])
	if wid == "" {
		return
	}
	title := asString(mget(w, "title"))
	if title == "" {
		title = "Workout"
	}
	at := asString(mget(w, "start_time", "startTime", "created_at"))
	a.store.WorkoutIDs[wid] = title
	keep := a.store.Sets[:0]
	for _, s := range a.store.Sets {
		if s.WorkoutID != wid {
			keep = append(keep, s)
		}
	}
	a.store.Sets = keep
	exs, _ := w["exercises"].([]any)
	for _, ev := range exs {
		ex, ok := ev.(map[string]any)
		if !ok {
			continue
		}
		et := asString(ex["title"])
		if et == "" {
			et = "Exercise"
		}
		tid := asString(mget(ex, "exercise_template_id", "exerciseTemplateId", "id"))
		if tid == "" {
			tid = et
		}
		if old, ok := a.store.Meta[tid]; ok {
			old.ExerciseTitle = et
			a.store.Meta[tid] = old
		} else {
			a.store.Meta[tid] = a.defaultMeta(tid, et)
		}
		sets, _ := ex["sets"].([]any)
		for _, sv := range sets {
			s, ok := sv.(map[string]any)
			if !ok {
				continue
			}
			st := strings.ToLower(asString(mget(s, "set_type", "type")))
			if st == "warmup" || st == "warm_up" {
				continue
			}
			row := SetRow{WorkoutID: wid, WorkoutTitle: title, PerformedAt: at, TemplateID: tid, RoutineID: asString(mget(w,"routine_id","routineId")), ExerciseTitle: et}
			if x, ok := asInt(s["index"]); ok {
				row.SetIndex = x
			}
			if x, ok := asFloat(mget(s, "weight_kg", "weightKg")); ok {
				row.WeightKg = x
				row.HasWeight = true
			}
			if x, ok := asInt(s["reps"]); ok {
				row.Reps = x
				row.HasReps = true
			}
			if x, ok := asFloat(s["rpe"]); ok {
				row.RPE = x
				row.HasRPE = true
			}
			a.store.Sets = append(a.store.Sets, row)
		}
	}
}

type SessionSummary struct {
	Date     string     `json:"date"`
	Workout  string     `json:"workout"`
	Sets     int        `json:"sets"`
	Volume   float64    `json:"volume"`
	BestE1RM float64    `json:"bestE1RM"`
	BestSet  [2]float64 `json:"bestSet"`
	AvgRPE   *float64   `json:"avgRpe"`
}

func round1(x float64) float64 { return math.Round(x*10) / 10 }
func epley(w float64, reps int) float64 {
	if reps <= 0 {
		return 0
	}
	return w * (1 + float64(reps)/30)
}
func summary(rows []SetRow) SessionSummary {
	s := SessionSummary{Sets: len(rows)}
	if len(rows) > 0 {
		s.Date = rows[0].PerformedAt
		s.Workout = rows[0].WorkoutTitle
	}
	var rs float64
	var rn int
	for _, r := range rows {
		if r.HasWeight && r.HasReps {
			s.Volume += r.WeightKg * float64(r.Reps)
			e := epley(r.WeightKg, r.Reps)
			if e > s.BestE1RM {
				s.BestE1RM = e
				s.BestSet = [2]float64{r.WeightKg, float64(r.Reps)}
			}
		}
		if r.HasRPE {
			rs += r.RPE
			rn++
		}
	}
	s.Volume = round1(s.Volume)
	s.BestE1RM = round1(s.BestE1RM)
	if rn > 0 {
		x := round1(rs / float64(rn))
		s.AvgRPE = &x
	}
	return s
}

func (a *App) sessionsFor(tid string, limit int) [][]SetRow {
	rows := []SetRow{}
	for _, s := range a.store.Sets {
		if s.TemplateID == tid && a.isSelectedWorkoutSetLocked(s) {
			rows = append(rows, s)
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].PerformedAt == rows[j].PerformedAt {
			return rows[i].SetIndex < rows[j].SetIndex
		}
		return rows[i].PerformedAt > rows[j].PerformedAt
	})
	groups := [][]SetRow{}
	var cur string
	for _, r := range rows {
		if r.WorkoutID != cur {
			if len(groups) >= limit {
				break
			}
			groups = append(groups, []SetRow{})
			cur = r.WorkoutID
		}
		groups[len(groups)-1] = append(groups[len(groups)-1], r)
	}
	return groups
}

type Rec struct {
	TemplateID string           `json:"templateId"`
	Exercise   string           `json:"exercise"`
	Muscle     string           `json:"muscle"`
	Stimulus   string           `json:"stimulus"`
	Role       string           `json:"role"`
	Status     string           `json:"status"`
	Headline   string           `json:"headline"`
	Rationale  string           `json:"rationale"`
	Confidence float64          `json:"confidence"`
	History    []SessionSummary `json:"history"`
}

func score(rows []SetRow) float64 {
	v := 0.0
	for _, r := range rows {
		if r.HasWeight && r.HasReps {
			v += r.WeightKg * float64(r.Reps)
		}
	}
	return v
}

// Muscle-volume totals represent broad muscle groups, not summed anatomical
// subheads. Rectus femoris and the vasti are parts of the quadriceps, never
// separate additional quadriceps-set exposures.
func canonicalVolumeMuscle(s string) string {
	k := strings.ToLower(strings.TrimSpace(s))
	if k == "" || k == "unmapped" {
		return ""
	}
	if strings.Contains(k, "rectus femoris") || strings.Contains(k, "vastus") ||
		strings.Contains(k, "quadricep") || strings.HasPrefix(k, "quads") {
		return "Quads"
	}
	if strings.Contains(k, "pectoralis major") || k == "pecs" || k == "chest" {
		return "Chest"
	}
	if strings.Contains(k, "triceps brachii") {
		return "Triceps"
	}
	if strings.Contains(k, "biceps brachii") {
		return "Biceps"
	}
	if strings.Contains(k, "deltoid") && strings.Contains(k, "anterior") {
		return "Front delts"
	}
	return strings.TrimSpace(s)
}
func isWalkingTitle(title string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	return t == "walking" || t == "walk" || strings.HasPrefix(t, "treadmill walk") || strings.HasPrefix(t, "brisk walk")
}
func isCarryTitle(title string) bool {
	t := strings.ToLower(strings.TrimSpace(title))
	return strings.Contains(t, "carry") || strings.Contains(t, "farmers walk") ||
		strings.Contains(t, "farmer's walk") || strings.Contains(t, "farmer walk") || strings.Contains(t, "suitcase walk")
}
func isNonRepTitle(title string) bool { return isWalkingTitle(title) || isCarryTitle(title) }
func isElbowExtensionPress(title string) bool {
	t := strings.ToLower(title)
	return strings.Contains(t, "chest press") || strings.Contains(t, "bench press") ||
		strings.Contains(t, "shoulder press") || strings.Contains(t, "overhead press") ||
		strings.Contains(t, "military press") || strings.Contains(t, "arnold press")
}
func containsBiceps(parts []string) bool {
	for _, p := range parts {
		if strings.Contains(strings.ToLower(p), "bicep") {
			return true
		}
	}
	return false
}
func hasMuscle(parts []string, canonical string) bool {
	for _, p := range parts {
		if canonicalVolumeMuscle(p) == canonical {
			return true
		}
	}
	return false
}

// Applies only to automatically generated mappings. A user's manually saved
// classifications remain unchanged, even when Signal suggests reviewing them.
// Keep quadriceps subdivisions as a single group in auto-derived exercise
// anatomy, not only in the summary math. Manual anatomical notes remain intact.
func canonicalAutoParts(major, secondary []string) ([]string, []string) {
	cleanedPrimary := []string{}
	primary := map[string]bool{}
	for _, p := range major {
		label := p
		if canonicalVolumeMuscle(p) == "Quads" {
			label = "Quads"
		}
		if label == "" || primary[label] {
			continue
		}
		primary[label] = true
		cleanedPrimary = append(cleanedPrimary, label)
	}
	cleanedSecondary := []string{}
	seen := map[string]bool{}
	for _, p := range secondary {
		label := p
		if canonicalVolumeMuscle(p) == "Quads" {
			label = "Quads"
		}
		if label == "" || primary[label] || seen[label] {
			continue
		}
		seen[label] = true
		cleanedSecondary = append(cleanedSecondary, label)
	}
	return cleanedPrimary, cleanedSecondary
}
func sanitizeAutomaticPress(title string, major, secondary []string) ([]string, []string) {
	if !isElbowExtensionPress(title) {
		return canonicalAutoParts(major, secondary)
	}
	filtered := []string{}
	for _, p := range secondary {
		if !strings.Contains(strings.ToLower(p), "bicep") {
			filtered = append(filtered, p)
		}
	}
	if !hasMuscle(major, "Triceps") && !hasMuscle(filtered, "Triceps") {
		filtered = append(filtered, "Triceps")
	}
	if strings.Contains(strings.ToLower(title), "chest press") || strings.Contains(strings.ToLower(title), "bench press") {
		if !hasMuscle(major, "Chest") && !hasMuscle(filtered, "Chest") {
			major = append(major, "Chest")
		}
	}
	return canonicalAutoParts(major, filtered)
}

// One exercise set contributes at most one direct set to one canonical group;
// secondary muscle exposure is modeled at 0.5, never added to an already
// directly trained canonical group for that same exercise set.
func addMuscleExposure(m Meta, mus map[string]float64, stim map[string]map[string]float64) {
	if isNonRepTitle(m.ExerciseTitle) {
		return
	} // carries/cardio use separate training metrics
	primary := map[string]bool{}
	for _, raw := range m.MajorMuscles {
		p := canonicalVolumeMuscle(raw)
		if p == "" || primary[p] {
			continue
		}
		primary[p] = true
		mus[p]++
		if stim[p] == nil {
			stim[p] = map[string]float64{"lengthened": 0, "shortened": 0, "mixed": 0}
		}
		kind := m.Stimulus
		if _, ok := stim[p][kind]; !ok {
			kind = "mixed"
		}
		stim[p][kind]++
	}
	secondary := map[string]bool{}
	for _, raw := range m.SupportingMuscles {
		sec := canonicalVolumeMuscle(raw)
		if sec == "" || primary[sec] || secondary[sec] {
			continue
		}
		secondary[sec] = true
		mus[sec] += .5
	}
}

// Large discontinuities in a named exercise's historical loads can signal
// that two machines, leverage configurations or execution variants were
// logged under the same Hevy template. Block comparisons until verified.
func suspectLoadVariant(ss [][]SetRow) bool {
	minKg, maxKg := 0.0, 0.0
	sessions := 0
	for _, session := range ss {
		best := 0.0
		for _, row := range session {
			if row.HasWeight && row.WeightKg > best {
				best = row.WeightKg
			}
		}
		if best <= 0 {
			continue
		}
		sessions++
		if minKg == 0 || best < minKg {
			minKg = best
		}
		if best > maxKg {
			maxKg = best
		}
	}
	return sessions >= 2 && minKg > 0 && maxKg/minKg >= 2.25
}
func shouldVerifyLoadHistory(title string) bool {
	t := strings.ToLower(title)
	return strings.Contains(t, "abduct") || strings.Contains(t, "kickback")
}

func (a *App) recFor(m Meta) Rec {
	base := Rec{TemplateID: m.TemplateID, Exercise: m.ExerciseTitle, Muscle: majorLabel(m), Stimulus: m.Stimulus, Role: m.Role}
	ss := a.sessionsFor(m.TemplateID, 6)
	for _, g := range ss {
		base.History = append(base.History, summary(g))
	}
	set := func(st, h, r string, c float64) Rec {
		x := base
		x.Status = st
		x.Headline = h
		x.Rationale = r
		x.Confidence = c
		return x
	}

	if isWalkingTitle(m.ExerciseTitle) {
		return set("tracking", "Track walking by time and pace", "Use duration, distance, pace, and perceived effort for walking. Hevy's rep-based entries cannot establish walking progression; no load/reps or e1RM recommendation is issued.", .9)
	}
	if isCarryTitle(m.ExerciseTitle) {
		return set("tracking", "Track carry distance, duration and load", "Loaded carries should progress using distance, time, implement load, control and posture; the existing workout export lacks reliable distance/time details, so rep-based advice and e1RM are suppressed.", .9)
	}
	if isElbowExtensionPress(m.ExerciseTitle) && (containsBiceps(m.SupportingMuscles) || containsBiceps(m.MajorMuscles)) {
		return set("review", "Review pressing muscle classification", "Biceps are listed for a conventional press where triceps provide elbow extension. If this was manually mapped, review it under Exercise setup; Signal will not overwrite your edit.", .95)
	}
	if shouldVerifyLoadHistory(m.ExerciseTitle) && suspectLoadVariant(a.sessionsFor(m.TemplateID, 12)) {
		return set("review", "Verify machine or movement variant", "Historical loads for the same exercise name differ by more than 2.25-fold. Verify the cable stack, machine, unilateral vs bilateral loading, setup and execution. Separate incomparable variants into distinct Hevy exercise templates before interpreting strength trends; no progression recommendation is issued.", .95)
	}

	if len(ss) < 2 {
		return set("insufficient", "Insufficient data", "Log at least two exposures before changing anything.", .35)
	}
	cur, prev := ss[0], ss[1]
	cs, ps := score(cur), score(prev)
	hit := 0
	var ws []float64
	high := false
	for _, r := range cur {
		if r.HasReps && r.Reps >= m.MaxReps {
			hit++
		}
		if r.HasWeight {
			ws = append(ws, r.WeightKg)
		}
		if r.HasRPE && r.RPE >= 9.5 {
			high = true
		}
	}
	stable := len(ws) > 0
	if len(ws) > 1 {
		for _, x := range ws[1:] {
			if math.Abs(x-ws[0]) > .01 {
				stable = false
			}
		}
	}
	decl := len(ss) >= 3 && score(ss[0]) < score(ss[1])*.96 && score(ss[1]) < score(ss[2])*.98
	if hit >= m.TargetSets && stable && !high {
		return set("progress", fmt.Sprintf("Ready to progress +%g kg", m.IncrementKg), fmt.Sprintf("You reached the top of the %d–%d target on %d+ working sets without a near-max RPE flag.", m.MinReps, m.MaxReps, m.TargetSets), .92)
	}
	if hit >= m.TargetSets && stable && high {
		return set("maintain", "Own this load once more", "You reached the rep ceiling, but at least one set was RPE 9.5+. Repeat the load before progressing.", .86)
	}
	if decl {
		return set("fatigue", "Performance trend is down", "Three exposures show declining load×rep performance. Hold progression and inspect recovery/fatigue before adding work.", .80)
	}
	if ps > 0 && cs >= ps*.98 {
		return set("maintain", "Maintain the prescription", fmt.Sprintf("Performance is stable (%+.1f%% load×reps vs prior). No programming change is justified.", (cs/ps-1)*100), .84)
	}
	if ps > 0 {
		return set("watch", "Watch one more exposure", fmt.Sprintf("Latest load×rep performance is %+.1f%% vs prior. One down session is not enough evidence to rewrite the program.", (cs/ps-1)*100), .68)
	}
	return set("watch", "Watch one more exposure", "Not enough comparable load×rep data yet.", .55)
}

func parseTime(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339, time.RFC3339Nano, "2006-01-02T15:04:05"} {
		if t, e := time.Parse(layout, s); e == nil {
			return t, true
		}
	}
	return time.Time{}, false
}
func (a *App) stimulus(days int) ([]map[string]any, map[string]map[string]float64) {
	cut := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	mus := map[string]float64{}
	stim := map[string]map[string]float64{}
	for _, r := range a.store.Sets {
		t, ok := parseTime(r.PerformedAt)
		if ok && t.Before(cut) {
			continue
		}
		m, ok := a.store.Meta[r.TemplateID]
		if !ok {
			continue
		}
		addMuscleExposure(m, mus, stim)
	}
	type kv struct {
		k string
		v float64
	}
	arr := []kv{}
	for k, v := range mus {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	out := []map[string]any{}
	for _, x := range arr {
		out = append(out, map[string]any{"muscle": x.k, "sets": round1(x.v)})
	}
	return out, stim
}

func (a *App) isCurrentRoutineExercise(tid string) bool {
	return a.store.CurrentRoutineExercises[tid]
}

func (a *App) currentRoutineMetadata() []Meta {
	out := []Meta{}
	for tid := range a.store.CurrentRoutineExercises {
		if m, ok := a.store.Meta[tid]; ok {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ExerciseTitle) < strings.ToLower(out[j].ExerciseTitle)
	})
	return out
}

func (a *App) currentRoutineStimulus(days int) ([]map[string]any, map[string]map[string]float64) {
	cut := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	mus := map[string]float64{}
	stim := map[string]map[string]float64{}
	for _, r := range a.store.Sets {
		if !a.isSelectedWorkoutSetLocked(r) {continue}
		if !a.store.CurrentRoutineExercises[r.TemplateID] {
			continue
		}
		if t, ok := parseTime(r.PerformedAt); ok && t.Before(cut) {
			continue
		}
		m, ok := a.store.Meta[r.TemplateID]
		if !ok {
			continue
		}
		addMuscleExposure(m, mus, stim)
	}
	type kv struct {
		k string
		v float64
	}
	arr := []kv{}
	for k, v := range mus {
		arr = append(arr, kv{k, v})
	}
	sort.Slice(arr, func(i, j int) bool { return arr[i].v > arr[j].v })
	out := []map[string]any{}
	for _, x := range arr {
		out = append(out, map[string]any{"muscle": x.k, "sets": round1(x.v)})
	}
	return out, stim
}

func (a *App) metadata() []Meta {
	out := []Meta{}
	for _, m := range a.store.Meta {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		return strings.ToLower(out[i].ExerciseTitle) < strings.ToLower(out[j].ExerciseTitle)
	})
	return out
}
func containsWord(q, w string) bool {
	return regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `\b`).MatchString(q)
}

var aliases = map[string][]string{"glute": {"Glute max", "Glute med/min"}, "glutes": {"Glute max", "Glute med/min"}, "butt": {"Glute max", "Glute med/min"}, "hamstring": {"Hamstrings"}, "hamstrings": {"Hamstrings"}, "quad": {"Quads"}, "quads": {"Quads"}, "delts": {"Side delts", "Rear delts"}, "shoulders": {"Side delts", "Rear delts"}, "back": {"Upper back", "Lats", "Erectors"}, "adductors": {"Adductors"}}

func (a *App) evidence(q string, history []ChatTurn) map[string]any {
	// Retrieval follows the conversation, not only the latest short follow-up.
	contextParts := []string{}
	start := 0
	if len(history) > 6 {
		start = len(history) - 6
	}
	for _, t := range history[start:] {
		if strings.EqualFold(strings.TrimSpace(t.Role), "user") && strings.TrimSpace(t.Text) != "" {
			contextParts = append(contextParts, t.Text)
		}
	}
	contextParts = append(contextParts, q)
	resolvedQuery := strings.Join(contextParts, " ")
	ql := strings.ToLower(resolvedQuery)
	selected := []Meta{}
	wanted := map[string]bool{}
	// If the user names a current routine, include that routine's exercises directly.
	for _, r := range a.store.Routines {
		rt := strings.ToLower(strings.TrimSpace(r.Title))
		if rt == "" {
			continue
		}
		matched := strings.Contains(ql, rt)
		if !matched {
			words := regexp.MustCompile(`\W+`).Split(rt, -1)
			hits := 0
			for _, w := range words {
				if len(w) > 3 && strings.Contains(ql, w) {
					hits++
				}
			}
			matched = hits >= 2 || (len(words) == 1 && hits == 1)
		}
		if matched {
			for _, rx := range r.Exercises {
				if m, ok := a.store.Meta[rx.TemplateID]; ok {
					selected = append(selected, m)
				}
			}
		}
	}
	for _, m := range a.currentRoutineMetadata() {
		for _, w := range regexp.MustCompile(`\W+`).Split(strings.ToLower(m.ExerciseTitle), -1) {
			if len(w) > 3 && strings.Contains(ql, w) {
				selected = append(selected, m)
				break
			}
		}
	}
	for alias, ms := range aliases {
		if containsWord(ql, alias) {
			for _, x := range ms {
				wanted[x] = true
			}
		}
	}
	if len(wanted) > 0 {
		for _, m := range a.currentRoutineMetadata() {
			ok := false
			for _, x := range m.MajorMuscles {
				if wanted[x] {
					ok = true
				}
			}
			for _, x := range m.SupportingMuscles {
				if wanted[x] {
					ok = true
				}
			}
			if ok {
				selected = append(selected, m)
			}
		}
	}
	if len(selected) == 0 {
		for _, m := range a.currentRoutineMetadata() {
			if !(len(m.MajorMuscles) == 1 && m.MajorMuscles[0] == "Unmapped") {
				selected = append(selected, m)
				if len(selected) >= 20 {
					break
				}
			}
		}
	}
	seen := map[string]bool{}
	recs := []Rec{}
	for _, m := range selected {
		if seen[m.TemplateID] {
			continue
		}
		seen[m.TemplateID] = true
		recs = append(recs, a.recFor(m))
		if len(recs) >= 12 {
			break
		}
	}
	mus7, stim7 := a.currentRoutineStimulus(7)
	mus14, stim14 := a.currentRoutineStimulus(14)
	mus28, stim28 := a.currentRoutineStimulus(28)
	if len(wanted) > 0 {
		for _, sm := range []map[string]map[string]float64{stim7, stim14, stim28} {
			for k := range sm {
				if !wanted[k] {
					delete(sm, k)
				}
			}
		}
	}
	setHistory := map[string]any{}
	setup := []Meta{}
	for _, m := range selected {
		if seen[m.TemplateID] {
			setup = append(setup, m)
			groups := a.sessionsFor(m.TemplateID, 6)
			setHistory[m.ExerciseTitle] = groups
		}
	}
	selectedNames := []string{}
	for _, m := range setup {
		selectedNames = append(selectedNames, m.ExerciseTitle)
	}
	return map[string]any{
		"question":              q,
		"retrievalContext":      resolvedQuery,
		"resolvedExerciseFocus": selectedNames,
		"currentRoutines":       a.store.Routines,
		"selectedExerciseSetup": setup,
		"exerciseSignals":       recs,
		"selectedSetHistory":    setHistory,
		"muscleSets":            map[string]any{"7d": mus7, "14d": mus14, "28d": mus28},
		"stimulusSplit":         map[string]any{"7d": stim7, "14d": stim14, "28d": stim28},
		"interpretationRules": []string{
			func() string {
				if a.store.ProgramSource == "csv-inferred" {
					return "CSV mode: the active program is inferred from recent workout history because Hevy workout exports do not include saved routine definitions; treat those routine cards as an inference, not a direct routine sync."
				}
				return "The current Hevy routines define the active program; historical exercises outside them are context only."
			}(),
			"Major-contributor sets count as 1.0 for every major muscle; supporting-contributor sets count as 0.5 in muscle-set summaries.",
			"Execution variants should not be compared as if they are the same lift when setup materially changes leverage or recruitment.",
			"A single down exposure never triggers a program rewrite.",
			"Progression recommendations require the configured rep ceiling across the configured target sets.",
			"Load×reps and estimated 1RM are performance context, not direct measures of hypertrophy."},
	}
}
func (a *App) chatGPTBrief() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Hevy Signal — ChatGPT Training Brief\n\nGenerated: %s\n\n", time.Now().Format("2006-01-02 15:04 MST"))
	if a.store.ProgramSource == "csv-inferred" {
		b.WriteString("## Instructions for ChatGPT\n\nUse this brief as the source of truth for recent Hevy workout history and exercise setup. IMPORTANT: this brief came from a Hevy workout CSV export, which does not include saved routine definitions. The current-program routine cards below were inferred from the latest occurrence of recent workout names. Treat them as an inference to review, not as a direct Hevy routine sync. Do not invent sets, loads, reps, RPE, recovery, body composition, or technique details not present here. Treat materially different execution variants as separate lifts. One poor exposure alone is not a reason to rewrite the program. Give conversational coaching analysis rather than repeating Signal status labels.\n\n")
		b.WriteString("## Current program inferred from CSV\n\n")
	} else {
		b.WriteString("## Instructions for ChatGPT\n\nUse this brief as the source of truth for my current training program and recent Hevy performance. Saved Hevy routines define the active program; historical exercises outside them are context only. Do not invent sets, loads, reps, RPE, recovery, body composition, or technique details not present here. Distinguish programmed routine volume from completed recent volume. Treat materially different execution variants as separate lifts. One poor exposure alone is not a reason to rewrite the program. Give conversational coaching analysis rather than repeating Signal status labels.\n\n")
		b.WriteString("## Current Hevy routines\n\n")
	}
	fmt.Fprintf(&b,"**Active Hevy program:** %s\n\n",a.selectedFolderTitleLocked())
	if a.store.ActiveFolderID!="" { b.WriteString("Only workouts explicitly linked to routines in this folder count in the selected-program strength and volume totals. Older sets without routine IDs stay saved but are excluded from this program\u0027s calculations.\n\n") }
	for _, r := range a.selectedRoutinesLocked() {
		fmt.Fprintf(&b, "### %s\n", r.Title)
		for _, x := range r.Exercises {
			extra := ""
			if strings.TrimSpace(x.RepTargets) != "" {
				extra = " · " + x.RepTargets + " reps"
			}
			fmt.Fprintf(&b, "- %s — %d sets%s\n", x.ExerciseTitle, x.SetCount, extra)
		}
		b.WriteString("\n")
	}
	b.WriteString("## Exercise setup, signals, and recent exposures\n\n")
	for _, m := range a.currentRoutineMetadata() {
		r := a.recFor(m)
		fmt.Fprintf(&b, "### %s\n- Major contributors: %s\n- Supporting contributors: %s\n- Movement/stimulus: %s · %s\n- Programming role: %s · fatigue cost %d/5\n", m.ExerciseTitle, majorLabel(m), strings.Join(m.SupportingMuscles, ", "), m.Movement, m.Stimulus, m.Role, m.FatigueCost)
		if isWalkingTitle(m.ExerciseTitle) {
			fmt.Fprint(&b, "- Tracking: duration, distance, pace and perceived effort; not rep targets or strength e1RM\n")
		} else if isCarryTitle(m.ExerciseTitle) {
			fmt.Fprint(&b, "- Tracking: load, distance or duration and technique quality; not rep targets or strength e1RM\n")
		} else {
			fmt.Fprintf(&b, "- Progression target: %d–%d reps across %d sets; load jump %.2f kg\n", m.MinReps, m.MaxReps, m.TargetSets, m.IncrementKg)
		}
		if strings.TrimSpace(m.Notes) != "" {
			fmt.Fprintf(&b, "- Notes: %s\n", m.Notes)
		}
		fmt.Fprintf(&b, "- Signal: %s — %s\n- Rationale: %s\n", r.Status, r.Headline, r.Rationale)
		groups := a.sessionsFor(m.TemplateID, 4)
		if len(groups) == 0 {
			b.WriteString("- Recent exposures: none available\n\n")
			continue
		}
		b.WriteString("- Recent exposures:\n")
		for _, g := range groups {
			su := summary(g)
			date := su.Date
			if len(date) > 10 {
				date = date[:10]
			}
			if isNonRepTitle(m.ExerciseTitle) {
				fmt.Fprintf(&b, "  - %s · %s: %d logged exposures; duration/distance unavailable in current Hevy import", date, su.Workout, su.Sets)
			} else {
				fmt.Fprintf(&b, "  - %s · %s: %.1f kg × %d best set; %d sets; e1RM %.1f kg", date, su.Workout, su.BestSet[0], int(su.BestSet[1]), su.Sets, su.BestE1RM)
			}
			if su.AvgRPE != nil {
				fmt.Fprintf(&b, "; avg RPE %.1f", *su.AvgRPE)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("## Completed current-program estimated muscle exposure\n\n")
	for _, days := range []int{7, 14, 28} {
		mus, stim := a.currentRoutineStimulus(days)
		fmt.Fprintf(&b, "### Last %d days\n", days)
		for _, x := range mus {
			fmt.Fprintf(&b, "- %s: %v modeled sets (not direct measured hypertrophy)\n", asString(x["muscle"]), x["sets"])
		}
		keys := make([]string, 0, len(stim))
		for k := range stim {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			parts := []string{}
			for _, kind := range []string{"lengthened", "shortened", "mixed"} {
				if v := stim[k][kind]; v > 0 {
					parts = append(parts, fmt.Sprintf("%s %.1f", kind, v))
				}
			}
			if len(parts) > 0 {
				fmt.Fprintf(&b, "  - %s stimulus: %s\n", k, strings.Join(parts, ", "))
			}
		}
		b.WriteString("\n")
	}
	b.WriteString("## Interpretation rules\n\n- Canonical muscle groups are counted once per set (rectus femoris and vasti belong to quads); supporting exposures count at an optional 0.5 modeling weight, not a validated hypertrophy equivalence. Carries and walking do not count as standard lifting sets.\n- Execution variants should not share a strength baseline if setup materially changes leverage or recruitment.\n- Load×reps and estimated 1RM are performance context, not direct hypertrophy measures.\n- Signal labels are deterministic guardrails, not coaching conclusions.\n")
	return b.String()
}

func (a *App) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{"ok": true, "version": version})
	})
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			a.mu.RLock()
			defer a.mu.RUnlock()
			writeJSON(w, 200, map[string]any{"hevyConfigured": strings.TrimSpace(a.cfg.HevyAPIKey) != ""})
			return
		}
		var b map[string]any
		if err := readJSON(r, &b); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		a.mu.Lock()
		if v := strings.TrimSpace(asString(b["hevy_api_key"])); v != "" {
			a.cfg.HevyAPIKey = v
		}
		if x, ok := b["clear_hevy"].(bool); ok && x {
			a.cfg.HevyAPIKey = ""
		}
		err := a.saveConfig()
		hc := a.cfg.HevyAPIKey != ""
		a.mu.Unlock()
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "hevyConfigured": hc})
	})
	mux.HandleFunc("/api/hevy-test", func(w http.ResponseWriter, r *http.Request) {
		n, err := a.hevyTest()
		if err != nil {
			writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "workoutCount": n})
	})
	mux.HandleFunc("/api/sync", func(w http.ResponseWriter, r *http.Request) {
		n, rn, err := a.syncHevy()
		if err != nil {
			writeJSON(w, 502, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "synced": n, "routines": rn})
	})

	mux.HandleFunc("/api/import-hevy-csv", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, 405, map[string]any{"error": "POST required"})
			return
		}
		if err := r.ParseMultipartForm(45 << 20); err != nil {
			writeJSON(w, 400, map[string]any{"error": "Could not read upload: " + err.Error()})
			return
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, 400, map[string]any{"error": "Choose a Hevy workout CSV file first."})
			return
		}
		defer f.Close()
		n, sets, routines, err := a.importHevyCSV(f)
		if err != nil {
			writeJSON(w, 400, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "workouts": n, "sets": sets, "routines": routines, "programSource": "csv-inferred"})
	})
	mux.HandleFunc("/api/analysis", func(w http.ResponseWriter, r *http.Request) {
		a.mu.RLock()
		defer a.mu.RUnlock()
		recs := []Rec{}
		for _, m := range a.currentRoutineMetadata() {
			recs = append(recs, a.recFor(m))
		}
		mus, stim := a.currentRoutineStimulus(14)
		unmapped := 0
		for tid := range a.store.CurrentRoutineExercises {
			if m, ok := a.store.Meta[tid]; ok && len(m.MajorMuscles) == 1 && m.MajorMuscles[0] == "Unmapped" {
				unmapped++
			}
		}
		writeJSON(w, 200, map[string]any{"recs": recs, "routines": a.selectedRoutinesLocked(), "activeFolderId":a.store.ActiveFolderID,"activeFolderName":a.selectedFolderTitleLocked(), "counts": map[string]any{"workouts": len(a.store.WorkoutIDs), "sets": len(a.store.Sets), "unmapped": unmapped, "routines": len(a.selectedRoutinesLocked()), "routineExercises": len(a.store.CurrentRoutineExercises)}, "muscle": mus, "stimulus": stim, "hevyConfigured": strings.TrimSpace(a.cfg.HevyAPIKey) != "", "programSource": a.store.ProgramSource})
	})
	mux.HandleFunc("/api/program-folders",func(w http.ResponseWriter,r *http.Request){
  if r.Method!="GET" {writeJSON(w,405,map[string]any{"error":"GET required"});return}
  a.mu.RLock();defer a.mu.RUnlock()
  writeJSON(w,200,map[string]any{"folders":a.store.RoutineFolders,
    "activeFolderId":a.store.ActiveFolderID,"activeFolderName":a.selectedFolderTitleLocked(),
    "currentRoutines":len(a.selectedRoutinesLocked()),"programSource":a.store.ProgramSource,
    "hevyConfigured":strings.TrimSpace(a.cfg.HevyAPIKey)!=""})
 })
 mux.HandleFunc("/api/select-program-folder",func(w http.ResponseWriter,r *http.Request){
  if r.Method!="POST" {writeJSON(w,405,map[string]any{"error":"POST required"});return}
  var body struct{FolderID string `json:"folder_id"`}
  if err:=readJSON(r,&body);err!=nil {writeJSON(w,400,map[string]any{"error":err.Error()});return}
  a.mu.Lock();defer a.mu.Unlock()
  if a.store.ProgramSource!="hevy-api" {writeJSON(w,409,map[string]any{"error":"Program folder selection requires Hevy API sync. Workout CSV exports do not contain saved folders."});return}
  id:=strings.TrimSpace(body.FolderID)
  if id!="" {
   valid:=false
   for _,f:=range a.store.RoutineFolders {if f.ID==id {valid=true;break}}
   if !valid {writeJSON(w,404,map[string]any{"error":"Folder not available. Sync Hevy to refresh its folder list."});return}
  }
  oldID,oldExercises:=a.store.ActiveFolderID,a.store.CurrentRoutineExercises
  a.store.ActiveFolderID=id
  a.rebuildCurrentRoutineExercisesLocked()
  if err:=a.saveStore();err!=nil {
   a.store.ActiveFolderID=oldID;a.store.CurrentRoutineExercises=oldExercises
   writeJSON(w,500,map[string]any{"error":"Could not persist program folder: "+err.Error()});return
  }
  writeJSON(w,200,map[string]any{"ok":true,"activeFolderId":id,"activeFolderName":a.selectedFolderTitleLocked(),"currentRoutines":len(a.selectedRoutinesLocked())})
 })

	mux.HandleFunc("/api/routines", func(w http.ResponseWriter, r *http.Request) {
		a.mu.RLock()
		defer a.mu.RUnlock()
		writeJSON(w, 200, a.selectedRoutinesLocked())
	})

	mux.HandleFunc("/api/auto-remap", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, 405, map[string]any{"error": "POST required"})
			return
		}
		a.mu.Lock()
		defer a.mu.Unlock()
		if err := a.backupBeforeAutoRepair(); err != nil {
			writeJSON(w, 500, map[string]any{"error": "Unable to back up mappings: " + err.Error()})
			return
		}
		// Catalog suggestions must be exact-title, high-confidence wger references.
		// The client never uploads the user's workout data to wger. The source lookup
		// only retrieves a general public exercise catalog.
		var request struct {
			Suggestions []struct {
				TemplateID string   `json:"template_id"`
				Title      string   `json:"exercise_title"`
				Matched    string   `json:"matched"`
				Source     string   `json:"source"`
				Confidence string   `json:"confidence"`
				Primary    []string `json:"primary"`
				Secondary  []string `json:"secondary"`
			} `json:"suggestions"`
		}
		if err := readJSON(r, &request); err != nil {
			writeJSON(w, 400, map[string]any{"error": "Invalid mapping suggestions: " + err.Error()})
			return
		}
		changed := a.repairAutomaticMappings()
		fromReference := 0
		for index, proposal := range request.Suggestions {
			if index >= 300 {
				break
			}
			old, ok := a.store.Meta[proposal.TemplateID]
			if !ok || !isAutoEditable(old) || len(proposal.Primary) == 0 || len(proposal.Primary) > 8 || len(proposal.Secondary) > 16 {
				continue
			}
			if proposal.Source != "wger" || proposal.Confidence != "high" {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(proposal.Title), strings.TrimSpace(old.ExerciseTitle)) {
				continue
			}
			if !strings.EqualFold(strings.TrimSpace(proposal.Matched), strings.TrimSpace(old.ExerciseTitle)) {
				continue
			}
			nextPrimary := cleanMuscles(proposal.Primary)
			nextSecondary := cleanMuscles(proposal.Secondary)
			nextPrimary, nextSecondary = sanitizeAutomaticPress(old.ExerciseTitle, nextPrimary, nextSecondary)
			if len(nextPrimary) == 0 {
				continue
			}
			// Exact catalog matches are allowed to revise any auto-generated profile;
			// explicitly manual classifications are blocked above.
			if strings.Join(old.MajorMuscles, "\x1f") == strings.Join(nextPrimary, "\x1f") && strings.Join(old.SupportingMuscles, "\x1f") == strings.Join(nextSecondary, "\x1f") {
				continue
			}
			old.MajorMuscles = nextPrimary
			old.SupportingMuscles = nextSecondary
			old.AutoTagged = true
			a.store.Meta[proposal.TemplateID] = normalizeMeta(old)
			changed++
			fromReference++
		}
		manual := 0
		unknown := 0
		for _, m := range a.store.Meta {
			if !isAutoEditable(m) {
				manual++
			}
			if len(m.MajorMuscles) == 0 || (len(m.MajorMuscles) == 1 && strings.EqualFold(m.MajorMuscles[0], "Unmapped")) {
				unknown++
			}
		}
		if changed > 0 {
			if err := a.saveStore(); err != nil {
				writeJSON(w, 500, map[string]any{"error": "Unable to save remapping: " + err.Error()})
				return
			}
		}
		writeJSON(w, 200, map[string]any{"ok": true, "changed": changed, "referenceMapped": fromReference, "manualPreserved": manual, "unmapped": unknown})
	})

	mux.HandleFunc("/api/metadata", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			a.mu.RLock()
			defer a.mu.RUnlock()
			writeJSON(w, 200, a.currentRoutineMetadata())
			return
		}
		var m Meta
		if err := readJSON(r, &m); err != nil {
			writeJSON(w, 400, map[string]any{"error": err.Error()})
			return
		}
		m = normalizeMeta(m)
		a.mu.Lock()
		old, ok := a.store.Meta[m.TemplateID]
		if !ok {
			a.mu.Unlock()
			writeJSON(w, 404, map[string]any{"error": "exercise not found"})
			return
		}
		m.ExerciseTitle = old.ExerciseTitle
		m.AutoTagged = false
		a.store.Meta[m.TemplateID] = m
		err := a.saveStore()
		a.mu.Unlock()
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/about", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]any{
			"version": version, "platform": runtime.GOOS, "arch": runtime.GOARCH,
			"dataDir": a.dir, "updateChannel": "manual",
			"signed": false,
		})
	})
	mux.HandleFunc("/api/export-metadata", func(w http.ResponseWriter, r *http.Request) {
		a.mu.RLock()
		defer a.mu.RUnlock()
		md := map[string]Meta{}
		for tid, m := range a.store.Meta {
			md[tid] = normalizeMeta(m)
		}
		ex := MappingExport{FormatVersion: 1, AppVersion: version, ExportedAt: time.Now().UTC().Format(time.RFC3339), Metadata: md}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="hevy-signal-exercise-mappings.json"`)
		_ = json.NewEncoder(w).Encode(ex)
	})
	mux.HandleFunc("/api/import-metadata", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, 405, map[string]any{"error": "POST required"})
			return
		}
		var ex MappingExport
		if err := readJSON(r, &ex); err != nil {
			writeJSON(w, 400, map[string]any{"error": "Invalid mapping file: " + err.Error()})
			return
		}
		if ex.FormatVersion != 1 || ex.Metadata == nil {
			writeJSON(w, 400, map[string]any{"error": "Unsupported mapping file format."})
			return
		}
		a.mu.Lock()
		count := 0
		for tid, incoming := range ex.Metadata {
			old, ok := a.store.Meta[tid]
			if !ok {
				continue
			}
			incoming.TemplateID = tid
			incoming.ExerciseTitle = old.ExerciseTitle
			incoming.AutoTagged = false
			a.store.Meta[tid] = normalizeMeta(incoming)
			count++
		}
		err := a.saveStore()
		a.mu.Unlock()
		if err != nil {
			writeJSON(w, 500, map[string]any{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true, "imported": count})
	})
	mux.HandleFunc("/api/reset", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, 405, map[string]any{"error": "POST required"})
			return
		}
		var b map[string]any
		_ = readJSON(r, &b)
		mode := asString(b["mode"])
		a.mu.Lock()
		switch mode {
		case "data":
			a.store = Store{WorkoutIDs: map[string]string{}, Meta: map[string]Meta{}, CurrentRoutineExercises: map[string]bool{}}
			_ = a.saveStore()
		case "everything":
			a.store = Store{WorkoutIDs: map[string]string{}, Meta: map[string]Meta{}, CurrentRoutineExercises: map[string]bool{}}
			a.cfg = Config{}
			_ = a.saveStore()
			_ = a.saveConfig()
		default:
			a.mu.Unlock()
			writeJSON(w, 400, map[string]any{"error": "mode must be data or everything"})
			return
		}
		a.mu.Unlock()
		writeJSON(w, 200, map[string]any{"ok": true})
	})
	mux.HandleFunc("/api/history", func(w http.ResponseWriter, r *http.Request) {
		tid := r.URL.Query().Get("template_id")
		a.mu.RLock()
		defer a.mu.RUnlock()
		m, ok := a.store.Meta[tid]
		if !ok {
			writeJSON(w, 404, map[string]any{"error": "not found"})
			return
		}
		ss := []SessionSummary{}
		for _, g := range a.sessionsFor(tid, 12) {
			ss = append(ss, summary(g))
		}
		writeJSON(w, 200, map[string]any{"meta": m, "sessions": ss})
	})
	mux.HandleFunc("/api/chatgpt-brief", func(w http.ResponseWriter, r *http.Request) {
		a.mu.RLock()
		brief := a.chatGPTBrief()
		a.mu.RUnlock()
		if r.URL.Query().Get("download") == "1" {
			w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
			w.Header().Set("Content-Disposition", `attachment; filename="hevy-signal-chatgpt-brief.md"`)
			_, _ = io.WriteString(w, brief)
			return
		}
		writeJSON(w, 200, map[string]any{"brief": brief})
	})
	mux.HandleFunc("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			writeJSON(w, 405, map[string]any{"error": "POST required"})
			return
		}
		writeJSON(w, 200, map[string]any{"ok": true})
		go func() {
			time.Sleep(150 * time.Millisecond)
			_ = os.Remove(filepath.Join(a.dir, "pid.txt"))
			_ = os.Remove(filepath.Join(a.dir, "port.txt"))
			os.Exit(0)
		}()
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, "/")
		if p == "" {
			p = "index.html"
		}
		b, err := staticFS.ReadFile("static/" + p)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		switch filepath.Ext(p) {
		case ".css":
			w.Header().Set("Content-Type", "text/css")
		case ".js":
			w.Header().Set("Content-Type", "application/javascript")
		default:
			w.Header().Set("Content-Type", "text/html")
		}
		w.Write(b)
	})
	return mux
}

func openBrowser(u string) {
	time.Sleep(450 * time.Millisecond)
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("/usr/bin/open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	_ = cmd.Start()
}
func main() {
	stateDir := stateDirectory()
	_ = os.MkdirAll(stateDir, 0700)
	if f, err := os.OpenFile(filepath.Join(stateDir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600); err == nil {
		defer f.Close()
		log.SetOutput(f)
	}
	log.Printf("starting Hevy Signal %s (%s/%s)", version, runtime.GOOS, runtime.GOARCH)

	// On Windows, reuse an already-running copy of this exact build instead of
	// spawning multiple invisible local servers when the user double-clicks twice.
	if runtime.GOOS == "windows" {
		if b, err := os.ReadFile(filepath.Join(stateDir, "port.txt")); err == nil {
			port := strings.TrimSpace(string(b))
			if port != "" {
				u := "http://127.0.0.1:" + port
				client := http.Client{Timeout: 500 * time.Millisecond}
				if r, err := client.Get(u + "/api/health"); err == nil {
					var h map[string]any
					_ = json.NewDecoder(r.Body).Decode(&h)
					r.Body.Close()
					if r.StatusCode == 200 && h["ok"] == true && asString(h["version"]) == version {
						openBrowser(u)
						return
					}
				}
			}
		}
	}

	app := newApp()
	port := os.Getenv("HEVY_SIGNAL_PORT")
	if port == "" {
		if runtime.GOOS == "windows" {
			port = "0"
		} else {
			port = "8765"
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Printf("listen failed: %v", err)
		return
	}
	addr := ln.Addr().String()
	u := "http://" + addr
	if runtime.GOOS == "windows" {
		if _, p, ok := strings.Cut(addr, ":"); ok {
			_ = os.WriteFile(filepath.Join(stateDir, "port.txt"), []byte(p+"\n"), 0600)
		}
		_ = os.WriteFile(filepath.Join(stateDir, "pid.txt"), []byte(strconv.Itoa(os.Getpid())+"\n"), 0600)
	}
	srv := &http.Server{Handler: app.handler(), ReadHeaderTimeout: 10 * time.Second}
	log.Printf("serving at %s", u)
	if runtime.GOOS == "windows" {
		go openBrowser(u)
	}
	if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
		log.Printf("server stopped: %v", err)
	}
	if runtime.GOOS == "windows" {
		_ = os.Remove(filepath.Join(stateDir, "port.txt"))
		_ = os.Remove(filepath.Join(stateDir, "pid.txt"))
	}
}
