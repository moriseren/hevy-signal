#!/usr/bin/env python3
"""Scientific-interpretation safeguards for Hevy Signal v3.5 desktop."""
import sys,re
from pathlib import Path
root=Path(sys.argv[1])
main=root/"main.go"; launcher=root/"cmd/macos-launcher/main.go";html=root/"static/index.html"
s=main.read_text()
assert '3.4-remap-auto' in s
s=s.replace('3.4-remap-auto','3.5-volume-validation')
extra=r'''
// Muscle-volume totals represent broad muscle groups, not summed anatomical
// subheads. Rectus femoris and the vasti are parts of the quadriceps, never
// separate additional quadriceps-set exposures.
func canonicalVolumeMuscle(s string) string {
 k:=strings.ToLower(strings.TrimSpace(s))
 if k=="" || k=="unmapped" {return ""}
 if strings.Contains(k,"rectus femoris") || strings.Contains(k,"vastus") ||
    strings.Contains(k,"quadricep") || strings.HasPrefix(k,"quads") {return "Quads"}
 if strings.Contains(k,"pectoralis major") || k=="pecs" || k=="chest" {return "Chest"}
 if strings.Contains(k,"triceps brachii") {return "Triceps"}
 if strings.Contains(k,"biceps brachii") {return "Biceps"}
 if strings.Contains(k,"deltoid") && strings.Contains(k,"anterior") {return "Front delts"}
 return strings.TrimSpace(s)
}
func isWalkingTitle(title string) bool {
 t:=strings.ToLower(strings.TrimSpace(title))
 return t=="walking"||t=="walk"||strings.HasPrefix(t,"treadmill walk")||strings.HasPrefix(t,"brisk walk")
}
func isCarryTitle(title string) bool {
 t:=strings.ToLower(strings.TrimSpace(title))
 return strings.Contains(t,"carry")||strings.Contains(t,"farmers walk")||
 strings.Contains(t,"farmer's walk")||strings.Contains(t,"farmer walk")||strings.Contains(t,"suitcase walk")
}
func isNonRepTitle(title string) bool {return isWalkingTitle(title)||isCarryTitle(title)}
func isElbowExtensionPress(title string) bool {
 t:=strings.ToLower(title)
 return strings.Contains(t,"chest press")||strings.Contains(t,"bench press")||
 strings.Contains(t,"shoulder press")||strings.Contains(t,"overhead press")||
 strings.Contains(t,"military press")||strings.Contains(t,"arnold press")
}
func containsBiceps(parts []string) bool {
 for _,p:=range parts {if strings.Contains(strings.ToLower(p),"bicep"){return true}}
 return false
}
func hasMuscle(parts []string,canonical string) bool {
 for _,p:=range parts {if canonicalVolumeMuscle(p)==canonical{return true}}
 return false
}
// Applies only to automatically generated mappings. A user's manually saved
// classifications remain unchanged, even when Signal suggests reviewing them.
// Keep quadriceps subdivisions as a single group in auto-derived exercise
// anatomy, not only in the summary math. Manual anatomical notes remain intact.
func canonicalAutoParts(major,secondary []string) ([]string,[]string) {
 cleanedPrimary:=[]string{}
 primary:=map[string]bool{}
 for _,p:=range major {
  label:=p
  if canonicalVolumeMuscle(p)=="Quads" {label="Quads"}
  if label==""||primary[label] {continue}
  primary[label]=true
  cleanedPrimary=append(cleanedPrimary,label)
 }
 cleanedSecondary:=[]string{}
 seen:=map[string]bool{}
 for _,p:=range secondary {
  label:=p
  if canonicalVolumeMuscle(p)=="Quads" {label="Quads"}
  if label==""||primary[label]||seen[label] {continue}
  seen[label]=true
  cleanedSecondary=append(cleanedSecondary,label)
 }
 return cleanedPrimary,cleanedSecondary
}
func sanitizeAutomaticPress(title string, major, secondary []string) ([]string,[]string) {
 if !isElbowExtensionPress(title){return canonicalAutoParts(major,secondary)}
 filtered:=[]string{}
 for _,p:=range secondary {if !strings.Contains(strings.ToLower(p),"bicep"){filtered=append(filtered,p)}}
 if !hasMuscle(major,"Triceps") && !hasMuscle(filtered,"Triceps") {
  filtered=append(filtered,"Triceps")
 }
 if strings.Contains(strings.ToLower(title),"chest press") || strings.Contains(strings.ToLower(title),"bench press") {
  if !hasMuscle(major,"Chest")&&!hasMuscle(filtered,"Chest") {major=append(major,"Chest")}
 }
 return canonicalAutoParts(major,filtered)
}
// One exercise set contributes at most one direct set to one canonical group;
// secondary muscle exposure is modeled at 0.5, never added to an already
// directly trained canonical group for that same exercise set.
func addMuscleExposure(m Meta, mus map[string]float64, stim map[string]map[string]float64) {
 if isNonRepTitle(m.ExerciseTitle) {return} // carries/cardio use separate training metrics
 primary:=map[string]bool{}
 for _,raw:=range m.MajorMuscles {
  p:=canonicalVolumeMuscle(raw)
  if p==""||primary[p]{continue}
  primary[p]=true
  mus[p]++
  if stim[p]==nil{stim[p]=map[string]float64{"lengthened":0,"shortened":0,"mixed":0}}
  kind:=m.Stimulus
  if _,ok:=stim[p][kind];!ok{kind="mixed"}
  stim[p][kind]++
 }
 secondary:=map[string]bool{}
 for _,raw:=range m.SupportingMuscles {
  sec:=canonicalVolumeMuscle(raw)
  if sec==""||primary[sec]||secondary[sec]{continue}
  secondary[sec]=true
  mus[sec]+=.5
 }
}
// Large discontinuities in a named exercise's historical loads can signal
// that two machines, leverage configurations or execution variants were
// logged under the same Hevy template. Block comparisons until verified.
func suspectLoadVariant(ss [][]SetRow) bool {
 minKg,maxKg:=0.0,0.0
 sessions:=0
 for _,session:=range ss {
  best:=0.0
  for _,row:=range session {
   if row.HasWeight && row.WeightKg>best {best=row.WeightKg}
  }
  if best<=0{continue}
  sessions++
  if minKg==0||best<minKg {minKg=best}
  if best>maxKg {maxKg=best}
 }
 return sessions>=2 && minKg>0 && maxKg/minKg>=2.25
}
func shouldVerifyLoadHistory(title string) bool {
 t:=strings.ToLower(title)
 return strings.Contains(t,"abduct")||strings.Contains(t,"kickback")
}
'''
anchor='func (a *App) recFor(m Meta) Rec {'
assert s.count(anchor)==1
s=s.replace(anchor,extra+'\n'+anchor,1)
anchor='\tif len(ss) < 2 {'
# only modify within recFor first!
at=s.index('func (a *App) recFor(m Meta) Rec {'); end=s.index('func parseTime(',at)
sub=s[at:end]
assert anchor in sub
checks=r'''
 if isWalkingTitle(m.ExerciseTitle) {
  return set("tracking","Track walking by time and pace","Use duration, distance, pace, and perceived effort for walking. Hevy's rep-based entries cannot establish walking progression; no load/reps or e1RM recommendation is issued.",.9)
 }
 if isCarryTitle(m.ExerciseTitle) {
  return set("tracking","Track carry distance, duration and load","Loaded carries should progress using distance, time, implement load, control and posture; the existing workout export lacks reliable distance/time details, so rep-based advice and e1RM are suppressed.",.9)
 }
 if isElbowExtensionPress(m.ExerciseTitle) && (containsBiceps(m.SupportingMuscles)||containsBiceps(m.MajorMuscles)) {
  return set("review","Review pressing muscle classification","Biceps are listed for a conventional press where triceps provide elbow extension. If this was manually mapped, review it under Exercise setup; Signal will not overwrite your edit.",.95)
 }
 if shouldVerifyLoadHistory(m.ExerciseTitle) && suspectLoadVariant(a.sessionsFor(m.TemplateID,12)) {
  return set("review","Verify machine or movement variant","Historical loads for the same exercise name differ by more than 2.25-fold. Verify the cable stack, machine, unilateral vs bilateral loading, setup and execution. Separate incomparable variants into distinct Hevy exercise templates before interpreting strength trends; no progression recommendation is issued.",.95)
 }
'''
sub=sub.replace(anchor,checks+'\n'+anchor,1);s=s[:at]+sub+s[end:]
# Aggregate both timeline methods through one canonicalized set-level function.
for name in ['stimulus','currentRoutineStimulus']:
 start=s.index('func (a *App) '+name+'(days int)')
 end=s.index('\nfunc ',start+10)
 part=s[start:end]
 a=part.index('\t\tfor _, p := range m.MajorMuscles {')
 b=part.index('\n\t}\n\ttype kv struct',a)
 part=part[:a]+'\t\taddMuscleExposure(m,mus,stim)'+part[b:]
 s=s[:start]+part+s[end:]
