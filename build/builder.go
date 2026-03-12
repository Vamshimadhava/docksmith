package build

import (
	"docksmith/cache"
	"docksmith/images"
	"docksmith/layers"
	"docksmith/runtime"
	"docksmith/util"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BuildContext holds the state during a build
type BuildContext struct {
	ContextDir   string
	WorkDir      string
	Env          map[string]string
	Cmd          []string
	Layers       []layers.Layer
	CurrentLayer string // digest of the most recent layer
}

// Builder handles the build process
type Builder struct {
	context      *BuildContext
	instructions []Instruction
	stagingDir   string
}

// NewBuilder creates a new builder
func NewBuilder(contextDir string, instructions []Instruction) *Builder {
	return &Builder{
		context: &BuildContext{
			ContextDir: contextDir,
			WorkDir:    "/",
			Env:        make(map[string]string),
			Cmd:        []string{},
			Layers:     []layers.Layer{},
		},
		instructions: instructions,
	}
}

// Build executes the build process and returns the final image manifest
func (b *Builder) Build(name, tag string) (*images.ImageManifest, error) {
	var err error
	
	// Create staging directory
	b.stagingDir, err = os.MkdirTemp("", "docksmith-build-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(b.stagingDir)
	
	// Load cache
	cacheIndex := cache.GetCache()
	
	// Execute instructions
	for i, instr := range b.instructions {
		stepNum := i + 1
		totalSteps := len(b.instructions)
		
		fmt.Printf("Step %d/%d : %s\n", stepNum, totalSteps, instr.Raw)
		
		start := time.Now()
		
		switch instr.Type {
		case InstructionFROM:
			err = b.executeFROM(instr)
		case InstructionCOPY:
			err = b.executeCOPY(instr, cacheIndex)
		case InstructionRUN:
			err = b.executeRUN(instr, cacheIndex)
		case InstructionWORKDIR:
			err = b.executeWORKDIR(instr)
		case InstructionENV:
			err = b.executeENV(instr)
		case InstructionCMD:
			err = b.executeCMD(instr)
		}
		
		if err != nil {
			return nil, fmt.Errorf("step %d failed: %w", stepNum, err)
		}
		
		elapsed := time.Since(start)
		if instr.Type == InstructionCOPY || instr.Type == InstructionRUN {
			fmt.Printf("  %.2fs\n", elapsed.Seconds())
		}
	}
	
	// Create final manifest
	manifest := &images.ImageManifest{
		Name:   name,
		Tag:    tag,
		Layers: b.context.Layers,
		Config: images.ImageConfig{
			WorkingDir: b.context.WorkDir,
			Cmd:        b.context.Cmd,
		},
	}
	
	// Convert env map to slice
	for k, v := range b.context.Env {
		manifest.Config.Env = append(manifest.Config.Env, fmt.Sprintf("%s=%s", k, v))
	}
	
	// Save manifest
	if err := images.SaveImage(manifest); err != nil {
		return nil, err
	}
	
	fmt.Printf("Successfully built %s %s:%s\n", manifest.Digest, name, tag)
	
	return manifest, nil
}

// executeFROM loads the base image
func (b *Builder) executeFROM(instr Instruction) error {
	imageRef := instr.Args[0]
	
	// Try to load existing image
	baseImage, err := images.LoadImage(imageRef)
	if err != nil {
		// Image doesn't exist - create an empty base
		fmt.Printf("  Creating empty base image\n")
		b.context.CurrentLayer = ""
		return nil
	}
	
	// Load layers from base image
	b.context.Layers = baseImage.Layers
	if len(baseImage.Layers) > 0 {
		b.context.CurrentLayer = baseImage.Layers[len(baseImage.Layers)-1].Digest
	}
	
	// Load config
	b.context.Env = baseImage.GetEnvMap()
	b.context.WorkDir = baseImage.Config.WorkingDir
	if b.context.WorkDir == "" {
		b.context.WorkDir = "/"
	}
	b.context.Cmd = baseImage.Config.Cmd
	
	fmt.Printf("  Loaded base image with %d layers\n", len(baseImage.Layers))
	
	return nil
}

// executeCOPY handles COPY instructions with caching
func (b *Builder) executeCOPY(instr Instruction, cacheIndex *cache.CacheIndex) error {
	if len(instr.Args) < 2 {
		return fmt.Errorf("COPY requires source and destination")
	}
	
	src := instr.Args[0]
	dst := instr.Args[len(instr.Args)-1]
	
	// Compute source file hashes
	srcPaths, err := util.MatchGlob(b.context.ContextDir, src)
	if err != nil {
		return err
	}
	
	if len(srcPaths) == 0 {
		return fmt.Errorf("no files match pattern: %s", src)
	}
	
	var srcHashes []string
	for _, path := range srcPaths {
		hash, err := util.ComputeFileSHA256(path)
		if err != nil {
			return err
		}
		srcHashes = append(srcHashes, hash)
	}
	
	// Compute cache key
	cacheKey := cache.ComputeCacheKey(b.context.CurrentLayer, instr.Raw, b.context.WorkDir, b.context.Env, srcHashes)
	
	// Check cache
	if layer, hit := cacheIndex.Lookup(cacheKey); hit {
		fmt.Printf("  [CACHE HIT]\n")
		b.context.Layers = append(b.context.Layers, *layer)
		b.context.CurrentLayer = layer.Digest
		return nil
	}
	
	fmt.Printf("  [CACHE MISS]\n")
	
	// Create layer staging area
	layerDir, err := os.MkdirTemp(b.stagingDir, "layer-*")
	if err != nil {
		return err
	}
	
	// Copy files
	for _, srcPath := range srcPaths {
		relPath, _ := filepath.Rel(b.context.ContextDir, srcPath)
		dstPath := filepath.Join(layerDir, dst, relPath)
		
		if err := util.CopyFile(srcPath, dstPath); err != nil {
			return fmt.Errorf("failed to copy %s: %w", srcPath, err)
		}
	}
	
	// Create layer
	layer, err := layers.CreateLayer(layerDir, instr.Raw)
	if err != nil {
		return err
	}
	
	// Store in cache
	cacheIndex.Store(cacheKey, *layer)
	
	b.context.Layers = append(b.context.Layers, *layer)
	b.context.CurrentLayer = layer.Digest
	
	return nil
}

// executeRUN handles RUN instructions with caching and isolation
func (b *Builder) executeRUN(instr Instruction, cacheIndex *cache.CacheIndex) error {
	command := instr.Args[0]
	
	// Compute cache key (no source hashes for RUN)
	cacheKey := cache.ComputeCacheKey(b.context.CurrentLayer, instr.Raw, b.context.WorkDir, b.context.Env, nil)
	
	// Check cache
	if layer, hit := cacheIndex.Lookup(cacheKey); hit {
		fmt.Printf("  [CACHE HIT]\n")
		b.context.Layers = append(b.context.Layers, *layer)
		b.context.CurrentLayer = layer.Digest
		return nil
	}
	
	fmt.Printf("  [CACHE MISS]\n")
	
	// Create rootfs from existing layers
	rootfsDir, err := os.MkdirTemp(b.stagingDir, "rootfs-*")
	if err != nil {
		return err
	}
	
	// Extract existing layers
	layerDigests := make([]string, len(b.context.Layers))
	for i, l := range b.context.Layers {
		layerDigests[i] = l.Digest
	}
	if err := layers.ExtractLayers(layerDigests, rootfsDir); err != nil {
		return fmt.Errorf("failed to extract layers: %w", err)
	}
	
	// Create output directory for changes
	outputDir, err := os.MkdirTemp(b.stagingDir, "output-*")
	if err != nil {
		return err
	}
	
	// Execute command in isolated environment
	executor := runtime.NewExecutor(rootfsDir, outputDir)
	executor.SetWorkDir(b.context.WorkDir)
	executor.SetEnv(b.context.Env)
	
	if err := executor.Run([]string{"/bin/sh", "-c", command}); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}
	
	// Create layer from output (changes only)
	layer, err := layers.CreateLayer(outputDir, instr.Raw)
	if err != nil {
		return err
	}
	
	// Store in cache
	cacheIndex.Store(cacheKey, *layer)
	
	b.context.Layers = append(b.context.Layers, *layer)
	b.context.CurrentLayer = layer.Digest
	
	return nil
}

// executeWORKDIR sets the working directory
func (b *Builder) executeWORKDIR(instr Instruction) error {
	b.context.WorkDir = instr.Args[0]
	return nil
}

// executeENV sets environment variables
func (b *Builder) executeENV(instr Instruction) error {
	envStr := instr.Args[0]
	parts := strings.SplitN(envStr, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid ENV format, expected key=value")
	}
	b.context.Env[parts[0]] = parts[1]
	return nil
}

// executeCMD sets the default command
func (b *Builder) executeCMD(instr Instruction) error {
	b.context.Cmd = instr.Args
	return nil
}
