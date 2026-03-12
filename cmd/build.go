package cmd

import (
	"docksmith/build"
	"docksmith/util"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var (
	buildTag string
)

// BuildCmd represents the build command
var BuildCmd = &cobra.Command{
	Use:   "build [OPTIONS] PATH",
	Short: "Build an image from a Docksmithfile",
	Long:  `Parse a Docksmithfile and build a container image with layer caching.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runBuild,
}

func init() {
	BuildCmd.Flags().StringVarP(&buildTag, "tag", "t", "", "Name and optionally a tag in the 'name:tag' format (required)")
	BuildCmd.MarkFlagRequired("tag")
}

func runBuild(cmd *cobra.Command, args []string) error {
	contextPath := args[0]
	
	// Ensure Docksmith directories exist
	if err := util.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to initialize directories: %w", err)
	}
	
	// Resolve context path
	contextPath, err := filepath.Abs(contextPath)
	if err != nil {
		return fmt.Errorf("invalid context path: %w", err)
	}
	
	// Check if context exists
	if _, err := os.Stat(contextPath); os.IsNotExist(err) {
		return fmt.Errorf("context path does not exist: %s", contextPath)
	}
	
	// Find Docksmithfile
	docksmithfilePath := filepath.Join(contextPath, "Docksmithfile")
	if _, err := os.Stat(docksmithfilePath); os.IsNotExist(err) {
		return fmt.Errorf("Docksmithfile not found in context: %s", contextPath)
	}
	
	// Parse name and tag
	parts := strings.SplitN(buildTag, ":", 2)
	name := parts[0]
	tag := "latest"
	if len(parts) > 1 {
		tag = parts[1]
	}
	
	if name == "" {
		return fmt.Errorf("image name cannot be empty")
	}
	
	// Parse Docksmithfile
	fmt.Printf("Parsing Docksmithfile...\n")
	instructions, err := build.ParseDocksmithfile(docksmithfilePath)
	if err != nil {
		return fmt.Errorf("failed to parse Docksmithfile: %w", err)
	}
	
	// Build image
	fmt.Printf("Building image %s:%s...\n", name, tag)
	builder := build.NewBuilder(contextPath, instructions)
	
	_, err = builder.Build(name, tag)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	
	return nil
}
