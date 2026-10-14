package main

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestListYamlFiles(t *testing.T) {
	dir := t.TempDir()
	write(t, filepath.Join(dir, "kubernetes", "TestA.yaml"), "a")
	write(t, filepath.Join(dir, "network", "TestB.yaml"), "b")
	write(t, filepath.Join(dir, "junk.txt"), "x")

	files, err := listYamlFiles(dir)
	if err != nil {
		t.Fatalf("listYamlFiles: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2: %v", len(files), files)
	}
	for _, rel := range []string{filepath.Join("kubernetes", "TestA.yaml"), filepath.Join("network", "TestB.yaml")} {
		found := false
		for _, f := range files {
			if f == rel {
				found = true
			}
		}
		if !found {
			t.Errorf("missing %s in %v", rel, files)
		}
	}
}

func assertFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("%s = %q, want %q", path, got, want)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("%s should not exist", path)
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
