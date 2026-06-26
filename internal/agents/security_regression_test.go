package agents

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HoangP8/tokless/internal/util"
)

func TestCodexWiringPreservesApprovalPolicy(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("CODEX_HOME", filepath.Join(dir, ".codex"))
	p := util.CodexPathsResolved()
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Config, []byte("approval_policy = \"on-request\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ConfigureCodexMcp("context-mode")
	InstallCodexRtkHook()

	raw, err := os.ReadFile(p.Config)
	if err != nil {
		t.Fatal(err)
	}
	cfg := string(raw)
	if !strings.Contains(cfg, "approval_policy = \"on-request\"") {
		t.Fatalf("existing approval policy was not preserved:\n%s", cfg)
	}
	if strings.Contains(cfg, "approval_policy = \"never\"") {
		t.Fatalf("tokless must not force global Codex approval_policy=never:\n%s", cfg)
	}
}

func TestClaudeMcpUninstallRemovesOwnedAllowEntry(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	util.SetHomeOverride(dir)
	defer util.SetHomeOverride("")

	p := util.ClaudeCodePaths()
	if err := os.MkdirAll(p.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.GlobalJSON, []byte(`{"mcpServers":{"context-mode":{"type":"stdio"},"other":{"type":"stdio"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p.Settings, []byte(`{"permissions":{"allow":["mcp__context-mode__.*","mcp__other__.*","Bash(echo *)"]}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	if !RemoveClaudeMcp("context-mode") {
		t.Fatal("RemoveClaudeMcp returned false")
	}

	global, err := os.ReadFile(p.GlobalJSON)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(global), "context-mode") {
		t.Fatalf("context-mode MCP server was not removed:\n%s", string(global))
	}
	if !strings.Contains(string(global), "other") {
		t.Fatalf("unrelated MCP server was removed:\n%s", string(global))
	}

	settings, err := os.ReadFile(p.Settings)
	if err != nil {
		t.Fatal(err)
	}
	s := string(settings)
	if strings.Contains(s, "mcp__context-mode__.*") {
		t.Fatalf("owned Claude allow entry was not removed:\n%s", s)
	}
	if !strings.Contains(s, "mcp__other__.*") || !strings.Contains(s, "Bash(echo *)") {
		t.Fatalf("unrelated Claude allow entries were not preserved:\n%s", s)
	}
}

func TestAntigravityRtkPermissionOwnedByRtkHook(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	util.SetHomeOverride(dir)
	defer util.SetHomeOverride("")

	ConfigureAntigravityMcp("context-mode")
	for _, settings := range antigravitySettingsFiles() {
		raw, err := os.ReadFile(settings)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "command(rtk)") {
			t.Fatalf("generic MCP setup should not allow command(rtk):\n%s", string(raw))
		}
	}

	InstallAntigravityRtkHook()
	for _, settings := range antigravitySettingsFiles() {
		raw, err := os.ReadFile(settings)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), "command(rtk)") {
			t.Fatalf("RTK hook setup did not allow command(rtk):\n%s", string(raw))
		}
	}

	RemoveAntigravityRtkHook()
	for _, settings := range antigravitySettingsFiles() {
		raw, err := os.ReadFile(settings)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "command(rtk)") {
			t.Fatalf("RTK hook teardown did not remove command(rtk):\n%s", string(raw))
		}
		if !strings.Contains(string(raw), "mcp(context-mode/*)") {
			t.Fatalf("RTK teardown removed unrelated MCP permission:\n%s", string(raw))
		}
	}
}