# Labels must not present rectus femoris separately as a second quad target.
major='return strings.Join(m.MajorMuscles, " + ")'
assert major in s
s=s.replace(major,'''seen:=map[string]bool{}; labels:=[]string{}
 for _,raw:=range m.MajorMuscles {label:=canonicalVolumeMuscle(raw); if label!=""&&!seen[label]{seen[label]=true;labels=append(labels,label)}}
 if len(labels)==0{return "Unmapped"}
 return strings.Join(labels," + ")''',1)
# The older automatic presets and wger matches are cleaned, preserving manual overrides.
old='updated:=a.defaultMeta(tid,old.ExerciseTitle)'
assert old in s
s=s.replace(old,'''updated:=a.defaultMeta(tid,old.ExerciseTitle)
  updated.MajorMuscles,updated.SupportingMuscles=sanitizeAutomaticPress(old.ExerciseTitle,updated.MajorMuscles,updated.SupportingMuscles)''',1)
anchor='nextPrimary:=cleanMuscles(proposal.Primary)'
assert anchor in s
s=s.replace('nextSecondary:=cleanMuscles(proposal.Secondary)',
'''nextSecondary:=cleanMuscles(proposal.Secondary)
   nextPrimary,nextSecondary=sanitizeAutomaticPress(old.ExerciseTitle,nextPrimary,nextSecondary)''',1)
