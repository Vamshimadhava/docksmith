package util

import (
	"os"
	"path/filepath"
)

// GetDocksmithHome returns the base directory for all Docksmith data (~/.docksmith)
func GetDocksmithHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".docksmith"), nil
}

// GetImagesDir returns the directory where image manifests are stored
func GetImagesDir() (string, error) {
	home, err := GetDocksmithHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "images"), nil
}

// GetLayersDir returns the directory where layers are stored
func GetLayersDir() (string, error) {
	home, err := GetDocksmithHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "layers"), nil
}

// GetCacheDir returns the directory where cache data is stored
func GetCacheDir() (string, error) {
	home, err := GetDocksmithHome()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "cache"), nil
}

// EnsureDirectories creates all necessary Docksmith directories if they don't exist
func EnsureDirectories() error {
	dirs := []func() (string, error){
		GetImagesDir,
		GetLayersDir,
		GetCacheDir,
	}
	
	for _, dirFunc := range dirs {
		dir, err := dirFunc()
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	
	return nil
}
