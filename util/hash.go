package util

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"sort"
)

// ComputeSHA256 computes the SHA256 hash of data and returns it as "sha256:<hex>"
func ComputeSHA256(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(hash[:]))
}

// ComputeFileSHA256 computes the SHA256 hash of a file
func ComputeFileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil))), nil
}

// ComputeReaderSHA256 computes the SHA256 hash of data from a reader
func ComputeReaderSHA256(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("sha256:%s", hex.EncodeToString(h.Sum(nil))), nil
}

// ComputeCacheKey generates a cache key from instruction context
func ComputeCacheKey(prevDigest, instruction, workDir string, env map[string]string, srcHashes []string) string {
	// Sort environment variables for deterministic key
	envKeys := make([]string, 0, len(env))
	for k := range env {
		envKeys = append(envKeys, k)
	}
	sort.Strings(envKeys)
	
	envStr := ""
	for _, k := range envKeys {
		envStr += fmt.Sprintf("%s=%s\n", k, env[k])
	}
	
	srcHashStr := ""
	for _, h := range srcHashes {
		srcHashStr += h + "\n"
	}
	
	data := fmt.Sprintf("%s\n%s\n%s\n%s\n%s",
		prevDigest, instruction, workDir, envStr, srcHashStr)
	
	return ComputeSHA256([]byte(data))
}
