package util

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestVerifyFileSHA256Sums(t *testing.T) {
	path := writeSHAFixture(t, "payload")
	hash := sha256.Sum256([]byte("payload"))
	sums := []byte(fmt.Sprintf("%x  payload.tar.gz\n", hash))

	if err := VerifyFileSHA256Sums(path, sums, "payload.tar.gz"); err != nil {
		t.Fatalf("VerifyFileSHA256Sums returned error: %v", err)
	}
}

func TestVerifyFileSHA256SumsRejectsUnsafeInputs(t *testing.T) {
	path := writeSHAFixture(t, "payload")
	other := sha256.Sum256([]byte("other"))

	tests := []struct {
		name string
		sums []byte
	}{
		{"missing asset", []byte(fmt.Sprintf("%x  other.tar.gz\n", other))},
		{"mismatch", []byte(fmt.Sprintf("%x  payload.tar.gz\n", other))},
		{"short digest", []byte("abcd  payload.tar.gz\n")},
		{"non hex digest", []byte(strings.Repeat("z", 64) + "  payload.tar.gz\n")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := VerifyFileSHA256Sums(path, tt.sums, "payload.tar.gz"); err == nil {
				t.Fatal("expected checksum verification to fail")
			}
		})
	}
}

func TestDownloadToTempVerified(t *testing.T) {
	payload := []byte("payload")
	hash := sha256.Sum256(payload)
	sums := []byte(fmt.Sprintf("%x  payload.tar.gz\n", hash))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/payload.tar.gz":
			_, _ = w.Write(payload)
		case "/SHASUMS256.txt":
			_, _ = w.Write(sums)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	path, err := DownloadToTempVerified(srv.URL+"/payload.tar.gz", srv.URL+"/SHASUMS256.txt", "payload.tar.gz")
	if err != nil {
		t.Fatalf("DownloadToTempVerified returned error: %v", err)
	}
	defer os.Remove(path)
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(payload) {
		t.Fatalf("downloaded payload = %q, want %q", got, payload)
	}
}

func TestAllowUnverifiedBootstrap(t *testing.T) {
	t.Setenv(AllowUnverifiedBootstrapEnv, "")
	if AllowUnverifiedBootstrap() {
		t.Fatal("empty opt-in env should not allow unverified bootstrap")
	}
	t.Setenv(AllowUnverifiedBootstrapEnv, "yes")
	if !AllowUnverifiedBootstrap() {
		t.Fatal("truthy opt-in env should allow unverified bootstrap")
	}
}

func TestDownloadAndExtractTarGzBlocksUnverifiedByDefault(t *testing.T) {
	t.Setenv(AllowUnverifiedBootstrapEnv, "")
	if err := DownloadAndExtractTarGz("https://example.test/payload.tar.gz", t.TempDir()); err == nil {
		t.Fatal("expected unverified tarball download to be blocked")
	}
}

func writeSHAFixture(t *testing.T, content string) string {
	t.Helper()
	path := t.TempDir() + "/payload"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}
