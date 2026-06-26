package commands

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HoangP8/tokless/internal/util"
)

func writeExecutable(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestRtkRewriteIgnoresUntrustedPathBinary(t *testing.T) {
	if util.IsWin {
		t.Skip("shell fixture is unix-only")
	}
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	untrusted := t.TempDir()
	writeExecutable(t, filepath.Join(untrusted, "rtk"), "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'rtk 1.2.3'; exit 0; fi\necho 'rtk injected'\n")
	t.Setenv("PATH", untrusted)

	if got, changed := rtkRewrite("echo hello"); changed || got != "" {
		t.Fatalf("untrusted PATH rtk should be ignored, got changed=%v command=%q", changed, got)
	}
}

func TestRunRtkHookCodexDoesNotAutoAllowRewrite(t *testing.T) {
	if util.IsWin {
		t.Skip("shell fixture is unix-only")
	}
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	rtk := filepath.Join(home, ".local", "bin", "rtk")
	writeExecutable(t, rtk, "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'rtk 1.2.3'; exit 0; fi\nif [ \"$1\" = \"rewrite\" ]; then echo 'rtk safe-rewrite'; exit 0; fi\n")
	t.Setenv("PATH", "")

	inR, inW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldStdin, oldStdout := os.Stdin, os.Stdout
	defer func() {
		os.Stdin = oldStdin
		os.Stdout = oldStdout
	}()
	os.Stdin = inR
	os.Stdout = outW
	if _, err := inW.Write([]byte(`{"tool_name":"Bash","tool_input":{"command":"echo hello"}}`)); err != nil {
		t.Fatal(err)
	}
	inW.Close()

	if code := RunRtkHookCodex(); code != 0 {
		t.Fatalf("RunRtkHookCodex exit=%d", code)
	}
	outW.Close()
	out, err := io.ReadAll(outR)
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, `"command":"rtk safe-rewrite"`) {
		t.Fatalf("rewrite output missing updated command: %s", s)
	}
	if strings.Contains(s, "permissionDecision") {
		t.Fatalf("hook must not auto-allow rewritten commands: %s", s)
	}
}

func TestResolveHookProjectDirValidatesWorkspace(t *testing.T) {
	root := t.TempDir()
	project := filepath.Join(root, "project")
	subdir := filepath.Join(project, "sub")
	other := filepath.Join(root, "other")
	for _, dir := range []string{filepath.Join(project, ".git"), subdir, filepath.Join(other, ".git")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldWd)

	input := []byte(`{"workspacePaths":["` + filepath.ToSlash(subdir) + `"]}`)
	if got := resolveHookProjectDirFromInput(input); got != project {
		t.Fatalf("workspace inside cwd project resolved to %q, want %q", got, project)
	}

	input = []byte(`{"workspacePaths":["` + filepath.ToSlash(other) + `"]}`)
	if got := resolveHookProjectDirFromInput(input); got != "" {
		t.Fatalf("workspace outside cwd project should be rejected, got %q", got)
	}
}

func TestResolveCodegraphBinIgnoresUntrustedPath(t *testing.T) {
	if util.IsWin {
		t.Skip("shell fixture is unix-only")
	}
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	untrusted := t.TempDir()
	writeExecutable(t, filepath.Join(untrusted, "codegraph"), "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'codegraph 1.2.3'; exit 0; fi\necho should-not-run\n")
	t.Setenv("PATH", untrusted)
	if got := resolveCodegraphBin(); got != "" {
		t.Fatalf("untrusted PATH codegraph should be ignored, got %q", got)
	}

	trusted := filepath.Join(home, ".local", "bin", "codegraph")
	writeExecutable(t, trusted, "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo 'codegraph 1.2.3'; exit 0; fi\n")
	if got := resolveCodegraphBin(); got != trusted {
		t.Fatalf("trusted codegraph = %q, want %q", got, trusted)
	}
}
