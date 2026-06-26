package util

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// ExecResult mirrors the TS { code, stdout, stderr } shape.
type ExecResult struct {
	Code   int
	Stdout string
	Stderr string
}

// RunOptions controls stdio handling for Run.
type RunOptions struct {
	Capture bool
	Quiet   bool
	Cwd     string
	Env     []string
}

// Run executes a command; Capture pipes stdio, Quiet discards it, else inherit.
func Run(cmd string, args []string, opts RunOptions) ExecResult {
	c := exec.Command(cmd, args...)
	if opts.Cwd != "" {
		c.Dir = opts.Cwd
	}
	if opts.Env != nil {
		c.Env = append(os.Environ(), opts.Env...)
	}
	var outBuf, errBuf bytes.Buffer
	if opts.Capture {
		c.Stdout = &outBuf
		c.Stderr = &errBuf
	} else if opts.Quiet {
		// discard
	} else {
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Stdin = os.Stdin
	}
	err := c.Run()
	res := ExecResult{Stdout: outBuf.String(), Stderr: errBuf.String()}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			res.Code = ee.ExitCode()
		} else {
			// spawn failure (ENOENT etc.)
			res.Code = 127
			res.Stderr += err.Error()
		}
		return res
	}
	res.Code = 0
	return res
}

// Which finds an executable on PATH, honoring PATHEXT on Windows.
func Which(bin string) string {
	pathEnv := os.Getenv("PATH")
	var exts []string
	sep := ":"
	if IsWin {
		sep = ";"
		pe := os.Getenv("PATHEXT")
		if pe == "" {
			pe = ".EXE;.CMD;.BAT"
		}
		exts = strings.Split(pe, ";")
	} else {
		exts = []string{""}
	}
	for _, dir := range strings.Split(pathEnv, sep) {
		if dir == "" {
			continue
		}
		for _, ext := range exts {
			p := filepath.Join(dir, bin+ext)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return p
			}
		}
	}
	return ""
}

func FindBinary(bin string, extraDirs []string) string {
	if p := Which(bin); p != "" {
		return p
	}
	names := []string{bin}
	if IsWin {
		names = []string{bin + ".exe", bin + ".cmd", bin + ".bat", bin}
	}
	for _, dir := range extraDirs {
		if dir == "" {
			continue
		}
		for _, n := range names {
			p := filepath.Join(dir, n)
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				PrependProcessPath(dir)
				return p
			}
		}
	}
	return ""
}

// PrependProcessPath puts dir at the front of this process's PATH (idempotent).
func PrependProcessPath(dir string) {
	sep := ":"
	if IsWin {
		sep = ";"
	}
	cur := os.Getenv("PATH")
	for _, d := range strings.Split(cur, sep) {
		if d == dir {
			return
		}
	}
	os.Setenv("PATH", dir+sep+cur)
}

// WhichAny returns the first found bin and its path.
func WhichAny(bins []string) (string, string) {
	for _, b := range bins {
		if p := Which(b); p != "" {
			return b, p
		}
	}
	return "", ""
}

func pathExts() []string {
	if IsWin {
		pe := os.Getenv("PATHEXT")
		if pe == "" {
			pe = ".EXE;.CMD;.BAT"
		}
		return strings.Split(pe, ";")
	}
	return []string{""}
}

func executableInDir(dir, bin string) string {
	if dir == "" {
		return ""
	}
	for _, ext := range pathExts() {
		p := filepath.Join(dir, bin+ext)
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}
	return ""
}

func npmPrefixFromEnv() string {
	for _, k := range []string{"npm_config_prefix", "NPM_CONFIG_PREFIX"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func trustedBinDirs(extraDirs []string) []string {
	seen := map[string]bool{}
	var dirs []string
	add := func(dir string) {
		if dir == "" {
			return
		}
		clean := filepath.Clean(dir)
		if !seen[clean] {
			seen[clean] = true
			dirs = append(dirs, clean)
		}
	}
	for _, d := range extraDirs {
		add(d)
	}
	home := Home()
	if home != "" {
		add(filepath.Join(home, ".local", "bin"))
		add(filepath.Join(home, ".cargo", "bin"))
		if matches, _ := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin")); len(matches) > 0 {
			for _, d := range matches {
				add(d)
			}
		}
	}
	if prefix := npmPrefixFromEnv(); prefix != "" {
		add(npmGlobalBinDir(prefix, IsWin))
	}
	if IsWin {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			add(filepath.Join(local, "Programs", "nodejs"))
			add(filepath.Join(local, "tokless", "node"))
			add(filepath.Join(local, "tokless", "git", "cmd"))
			add(filepath.Join(local, "rtk", "bin"))
		}
		return dirs
	}
	for _, d := range []string{"/usr/local/bin", "/usr/bin", "/bin", "/opt/homebrew/bin"} {
		add(d)
	}
	if prefix := os.Getenv("PREFIX"); prefix != "" && runtime.GOOS == "android" {
		add(filepath.Join(prefix, "bin"))
	}
	return dirs
}

// FindTrustedBinary resolves bin only from expected install roots, not arbitrary
// earlier PATH entries from a project or temp directory.
func FindTrustedBinary(bin string, extraDirs []string) string {
	for _, dir := range trustedBinDirs(extraDirs) {
		if p := executableInDir(dir, bin); p != "" {
			return p
		}
	}
	return ""
}

// RtkInstallDirs returns well-known rtk install locations.
func RtkInstallDirs() []string {
	if IsWin {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			return []string{filepath.Join(local, "rtk", "bin")}
		}
		return nil
	}
	return []string{filepath.Join(Home(), ".local", "bin")}
}

// BinaryHealthy probes --version for a dot — rejects shims and 0-byte files.
func BinaryHealthy(p string) bool {
	r := Run(p, []string{"--version"}, RunOptions{Capture: true})
	return r.Code == 0 && strings.Contains(r.Stdout, ".")
}

// ResolveRtkBin finds a working rtk binary, surviving PATH drift.
func ResolveRtkBin() string {
	if p := FindTrustedBinary("rtk", RtkInstallDirs()); p != "" && BinaryHealthy(p) {
		return p
	}
	return ""
}
