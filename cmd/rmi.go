package cmd

import (
	"docksmith/images"
	"docksmith/util"
	"fmt"

	"github.com/spf13/cobra"
)

// RmiCmd represents the rmi (remove image) command
var RmiCmd = &cobra.Command{
	Use:   "rmi IMAGE[:TAG]",
	Short: "Remove one or more images",
	Long:  `Remove container images by name and tag.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRmi,
}

func runRmi(cmd *cobra.Command, args []string) error {
	imageRef := args[0]
	
	// Ensure Docksmith directories exist
	if err := util.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to initialize directories: %w", err)
	}
	
	// Check if image exists
	if !images.ImageExists(imageRef) {
		return fmt.Errorf("image not found: %s", imageRef)
	}
	
	// Delete image
	if err := images.DeleteImage(imageRef); err != nil {
		return fmt.Errorf("failed to delete image: %w", err)
	}
	
	fmt.Printf("Deleted: %s\n", imageRef)
	
	return nil
}
