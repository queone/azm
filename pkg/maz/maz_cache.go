package maz

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"github.com/queone/utl"
)

// Cache type
type Cache struct {
	filePath        string
	deltaLinkFile   string
	partialFilePath string // file path for saving in-progress deltaSet
	data            AzureObjectList
	mu              sync.Mutex
}

// Cache initialization flow:
// 1. NewCache() - Creates instance with paths
// 2. Either:
//    a) GetCache() - Normal usage (loads or creates)
//    b) Manual setup - For special cases (resume, purge, etc)

// Creates a Cache instance with properly initialized paths but doesn't load data
func NewCache(mazType string, z *Config) (*Cache, error) {
	if z == nil || z.TenantId == "" {
		return nil, errors.New("invalid config: missing tenant ID")
	}

	suffix, ok := CacheSuffix[mazType]
	if !ok {
		return nil, fmt.Errorf("invalid object type code: %s", utl.Red(mazType))
	}

	cacheFile := filepath.Join(MazConfigDir, z.TenantId+suffix+".bin")
	return &Cache{
		filePath:        cacheFile,
		deltaLinkFile:   cacheFile[:len(cacheFile)-4] + "_link.bin",
		partialFilePath: cacheFile[:len(cacheFile)-4] + "_partial.bin",
		data:            AzureObjectList{},
	}, nil
}

