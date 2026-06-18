package main

import (
	"os"
	"testing"
)

// TestFindOverridesPath_CwdIndependent guards the codegen bug where
// findOverridesPath resolved overrides.json relative to the working directory
// only. `go generate ./...` runs the internal/generated/doc.go directive with
// cwd=internal/generated, where the cwd-relative candidates do not resolve, so
// ZERO overrides loaded and every map_string/map_nested field regenerated as a
// stale auto-parsed scalar. The fix resolves the path via runtime.Caller
// (tfgenSourceDir), which is cwd-independent. This test changes into a
// scratch directory and asserts overrides.json is still found and parses.
func TestFindOverridesPath_CwdIndependent(t *testing.T) {
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	path := findOverridesPath()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("findOverridesPath returned %q which does not exist from a foreign cwd: %v", path, err)
	}

	overrides, err := LoadOverrides(path)
	if err != nil {
		t.Fatalf("LoadOverrides(%q): %v", path, err)
	}
	if len(overrides.FieldOverrides) == 0 {
		t.Fatalf("expected field overrides to load from %q, got 0 — map overrides would regenerate as stale scalars", path)
	}
}
