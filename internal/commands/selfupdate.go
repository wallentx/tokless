package commands

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/HoangP8/tokless/internal/util"
)

const owner = "HoangP8"
const repo = "tokless"
const releaseLatestURL = "https://github.com/HoangP8/tokless/releases/latest"

var selfUpdateHTTPClient = &http.Client{Timeout: 30 * time.Second}

func RunSelfUpdate() int {
	util.L.Banner("tokless self-update", "Update the tokless CLI itself")
	local := util.ToklessVersion()
	util.L.Sub("local: " + local)
	latest := fetchLatestReleaseTag()
	if latest == "" {
		util.L.Err("Could not reach GitHub Releases. Try again later.")
		printManualUpdateHint()
		return 1
	}
	util.L.Sub("latest: " + latest)
	if util.SemverGte(local, latest) {
		util.L.Ok("Already up to date.")
		return 0
	}
	asset, ok := toklessReleaseAsset(runtime.GOOS, runtime.GOARCH)
	if !ok {
		util.L.Err("No release asset for " + runtime.GOOS + "/" + runtime.GOARCH + ".")
		printManualUpdateHint()
		return 1
	}
	if util.IsWin {
		util.L.Warn("Verified self-update cannot replace the running Windows executable.")
		printManualUpdateHint()
		return 0
	}
	util.L.Step("Updating " + local + " → " + latest + "…")
	if err := installVerifiedReleaseAsset(latest, asset); err != nil {
		util.L.Err("Verified auto-update failed: " + err.Error())
		printManualUpdateHint()
		return 1
	}
	util.L.Ok("Updated to " + latest + ". Restart your shell if needed.")
	return 0
}

func printManualUpdateHint() {
	util.L.Warn("Manual update:")
	util.L.Raw("  " + util.C.Cyan("Download the release asset and SHA256SUMS from "+releaseLatestURL))
	util.L.Raw("  " + util.C.Cyan("Verify the SHA-256 digest before replacing your tokless binary."))
}

func fetchLatestReleaseTag() string {
	req, _ := http.NewRequest("GET", "https://api.github.com/repos/"+owner+"/"+repo+"/releases/latest", nil)
	req.Header.Set("User-Agent", "tokless-self-update")
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := selfUpdateHTTPClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	var j struct {
		TagName string `json:"tag_name"`
	}
	if json.NewDecoder(resp.Body).Decode(&j) != nil {
		return ""
	}
	return strings.TrimPrefix(j.TagName, "v")
}

func toklessReleaseAsset(goos, goarch string) (string, bool) {
	arch := ""
	switch goarch {
	case "amd64":
		arch = "x64"
	case "arm64":
		arch = "arm64"
	default:
		return "", false
	}
	switch goos {
	case "linux", "darwin":
		return "tokless-" + goos + "-" + arch, true
	case "windows":
		if arch != "x64" {
			return "", false
		}
		return "tokless-windows-x64.exe", true
	default:
		return "", false
	}
}

func releaseTag(tag string) string {
	if strings.HasPrefix(tag, "v") {
		return tag
	}
	return "v" + tag
}

func releaseDownloadURL(tag, asset string) string {
	return "https://github.com/" + owner + "/" + repo + "/releases/download/" + releaseTag(tag) + "/" + asset
}

func installVerifiedReleaseAsset(tag, asset string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}

	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".tokless-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	_ = tmp.Close()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := downloadReleaseFile(tag, asset, tmpPath); err != nil {
		return err
	}
	sums, err := downloadReleaseBytes(tag, "SHA256SUMS")
	if err != nil {
		return err
	}
	if err := verifyFileSHA256(tmpPath, sums, asset); err != nil {
		return err
	}
	if err := os.Chmod(tmpPath, 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, exe); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func downloadReleaseFile(tag, asset, dst string) error {
	resp, err := getReleaseAsset(tag, asset)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	n, err := io.Copy(out, resp.Body)
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%s was empty", asset)
	}
	return nil
}

func downloadReleaseBytes(tag, asset string) ([]byte, error) {
	resp, err := getReleaseAsset(tag, asset)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("%s was empty", asset)
	}
	return data, nil
}

func getReleaseAsset(tag, asset string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, releaseDownloadURL(tag, asset), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "tokless-self-update")
	resp, err := selfUpdateHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download %s failed with HTTP %d", asset, resp.StatusCode)
	}
	return resp, nil
}

func checksumFromSums(sums []byte, asset string) (string, bool) {
	for _, line := range strings.Split(string(sums), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == asset {
			return strings.ToLower(fields[0]), true
		}
	}
	return "", false
}

func verifyFileSHA256(path string, sums []byte, asset string) error {
	expected, ok := checksumFromSums(sums, asset)
	if !ok {
		return fmt.Errorf("checksum file does not list %s", asset)
	}
	if len(expected) != 64 {
		return fmt.Errorf("checksum for %s is not a SHA-256 digest", asset)
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return fmt.Errorf("checksum for %s is not hex", asset)
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := fmt.Sprintf("%x", h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch for %s", asset)
	}
	return nil
}
