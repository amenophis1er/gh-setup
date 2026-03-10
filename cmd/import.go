package cmd

import (
	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/importer"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var importRepoFlag string

var importCmd = &cobra.Command{
	Use:   "import <account>",
	Short: "Import existing GitHub state into a config file",
	Long:  "Reverse-engineer an existing GitHub account or repo into a gh-setup.yaml config file.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := importer.Options{
			Account:  args[0],
			RepoName: importRepoFlag,
		}

		cfg, err := importer.Run(opts)
		if err != nil {
			return err
		}

		if err := config.Save(cfgFile, cfg); err != nil {
			return err
		}

		log.Info("Config imported", "path", cfgFile)
		log.Info("Review it, then run `gh-setup apply` to enforce.")
		return nil
	},
}

func init() {
	importCmd.Flags().StringVar(&importRepoFlag, "repo", "", "import only a specific repo")
	rootCmd.AddCommand(importCmd)
}
