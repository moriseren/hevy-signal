#!/usr/bin/env python3
"""Turn released v3.2 Go source into a v3.3 desktop build."""
import sys, re
from pathlib import Path
root=Path(sys.argv[1])
p=root/"main.go"; launcher=root/"cmd/macos-launcher/main.go"
s=p.read_text()
s,n=re.subn(r'(version\s*=\s*")3\.2-public-beta(")',r'\g<1>3.3-muscle-mapping\2',s)
assert n>=1,"Unknown backend version"
# Prioritized exercise heuristics; confidence remains estimated.
rules=[
 ("archer pull","Rear delts","Mid traps/rhomboids, External rotators","Horizontal pull"),
 ("serratus press","Serratus anterior","Chest","Scapular protraction"),
 ("scapular (pull.?down|depression)|straight.arm scapular","Lower traps, Lats","Pectoralis minor","Scapular depression"),
 ("suitcase front carry","Obliques, Abs","Forearms, Front delts","Loaded carry"),
 ("suitcase (carry|walk|hold)","Obliques, Abs","Forearms, Quadratus lumborum","Loaded carry"),
 ("farmer.?s? (walk|carry|hold)","Upper traps, Forearms","Abs","Loaded carry"),
 (r"^(walking|walk|treadmill walk|brisk walk)(\s*\(.*\))?$","Cardio / walking","","Cardio"),
 ("triceps? kickback|skull.?crusher|triceps? pushdown|triceps? extension","Triceps","","Elbow extension"),
 ("glute kickback|cable kickback|donkey kick|hip extension","Glute max","Hamstrings","Hip extension"),
 ("abduct|clamshell|lateral band walk|monster walk","Glute med/min","Glute max","Hip abduction"),
 ("adduct|copenhagen","Adductors","","Hip adduction"),
 ("hip thrust|glute bridge|glute drive|frog pump","Glute max","Hamstrings","Hip extension"),
 ("leg extension|reverse nordic","Quads","Rectus femoris","Knee extension"),
 ("leg curl|hamstring curl|nordic curl|glute ham raise","Hamstrings","","Knee flexion"),
 (r"romanian deadlift|\brdl\b|stiff.?leg|good morning|cable rdl","Hamstrings, Glute max","Erectors","Hip hinge"),
 ("reverse hyper|cable pull through","Glute max","Hamstrings, Erectors","Hip extension"),
 ("back extension|hyperextension|45.degree","Glute max, Spinal erectors","Hamstrings","Hip extension"),
 ("sumo deadlift","Glute max, Adductors","Quads, Erectors","Hip hinge"),
 ("deadlift|rack pull|trap bar","Glute max, Erectors","Hamstrings, Quads","Hip hinge"),
 ("bulgarian|split squat|lunge|step.?up","Quads, Glute max","Adductors","Lunge"),
 (r"leg press|hack squat|pendulum|belt squat|front squat|back squat|goblet squat|smith squat|\bsquat\b","Quads, Glute max","Adductors","Squat/press"),
 ("calf raise|calf press|soleus raise|toe press","Calves","","Plantar flexion"),
 ("face pull|reverse fly|reverse pec|rear delt","Rear delts","Mid traps/rhomboids","Horizontal abduction"),
 ("lateral raise|side raise|cable y.?raise|scaption","Side delts","Upper traps","Shoulder abduction"),
 ("front raise","Front delts","","Shoulder flexion"),
 ("shrug|high pull","Upper traps","","Scapular elevation"),
 ("upright row","Side delts, Upper traps","Biceps","Shoulder abduction"),
 ("shoulder press|overhead press|arnold press|military press|landmine press","Front delts","Side delts, Triceps","Shoulder press"),
 ("chest.supported row|high row|wide.grip row|upper.back row","Mid traps/rhomboids, Rear delts","Lats, Biceps","Upper-back row"),
 (r"\brow\b","Lats, Mid traps/rhomboids","Rear delts, Biceps","Horizontal pull"),
 ("lat pulldown|pull.?down|pull.?up|chin.?up|pullover|lat prayer","Lats","Biceps","Vertical pull"),
 ("close.grip bench|jm press|tricep dip|bench dip","Triceps","Chest","Elbow extension"),
 (r"chest dip|parallel bar dip|\bdips?\b","Chest, Triceps","Front delts","Push"),
 ("incline (bench|chest|dumbbell|barbell|press)","Upper chest","Front delts, Triceps","Push"),
 ("bench|chest press|push.?up|pec.?deck|chest fly|cable fly|dumbbell fly|butterfly","Chest","Front delts, Triceps","Push"),
 (r"wrist curl|wrist extension|reverse curl|wrist roller","Forearms","","Forearm"),
 (r"hammer curl|preacher curl|bicep|barbell curl|dumbbell curl|cable curl|incline curl|concentration curl|\bcurl\b","Biceps","Brachialis/brachioradialis","Elbow flexion"),
 ("pallof|woodchop|wood chop|russian twist|side plank|oblique","Obliques","Abs","Core"),
 (r"crunch|sit.?up|leg raise|knee raise|ab wheel|\bplank\b|dead bug|hollow hold|abdominal","Abs","Hip flexors","Core")
]
def vals(x):return "[]string{"+", ".join('"'+v.strip()+'"' for v in x.split(",") if v.strip())+"}"
lines=[]
for pat,primary,secondary,movement in rules:
 lines.append(' {regexp.MustCompile(__TICK__(?i)'+pat+'__TICK__), Meta{MajorMuscles: '+vals(primary)+', SupportingMuscles: '+vals(secondary)+', Movement: "'+movement+'", Stimulus: "mixed", FatigueCost: 2, Role: "accessory", MinReps: 8, MaxReps: 15, TargetSets: 3, IncrementKg: 2.5}},')
