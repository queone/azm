package maz

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/queone/utl"
)

// Cache type
type Cache struct {
	filePath      string
	deltaLinkFile string
	data          AzureObjectList
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
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+suffix+".bin")
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

// Removes cache files for a given type code and configuration.
// It ensures both the cache file and deltaLink file associated with the type are deleted.
func RemoveCacheFiles(t string, z *Config) error {
	// Validate the input type and get the suffix.
	suffix, ok := CacheSuffix[t]
	if !ok {
		return fmt.Errorf("invalid object type code: %s", utl.Red(t))
	}

	// Construct the cache file and delta link file paths without loading the cache.
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+suffix+".bin")
	deltaLinkFile := cacheFile[:len(cacheFile)-4] + "_link.bin" // Replace ".bin" with "_link.bin"

	// Remove the cache file.
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove cache file: %w", err)
	}

	// Remove the delta link file.
	if err := os.Remove(deltaLinkFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove deltaLink file: %w", err)
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
	return SaveFileBinaryList(c.filePath, c.data, 0600, false)
}

// Count returns the number of entries in the cache.
func (c *Cache) Count() int64 {
	return int64(len(c.data))
}

// Removes an object by its ID from the cache and saves the updated cache to disk.
func (c *Cache) Delete(id string) error {
	// Attempt to delete the object from the cache data
	if !c.data.DeleteById(id) {
		return fmt.Errorf("failed to delete object %s from cache", id)
	}

	// Save the updated cache back to the file
	if err := c.Save(); err != nil {
		return fmt.Errorf("failed to save updated cache: %w", err)
	}

	return nil
}

func (c *Cache) Upsert(obj AzureObject) error {
	id := utl.Str(obj["id"])
	if id == "" {
		id = utl.Str(obj["name"]) // Some objects use 'name' for ID
		if id == "" {
			id = utl.Str(obj["subscriptionId"]) // Subscriptions use this for ID
			if id == "" {
				return fmt.Errorf("object with blank ID not added to cache")
			}
		}
	}

	// Check if the object already exists in the cache
	existingObj := c.data.FindById(id) // Use FindById to locate the existing object
	if existingObj != nil {
		// Merge the new object into the existing one in place
		MergeAzureObjects(obj, *existingObj)
	} else {
		// Add the new object to the cache
		c.data.Add(obj)
	}

	// Save the updated cache to ensure persistence
	if err := c.Save(); err != nil {
		return fmt.Errorf("failed to save updated cache: %w", err)
	}

	return nil
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
	deletedIds := utl.StringSet{} // Track IDs to delete
	uniqueIds := utl.StringSet{}  // Track unique IDs in the deltaSet
	mergeSet := AzureObjectList{} // List for new/updated objects in deltaSet

	// 1. Process deltaSet to build mergeSet and track deleted IDs
	for i := range deltaSet {
		item := &deltaSet[i] // Access the element directly via pointer
		id := utl.Str((*item)["id"])
		if id == "" {
			continue // Skip items without a valid "id"
		}

		if (*item)["@removed"] == nil && (*item)["members@delta"] == nil {
			// New or updated object
			if !uniqueIds.Exists(id) {
				mergeSet.Add(*item) // Add to mergeSet
				uniqueIds.Add(id)
			}
		} else {
			// Deleted object
			deletedIds.Add(id)
		}
	}

	// 2. Remove deleted objects from the cache
	for id := range deletedIds {
		c.data.DeleteById(id) // Use the DeleteById method to remove objects
	}

	// 3. Add new entries from mergeSet to the cache
	for i := range mergeSet {
		item := &mergeSet[i] // Access the element directly via pointer
		if err := c.Upsert(*item); err != nil {
			fmt.Printf("WARNING: Failed to upsert cache object with ID '%s': %v\n", (*item)["id"], err)
		}
	}
}
