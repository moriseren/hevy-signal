#!/usr/bin/env python3
"""Patch released v3.3 desktop Go source for safe v3.4 automatic remapping."""
import re, sys
from pathlib import Path
root=Path(sys.argv[1])
src=root/"main.go"
launcher=root/"cmd/macos-launcher/main.go"
s=src.read_text()
assert "3.3-muscle-mapping" in s
s=s.replace("3.3-muscle-mapping","3.4-remap-auto")
old_start=s.index("// repairUnmapped")
old_end=s.index("func (a *App) defaultMeta(",old_start)
replacement=r'''
// isAutoEditable protects explicitly saved manual muscle classifications.
// v3.3's browser catalog accidentally marked machine-generated classifications
// as manual, recognizable by its distinctive Movement marker.
func isAutoEditable(m Meta) bool {
 unknown := len(m.MajorMuscles)==0 ||
   (len(m.MajorMuscles)==1 && strings.EqualFold(strings.TrimSpace(m.MajorMuscles[0]),"Unmapped"))
 defaultUnknown := unknown && (m.Movement=="" || m.Movement=="Unmapped")
 return m.AutoTagged || defaultUnknown ||
   strings.HasPrefix(m.Movement,"Automatically classified")
}

// repairAutomaticMappings re-evaluates ALL previously auto-tagged classifications,
// not just unmapped exercises. Never overwrites explicitly saved manual mapping
// or workout settings (reps, progression, training role, notes, stimulus).
func (a *App) repairAutomaticMappings() int {
 changed:=0
 for tid,old:=range a.store.Meta {
  if !isAutoEditable(old) || strings.TrimSpace(old.ExerciseTitle)=="" {continue}
  updated:=a.defaultMeta(tid,old.ExerciseTitle)
  if len(updated.MajorMuscles)==0 || (len(updated.MajorMuscles)==1 && updated.MajorMuscles[0]=="Unmapped") {continue}
  majorChanged:=strings.Join(old.MajorMuscles,"\x1f")!=strings.Join(updated.MajorMuscles,"\x1f")
  secondaryChanged:=strings.Join(old.SupportingMuscles,"\x1f")!=strings.Join(updated.SupportingMuscles,"\x1f")
  provenanceChanged:=!old.AutoTagged
  movementChanged:=old.Movement=="" || old.Movement=="Unmapped" || strings.HasPrefix(old.Movement,"Automatically classified")
  if !majorChanged && !secondaryChanged && !provenanceChanged && !movementChanged {continue}
  old.MajorMuscles=updated.MajorMuscles
  old.SupportingMuscles=updated.SupportingMuscles
  if movementChanged {old.Movement=updated.Movement}
  old.AutoTagged=true
  a.store.Meta[tid]=normalizeMeta(old)
  changed++
 }
 return changed
}

// Preserve the original local data.json before the first v3.4 repair, once.
// A separate backup is never included in exercise data exports.
func (a *App) backupBeforeAutoRepair() error {
 source:=filepath.Join(a.dir,"data.json")
 target:=filepath.Join(a.dir,"data-before-v3.4-remap.json")
 if _,err:=os.Stat(target);err==nil{return nil}
 data,err:=os.ReadFile(source)
 if os.IsNotExist(err){return nil}
 if err!=nil{return err}
 return os.WriteFile(target,data,0600)
}
'''
s=s[:old_start]+replacement+"\n"+s[old_end:]
anchor='a.load()\n\tif a.repairUnmapped() { _=a.saveStore() }\n\treturn a'
if anchor not in s:
 anchor='a.load()\n\tif a.repairUnmapped() { _ = a.saveStore() }\n\treturn a'
assert anchor in s,"Could not identify startup auto mapping path"
s=s.replace(anchor,'a.load()\n\tif err:=a.backupBeforeAutoRepair();err!=nil { log.Printf("auto-mapping backup failed: %v",err) } else if a.repairAutomaticMappings()>0 { _=a.saveStore() }\n\treturn a',1)
# Insert API that modifies only eligible profiles; separate from the manual-edit API.
api=r'''
 mux.HandleFunc("/api/auto-remap",func(w http.ResponseWriter,r *http.Request){
  if r.Method!="POST" {writeJSON(w,405,map[string]any{"error":"POST required"});return}
  a.mu.Lock()
  defer a.mu.Unlock()
  if err:=a.backupBeforeAutoRepair();err!=nil {
   writeJSON(w,500,map[string]any{"error":"Unable to back up mappings: "+err.Error()});return
  }
  changed:=a.repairAutomaticMappings()
  manual:=0
  unknown:=0
  for _,m:=range a.store.Meta {
   if !isAutoEditable(m) {manual++}
   if len(m.MajorMuscles)==0 || (len(m.MajorMuscles)==1 && strings.EqualFold(m.MajorMuscles[0],"Unmapped")) {unknown++}
  }
  if changed>0 {
   if err:=a.saveStore();err!=nil {
    writeJSON(w,500,map[string]any{"error":"Unable to save remapping: "+err.Error()});return
   }
  }
  writeJSON(w,200,map[string]any{"ok":true,"changed":changed,"manualPreserved":manual,"unmapped":unknown})
 })
'''
anchor='\tmux.HandleFunc("/api/metadata",'
assert s.count(anchor)==1,"Cannot add remapping handler"
s=s.replace(anchor,api+'\n'+anchor,1)
src.write_text(s)
t=launcher.read_text()
assert "3.3-muscle-mapping" in t
launcher.write_text(t.replace("3.3-muscle-mapping","3.4-remap-auto"))
html=root/"static/index.html"
h=html.read_text().replace("v3.3 PUBLIC BETA","v3.4 PUBLIC BETA")
html.write_text(h)
print("v3.4 auto-mapping patch applied; manual and legacy-auto detection included")
