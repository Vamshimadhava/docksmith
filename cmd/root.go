package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "docksmith",
	Short: "Docksmith - A simplified container system",
	Long: `Docksmith is a simplified Docker-like container system for building and running containerized applications.

It supports building images from Docksmithfiles, running containers with isolation,
and managing images and layers.`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Add all subcommands
	rootCmd.AddCommand(BuildCmd)
	rootCmd.AddCommand(RunCmd)
	rootCmd.AddCommand(ImagesCmd)
	rootCmd.AddCommand(RmiCmd)
}
