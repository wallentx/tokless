package util

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestManualRtkTarball(t *testing.T) {
	if os.Getenv("TOKLESS_MANUAL_RTK") != "1" {
		t.Skip("manual: TOKLESS_MANUAL_RTK=1")
	}
	dest := t.TempDir()
	asset := "rtk-x86_64-unknown-linux-musl.tar.gz"
	url := "https://github.com/rtk-ai/rtk/releases/latest/download/" + asset
	checksumsURL := "https://github.com/rtk-ai/rtk/releases/latest/download/checksums.txt"
	if err := DownloadAndExtractTarGzVerified(url, checksumsURL, asset, dest); err != nil {
		t.Fatalf("download/extract: %v", err)
	}
	bin := filepath.Join(dest, "rtk")
	_ = os.Chmod(bin, 0o755)
	out, err := exec.Command(bin, "--version").Output()
	if err != nil {
		t.Fatalf("rtk --version: %v", err)
	}
	t.Logf("rtk %s", out)
}
