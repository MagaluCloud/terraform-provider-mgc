package main

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MagaluCloud/terraform-provider-mgc/mgc/internal/acctest"
)

var setNameRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// sanitizeSetName turns a branch name into a cassette-set name usable as a
// directory and bucket prefix. Reserved and degenerate names are refused.
func sanitizeSetName(name string) (string, error) {
	s := setNameRe.ReplaceAllString(name, "-")
	s = strings.Trim(s, "-")
	if s == "" {
		return "", fmt.Errorf("branch name %q yields an empty set name", name)
	}
	if s == acctest.StagingDir {
		return "", fmt.Errorf("%q is reserved for unpublished recordings", acctest.StagingDir)
	}
	return s, nil
}

// listYamlFiles returns the relative paths of every .yaml under dir.
func listYamlFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(d.Name(), ".yaml") {
			return err
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}
