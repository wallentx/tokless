package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/wallentx/tokless/internal/util"
)

// relocateCavemanSkills must move staged skills out of the shared canonical
// ~/.agents/skills into the agent's own dir — selected agent only, no leftovers.
func TestRelocateCavemanSkills(t *testing.T) {
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	staging := filepath.Join(home, ".agents", "skills")
	for _, name := range cavemanSkillNames {
		os.MkdirAll(filepath.Join(staging, name), 0o755)
		os.WriteFile(filepath.Join(staging, name, "SKILL.md"), []byte("x"), 0o644)
	}
	os.MkdirAll(filepath.Join(staging, "my-skill"), 0o755)

	dst := filepath.Join(home, ".codex", "skills")
	relocateCavemanSkills(dst)

	for _, name := range cavemanSkillNames {
		if !util.Exists(filepath.Join(dst, name, "SKILL.md")) {
			t.Fatalf("%s not relocated to agent dir", name)
		}
		if util.Exists(filepath.Join(staging, name)) {
			t.Fatalf("%s left behind in canonical staging", name)
		}
	}
	if !util.Exists(filepath.Join(staging, "my-skill")) {
		t.Fatal("unrelated skill in ~/.agents/skills must not be touched")
	}
}

func TestRemoveCavemanSkillCopies(t *testing.T) {
	dir := t.TempDir()
	for _, name := range cavemanSkillNames {
		os.MkdirAll(filepath.Join(dir, name), 0o755)
	}
	os.MkdirAll(filepath.Join(dir, "my-skill"), 0o755)

	removeCavemanSkillCopies(dir)

	for _, name := range cavemanSkillNames {
		if util.Exists(filepath.Join(dir, name)) {
			t.Fatalf("%s not removed", name)
		}
	}
	if !util.Exists(filepath.Join(dir, "my-skill")) {
		t.Fatal("unrelated skill must survive unwire")
	}
}
