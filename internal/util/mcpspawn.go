package util

import (
	"os"
	"path/filepath"
	"strings"
)

// McpSpawn is the command shape written into an agent's MCP config entry.
type McpSpawn struct {
	Command string
	Args    []string
}

var pkgForBin = map[string]string{
	"context-mode": "context-mode",
	"codegraph":    "@colbymchenry/codegraph",
}

// PickMcpSpawn prefers a binary from known install roots, else falls back to
// trusted npx --no-install. It deliberately ignores arbitrary first-PATH hits so
// a project or temp directory cannot become a persisted MCP command.
func PickMcpSpawn(bin string, extraArgs ...string) McpSpawn {
	if extraArgs == nil {
		extraArgs = []string{}
	}
	if p := FindTrustedBinary(bin, nil); p != "" {
		return wrapCmdShim(McpSpawn{Command: spawnCommand(bin, p), Args: extraArgs})
	}
	pkg, ok := pkgForBin[bin]
	if !ok {
		pkg = bin
	}
	args := append([]string{"--no-install", pkg}, extraArgs...)
	cmd := "npx"
	if p := FindTrustedBinary("npx", nil); p != "" {
		cmd = spawnCommand("npx", p)
	}
	return wrapCmdShim(McpSpawn{Command: cmd, Args: args})
}

// spawnCommand picks what goes into the config.
func spawnCommand(bin, resolved string) string {
	if resolved != "" {
		return resolved
	}
	return bin
}

func wrapCmdShim(s McpSpawn) McpSpawn {
	if !IsWin {
		return s
	}
	p := s.Command
	if !filepath.IsAbs(p) {
		p = Which(s.Command)
	}
	ext := strings.ToLower(filepath.Ext(p))
	if ext != ".cmd" && ext != ".bat" {
		return s
	}
	return McpSpawn{Command: "cmd", Args: append([]string{"/c", p}, s.Args...)}
}

// WrapAutoIndex routes an MCP launch through `tokless run-mcp --agent <id>` so
// the per-project index is built/checked before the real server starts.
func WrapAutoIndex(agent string, s McpSpawn) McpSpawn {
	self, err := os.Executable()
	if err != nil {
		return s
	}
	args := append([]string{"run-mcp", "--agent", agent, s.Command}, s.Args...)
	return McpSpawn{Command: self, Args: args}
}
