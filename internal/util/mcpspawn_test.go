package util

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestPickMcpSpawnWindowsCmdShim(t *testing.T) {
	origIsWin := IsWin
	defer func() { IsWin = origIsWin }()
	home := t.TempDir()
	SetHomeOverride(home)
	defer SetHomeOverride("")

	IsWin = true

	t.Setenv("PATHEXT", ".EXE;.CMD")

	// 1. file is codegraph.cmd → must be wrapped in `cmd /c`.
	cmdDir := filepath.Join(home, ".local", "bin")
	os.MkdirAll(cmdDir, 0o755)
	os.WriteFile(filepath.Join(cmdDir, "codegraph.cmd"), []byte("dummy"), 0755)
	os.WriteFile(filepath.Join(cmdDir, "codegraph.CMD"), []byte("dummy"), 0755)

	t.Setenv("PATH", cmdDir)

	spawnCmd := PickMcpSpawn("codegraph", "serve", "--mcp")
	if spawnCmd.Command != "cmd" {
		t.Errorf("Expected Command == cmd, got %s", spawnCmd.Command)
	}
	expectedArgs := []string{"/c", FindTrustedBinary("codegraph", nil), "serve", "--mcp"}
	if !filepath.IsAbs(expectedArgs[1]) {
		t.Fatalf("test setup: trusted codegraph not absolute: %q", expectedArgs[1])
	}
	if !reflect.DeepEqual(spawnCmd.Args, expectedArgs) {
		t.Errorf("Expected Args == %v, got %v", expectedArgs, spawnCmd.Args)
	}

	// 2. file is codegraph.exe → spawned directly, no wrapper.
	os.Remove(filepath.Join(cmdDir, "codegraph.cmd"))
	os.Remove(filepath.Join(cmdDir, "codegraph.CMD"))
	exeDir := filepath.Join(home, ".cargo", "bin")
	os.MkdirAll(exeDir, 0o755)
	os.WriteFile(filepath.Join(exeDir, "codegraph.exe"), []byte("dummy"), 0755)
	os.WriteFile(filepath.Join(exeDir, "codegraph.EXE"), []byte("dummy"), 0755)
	t.Setenv("PATH", exeDir+";"+cmdDir)

	spawnExe := PickMcpSpawn("codegraph", "serve", "--mcp")
	if spawnExe.Command != FindTrustedBinary("codegraph", nil) || !filepath.IsAbs(spawnExe.Command) {
		t.Errorf("Expected Command == absolute trusted exe path %q, got %s", FindTrustedBinary("codegraph", nil), spawnExe.Command)
	}
	expectedExeArgs := []string{"serve", "--mcp"}
	if !reflect.DeepEqual(spawnExe.Args, expectedExeArgs) {
		t.Errorf("Expected Args == %v, got %v", expectedExeArgs, spawnExe.Args)
	}

	// 3. binary absent → npx fallback, npx itself is a .cmd shim → wrapped.
	os.Remove(filepath.Join(exeDir, "codegraph.exe"))
	os.Remove(filepath.Join(exeDir, "codegraph.EXE"))
	npxDir := cmdDir
	os.WriteFile(filepath.Join(npxDir, "npx.cmd"), []byte("dummy"), 0755)
	os.WriteFile(filepath.Join(npxDir, "npx.CMD"), []byte("dummy"), 0755)
	t.Setenv("PATH", npxDir)

	spawnFallback := PickMcpSpawn("codegraph")
	if spawnFallback.Command != "cmd" {
		t.Errorf("Expected fallback Command == cmd, got %s", spawnFallback.Command)
	}
	expectedFallbackArgs := []string{"/c", FindTrustedBinary("npx", nil), "--no-install", "@colbymchenry/codegraph"}
	if !filepath.IsAbs(expectedFallbackArgs[1]) {
		t.Fatalf("test setup: trusted npx not absolute: %q", expectedFallbackArgs[1])
	}
	if !reflect.DeepEqual(spawnFallback.Args, expectedFallbackArgs) {
		t.Errorf("Expected fallback Args == %v, got %v", expectedFallbackArgs, spawnFallback.Args)
	}
}

func TestPickMcpSpawnIsWinFalse(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix-semantics emulation not runnable on windows")
	}
	origIsWin := IsWin
	defer func() { IsWin = origIsWin }()
	home := t.TempDir()
	SetHomeOverride(home)
	defer SetHomeOverride("")

	IsWin = false
	tempDir := filepath.Join(home, ".local", "bin")
	os.MkdirAll(tempDir, 0o755)
	untrustedDir := t.TempDir()
	t.Setenv("PATH", untrustedDir+":"+tempDir)

	// 3. file is codegraph (chmod 0755)
	binPath := filepath.Join(tempDir, "codegraph")
	os.WriteFile(binPath, []byte("dummy"), 0755)

	spawn := PickMcpSpawn("codegraph")
	if spawn.Command != binPath {
		t.Errorf("Expected Command == %s (absolute resolved path), got %s", binPath, spawn.Command)
	}
}

func TestPickMcpSpawnIgnoresUntrustedPathHit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix path semantics")
	}
	origIsWin := IsWin
	defer func() { IsWin = origIsWin }()
	IsWin = false

	home := t.TempDir()
	SetHomeOverride(home)
	defer SetHomeOverride("")

	untrustedDir := t.TempDir()
	os.WriteFile(filepath.Join(untrustedDir, "codegraph"), []byte("dummy"), 0o755)
	trustedDir := filepath.Join(home, ".local", "bin")
	os.MkdirAll(trustedDir, 0o755)
	trustedNpx := filepath.Join(trustedDir, "npx")
	os.WriteFile(trustedNpx, []byte("dummy"), 0o755)
	t.Setenv("PATH", untrustedDir+":"+trustedDir)

	spawn := PickMcpSpawn("codegraph", "serve", "--mcp")
	if spawn.Command != trustedNpx {
		t.Fatalf("expected untrusted codegraph to be ignored in favor of trusted npx, got %#v", spawn)
	}
	wantArgs := []string{"--no-install", "@colbymchenry/codegraph", "serve", "--mcp"}
	if !reflect.DeepEqual(spawn.Args, wantArgs) {
		t.Fatalf("args = %v, want %v", spawn.Args, wantArgs)
	}
}
