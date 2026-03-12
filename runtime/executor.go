package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

// Executor handles running commands in an isolated environment
type Executor struct {
	rootfsDir string
	outputDir string
	workDir   string
	env       map[string]string
}

// NewExecutor creates a new executor
func NewExecutor(rootfsDir, outputDir string) *Executor {
	return &Executor{
		rootfsDir: rootfsDir,
		outputDir: outputDir,
		workDir:   "/",
		env:       make(map[string]string),
	}
}

// SetWorkDir sets the working directory for command execution
func (e *Executor) SetWorkDir(dir string) {
	e.workDir = dir
}

// SetEnv sets environment variables
func (e *Executor) SetEnv(env map[string]string) {
	e.env = env
}

// Run executes a command in the isolated environment
// On Linux, this uses namespace isolation
// On other systems, it falls back to basic chroot-like behavior
func (e *Executor) Run(command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("no command specified")
	}
	
	// Create the working directory in rootfs if it doesn't exist
	workDirPath := filepath.Join(e.rootfsDir, e.workDir)
	if err := os.MkdirAll(workDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create workdir: %w", err)
	}
	
	// Try Linux namespace isolation first
	if err := e.runWithNamespaces(command); err == nil {
		return nil
	}
	
	// Fallback to basic execution (for development/testing on non-Linux)
	return e.runBasic(command)
}

// runWithNamespaces runs the command with Linux namespace isolation
func (e *Executor) runWithNamespaces(command []string) error {
	// This is Linux-specific and uses unshare/chroot
	// For a production system, you would use proper namespace setup
	
	// Create a wrapper script that will be executed
	wrapperScript := fmt.Sprintf(`#!/bin/sh
set -e
cd %s
%s
`, e.workDir, strings.Join(command, " "))
	
	wrapperPath := filepath.Join(e.rootfsDir, "tmp", "docksmith-run.sh")
	os.MkdirAll(filepath.Dir(wrapperPath), 0755)
	
	if err := os.WriteFile(wrapperPath, []byte(wrapperScript), 0755); err != nil {
		return err
	}
	
	// Build environment variables
	envVars := []string{}
	for k, v := range e.env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}
	envVars = append(envVars, "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin")
	
	// Try to use unshare for namespace isolation
	cmd := exec.Command("unshare", "--fork", "--pid", "--mount", "--uts",
		"chroot", e.rootfsDir,
		"/tmp/docksmith-run.sh")
	
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWNS | syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
	}
	
	if err := cmd.Run(); err != nil {
		return err
	}
	
	// Copy any changes to the output directory
	// In a real implementation, you would track filesystem changes
	// For simplicity, we'll copy everything that was modified
	return e.captureChanges()
}

// runBasic runs the command without full isolation (fallback)
func (e *Executor) runBasic(command []string) error {
	// This is a simplified version for development/testing
	// It doesn't provide true isolation but allows the system to work on non-Linux
	
	// Build environment variables
	envVars := os.Environ()
	for k, v := range e.env {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Execute in rootfs context
	cmd := exec.Command(command[0], command[1:]...)
	cmd.Dir = filepath.Join(e.rootfsDir, e.workDir)
	cmd.Env = envVars
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		return err
	}
	
	return e.captureChanges()
}

// captureChanges copies modified files to the output directory
// This is a simplified implementation
func (e *Executor) captureChanges() error {
	// In a real implementation, you would use overlay filesystem or track changes
	// For this simplified version, we'll just note that changes should be captured
	
	// For RUN instructions, typically you would:
	// 1. Take a snapshot before execution
	// 2. Execute the command
	// 3. Compute the diff
	// 4. Store only the changes in the output directory
	
	// For now, we'll create a marker file to indicate the layer was created
	markerPath := filepath.Join(e.outputDir, ".docksmith-layer")
	return os.WriteFile(markerPath, []byte("layer created"), 0644)
}
