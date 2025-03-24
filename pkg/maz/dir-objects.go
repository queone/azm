package maz

import (
	"fmt"

	"github.com/queone/utl"
)

// Returns the number of object entries in the local cache file for the given type.
func ObjectCountLocal(mazType string, z *Config) int64 {
	// This function works for any mazType and should really be in helper.go, not here.
	// But it's so closely tied to below ObjectCountAzure() that we are leaving here.

	// Initialize cache
	cache, err := GetCache(mazType, z)
	if err != nil {
		return 0 // If the cache cannot be loaded, return 0
	}
	return cache.Count() // Return the count of entries in the cache
}

// Returns the number of objects of given type in the Azure tenant.
func ObjectCountAzure(t string, z *Config) int64 {
	z.AddMgHeader("ConsistencyLevel", "eventual")
	apiUrl := ConstMgUrl + ApiEndpoint[t] + "/$count"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		return 0
	}
	count := utl.Int64(resp["value"]) // Try asserting response as a int64 value
	return count
}

// Returns an id:name map of objects of the given type.
func GetIdMapDirObjects(mazType string, z *Config) map[string]string {
	nameMap := make(map[string]string)
	objects := GetMatchingDirObjects(mazType, "", false, z) // false = get from cache, not Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy

	// Memory-walk the slice to gather these values more efficiently
	for i := range objects {
		obj := objects[i]                   // No need to cast; should already be AzureObject type
		id := utl.Str(obj["id"])            // Try casting as a string
		name := utl.Str(obj["displayName"]) // Try casting as a string
		if id == "" || name == "" {
			continue // Skip is either is not a string or empty
		}
		nameMap[id] = name
	}

	return nameMap
}

// Gets object of given type from Azure by id. Updates entry in local cache.
func GetObjectFromAzureById(mazType, targetId string, z *Config) AzureObject {
	obj := AzureObject{}
	baseUrl := ConstMgUrl + ApiEndpoint[mazType]
	apiUrl := baseUrl + "/" + targetId
	resp, _, _ := ApiGet(apiUrl, z, nil)
	// TODO: Maybe improve error checking and reporting?
	id := utl.Str(resp["id"]) // Try casting to a string
	if id != "" {
		// If we have an ID, then we found an object
		objMap := utl.Map(resp)   // Cast object to a map
		obj = AzureObject(objMap) // then to an AzureObject
	} else {
		// Check if the targetId is a Client ID/appId belonging to an AppSP pair
		if mazType == Application || mazType == ServicePrincipal {
			apiUrl := baseUrl
			params := map[string]string{"$filter": "appId eq '" + targetId + "'"}
			resp, _, _ := ApiGet(apiUrl, z, params)
			objList := utl.Slice(resp["value"]) // Try casting to a slice
			if objList != nil {
				count := len(objList)
				if count >= 1 {
					objMap := utl.Map(objList[0]) // Try casting the first object to a map
					if objMap != nil {
						obj = AzureObject(objMap) // then cast to an AzureObject
					}
					if count > 1 {
						msg := fmt.Sprintf("Warning! Found %d entries with this appId. Returning entry 0.", count)
						fmt.Println(utl.Yel(msg))
					}
				}
			}
		}
	}
	if obj == nil {
		return nil // No valid object found after all attempts
	}

	obj["maz_from_azure"] = true // Mark it as being from Azure

	// Update the object in the local cache
	cache, err := GetCache(mazType, z)
	if err != nil {
		fmt.Printf("Warning: Failed to load cache for type '%s': %v\n", mazType, err)
		return obj // Return the fetched object even if cache update fails
	}
	cache.Upsert(obj.TrimForCache(mazType))

	return obj // Return the found object or nil
}

// Fetches objects of the given type from Azure by displayName. It returns a list of
// matching objects, accounting for the possibility of multiple objects with the
// same displayName.
func GetObjectFromAzureByName(mazType, displayName string, z *Config) AzureObjectList {
	result := AzureObjectList{} // Initialize the result list
	apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "?$filter=displayName eq '" + displayName + "'"
	resp, _, _ := ApiGet(apiUrl, z, nil)
	matchingObjects := utl.Slice(resp["value"]) // Try casting to a slice
	if matchingObjects != nil {
		// It is a slice, let's process it
		for i := range matchingObjects {
			obj := utl.Map(matchingObjects[i]) // Try casting to a map
			if obj == nil {
				continue // Skip if not a map
			}
			result = append(result, AzureObject(obj))
		}
		return result
	}
	return result
}

