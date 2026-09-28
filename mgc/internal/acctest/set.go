package acctest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MainSet is the published cassette set, the base layer every run replays
// from. Only the promote step writes it: recordings land in the branch layer.
const MainSet = "main"

var setNameRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// SetName turns a branch name into a cassette-set name usable as a directory
// and as a bucket prefix.
func SetName(branch string) (string, error) {
	s := strings.Trim(setNameRe.ReplaceAllString(branch, "-"), "-")
	if s == "" {
		return "", fmt.Errorf("branch name %q yields an empty set name", branch)
	}
	return s, nil
}

// CurrentSet is the cassette set of the checked-out branch. The branch is the
// only source of truth for which layer a run records into and replays from,
// so CI checks out a branch named after the PR before running.
func CurrentSet() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolving working directory: %w", err)
	}
	branch, err := branchAt(wd)
	if err != nil {
		return "", err
	}
	return SetName(branch)
}

// branchAt reads the checked-out branch from .git/HEAD, walking up from dir.
// Reading the file rather than shelling out to git keeps replay working in a
// stripped environment, where there is no PATH to find the binary in.
func branchAt(dir string) (string, error) {
	gitDir, err := findGitDir(dir)
	if err != nil {
		return "", err
	}
	head, err := os.ReadFile(filepath.Join(gitDir, "HEAD"))
	if err != nil {
		return "", fmt.Errorf("reading git HEAD: %w", err)
	}
	branch, ok := strings.CutPrefix(strings.TrimSpace(string(head)), "ref: refs/heads/")
	if !ok {
		return "", fmt.Errorf("detached HEAD in %s: cassette sets are named after the branch, so check one out first", gitDir)
	}
	return branch, nil
}

func findGitDir(dir string) (string, error) {
	for {
		p := filepath.Join(dir, ".git")
		switch info, err := os.Stat(p); {
		case err == nil && info.IsDir():
			return p, nil
		case err == nil:
			return worktreeGitDir(p)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no git repository found above %s", dir)
		}
		dir = parent
	}
}

// worktreeGitDir resolves the ".git" file a linked worktree carries in place
// of a directory.
func worktreeGitDir(file string) (string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return "", err
	}
	p, ok := strings.CutPrefix(strings.TrimSpace(string(data)), "gitdir: ")
	if !ok {
		return "", fmt.Errorf("%s: not a git worktree file", file)
	}
	if !filepath.IsAbs(p) {
		p = filepath.Join(filepath.Dir(file), p)
	}
	return p, nil
}
