package util

import (
	"os"
	"path/filepath"
	"strings"
)

// MatchGlob matches files using glob patterns (* and **)
// Returns a list of matching file paths
func MatchGlob(baseDir, pattern string) ([]string, error) {
	var matches []string
	
	// Handle "." as "all files recursively"
	if pattern == "." {
		err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if info.Mode().IsRegular() {
				matches = append(matches, path)
			}
			return nil
		})
		return matches, err
	}
	
	// Handle ** recursive glob
	if strings.Contains(pattern, "**") {
		parts := strings.Split(pattern, "**")
		prefix := strings.TrimSuffix(parts[0], "/")
		suffix := ""
		if len(parts) > 1 {
			suffix = strings.TrimPrefix(parts[1], "/")
		}
		
		startDir := filepath.Join(baseDir, prefix)
		
		err := filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			
			relPath, err := filepath.Rel(baseDir, path)
			if err != nil {
				return err
			}
			
			// Check if it matches the suffix pattern
			if suffix == "" || strings.HasSuffix(relPath, suffix) {
				if info.Mode().IsRegular() {
					matches = append(matches, path)
				}
			} else if suffix != "" {
				// Try glob matching for the suffix
				matched, _ := filepath.Match(suffix, filepath.Base(path))
				if matched && info.Mode().IsRegular() {
					matches = append(matches, path)
				}
			}
			
			return nil
		})
		
		if err != nil {
			return nil, err
		}
	} else {
		// Simple glob pattern
		fullPattern := filepath.Join(baseDir, pattern)
		globMatches, err := filepath.Glob(fullPattern)
		if err != nil {
			return nil, err
		}
		
		for _, match := range globMatches {
			info, err := os.Stat(match)
			if err != nil {
				continue
			}
			if info.Mode().IsRegular() {
				matches = append(matches, match)
			}
		}
	}
	
	return matches, nil
}

// CopyFile copies a single file from src to dst
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	
	// Create parent directory if needed
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	
	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()
	
	_, err = srcFile.WriteTo(dstFile)
	return err
}
