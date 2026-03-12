package runtime

import (
	"docksmith/images"
	"docksmith/layers"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// Container represents a running container instance
type Container struct {
	image     *images.ImageManifest
	rootfsDir string
	command   []string
}

// NewContainer creates a new container from an image
func NewContainer(image *images.ImageManifest, overrideCmd []string) *Container {
	cmd := image.Config.Cmd
	if len(overrideCmd) > 0 {
		cmd = overrideCmd
	}
	
	return &Container{
		image:   image,
		command: cmd,
	}
}

// Run executes the container
func (c *Container) Run() error {
	// Create temporary rootfs
	tmpDir, err := os.MkdirTemp("", "docksmith-run-*")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	c.rootfsDir = tmpDir
	defer c.Cleanup()
	
	// Extract all layers
	fmt.Printf("Extracting %d layers...\n", len(c.image.Layers))
	if err := layers.ExtractLayers(c.image.GetLayerDigests(), c.rootfsDir); err != nil {
		return fmt.Errorf("failed to extract layers: %w", err)
	}
	
	// Run the command
	fmt.Printf("Running: %s\n", strings.Join(c.command, " "))
	return c.execute()
}

// execute runs the container command with isolation
func (c *Container) execute() error {
	if len(c.command) == 0 {
		return fmt.Errorf("no command specified")
	}
	
	// Set up working directory
	workDir := c.image.Config.WorkingDir
	if workDir == "" {
		workDir = "/"
	}
	
	// Create working directory in rootfs
	workDirPath := filepath.Join(c.rootfsDir, workDir)
	if err := os.MkdirAll(workDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create workdir: %w", err)
	}
	
	// Build environment
	env := c.image.Config.Env
	if len(env) == 0 {
		env = []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"}
	}
	
	// Try running with namespace isolation
	if err := c.runWithIsolation(workDir, env); err != nil {
		// Fallback to basic execution
		return c.runBasic(workDir, env)
	}
	
	return nil
}

// runWithIsolation executes with Linux namespace isolation
func (c *Container) runWithIsolation(workDir string, env []string) error {
	// Create wrapper script
	cmdStr := strings.Join(c.command, " ")
	wrapperScript := fmt.Sprintf(`#!/bin/sh
cd %s || exit 1
exec %s
`, workDir, cmdStr)
	
	wrapperPath := filepath.Join(c.rootfsDir, "tmp", "docksmith-exec.sh")
	os.MkdirAll(filepath.Dir(wrapperPath), 0755)
	
	if err := os.WriteFile(wrapperPath, []byte(wrapperScript), 0755); err != nil {
		return err
	}
	
	// Use unshare and chroot for isolation
	cmd := exec.Command("unshare", "--fork", "--pid", "--mount", "--uts",
		"chroot", c.rootfsDir,
		"/tmp/docksmith-exec.sh")
	
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
	}
	
	return cmd.Run()
}

// runBasic executes without full isolation (fallback)
func (c *Container) runBasic(workDir string, env []string) error {
	// This is a fallback for non-Linux systems or when unshare is not available
	
	// Build full environment
	fullEnv := os.Environ()
	fullEnv = append(fullEnv, env...)
	
	// Execute command
	cmd := exec.Command(c.command[0], c.command[1:]...)
	cmd.Dir = filepath.Join(c.rootfsDir, workDir)
	cmd.Env = fullEnv
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	return cmd.Run()
}

// Cleanup removes temporary files
func (c *Container) Cleanup() {
	if c.rootfsDir != "" {
		os.RemoveAll(c.rootfsDir)
	}
}
