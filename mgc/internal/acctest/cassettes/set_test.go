package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSanitizeSetName(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "main", want: "main"},
		{in: "feat/tags", want: "feat-tags"},
		{in: "chore/tf-acctest-with-vcr", want: "chore-tf-acctest-with-vcr"},
		{in: "Fix_Bug.2", want: "Fix_Bug.2"},
		{in: "weird//name", want: "weird-name"},
		{in: "/leading/trailing/", want: "leading-trailing"},
		// Reserved and degenerate names must be refused.
		{in: "staging", wantErr: true},
		{in: "///", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		got, err := sanitizeSetName(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("sanitizeSetName(%q) = %q, want error", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("sanitizeSetName(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("sanitizeSetName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

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
