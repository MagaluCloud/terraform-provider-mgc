package acctest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{in: "main", want: "main"},
		{in: "feat/tags", want: "feat-tags"},
		{in: "chore/acctest-vcr", want: "chore-acctest-vcr"},
		{in: "Fix_Bug.2", want: "Fix_Bug.2"},
		{in: "weird//name", want: "weird-name"},
		{in: "/leading/trailing/", want: "leading-trailing"},
		{in: "///", wantErr: true},
		{in: "", wantErr: true},
	}
	for _, tt := range tests {
		got, err := SetName(tt.in)
		if tt.wantErr {
			if err == nil {
				t.Errorf("SetName(%q) = %q, want error", tt.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("SetName(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("SetName(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestBranchAt(t *testing.T) {
	t.Parallel()

	t.Run("reads the checked out branch", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/feat/tags\n")

		got, err := branchAt(dir)
		if err != nil {
			t.Fatalf("branchAt: %v", err)
		}
		if got != "feat/tags" {
			t.Errorf("branch = %q, want %q", got, "feat/tags")
		}
	})

	t.Run("walks up from a package dir", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, ".git", "HEAD"), "ref: refs/heads/main\n")
		pkg := filepath.Join(dir, "mgc", "kubernetes")
		if err := os.MkdirAll(pkg, 0o755); err != nil {
			t.Fatal(err)
		}

		got, err := branchAt(pkg)
		if err != nil {
			t.Fatalf("branchAt: %v", err)
		}
		if got != "main" {
			t.Errorf("branch = %q, want %q", got, "main")
		}
	})

	t.Run("follows a worktree gitdir file", func(t *testing.T) {
		dir := t.TempDir()
		gitDir := filepath.Join(dir, "gitdirs", "wt")
		writeFile(t, filepath.Join(gitDir, "HEAD"), "ref: refs/heads/wt-branch\n")
		writeFile(t, filepath.Join(dir, "checkout", ".git"), "gitdir: "+gitDir+"\n")

		got, err := branchAt(filepath.Join(dir, "checkout"))
		if err != nil {
			t.Fatalf("branchAt: %v", err)
		}
		if got != "wt-branch" {
			t.Errorf("branch = %q, want %q", got, "wt-branch")
		}
	})

	t.Run("detached HEAD is refused", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, ".git", "HEAD"), "6965bb6ae0f8bdc6e2b6c3b3f0a2a2a9e2f1c0d4\n")

		if _, err := branchAt(dir); err == nil {
			t.Error("detached HEAD should fail: the cassette set is named after the branch")
		}
	})

	t.Run("no repository is refused", func(t *testing.T) {
		if _, err := branchAt(t.TempDir()); err == nil {
			t.Error("a dir outside a repository should fail")
		}
	})
}
