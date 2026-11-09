// Package gitutil provides small git helpers.
package gitutil

import (
	"os/exec"
	"strings"
)

// ChangedFiles returns files changed between from and to commits (or working tree if to empty).
func ChangedFiles(repoRoot, from, to string) ([]string, error) {
	args := []string{"-C", repoRoot, "diff", "--name-only"}
	if from != "" {
		args = append(args, from)
	}
	if to != "" {
		args = append(args, to)
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		// try diff against HEAD for working tree
		out, err = exec.Command("git", "-C", repoRoot, "diff", "--name-only", "HEAD").Output()
		if err != nil {
			return nil, err
		}
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var files []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			files = append(files, l)
		}
	}
	return files, nil
}

