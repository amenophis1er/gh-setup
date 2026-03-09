package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
)

// version is set via ldflags at build time.
var version = "dev"

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "gh-setup",
	Short: "Declaratively configure GitHub accounts, repos, and settings",
	Long:  "gh-setup lets you define your entire GitHub setup in a YAML file and apply it idempotently.",
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("gh-setup", version)
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "gh-setup.yaml", "config file path")
	rootCmd.AddCommand(versionCmd)
}
