package tools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wallentx/tokless/internal/core"
	"github.com/wallentx/tokless/internal/util"
)

func TestContextModeUnwirePreservesProjectGeminiMd(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	util.SetHomeOverride(dir)
	defer util.SetHomeOverride("")

	project := filepath.Join(dir, "project")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	gemini := filepath.Join(project, "GEMINI.md")
	content := "# Project instructions\n\nKeep this file.\n\n" + ctxGeminiMarker + "\n\nProject-specific safety text.\n"
	if err := os.WriteFile(gemini, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	if ok, err := ctxUnwireAntigravity(core.RunOpts{}); !ok || err != nil {
		t.Fatalf("ctxUnwireAntigravity ok=%v err=%v", ok, err)
	}

	raw, err := os.ReadFile(gemini)
	if err != nil {
		t.Fatalf("project GEMINI.md should be preserved: %v", err)
	}
	if !strings.Contains(string(raw), "Project-specific safety text") {
		t.Fatalf("project GEMINI.md content was not preserved:\n%s", string(raw))
	}
}
