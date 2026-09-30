package acctest

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const MainSet = "main"

var setNameRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func SetName(branch string) (string, error) {
	s := strings.Trim(setNameRe.ReplaceAllString(branch, "-"), "-")
	if s == "" {
		return "", fmt.Errorf("branch name %q yields an empty set name", branch)
	}
	return s, nil
}

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
