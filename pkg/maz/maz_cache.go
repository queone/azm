package maz

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sync"

	"github.com/queone/utl"
)

// Cache type
type Cache struct {
	filePath      string
	deltaLinkFile string
	data          AzureObjectList
	mu            sync.Mutex
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

// Initializes a Cache instance for a given type.
// If the cache file exists, it loads the existing cache; otherwise, it creates a new one.
func GetCache(mazType string, z *Config) (*Cache, error) {
	// Ensure the type is valid
	suffix, ok := CacheSuffix[mazType]
	if !ok {
		return nil, fmt.Errorf("invalid object type code: %s", utl.Red(mazType))
	}

	// Construct both file paths
	cacheFile := filepath.Join(MazConfigDir, z.TenantId+suffix+".bin")
	deltaLinkFile := cacheFile[:len(cacheFile)-4] + "_link.bin" // Replace ".bin" with "_link.bin"

	cache := &Cache{
		filePath:      cacheFile,
		deltaLinkFile: deltaLinkFile,
	}

	// Try loading the cache
	if err := cache.Load(); err != nil {
		if os.IsNotExist(err) {
			// If the file doesn't exist, initialize an empty cache and create the file
			cache.data = AzureObjectList{}
			if saveErr := cache.Save(); saveErr != nil {
				return nil, fmt.Errorf("failed to create new cache file: %w", saveErr)
			}
		} else {
			return nil, fmt.Errorf("unexpected error while loading cache: %w", err)
		}
	}
	return cache, nil
}

// Purges cache and delta link files for a given type.
func PurgeCacheFiles(mazType string, z *Config) error {
	// Validate the input type and get the suffix.
	suffix, ok := CacheSuffix[mazType]
	if !ok {
		msg := "invalid object type code"
		Logf("Error: %s %s\n", msg, utl.Red(mazType))
		return fmt.Errorf("%s: %s", msg, mazType)
	}

	// Build the cache and delta link file paths without loading the cache
	cacheFile := filepath.Join(MazConfigDir, z.TenantId+suffix+".bin")
	deltaLinkFile := cacheFile[:len(cacheFile)-4] + "_link.bin" // Replace ".bin" with "_link.bin"

	// Remove the cache file.
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		msg := "failed to remove cache file"
		Logf("Error: %s %s\n", msg, utl.Red(cacheFile))
		return fmt.Errorf("%s: %w", msg, err)
	}

	// Remove the delta link file.
	if err := os.Remove(deltaLinkFile); err != nil && !os.IsNotExist(err) {
		msg := "failed to remove deltaLink file"
		Logf("Error: %s %s\n", msg, utl.Red(deltaLinkFile))
		return fmt.Errorf("%s: %w", msg, err)
	}

	return nil
}

// Deletes both the cache file and the deltaLink file from the filesystem.
func (c *Cache) Erase() error {
	if err := os.Remove(c.filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}
	if err := os.Remove(c.deltaLinkFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove deltaLink file: %w", err)
	}
	return nil
}

// LoadDeltaLink loads the delta link from the file, if it exists and is valid.
func (c *Cache) LoadDeltaLink() (AzureObject, error) {
	if !utl.FileUsable(c.deltaLinkFile) || utl.FileAge(c.deltaLinkFile) >= (3660*24*27) {
		// Delta link file is either unusable or expired
		// Note that deltaLink file age has to be within 30 days (we do 27)
		return nil, nil
	}
	deltaLinkMap, err := LoadFileBinaryObject(c.deltaLinkFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load delta link: %w", err)
	}
	return deltaLinkMap, nil
}

// SaveDeltaLink saves the provided delta link to the file.
func (c *Cache) SaveDeltaLink(deltaLinkMap AzureObject) error {
	return SaveFileBinaryObject(c.deltaLinkFile, deltaLinkMap, 0600)
}

// Age returns the age of the cache file in seconds. If the file does not
// exist or is empty, it returns -1.
func (c *Cache) Age() int64 {
	return utl.FileAge(c.filePath)
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

func (c *Cache) upsertLocked(obj AzureObject) error {
	id := ExtractID(obj)
	if id == "" {
		return fmt.Errorf("object with blank ID not added to cache")
	}

	if existingObj := c.data.FindById(id); existingObj != nil {
		MergeAzureObjects(obj, *existingObj)
	} else {
		c.data = append(c.data, obj)
	}
	return nil
}

// Merges the deltaSet with the current cache data.
func (c *Cache) Normalize(mazType string, deltaSet AzureObjectList) {
	// Process changes under single lock
	c.mu.Lock()
	defer c.mu.Unlock()

	// 1. Process deltaSet to track changes
	deletedIds := make(utl.StringSet)                   // Track IDs to delete
	uniqueIds := make(utl.StringSet)                    // Track unique IDs in the deltaSet
	mergeSet := make(AzureObjectList, 0, len(deltaSet)) // List for new/updated objects in deltaSet

	for _, obj := range deltaSet {
		id := utl.Str(obj["id"])
		if id == "" {
			continue
		}
		// Check for deletions first (most delta sets are <5% deletions)
		if obj["@removed"] != nil || obj["members@delta"] != nil {
			deletedIds[id] = struct{}{}
			continue
		}
		// Dedupe in mergeSet
		if _, exists := uniqueIds[id]; !exists {
			uniqueIds[id] = struct{}{}
			mergeSet = append(mergeSet, obj)
		}
	}

	// 2. Batch deletion optimized for AzureObjectList
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
				fmt.Printf("WARNING: object with blank ID not added to cache\n")
				continue
			}
			c.data = append(c.data, obj)
		}
	} else {
		for _, obj := range mergeSet {
			if err := c.upsertLocked(obj); err != nil {
				fmt.Printf("WARNING: %v\n", err)
			}
		}
	}
}
