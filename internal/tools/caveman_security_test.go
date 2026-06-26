package tools

import (
	"strings"
	"testing"

	"github.com/HoangP8/tokless/internal/core"
	"github.com/HoangP8/tokless/internal/util"
)

func TestCavemanExecBlocksMutableNpxByDefault(t *testing.T) {
	t.Setenv("TOKLESS_TEST", "")
	t.Setenv(allowMutableInstallersEnv, "")
	t.Setenv("TOKLESS_ALLOW_MUTABLE_NPX", "")

	ran, err := cavemanExec("npx", []string{"-y", "skills", "add", "JuliusBrussee/caveman"}, core.RunOpts{}, "")
	if err == nil {
		t.Fatal("expected mutable npx installer to be blocked")
	}
	if ran {
		t.Fatal("blocked command reported as run")
	}
	if !strings.Contains(err.Error(), allowMutableInstallersEnv) {
		t.Fatalf("error should name opt-in env, got %v", err)
	}
}

func TestCavemanExecBlocksClaudePluginInstallByDefault(t *testing.T) {
	t.Setenv("TOKLESS_TEST", "")
	t.Setenv(allowMutableInstallersEnv, "")
	t.Setenv("TOKLESS_ALLOW_MUTABLE_NPX", "")

	ran, err := cavemanExec("claude", []string{"plugin", "marketplace", "add", "JuliusBrussee/caveman"}, core.RunOpts{}, "")
	if err == nil {
		t.Fatal("expected mutable Claude plugin installer to be blocked")
	}
	if ran {
		t.Fatal("blocked command reported as run")
	}
}

func TestCavemanExecRunsMutableInstallerWithOptIn(t *testing.T) {
	t.Setenv("TOKLESS_TEST", "")
	t.Setenv(allowMutableInstallersEnv, "1")

	orig := cavemanRun
	defer func() { cavemanRun = orig }()

	called := false
	cavemanRun = func(cmd string, args []string, opts util.RunOptions) util.ExecResult {
		called = true
		if cmd != "npx" {
			t.Fatalf("cmd = %q, want npx", cmd)
		}
		if !opts.Capture {
			t.Fatal("caveman command should capture output")
		}
		return util.ExecResult{Code: 0}
	}

	ran, err := cavemanExec("npx", []string{"-y", "skills", "add", "JuliusBrussee/caveman"}, core.RunOpts{}, "")
	if err != nil || !ran {
		t.Fatalf("expected opt-in command to run, ran=%v err=%v", ran, err)
	}
	if !called {
		t.Fatal("runner was not called")
	}
}
