package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/farishan-labs/git-dbms/internal/config"
)

// doctorCmd is defined in cli.go? We'll add via init in this file's init.
func init() {
	// register after rootCmd exists
}

var doctorJSON bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check repository health",
	RunE: func(cmd *cobra.Command, args []string) error {
		type checkResult struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}
		ok := true
		results := []checkResult{}
		record := func(name string, err error) {
			if err != nil {
				ok = false
				results = append(results, checkResult{Name: name, Status: "FAIL", Error: err.Error()})
				if !doctorJSON {
					fmt.Printf("FAIL %s: %v\n", name, err)
				}
			} else {
				results = append(results, checkResult{Name: name, Status: "OK"})
				if !doctorJSON {
					fmt.Printf("OK   %s\n", name)
				}
			}
		}
		root, err := config.LoadRoot(repoRoot)
		record("load root config", err)
		if err == nil {
			for _, ref := range root.Collections {
				def, derr := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
				record("collection "+ref.Name, derr)
				if derr == nil {
					dir := def.DataDir
					if !filepath.IsAbs(dir) {
						dir = filepath.Join(repoRoot, dir)
					}
					if _, err := os.Stat(dir); err != nil {
						record("data dir "+ref.Name, err)
					} else {
						record("data dir "+ref.Name, nil)
					}
				}
			}
		}
		if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err != nil {
			results = append(results, checkResult{Name: "git repository", Status: "WARN", Error: "not a git repository"})
			if !doctorJSON {
				fmt.Println("WARN not a git repository")
			}
		} else {
			results = append(results, checkResult{Name: "git repository", Status: "OK"})
			if !doctorJSON {
				fmt.Println("OK   git repository")
			}
		}
		if doctorJSON {
			b, _ := json.MarshalIndent(results, "", "  ")
			fmt.Println(string(b))
		}
		if !ok {
			return fmt.Errorf("doctor found issues")
		}
		return nil
	},
}
