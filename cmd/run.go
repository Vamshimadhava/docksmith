package cmd

import (
	"docksmith/images"
	"docksmith/runtime"
	"docksmith/util"
	"fmt"

	"github.com/spf13/cobra"
)

// RunCmd represents the run command
var RunCmd = &cobra.Command{
	Use:   "run IMAGE[:TAG] [COMMAND]",
	Short: "Run a command in a new container",
	Long:  `Create and run a container from an image. Optionally override the default command.`,
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRun,
}

func runRun(cmd *cobra.Command, args []string) error {
	imageRef := args[0]
	var overrideCmd []string
	if len(args) > 1 {
		overrideCmd = args[1:]
	}
	
	// Ensure Docksmith directories exist
	if err := util.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to initialize directories: %w", err)
	}
	
	// Load image
	image, err := images.LoadImage(imageRef)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	
	// Create container
	container := runtime.NewContainer(image, overrideCmd)
	
	// Run container
	if err := container.Run(); err != nil {
		return fmt.Errorf("container execution failed: %w", err)
	}
	
	return nil
}