// Retrieves existing object from Azure by its ID or displayName. This is
// typically used as preprocessing for operations like renaming, deleting,
// or updating the object.
func PreFetchAzureObject(mazType, identifier string, z *Config) AzureObject {
	if utl.ValidUuid(identifier) {
		return GetObjectFromAzureById(mazType, identifier, z)
	}

	matchingObjects := GetObjectFromAzureByName(mazType, identifier, z)
	if len(matchingObjects) == 0 {
		return nil
	}

	if len(matchingObjects) > 1 {
		fmt.Printf("Found multiple '%s' objects with same name '%s'\n", mazType, utl.Red(identifier))
		for _, x := range matchingObjects {
			fmt.Printf("  %s  %s\n", x["id"], x["displayName"])
		}
		utl.Die("%s. Pry processing by ID instead of name.\n", utl.Red("Aborting"))
	}

	return matchingObjects[0]
}

// Gets all objects of given type, matching on 'filter'. Returns the entire list if filter is empty "".
func GetMatchingDirObjects(mazType, filter string, force bool, z *Config) AzureObjectList {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		x := GetObjectFromAzureById(mazType, filter, z)
		if x != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{x}
		}
		// If not found, then filter will be used below in obj.HasString()
	}

	// Get current cache data, or initialize a new cache for this type
	cache, err := GetCache(mazType, z)
	if err != nil {
		utl.Die("Error: %v\n", err)
	}

	// Return an empty list if cache is nil and internet is not available
	internetIsAvailable := utl.IsInternetAvailable()
	if cache == nil && !internetIsAvailable {
		return AzureObjectList{} // Return empty list
	}

	// Determine if cache is empty or outdated and needs to be refreshed from Azure
	cacheNeedsRefreshing := force || cache.Count() < 1 || cache.Age() == 0 || cache.Age() > ConstMgCacheFileAgePeriod
	if internetIsAvailable && cacheNeedsRefreshing {
		RefreshLocalCacheWithAzure(mazType, cache, z, true) // Call Azure to refresh cache
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}

	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.StringSet{}            // Keep track of unique IDs to eliminate duplicates

	for i := range cache.data {
		obj := cache.data[i] // No need to cast; should already be AzureObject type

		// Extract ID and skip if it is empty or has already been seen
		id := utl.Str(obj["id"])
		if id == "" || ids.Exists(id) {
			continue
		}

		// Check if the object matches the filter
		if obj.HasString(filter) {
			matchingList = append(matchingList, obj) // Add matching object to the list
			ids.Add(id)                              // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all directory objects of given type from Azure and syncs them to local cache.
// Note that we are updating the cache via its pointer. Shows progress if verbose = true.
func RefreshLocalCacheWithAzure(mazType string, cache *Cache, z *Config, verbose bool) {
	// Setup REST API URL for the specific type
	apiUrl := ConstMgUrl + ApiEndpoint[mazType] // e.g., 'https://graph.microsoft.com/v1.0/groups'

	// Setup select criteria for each object type: what fields will trigger delta updates upon changing
	switch mazType {
	case DirectoryUser:
		apiUrl += "/delta?$select=id,displayName,userPrincipalName,onPremisesSamAccountName&$top=999"
	case DirectoryGroup:
		apiUrl += "/delta?$select=id,displayName,description,isAssignableToRole,createdDateTime&$top=999"
	case Application:
		apiUrl += "?$select=id,displayName,appId,requiredResourceAccess,passwordCredentials&$top=999"
	case ServicePrincipal:
		apiUrl += "?$select=id,displayName,appId,accountEnabled,appOwnerOrganizationId,passwordCredentials&$top=999"
	case DirRoleDefinition, DirRoleAssignment:
		//No additional adjustment required
	}

	if len(cache.data) < 1 {
		// These headers are only needed on the initial cache run
		z.AddMgHeader("Prefer", "return=minimal") // Focus on $select attributes deltas
		z.AddMgHeader("deltaToken", "latest")
	}

	// Prep to do a delta query if it is possible
	deltaLinkMap, err := cache.LoadDeltaLink() // Attempt to load a valid delta link
	if err != nil {
		utl.Die("Error loading delta link: %s\n", err.Error())
	}
	if deltaLinkMap != nil {
		// Try using delta link for the API call
		if deltaLink := utl.Str(deltaLinkMap["@odata.deltaLink"]); deltaLink != "" {
			apiUrl = deltaLink // Use delta link for the API call
		}
	}

	// Fetch Azure objects using the updated URL (either a full or a delta query)
	var deltaSet AzureObjectList
	deltaSet, deltaLinkMap = FetchDirObjectsDelta(apiUrl, z, verbose)

	// Save the new delta link for future calls
	if err := cache.SaveDeltaLink(deltaLinkMap); err != nil {
		utl.Die("Error saving delta link: %s\n", err.Error())
	}

	// Merge the deltaSet with the cache
	cache.Normalize(mazType, deltaSet)

	// Save the updated cache back to file
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated cache: %s\n", err.Error())
	}
}

// Retrieves Azure directory object deltas. Returns the set of new or updated items, and
// a deltaLink for running the next future Azure query. Implements the code logic pattern
// described at https://docs.microsoft.com/en-us/graph/delta-query-overview
func FetchDirObjectsDelta(apiUrl string, z *Config, verbose bool) (AzureObjectList, AzureObject) {
	callCount := 1 // Track number of API calls
	deltaSet := AzureObjectList{}
	deltaLinkMap := AzureObject{}

	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	for {
		if verbose && statCode != 200 {
			msg := fmt.Sprintf("%sHTTP %d: %s: Continuing to try...", rUp, statCode, ApiErrorMsg(resp))
			fmt.Printf("%s", utl.Yel(msg))
		}
		// Infinite for-loop until deltaLink appears (meaning we're done getting current delta set)
		objCount := 0
		thisBatch := utl.Slice(resp["value"]) // Try casting value as a slice
		if thisBatch != nil {
			// If its a valid slice
			objCount = len(thisBatch)
			for i := range thisBatch {
				objMap := utl.Map(thisBatch[i]) // Try casting element as a map
				if objMap == nil {
					continue // Skip this entry if not a map
				}
				deltaSet = append(deltaSet, AzureObject(objMap))
			}
		}

		if verbose {
			// Progress count indicator. Using global var rUp to overwrite last line. Defer newline until done
			fmt.Printf("%sCall %05d : count %05d", rUp, callCount, objCount)
		}

		// Return immediately when deltaLink appears
		if deltaLinkMap := utl.Map(resp["@odata.deltaLink"]); deltaLinkMap != nil {
			if verbose {
				fmt.Print(rUp) // Go up to overwrite progress line
			}
			return deltaSet, deltaLinkMap
		}

		// Get nextLink value
		nextLink := utl.Str(resp["@odata.nextLink"])
		if nextLink != "" {
			resp, statCode, _ = ApiGet(nextLink, z, nil) // Get next batch
			callCount++
		} else {
			if verbose {
				fmt.Print(rUp) // Go up to overwrite progress line
			}
			break // If nextLink is empty, we can break out of the loop
		}
	}
	return deltaSet, deltaLinkMap
}

// Deletes directory object of given type in Azure, and updates local cache.
func DeleteDirObjectInAzure(mazType, id string, z *Config) error {
	mazTypeName := MazTypeNames[mazType]
	apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "/" + id
	resp, statCode, _ := ApiDelete(apiUrl, z, nil)
	if statCode == 204 {
		msg := fmt.Sprintf("Successfully DELETED %s!", mazTypeName)
		fmt.Printf("%s\n", utl.Gre(msg))

		// Also remove from local cache
		cache, err := GetCache(mazType, z)
		if err != nil {
			return fmt.Errorf("failed to get cache for %s: %w", mazTypeName, err)
		}
		cache.Delete(id)
		// Ignoring the error for now, because many time it just doesn exist,
		// which is not an error. We'll need to revisit this code:
		// err = cache.Delete(id)
		// if err != nil {
		// 	return fmt.Errorf("failed to delete object with ID %s: %w", id, err)
		// }
	} else {
		return fmt.Errorf("http %d: %s", statCode, ApiErrorMsg(resp))
	}
	return nil
}

// Deletes directory object of given type in Azure, with a confirmation prompt.
func DeleteDirObject(force bool, id, mazType string, z *Config) error {
	// Note that 'id' may be a UUID or a displayName

	mazTypeName := MazTypeNames[mazType]
	obj := PreFetchAzureObject(mazType, id, z)
	if obj == nil {
		return fmt.Errorf("no %s with identifier '%s'", mazTypeName, id)
	}

	// Confirmation prompt
	PrintObject(mazType, obj, z)
	if !force {
		msg := utl.Yel("Delete " + mazTypeName + "? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			return fmt.Errorf("operation aborted by user")
		}
	}

	// Delete object in Azure
	id = utl.Str(obj["id"])
	err := DeleteDirObjectInAzure(mazType, id, z)
	if err != nil {
		return fmt.Errorf("issue with delete: %w", err)
	}

	return nil
}

// Creates directory object of given type in Azure, and updates local cache.
func CreateDirObjectInAzure(mazType string, obj AzureObject, z *Config) (AzureObject, error) {
	mazTypeName := MazTypeNames[mazType]

	// Creates object in Azure using obj as payload
	apiUrl := ConstMgUrl + ApiEndpoint[mazType]
	payload := obj
	resp, statCode, _ := ApiPost(apiUrl, z, payload, nil)
	if statCode == 201 {
		msg := fmt.Sprintf("Successfully CREATED %s!", mazTypeName)
		fmt.Printf("%s\n", utl.Gre(msg))

		azObj := AzureObject(resp) // Newly created object
		id := utl.Str(azObj["id"])

		// Upsert object in local cache also
		cache, err := GetCache(mazType, z)
		if err != nil {
			return azObj, fmt.Errorf("failed to get cache for %s: %w", mazTypeName, err)
		}
		err = cache.Upsert(azObj.TrimForCache(mazType))
		if err != nil {
			return azObj, fmt.Errorf("failed to upsert object with ID %s: %w", id, err)
		}
		return azObj, nil
	} else {
		return nil, fmt.Errorf("http %d: filed to create %s:%s", statCode,
			mazTypeName, ApiErrorMsg(resp))
	}
}

// Creates directory object of given type in Azure, with a confirmation prompt.
func CreateDirObject(force bool, obj AzureObject, mazType string, z *Config) (AzureObject, error) {
	// Present confirmation prompt if force isn't set
	mazTypeName := MazTypeNames[mazType]
	fmt.Printf("%s\n", utl.Yel("Creating new "+mazTypeName+" with below attributes:"))
	utl.PrintYamlColor(obj)
	if !force {
		msg := utl.Yel("Create " + mazTypeName + "? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			return nil, fmt.Errorf("operation aborted by user")
		}
	}

	// Create the object in Azure
	var azObj AzureObject
	var err error
	if azObj, err = CreateDirObjectInAzure(mazType, obj, z); err != nil {
		return azObj, fmt.Errorf("%s", err)
	}

	return azObj, nil
}

// Updates directory object of given type in Azure, and updates local cache.
func UpdateDirObjectInAzure(mazType, id string, obj AzureObject, z *Config) error {
	mazTypeName := MazTypeNames[mazType]
	apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "/" + id
	payload := obj
	resp, statCode, _ := ApiPatch(apiUrl, z, payload, nil)
	if statCode == 204 {
		msg := fmt.Sprintf("Successfully UPDATED %s!", mazTypeName)
		fmt.Printf("%s\n", utl.Gre(msg))

		// Above API patch call does NOT return the updated object, so to update
		// the local cache we have to re-use our original item.
		obj["id"] = id // Ensure it has the id, so local cache update works

		// Upsert object in local cache also
		cache, err := GetCache(mazType, z)
		if err != nil {
			return fmt.Errorf("failed to get cache for %s: %w", mazTypeName, err)
		}
		err = cache.Upsert(obj.TrimForCache(mazType))
		if err != nil {
			return fmt.Errorf("failed to upsert object with ID %s: %w", id, err)
		}
	} else {
		return fmt.Errorf("http %d: %s", statCode, ApiErrorMsg(resp))
	}
	return nil
}

// Updates directory object of given type in Azure, with a confirmation prompt.
func UpdateDirObject(force bool, id string, obj AzureObject, mazType string, z *Config) {
	mazTypeName := MazTypeNames[mazType]

	// Present confirmation prompt if force isn't set
	fmt.Printf("%s\n", utl.Yel("Update "+mazTypeName+" with below attributes:"))
	utl.PrintYamlColor(obj)
	if !force {
		msg := utl.Yel("Update " + mazTypeName + "? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Update the object in Azure
	err := UpdateDirObjectInAzure(mazType, id, obj, z)
	if err != nil {
		fmt.Println(err)
	}
}

// Renames directory object of given type in Azure.
func RenameDirObject(force bool, from, newName, mazType string, z *Config) {
	// Note that 'from' can be ID or displayName
	mazTypeName := MazTypeNames[mazType]

	x := PreFetchAzureObject(mazType, from, z)
	if x == nil {
		utl.Die("No such %s\n", mazTypeName)
	}

	id := utl.Str(x["id"])

	// Confirmation prompt
	if !force {
		oldName := utl.Str(x["displayName"])
		msg := utl.Yel("Rename "+mazTypeName+" "+id+"\n  from \"") + utl.Blu(oldName) +
			utl.Yel("\"\n    to \"") + utl.Blu(newName) + utl.Yel("\"\n? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Update the object in Azure
	obj := AzureObject{"displayName": newName}
	// The obj payload only requires the displayName
	err := UpdateDirObjectInAzure(mazType, id, obj, z)
	if err != nil {
		fmt.Println(err)
	}
}

// Find JSON object with given ID in slice
func FindObjectOld(objSet []interface{}, id string) map[string]interface{} {
	for _, item := range objSet {
		if x := utl.Map(item); x != nil {
			if utl.Str(x["id"]) == id {
				return x
			}
		}
	}
	return nil
}
