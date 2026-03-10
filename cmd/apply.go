package cmd

import (
	"github.com/amenophis1er/gh-setup/internal/apply"
	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	dryRun           bool
	interactive      bool
	nonInteractive   bool
	applyConcurrency int
)

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply configuration to GitHub",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}

		if err := cfg.Validate(); err != nil {
			return err
		}

		opts := apply.Options{
			DryRun:         dryRun,
			Interactive:    interactive,
			NonInteractive: nonInteractive,
			Concurrency:    applyConcurrency,
		}

		if err := apply.Run(cfg, opts); err != nil {
			return err
		}

		log.Info("Apply complete")
		return nil
	},
}

func init() {
	applyCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would change without mutating")
	applyCmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "confirm each step before applying")
	applyCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "run without prompts (secrets must be provided via GH_SETUP_SECRET_<NAME> env vars)")
	applyCmd.Flags().IntVar(&applyConcurrency, "concurrency", 4, "number of repos to process in parallel (forced to 1 in interactive mode)")
	rootCmd.AddCommand(applyCmd)
}
