package cmd

import (
	"fmt"

	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/diff"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	diffOutputFormat string
	diffConcurrency  int
)

var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare config vs actual GitHub state",
	RunE: func(cmd *cobra.Command, args []string) error {
		if diffOutputFormat != "text" && diffOutputFormat != "json" {
			return fmt.Errorf("invalid --output value %q: allowed values are \"text\" and \"json\"", diffOutputFormat)
		}

		cfg, err := config.Load(cfgFile)
		if err != nil {
			return err
		}

		if err := cfg.Validate(); err != nil {
			return err
		}

		if err := diff.Run(cfg, diffOutputFormat, diffConcurrency); err != nil {
			return err
		}

		if diffOutputFormat != "json" {
			log.Info("Diff complete")
		}
		return nil
	},
}

func init() {
	diffCmd.Flags().StringVarP(&diffOutputFormat, "output", "o", "text", "output format (text or json)")
	diffCmd.Flags().IntVar(&diffConcurrency, "concurrency", 4, "number of repos to process in parallel")
	rootCmd.AddCommand(diffCmd)
}