# Also sanitize fresh imported Hevy exercise templates / inferred routines.
anchor='m.AutoTagged = true\n\t\t\treturn m'
if anchor in s:s=s.replace(anchor,'m.AutoTagged = true\n\t\t\tm.MajorMuscles,m.SupportingMuscles=sanitizeAutomaticPress(title,m.MajorMuscles,m.SupportingMuscles)\n\t\t\treturn m',1)
# The training brief must not issue rep prescriptions or e1RMs for carries/walking.
pattern=r'(?m)^(\s*)fmt.Fprintf\(&b, "### %s\\n- Major contributors: %s\\n- Supporting contributors: %s\\n- Movement/stimulus: %s · %s\\n- Programming role: %s · fatigue cost %d/5\\n- Progression target: %d–%d reps across %d sets; load jump %.2f kg\\n",.*\)$'
match=re.search(pattern,s)
assert match,"Failed to identify existing chat brief progression line"
indent=match.group(1)
new=r'''fmt.Fprintf(&b,"### %s\n- Major contributors: %s\n- Supporting contributors: %s\n- Movement/stimulus: %s · %s\n- Programming role: %s · fatigue cost %d/5\n",m.ExerciseTitle,majorLabel(m),strings.Join(m.SupportingMuscles,", "),m.Movement,m.Stimulus,m.Role,m.FatigueCost)
if isWalkingTitle(m.ExerciseTitle) {
 fmt.Fprint(&b,"- Tracking: duration, distance, pace and perceived effort; not rep targets or strength e1RM\n")
} else if isCarryTitle(m.ExerciseTitle) {
 fmt.Fprint(&b,"- Tracking: load, distance or duration and technique quality; not rep targets or strength e1RM\n")
} else {
 fmt.Fprintf(&b,"- Progression target: %d–%d reps across %d sets; load jump %.2f kg\n",m.MinReps,m.MaxReps,m.TargetSets,m.IncrementKg)
}'''
s=s[:match.start()]+indent+new.replace('\n','\n'+indent)+s[match.end():]
needle='fmt.Fprintf(&b, "  - %s · %s: %.1f kg × %d best set; %d sets; e1RM %.1f kg", date, su.Workout, su.BestSet[0], int(su.BestSet[1]), su.Sets, su.BestE1RM)'
assert needle in s
s=s.replace(needle,'''if isNonRepTitle(m.ExerciseTitle) {
   fmt.Fprintf(&b,"  - %s · %s: %d logged exposures; duration/distance unavailable in current Hevy import",date,su.Workout,su.Sets)
  } else {
   fmt.Fprintf(&b, "  - %s · %s: %.1f kg × %d best set; %d sets; e1RM %.1f kg", date, su.Workout, su.BestSet[0], int(su.BestSet[1]), su.Sets, su.BestE1RM)
  }''',1)
# Don't repeat misinterpreted subgroup/equal-growth-factor assumptions.
s=s.replace('Major-contributor sets count as 1.0 effective set for each major muscle; supporting-contributor sets count as 0.5.',
 'Canonical muscle groups are counted once per set (rectus femoris and vasti belong to quads); supporting exposures count at an optional 0.5 modeling weight, not a validated hypertrophy equivalence. Carries and walking do not count as standard lifting sets.')
s=s.replace('a single down exposure never triggers a program rewrite.','a single down exposure never triggers a program rewrite.')
# Strong warning next to per-exercise signal rationale.
s=s.replace('b.WriteString("## Completed current-program effective volume\\n\\n")',
 '''b.WriteString("## Completed current-program estimated muscle exposure\\n\\n")''')
s=s.replace('fmt.Fprintf(&b, "- %s: %v effective sets\\n", asString(x["muscle"]), x["sets"])',
 '''fmt.Fprintf(&b, "- %s: %v modeled sets (not direct measured hypertrophy)\\n", asString(x["muscle"]), x["sets"])''')
src=main
main.write_text(s)
l=launcher.read_text(); assert '3.4-remap-auto' in l
launcher.write_text(l.replace('3.4-remap-auto','3.5-volume-validation'))
h=html.read_text()
h=h.replace('v3.4 PUBLIC BETA','v3.5 PUBLIC BETA')
h=h.replace('Major contributor = 1 set','Major = 1 estimated set').replace('Supporting contributor = 0.5 set','Supporting = 0.5 modeled set (not validated equivalent)')
html.write_text(h)
print('Applied v3.5: anatomy, canonical quad volumes, non-rep progression, setup verification')
