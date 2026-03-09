package cmd

import (
	"github.com/amenophis1er/gh-setup/internal/apply"
	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	dryRun      bool
	interactive bool
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
			DryRun:      dryRun,
			Interactive: interactive,
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
	rootCmd.AddCommand(applyCmd)
}
