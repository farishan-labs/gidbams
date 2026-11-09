// Package triggers runs config-defined shell triggers on record lifecycle events.
package triggers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Trigger struct {
	Name  string   `yaml:"name"`
	On    []string `yaml:"on"` // created, updated, deleted, any
	Run   string   `yaml:"run"`
}

type File struct {
	Triggers []Trigger `yaml:"triggers"`
}

func Load(repoRoot string) (*File, error) {
	p := filepath.Join(repoRoot, ".gitdb", "triggers.yaml")
	b, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{}, nil
		}
		return nil, err
	}
	var f File
	if err := yaml.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	return &f, nil
}

func Run(repoRoot, event string) error {
	f, err := Load(repoRoot)
	if err != nil {
		return err
	}
	for _, t := range f.Triggers {
		match := false
		for _, e := range t.On {
			if e == event || e == "any" || e == "*" {
				match = true
				break
			}
		}
		if !match {
			continue
		}
		cmd := exec.Command("sh", "-c", t.Run)
		cmd.Dir = repoRoot
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("trigger %s failed: %w", t.Name, err)
		}
	}
	return nil
}

