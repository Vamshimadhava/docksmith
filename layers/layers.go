package layers

import (
	"bytes"
	"docksmith/util"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Layer represents a filesystem layer with metadata
type Layer struct {
	Digest    string `json:"digest"`
	Size      int64  `json:"size"`
	CreatedBy string `json:"createdBy"`
}

// CreateLayer creates a new layer from a directory and stores it
// Returns the Layer metadata
func CreateLayer(sourceDir, createdBy string) (*Layer, error) {
	// Create tar archive in memory first to compute digest
	var buf bytes.Buffer
	if err := util.CreateTarLayer(sourceDir, &buf); err != nil {
		return nil, fmt.Errorf("failed to create tar: %w", err)
	}
	
	tarBytes := buf.Bytes()
	digest := util.ComputeSHA256(tarBytes)
	
	// Write layer to disk
	layerPath := GetLayerPath(digest)
	if err := os.WriteFile(layerPath, tarBytes, 0644); err != nil {
		return nil, fmt.Errorf("failed to write layer: %w", err)
	}
	
	return &Layer{
		Digest:    digest,
		Size:      int64(len(tarBytes)),
		CreatedBy: createdBy,
	}, nil
}

// GetLayerPath returns the filesystem path for a layer digest
func GetLayerPath(digest string) string {
	layersDir, _ := util.GetLayersDir()
	// digest format is "sha256:hash", convert to "sha256_hash.tar"
	filename := strings.Replace(digest, ":", "_", 1) + ".tar"
	return filepath.Join(layersDir, filename)
}

// LayerExists checks if a layer file exists on disk
func LayerExists(digest string) bool {
	path := GetLayerPath(digest)
	_, err := os.Stat(path)
	return err == nil
}

// ExtractLayer extracts a layer to a destination directory
func ExtractLayer(digest, destDir string) error {
	layerPath := GetLayerPath(digest)
	
	f, err := os.Open(layerPath)
	if err != nil {
		return fmt.Errorf("failed to open layer %s: %w", digest, err)
	}
	defer f.Close()
	
	return util.ExtractTar(f, destDir)
}

// ExtractLayers extracts multiple layers in order to build a complete filesystem
func ExtractLayers(layerDigests []string, destDir string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return err
	}
	
	for _, digest := range layerDigests {
		if err := ExtractLayer(digest, destDir); err != nil {
			return err
		}
	}
	
	return nil
}

// DeleteLayer removes a layer file from disk
func DeleteLayer(digest string) error {
	layerPath := GetLayerPath(digest)
	return os.Remove(layerPath)
}

// CopyLayerFromReader reads a tar stream, computes its digest, and stores it
// This is useful for loading base images from external sources
func CopyLayerFromReader(r io.Reader, createdBy string) (*Layer, error) {
	// Read all data to compute digest
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	
	digest := util.ComputeSHA256(data)
	
	// Write to disk
	layerPath := GetLayerPath(digest)
	if err := os.WriteFile(layerPath, data, 0644); err != nil {
		return nil, err
	}
	
	return &Layer{
		Digest:    digest,
		Size:      int64(len(data)),
		CreatedBy: createdBy,
	}, nil
}
