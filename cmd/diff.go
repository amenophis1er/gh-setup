package cmd

import (
	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/diff"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare config vs actual GitHub state",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}

		if err := cfg.Validate(); err != nil {
			return err
		}

		if err := diff.Run(cfg); err != nil {
			return err
		}

		log.Info("Diff complete")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(diffCmd)
}
