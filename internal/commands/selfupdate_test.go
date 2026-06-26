package commands

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestToklessReleaseAsset(t *testing.T) {
	tests := []struct {
		goos, goarch string
		want         string
		ok           bool
	}{
		{"linux", "amd64", "tokless-linux-x64", true},
		{"linux", "arm64", "tokless-linux-arm64", true},
		{"darwin", "amd64", "tokless-darwin-x64", true},
		{"darwin", "arm64", "tokless-darwin-arm64", true},
		{"windows", "amd64", "tokless-windows-x64.exe", true},
		{"windows", "arm64", "", false},
		{"plan9", "amd64", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.goos+"/"+tt.goarch, func(t *testing.T) {
			got, ok := toklessReleaseAsset(tt.goos, tt.goarch)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("got %q/%v, want %q/%v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

func TestReleaseDownloadURLUsesPinnedTag(t *testing.T) {
	got := releaseDownloadURL("0.1.41", "tokless-linux-x64")
	if !strings.Contains(got, "/releases/download/v0.1.41/tokless-linux-x64") {
		t.Fatalf("download URL should use tag-specific release asset, got %q", got)
	}
	if strings.Contains(got, "/latest/download/") {
		t.Fatalf("download URL must not use mutable latest shortcut: %q", got)
	}
}

func TestVerifyFileSHA256(t *testing.T) {
	path := writeChecksumFixture(t, "tokless")
	hash := sha256.Sum256([]byte("tokless"))
	sums := []byte(fmt.Sprintf("%x  tokless-linux-x64\n", hash))

	if err := verifyFileSHA256(path, sums, "tokless-linux-x64"); err != nil {
		t.Fatalf("verifyFileSHA256 returned error: %v", err)
	}
}

func TestVerifyFileSHA256RejectsMissingOrMismatchedDigest(t *testing.T) {
	path := writeChecksumFixture(t, "tokless")
	otherHash := sha256.Sum256([]byte("other"))

	if err := verifyFileSHA256(path, []byte(fmt.Sprintf("%x  other-asset\n", otherHash)), "tokless-linux-x64"); err == nil {
		t.Fatal("expected missing asset checksum to fail")
	}
	if err := verifyFileSHA256(path, []byte(fmt.Sprintf("%x  tokless-linux-x64\n", otherHash)), "tokless-linux-x64"); err == nil {
		t.Fatal("expected checksum mismatch to fail")
	}
	if err := verifyFileSHA256(path, []byte("nothex  tokless-linux-x64\n"), "tokless-linux-x64"); err == nil {
		t.Fatal("expected malformed checksum to fail")
	}
}

func writeChecksumFixture(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/asset"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
