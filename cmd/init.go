package cmd

import (
	"github.com/amenophis1er/gh-setup/internal/config"
	"github.com/amenophis1er/gh-setup/internal/gitutil"
	"github.com/amenophis1er/gh-setup/internal/wizard"
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Interactive wizard to generate gh-setup.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		remote, _ := gitutil.DetectRemote()
		if remote.Owner != "" {
			log.Info("Detected from git remote", "account", remote.Owner, "repo", remote.Repo)
		}

		cfg, err := wizard.Run(remote)
		if err != nil {
			return err
		}
		if err := config.Save(cfgFile, cfg); err != nil {
			return err
		}
		log.Info("Config written", "path", cfgFile)
		log.Info("Review it, then run `gh-setup apply` to apply.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
