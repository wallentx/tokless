package util

import (
	"strings"
	"testing"
)

// Locks the install-attempt contract: package-token installs must use an
// explicit trusted registry instead of implicitly trusting user npm config.
func TestBuildNpmAttemptsForcesTrustedRegistry(t *testing.T) {
	attempts := buildNpmAttempts("pkg", "1.2.3", "https://registry.npmjs.org/pkg/-/pkg-1.2.3.tgz", "", defaultRegistryURL)
	if len(attempts) < 2 {
		t.Fatalf("expected several attempts, got %d", len(attempts))
	}

	joined := func(a []string) string { return strings.Join(a, " ") }

	for i, attempt := range attempts {
		if !strings.Contains(joined(attempt), "--registry "+defaultRegistryURL) {
			t.Errorf("attempt %d does not force public registry: %v", i, attempt)
		}
	}
}

func TestNpmRegistryBaseIgnoresCustomRegistryByDefault(t *testing.T) {
	t.Setenv(allowCustomNpmRegistryEnv, "")

	orig := npmConfigRegistry
	defer func() { npmConfigRegistry = orig }()
	npmConfigRegistry = func() string { return "https://registry.example.test/" }

	if got := npmRegistryBase(); got != defaultRegistryURL {
		t.Fatalf("got %q, want %q", got, defaultRegistryURL)
	}
}

func TestNpmRegistryBaseAllowsHttpsCustomRegistryWithOptIn(t *testing.T) {
	t.Setenv(allowCustomNpmRegistryEnv, "1")

	orig := npmConfigRegistry
	defer func() { npmConfigRegistry = orig }()
	npmConfigRegistry = func() string { return "https://registry.example.test" }

	if got, want := npmRegistryBase(), "https://registry.example.test/"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestNpmRegistryBaseRejectsHttpCustomRegistry(t *testing.T) {
	t.Setenv(allowCustomNpmRegistryEnv, "1")

	orig := npmConfigRegistry
	defer func() { npmConfigRegistry = orig }()
	npmConfigRegistry = func() string { return "http://registry.example.test/" }

	if got := npmRegistryBase(); got != defaultRegistryURL {
		t.Fatalf("got %q, want %q", got, defaultRegistryURL)
	}
}