anchor="var presets = []preset{"
assert s.count(anchor)==1
s=s.replace(anchor,anchor+"\n"+"\n".join(lines))
s=s.replace('__TICK__(?i)kickback__TICK__','__TICK__(?i)glute kickback|cable kickback|donkey kick__TICK__')
repair="""
// repairUnmapped repairs only unknown classifications; user edits are untouched.
func (a *App) repairUnmapped() bool {
 changed:=false
 for tid,old:=range a.store.Meta {
  if len(old.MajorMuscles)>0 && !(len(old.MajorMuscles)==1 && strings.EqualFold(strings.TrimSpace(old.MajorMuscles[0]),"Unmapped")) {continue}
  if strings.TrimSpace(old.ExerciseTitle)=="" {continue}
  next:=a.defaultMeta(tid,old.ExerciseTitle)
  if len(next.MajorMuscles)==0 || (len(next.MajorMuscles)==1 && next.MajorMuscles[0]=="Unmapped") {continue}
  old.MajorMuscles=next.MajorMuscles
  old.SupportingMuscles=next.SupportingMuscles
  if old.Movement=="" || old.Movement=="Unmapped" {old.Movement=next.Movement}
  old.AutoTagged=true
  a.store.Meta[tid]=normalizeMeta(old)
  changed=true
 }
 return changed
}
"""
anchor="func (a *App) defaultMeta(tid, title string) Meta {"
assert s.count(anchor)==1
s=s.replace(anchor,repair+"\n"+anchor)
assert "\ta.load()\n\treturn a" in s
s=s.replace("\ta.load()\n\treturn a","\ta.load()\n\tif a.repairUnmapped() { _=a.saveStore() }\n\treturn a",1)
s=s.replace('__TICK__',chr(96))
p.write_text(s)
t=launcher.read_text();assert '3.2-public-beta' in t
launcher.write_text(t.replace('3.2-public-beta','3.3-muscle-mapping'))
html=root/"static/index.html";h=html.read_text();assert "</body>" in h
h=h.replace("</body>",'<script src="/muscle-engine.js"></script><script src="/auto-map.js"></script></body>')
h=h.replace('PUBLIC BETA','v3.3 PUBLIC BETA')
html.write_text(h)
print("Patched v3.3 source:",len(lines),"movement rules and repair")