// Loads or creates a cache, initializing all required paths
func GetCache(mazType string, z *Config) (*Cache, error) {
	cache, err := NewCache(mazType, z)
	if err != nil {
		return nil, err
	}

	if err := cache.Load(); err != nil {
		if os.IsNotExist(err) {
			// Initialize new cache file
			if err := cache.Save(); err != nil {
				return nil, fmt.Errorf("failed to create new cache file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("unexpected error while loading cache: %w", err)
		}
	}
	return cache, nil
}

// Extracts the Azure object's ID
func ExtractID(obj AzureObject) string {
	// 'id' may be a full path or just a raw ID (e.g., Entra role assignment IDs)
	// Try 'id' first
	if id := path.Base(utl.Str(obj["id"])); id != "" && id != "." && id != "/" {
		return id
	}
	// Fallback to 'name'
	if id := path.Base(utl.Str(obj["name"])); id != "" && id != "." && id != "/" {
		return id
	}
	// Fallback to 'subscriptionId'
	if id := path.Base(utl.Str(obj["subscriptionId"])); id != "" && id != "." && id != "/" {
		return id
	}
	return ""
}

// Attempts to resume cache normalization from a partial delta set file if available.
func (c *Cache) ResumeFromPartialDelta(mazType string) error {
	// If a usable partial file exists, always normalize it into the cache
	if utl.FileUsable(c.partialFilePath) {
		Logf("Partial delta set detected - loading from: %s\n", c.partialFilePath)
		partialSet, err := LoadFileBinaryList(c.partialFilePath, false)
		if err == nil && len(partialSet) > 0 {
			Logf("Loaded %d items from partial delta set. Normalizing...\n", len(partialSet))

			// Normalize the cache with the partial set
			c.Normalize(mazType, partialSet)

			// Save the cache after normalization
			if err := c.Save(); err != nil {
				return fmt.Errorf("error saving cache after partial normalize: %w", err)
			}

			// Clean up the partial file once processed
			if removeErr := os.Remove(c.partialFilePath); removeErr != nil {
				// Retry once if deletion fails (e.g., transient file lock)
				time.Sleep(500 * time.Millisecond)
				removeErr = os.Remove(c.partialFilePath)
				if removeErr != nil {
					Logf("Error deleting partial file %s after retry: %v\n", c.partialFilePath, removeErr)
				}
			}
		} else {
			Logf("WARNING: Failed to read partial file: %v — continuing with delta fetch\n", err)
		}
	}
	return nil
}

// Purges files associated with cache for a given type.
func PurgeCacheFiles(mazType string, z *Config) error {
	// Create a minimal Cache instance just for file paths
	cache, err := NewCache(mazType, z)
	if err != nil {
		Logf("Error: %v\n", err)
		return err
	}
	return cache.Erase()
}

// Deletes files associated with the cache
func (c *Cache) Erase() error {
	files := []string{c.filePath, c.deltaLinkFile, c.partialFilePath}
	for _, f := range files {
		if err := os.Remove(f); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove %q: %w", f, err)
		}
	}
	return nil
}

// Loads the delta link map from the file, if it exists and is valid.
func (c *Cache) LoadDeltaLink() (AzureObject, error) {
	if !utl.FileUsable(c.deltaLinkFile) || utl.FileAge(c.deltaLinkFile) >= (3660*24*27) {
		// Delta link file is either unusable or expired
		// Note that deltaLink file age has to be within 30 days (we do 27)
		return nil, nil
	}
	deltaLinkMap, err := LoadFileBinaryMap(c.deltaLinkFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load delta link: %w", err)
	}
	return deltaLinkMap, nil
}

// Saves the provided delta link map to the file.
func (c *Cache) SaveDeltaLink(deltaLinkMap AzureObject) error {
	return SaveFileBinaryMap(c.deltaLinkFile, deltaLinkMap, 0600)
}

// Load cache from file
func (c *Cache) Load() error {
	// TODO: Maybe take 'compressed' boolean option?
	loadedData, err := LoadFileBinaryList(c.filePath, false) // false = not compressed
	if err != nil {
		return err
	}
	c.data = loadedData
	return nil
}

// Save cache to file
func (c *Cache) Save() error {
	// TODO: Maybe take 'compressed' boolean option?

	// Lock during in-memory operations only
	c.mu.Lock()
	defer c.mu.Unlock()

	return SaveFileBinaryList(c.filePath, c.data, 0600, false)
}

// Age returns the age of the cache file in seconds. If the file does not
// exist or is empty, it returns -1.
func (c *Cache) Age() int64 {
	return utl.FileAge(c.filePath)
}

// Count returns the number of entries in the cache.
func (c *Cache) Count() int64 {
	return int64(len(c.data))
}

// Removes an object by its ID from the in-memory cache.
func (c *Cache) Delete(id string) error {
	// Note: You must call Save() separately to persist changes to disk.

	// Lock during in-memory operations only
	c.mu.Lock()
	defer c.mu.Unlock()

	// Attempt to delete the object from the cache data
	if !c.data.DeleteById(id) {
		return fmt.Errorf("failed to delete object %s from cache", id)
	}
	return nil
}

// DeleteById removes a single object
func (c *Cache) DeleteById(id string) {
	newData := make(AzureObjectList, 0, len(c.data))
	for _, obj := range c.data {
		existingId := ExtractID(obj)
		if existingId != id {
			newData = append(newData, obj)
		}
	}
	c.data = newData
}

func (c *Cache) Upsert(obj AzureObject) error {
	// Lock during in-memory operations only
	c.mu.Lock()
	defer c.mu.Unlock()

	id := ExtractID(obj)
	if id == "" {
		return fmt.Errorf("invalid object ID (empty) — not cached")
	}
	if id == "." {
		return fmt.Errorf("invalid object ID ('.') — not cached")
	}
	if id == "/" {
		return fmt.Errorf("invalid object ID ('/') — not cached")
	}

	// Check if the object already exists in the cache
	existingObj := c.data.FindById(id) // Use FindById to locate the existing object
	if existingObj != nil {
		Logf("UPDATE cache object %s\n", utl.Mag(id))
		// Merge the new object into the existing one in place
		MergeAzureObjects(obj, *existingObj)
	} else {
		Logf("ADD cache object %s\n", utl.Mag(id))
		c.data = append(c.data, obj) // Add the new object to the cache
	}

	return nil
}

// BatchDeleteByIds removes multiple objects in one pass (O(n) instead of O(n*m))
func (c *Cache) BatchDeleteByIds(ids utl.StringSet) {
	newData := make(AzureObjectList, 0, len(c.data))
	for _, obj := range c.data {
		if _, deleted := ids[utl.Str(obj["id"])]; !deleted {
			newData = append(newData, obj)
		}
	}
	c.data = newData
}

// Recursively merges the keys from AzureObject a into b. Existing object b attributes
// are overwritten if there's a conflict.
func MergeAzureObjects(newObj, existingObj AzureObject) {
	for key, newValue := range newObj {
		if existingValue, exists := existingObj[key]; exists {
			// If both values are AzureObjects, recursively merge them
			if newMap, okNew := newValue.(AzureObject); okNew {
				if existingMap, okExisting := existingValue.(AzureObject); okExisting {
					// Recursively merge nested AzureObjects
					MergeAzureObjects(newMap, existingMap)
					continue
				}
			}
		}
		// Otherwise, overwrite or add the new value
		existingObj[key] = newValue
	}
}

// Merges the deltaSet with the current cache data.
func (c *Cache) Normalize(mazType string, deltaSet AzureObjectList) {
	Logf("Normalizing cache...\n")
	start := time.Now()

	// Early Exit for Empty Deltas
	if len(deltaSet) == 0 {
		Logf("Empty deltaSet received - no changes to process\n")
		return
	}

	// 1. Process deltaSet to track changes
	deletedIds := make(utl.StringSet)             // Track IDs to delete
	uniqueUpdates := make(map[string]AzureObject) // Track unique ID -> object
	for _, obj := range deltaSet {
		id := ExtractID(obj)
		if id == "" {
			continue
		}
		// Check for deletions first (most delta sets are <5% deletions)
		if obj["@removed"] != nil || obj["members@delta"] != nil {
			deletedIds[id] = struct{}{}
			continue
		}
		// Dedupe in update set (keep last seen object per ID)
		uniqueUpdates[id] = obj
	}

	// Build mergeSet from unique map
	mergeSet := make(AzureObjectList, 0, len(uniqueUpdates))
	for _, obj := range uniqueUpdates {
		mergeSet = append(mergeSet, obj)
	}

	// Metric collection
	Logf("Delta stats: %d total items, %d new/updated, %d deleted\n",
		len(deltaSet), len(mergeSet), len(deletedIds))

	// 2. Batch deletion optimized for AzureObjectList
	// Process changes under single lock
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(deletedIds) > 0 {
		c.BatchDeleteByIds(deletedIds)
	}

	// 3. Sequential upsert
	if c.Count() == 0 {
		// Optimized path for initial load
		// Pre-allocate slice
		c.data = make(AzureObjectList, 0, len(mergeSet))

		// Bulk append without per-item processing, with proper ID checking
		for _, obj := range mergeSet {
			id := ExtractID(obj)
			if id == "" {
				Logf("WARNING: object with blank ID not added to cache\n")
				continue
			}
			c.data = append(c.data, obj)
		}
	} else {
		// Fast index for current data
		existingIndex := make(map[string]int, len(c.data))
		for i, obj := range c.data {
			id := ExtractID(obj)
			if id != "" {
				existingIndex[id] = i
			}
		}

		for _, obj := range mergeSet {
			id := ExtractID(obj)
			if id == "" {
				Logf("WARNING: object with blank ID not processed\n")
				continue
			}
			if idx, exists := existingIndex[id]; exists {
				c.data[idx] = obj
			} else {
				c.data = append(c.data, obj)
			}
		}
	}

	Logf("Normalize completed in %v (%.1f items/sec)\n",
		time.Since(start),
		float64(len(deltaSet))/time.Since(start).Seconds())
}
