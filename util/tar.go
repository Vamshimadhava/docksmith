package util

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// CreateTarLayer creates a deterministic tar archive from a directory
// Timestamps are set to zero and files are sorted for reproducibility
func CreateTarLayer(sourceDir string, writer io.Writer) error {
	tw := tar.NewWriter(writer)
	defer tw.Close()
	
	// Collect all files first for sorting
	var files []string
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return err
	}
	
	// Sort for deterministic ordering
	sort.Strings(files)
	
	// Add files to tar
	for _, path := range files {
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		
		// Get relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		
		// Skip the source directory itself
		if relPath == "." {
			continue
		}
		
		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		
		// Use forward slashes and relative path
		header.Name = filepath.ToSlash(relPath)
		
		// Set timestamp to zero for reproducibility
		header.ModTime = time.Time{}
		header.AccessTime = time.Time{}
		header.ChangeTime = time.Time{}
		
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		
		// Write file content if it's a regular file
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			if _, err := io.Copy(tw, f); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	
	return nil
}

// ExtractTar extracts a tar archive to a destination directory
func ExtractTar(reader io.Reader, destDir string) error {
	tr := tar.NewReader(reader)
	
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		
		// Construct the full path
		target := filepath.Join(destDir, header.Name)
		
		// Ensure the target is within destDir (security check)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)) {
			return fmt.Errorf("illegal file path: %s", header.Name)
		}
		
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			// Create parent directories
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			
			// Create the file
			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	
	return nil
}
