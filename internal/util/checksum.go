package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const AllowUnverifiedBootstrapEnv = "TOKLESS_ALLOW_UNVERIFIED_BOOTSTRAP"

func bootstrapEnvTruthy(name string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(name)))
	return v == "1" || v == "true" || v == "yes"
}

// AllowUnverifiedBootstrap gates bootstrap channels without independent
// checksums or signatures.
func AllowUnverifiedBootstrap() bool {
	return bootstrapEnvTruthy(AllowUnverifiedBootstrapEnv)
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

// VerifyFileSHA256Sums verifies path against a sha256sum-style checksum file.
func VerifyFileSHA256Sums(path string, sums []byte, asset string) error {
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

// DownloadToTempVerified downloads url and verifies it against checksumURL.
func DownloadToTempVerified(url, checksumURL, asset string) (string, error) {
	tmp, err := downloadToTemp(url)
	if err != nil {
		return "", err
	}
	sums, err := downloadChecksumBytes(checksumURL)
	if err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := VerifyFileSHA256Sums(tmp, sums, asset); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return tmp, nil
}

func downloadChecksumBytes(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("checksum download failed with HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, fmt.Errorf("checksum file was empty")
	}
	return data, nil
}
