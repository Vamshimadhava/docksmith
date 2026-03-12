package cache

import (
	"docksmith/layers"
	"docksmith/util"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// CacheEntry represents a cached build step
type CacheEntry struct {
	Key    string        `json:"key"`
	Layer  layers.Layer  `json:"layer"`
}

// CacheIndex holds all cache entries
type CacheIndex struct {
	Entries map[string]layers.Layer `json:"entries"`
	mu      sync.RWMutex
}

var (
	globalCache *CacheIndex
	once        sync.Once
)

// GetCache returns the global cache index, initializing it if needed
func GetCache() *CacheIndex {
	once.Do(func() {
		globalCache = &CacheIndex{
			Entries: make(map[string]layers.Layer),
		}
		globalCache.Load()
	})
	return globalCache
}

// getCacheIndexPath returns the path to the cache index file
func getCacheIndexPath() (string, error) {
	cacheDir, err := util.GetCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(cacheDir, "index.json"), nil
}

// Load loads the cache index from disk
func (c *CacheIndex) Load() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	path, err := getCacheIndexPath()
	if err != nil {
		return err
	}
	
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Cache doesn't exist yet, start fresh
			c.Entries = make(map[string]layers.Layer)
			return nil
		}
		return err
	}
	
	var entries map[string]layers.Layer
	if err := json.Unmarshal(data, &entries); err != nil {
		return err
	}
	
	c.Entries = entries
	return nil
}

// Save persists the cache index to disk
func (c *CacheIndex) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	path, err := getCacheIndexPath()
	if err != nil {
		return err
	}
	
	data, err := json.MarshalIndent(c.Entries, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

// Lookup checks if a cache entry exists for the given key
// Returns the layer and true if found, otherwise nil and false
func (c *CacheIndex) Lookup(key string) (*layers.Layer, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	layer, exists := c.Entries[key]
	if !exists {
		return nil, false
	}
	
	// Verify the layer file still exists
	if !layers.LayerExists(layer.Digest) {
		return nil, false
	}
	
	return &layer, true
}

// Store adds a new cache entry
func (c *CacheIndex) Store(key string, layer layers.Layer) error {
	c.mu.Lock()
	c.Entries[key] = layer
	c.mu.Unlock()
	
	return c.Save()
}

// Clear removes all cache entries
func (c *CacheIndex) Clear() error {
	c.mu.Lock()
	c.Entries = make(map[string]layers.Layer)
	c.mu.Unlock()
	
	return c.Save()
}

// ComputeCacheKey is a convenience wrapper around util.ComputeCacheKey
func ComputeCacheKey(prevDigest, instruction, workDir string, env map[string]string, srcHashes []string) string {
	return util.ComputeCacheKey(prevDigest, instruction, workDir, env, srcHashes)
}
