package images

import (
	"docksmith/layers"
	"docksmith/util"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ImageConfig holds runtime configuration for the image
type ImageConfig struct {
	Env        []string `json:"Env"`
	Cmd        []string `json:"Cmd"`
	WorkingDir string   `json:"WorkingDir"`
}

// ImageManifest represents the complete image metadata
type ImageManifest struct {
	Name    string               `json:"name"`
	Tag     string               `json:"tag"`
	Digest  string               `json:"digest"`
	Created string               `json:"created"`
	Config  ImageConfig          `json:"config"`
	Layers  []layers.Layer       `json:"layers"`
}

// SaveImage saves an image manifest to disk
// The digest is computed after serializing the manifest
func SaveImage(manifest *ImageManifest) error {
	// Set creation time
	manifest.Created = time.Now().UTC().Format(time.RFC3339)
	
	// Serialize without digest first
	manifest.Digest = ""
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	
	// Compute digest
	manifest.Digest = util.ComputeSHA256(data)
	
	// Serialize again with digest
	data, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	
	// Write to file
	imagesDir, err := util.GetImagesDir()
	if err != nil {
		return err
	}
	
	filename := fmt.Sprintf("%s_%s.json", manifest.Name, manifest.Tag)
	path := filepath.Join(imagesDir, filename)
	
	return os.WriteFile(path, data, 0644)
}

// LoadImage loads an image manifest from disk
func LoadImage(nameTag string) (*ImageManifest, error) {
	parts := strings.SplitN(nameTag, ":", 2)
	name := parts[0]
	tag := "latest"
	if len(parts) > 1 {
		tag = parts[1]
	}
	
	imagesDir, err := util.GetImagesDir()
	if err != nil {
		return nil, err
	}
	
	filename := fmt.Sprintf("%s_%s.json", name, tag)
	path := filepath.Join(imagesDir, filename)
	
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("image not found: %s:%s", name, tag)
	}
	
	var manifest ImageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	
	return &manifest, nil
}

// ImageExists checks if an image exists
func ImageExists(nameTag string) bool {
	_, err := LoadImage(nameTag)
	return err == nil
}

// ListImages returns all available images
func ListImages() ([]*ImageManifest, error) {
	imagesDir, err := util.GetImagesDir()
	if err != nil {
		return nil, err
	}
	
	entries, err := os.ReadDir(imagesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*ImageManifest{}, nil
		}
		return nil, err
	}
	
	var images []*ImageManifest
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		
		data, err := os.ReadFile(filepath.Join(imagesDir, entry.Name()))
		if err != nil {
			continue
		}
		
		var manifest ImageManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			continue
		}
		
		images = append(images, &manifest)
	}
	
	return images, nil
}

// DeleteImage removes an image manifest from disk
func DeleteImage(nameTag string) error {
	parts := strings.SplitN(nameTag, ":", 2)
	name := parts[0]
	tag := "latest"
	if len(parts) > 1 {
		tag = parts[1]
	}
	
	imagesDir, err := util.GetImagesDir()
	if err != nil {
		return err
	}
	
	filename := fmt.Sprintf("%s_%s.json", name, tag)
	path := filepath.Join(imagesDir, filename)
	
	return os.Remove(path)
}

// GetLayerDigests returns a slice of layer digests from the manifest
func (m *ImageManifest) GetLayerDigests() []string {
	digests := make([]string, len(m.Layers))
	for i, layer := range m.Layers {
		digests[i] = layer.Digest
	}
	return digests
}

// GetEnvMap returns environment variables as a map
func (m *ImageManifest) GetEnvMap() map[string]string {
	envMap := make(map[string]string)
	for _, e := range m.Config.Env {
		parts := strings.SplitN(e, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	return envMap
}
