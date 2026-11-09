package cli

import (
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

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check repository health",
	RunE: func(cmd *cobra.Command, args []string) error {
		ok := true
		check := func(name string, err error) {
			if err != nil {
				ok = false
				fmt.Printf("FAIL %s: %v\n", name, err)
			} else {
				fmt.Printf("OK   %s\n", name)
			}
		}
		root, err := config.LoadRoot(repoRoot)
		check("load root config", err)
		if err == nil {
			for _, ref := range root.Collections {
				def, derr := config.LoadCollectionDefinition(repoRoot, root.DataDir, ref)
				check("collection "+ref.Name, derr)
				if derr == nil {
					dir := def.DataDir
					if !filepath.IsAbs(dir) {
						dir = filepath.Join(repoRoot, dir)
					}
					if _, err := os.Stat(dir); err != nil {
						check("data dir "+ref.Name, err)
					} else {
						fmt.Printf("OK   data dir %s\n", ref.Name)
					}
				}
			}
		}
		// git check
		if _, err := os.Stat(filepath.Join(repoRoot, ".git")); err != nil {
			fmt.Println("WARN not a git repository")
		} else {
			fmt.Println("OK   git repository")
		}
		if !ok {
			return fmt.Errorf("doctor found issues")
		}
		return nil
	},
}
