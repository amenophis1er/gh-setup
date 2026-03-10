package cmd

import (
	"fmt"
	"os"

	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/importer"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var (
	importRepoFlag   string
	importOutput     string
	importStdout     bool
)

var importCmd = &cobra.Command{
	Use:   "import <account>",
	Short: "Import existing GitHub state into a config file",
	Long: `Reverse-engineer an existing GitHub account or repo into a gh-setup.yaml config file.

By default the result is written to gh-setup.yaml (or the path given by --config).
Use --stdout to print to standard output instead, and --output to choose the format.`,
	Example: `  gh setup import myorg
  gh setup import myorg --repo my-repo
  gh setup import myorg --stdout
  gh setup import myorg --stdout -o json
  gh setup import myorg -o json -c config.json`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if importOutput != "yaml" && importOutput != "json" {
			return fmt.Errorf("invalid --output value %q: allowed values are \"yaml\" and \"json\"", importOutput)
		}

		opts := importer.Options{
			Account:  args[0],
			RepoName: importRepoFlag,
		}

		cfg, err := importer.Run(opts)
		if err != nil {
			return err
		}

		data, err := config.Marshal(cfg, importOutput)
		if err != nil {
			return err
		}

		if importStdout {
			_, err = os.Stdout.Write(data)
			return err
		}

		if err := os.WriteFile(cfgFile, data, 0644); err != nil {
			return err
		}

		log.Info("Config imported", "path", cfgFile)
		log.Info("Review it, then run `gh-setup apply` to enforce.")
		return nil
	},
}

func init() {
	importCmd.Flags().StringVar(&importRepoFlag, "repo", "", "import only a specific repo")
	importCmd.Flags().StringVarP(&importOutput, "output", "o", "yaml", "output format (yaml or json)")
	importCmd.Flags().BoolVar(&importStdout, "stdout", false, "print to stdout instead of writing a file")
	rootCmd.AddCommand(importCmd)
}
