package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/HoangP8/tokless/internal/core"
	"github.com/HoangP8/tokless/internal/util"
)

func TestRtkInstallPrebuiltUsesChecksumVerifiedDownload(t *testing.T) {
	if util.IsWin {
		t.Skip("unix tarball path only")
	}
	home := t.TempDir()
	util.SetHomeOverride(home)
	defer util.SetHomeOverride("")

	orig := rtkDownloadAndExtractTarGzVerified
	defer func() { rtkDownloadAndExtractTarGzVerified = orig }()

	called := false
	rtkDownloadAndExtractTarGzVerified = func(url, checksumURL, asset, dest string) error {
		called = true
		if checksumURL != rtkChecksumsURL {
			t.Fatalf("checksumURL = %q, want %q", checksumURL, rtkChecksumsURL)
		}
		if asset != "rtk-x86_64-unknown-linux-musl.tar.gz" {
			t.Fatalf("asset = %q", asset)
		}
		if err := os.MkdirAll(dest, 0o755); err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(dest, "rtk"), []byte("#!/bin/sh\necho rtk 1.2.3\n"), 0o755)
	}

	if !rtkInstallPrebuilt("rtk-x86_64-unknown-linux-musl.tar.gz", core.RunOpts{}) {
		t.Fatal("rtkInstallPrebuilt returned false")
	}
	if !called {
		t.Fatal("verified downloader was not called")
	}
}
