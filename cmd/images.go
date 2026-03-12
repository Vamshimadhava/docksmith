package cmd

import (
	"docksmith/images"
	"docksmith/util"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

// ImagesCmd represents the images command
var ImagesCmd = &cobra.Command{
	Use:   "images",
	Short: "List images",
	Long:  `List all available container images.`,
	RunE:  runImages,
}

func runImages(cmd *cobra.Command, args []string) error {
	// Ensure Docksmith directories exist
	if err := util.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to initialize directories: %w", err)
	}
	
	// List all images
	imageList, err := images.ListImages()
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}
	
	if len(imageList) == 0 {
		fmt.Println("No images found")
		return nil
	}
	
	// Print header
	fmt.Printf("%-30s %-15s %-70s %-10s\n", "REPOSITORY", "TAG", "IMAGE ID", "LAYERS")
	fmt.Println(strings.Repeat("-", 130))
	
	// Print images
	for _, img := range imageList {
		imageID := img.Digest
		if len(imageID) > 19 {
			// Show short form: sha256:abc123...
			parts := strings.SplitN(imageID, ":", 2)
			if len(parts) == 2 && len(parts[1]) > 12 {
				imageID = parts[0] + ":" + parts[1][:12]
			}
		}
		
		fmt.Printf("%-30s %-15s %-70s %-10d\n",
			img.Name,
			img.Tag,
			imageID,
			len(img.Layers))
	}
	
	return nil
}
