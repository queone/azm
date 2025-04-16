package maz

import (
	"fmt"
	"sync"
	"time"

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
	// Above indicates that we are okay with receiving data that may not be the most
	// up-to-date. For this function, performance is prioritized over immediate
	// consistency. It allows the system to return data that might be slightly
	// stale but can be retrieved more quickly.
	apiUrl := ConstMgUrl + ApiEndpoint[t] + "/$count"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		return 0
	}
	count := utl.Int64(resp["value"]) // Try asserting response as a int64 value
	return count
}

// Gets object of given type from Azure by id. Updates entry in local cache.
func GetObjectFromAzureById(mazType, targetId string, z *Config) AzureObject {
	obj := AzureObject{}
	baseUrl := ConstMgUrl + ApiEndpoint[mazType]
	apiUrl := baseUrl + "/" + targetId
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
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
			resp, statCode, _ := ApiGet(apiUrl, z, params)
			if statCode != 200 {
				Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
			}
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
	if err := cache.Save(); err != nil {
		Logf("Failed to save cache: %v", err)
	}

	return obj // Return the found object or nil
}

// Fetches objects of the given type from Azure by displayName. It returns a list of
// matching objects, accounting for the possibility of multiple objects with the
// same displayName.
func GetObjectFromAzureByName(mazType, displayName string, z *Config) AzureObjectList {
	result := AzureObjectList{} // Initialize the result list
	apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "?$filter=displayName eq '" + displayName + "'"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
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
	apiUrl := ConstMgUrl + ApiEndpoint[mazType]

	// Use regular pagination for initial sync, delta for updates
	if cache.Count() == 0 {
		// Full sync (faster)
		switch mazType {
		case DirectoryUser:
			apiUrl += "?$select=id,displayName,userPrincipalName,onPremisesSamAccountName&$top=999"
		case DirectoryGroup:
			apiUrl += "?$select=id,displayName,description,isAssignableToRole,createdDateTime&$top=999"
		case Application:
			apiUrl += "?$select=id,displayName,appId,requiredResourceAccess,passwordCredentials&$top=999"
		case ServicePrincipal:
			apiUrl += "?$select=id,displayName,appId,accountEnabled,appOwnerOrganizationId,passwordCredentials&$top=999"
		case DirRoleDefinition:
			apiUrl += "?$select=id,displayName,description,isBuiltIn,isEnabled,templateId"
		case DirRoleAssignment:
			apiUrl += "?$select=id,directoryScopeId,principalId,roleDefinitionId"
		}
	} else {
		// Delta sync (efficient updates)
		switch mazType {
		case DirectoryUser:
			apiUrl += "/delta?$select=id,displayName,userPrincipalName,onPremisesSamAccountName"
		case DirectoryGroup:
			apiUrl += "/delta?$select=id,displayName,description,isAssignableToRole,createdDateTime"
		}
	}

	if len(cache.data) < 1 {
		z.AddMgHeader("Prefer", "return=minimal")
		z.AddMgHeader("deltaToken", "latest")
	}

	deltaLinkMap, err := cache.LoadDeltaLink()
	if err != nil {
		// Fall back to full sync if delta token fails
		Logf("Delta token load failed, falling back to full sync: %v", err)
		queryParams := "?$select=" + map[string]string{
			DirectoryUser:     "id,displayName,userPrincipalName,onPremisesSamAccountName",
			DirectoryGroup:    "id,displayName,description,isAssignableToRole,createdDateTime",
			Application:       "id,displayName,appId,requiredResourceAccess,passwordCredentials",
			ServicePrincipal:  "id,displayName,appId,accountEnabled,appOwnerOrganizationId,passwordCredentials",
			DirRoleDefinition: "id,displayName,description,isBuiltIn,isEnabled,templateId",
			DirRoleAssignment: "id,directoryScopeId,principalId,roleDefinitionId",
		}[mazType]
		// Only add $top for supported object types
		if mazType != DirRoleDefinition && mazType != DirRoleAssignment {
			queryParams += "&$top=999"
		}
		apiUrl = ConstMgUrl + ApiEndpoint[mazType] + queryParams
	} else if deltaLinkMap != nil {
		if deltaLink := utl.Str(deltaLinkMap["@odata.deltaLink"]); deltaLink != "" {
			apiUrl = deltaLink
		}
	}

	deltaSet, deltaLinkMap := FetchDirObjectsDelta(apiUrl, z, verbose)

	// Retry delta token save once before dying
	if err := cache.SaveDeltaLink(deltaLinkMap); err != nil {
		Logf("Delta token save failed, retrying once: %v", err)
		time.Sleep(1 * time.Second)
		if err := cache.SaveDeltaLink(deltaLinkMap); err != nil {
			utl.Die("Error saving delta link after retry: %v", err)
		}
	}

	cache.Normalize(mazType, deltaSet)
	if err := cache.Save(); err != nil {
		utl.Die("Error saving cache: %v", err)
	}
}

// Retrieves Azure directory object deltas. Returns the set of new or updated items, and
// a deltaLink for running the next future Azure query. Implements the code logic pattern
// described at docs.microsoft.com/en-us/graph/delta-query-overview
func FetchDirObjectsDelta(apiUrl string, z *Config, verbose bool) (AzureObjectList, AzureObject) {
	callCount := 1
	deltaSet := AzureObjectList{}
	deltaLinkMap := AzureObject{}

	const (
		workerCount   = 10
		resultBufSize = 10000
	)

	workQueue := make(chan string, 10)
	results := make(chan AzureObject, resultBufSize)
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go deltaWorker(workQueue, results, z, verbose, &wg)
	}

	// Close results when all workers are done
	go func() {
		wg.Wait()
		close(results)
	}()

	workQueue <- apiUrl
	processRemaining := false

	for {
		if processRemaining {
			deltaSet = drainResults(results)
			if verbose {
				fmt.Print(clrLine)
			}
			return deltaSet, deltaLinkMap
		}

		select {
		case obj, ok := <-results:
			if !ok {
				processRemaining = true
				continue
			}
			deltaSet = append(deltaSet, obj)
			if verbose && len(deltaSet)%100 == 0 {
				fmt.Printf("%sCall %05d : count %07d", clrLine, callCount, len(deltaSet))
			}

		default:
			resp, err := apiGetWithRetry(apiUrl, z, verbose, 3)
			if err != nil {
				close(workQueue)
				processRemaining = true
				continue
			}

			processApiResponse(resp, results)

			// Handle pagination
			if deltaLink := utl.Map(resp["@odata.deltaLink"]); deltaLink != nil {
				deltaLinkMap = deltaLink
				close(workQueue)
				processRemaining = true
			} else if nextLink := utl.Str(resp["@odata.nextLink"]); nextLink != "" {
				workQueue <- nextLink
				callCount++
				apiUrl = nextLink
			} else {
				close(workQueue)
				processRemaining = true
			}
		}
	}
}

func deltaWorker(workQueue <-chan string, results chan<- AzureObject, z *Config, verbose bool, wg *sync.WaitGroup) {
	defer wg.Done()
	for url := range workQueue {
		resp, _ := apiGetWithRetry(url, z, verbose, 3)
		processApiResponse(resp, results)
	}
}

func processApiResponse(resp map[string]interface{}, results chan<- AzureObject) {
	if value := utl.Slice(resp["value"]); value != nil {
		for _, item := range value {
			if obj := utl.Map(item); obj != nil {
				results <- AzureObject(obj)
			}
		}
	}
}

func drainResults(results <-chan AzureObject) AzureObjectList {
	deltaSet := AzureObjectList{}
	for obj := range results {
		deltaSet = append(deltaSet, obj)
	}
	return deltaSet
}

func apiGetWithRetry(url string, z *Config, verbose bool, maxRetries int) (AzureObject, error) {
	var resp AzureObject
	var err error

	for i := 0; i < maxRetries; i++ {
		resp, _, err = ApiGet(url, z, nil) // statCode intentionally ignored
		if err == nil {
			return resp, nil
		}
		Logf("%v\n", err)
		if verbose {
			fmt.Printf("%sHTTP error (Retry %d/%d): %v\n", clrLine, i+1, maxRetries, err)
		}
		time.Sleep(time.Second * time.Duration(1<<i))
	}
	return resp, err
}

// Deletes directory object of given type in Azure, with a confirmation prompt.
func DeleteDirObject(force bool, id, mazType string, z *Config) {
	// Note that 'id' may be a UUID or a displayName

	mazTypeName := MazTypeNames[mazType]
	obj := PreFetchAzureObject(mazType, id, z)
	if obj == nil {
		utl.Die("No %s with identifier %s\n", utl.Yel(mazTypeName), utl.Yel(id))
	}

	// Confirmation prompt
	fmt.Printf("Deleting below %s:\n", utl.Yel(mazTypeName))
	PrintObject(mazType, obj, z)
	if !force {
		msg := fmt.Sprintf("%s %s? y/n ", utl.Yel("Delete"), mazTypeName)
		if utl.PromptMsg(msg) != 'y' {
			utl.Die("Operation aborted by user.\n")
		}
	}

	// Delete object in Azure
	id = utl.Str(obj["id"])
	DeleteDirObjectInAzure(mazType, id, z)
}

// Deletes directory object of given type in Azure, and updates local cache.
func DeleteDirObjectInAzure(mazType, id string, z *Config) error {
	mazTypeName := MazTypeNames[mazType]
	apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "/" + id
	resp, statCode, _ := ApiDelete(apiUrl, z, nil)
	if statCode != 204 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 204 {
		fmt.Printf("Successfully %s %s!\n", utl.Gre("DELETED"), mazTypeName)

		// Also remove from local cache
		cache, err := GetCache(mazType, z)
		if err != nil {
			Logf("Failed to get cache for %s: %w\n", mazTypeName, err)
		}
		err = cache.Delete(id)
		if err == nil { // Only save if deletion succeeded
			err = cache.Save()
		}
		if err != nil {
			Logf("Failed to delete object with ID %s: %w\n", id, err)
		}
	} else {
		fmt.Printf("HTTP %d: Error creating %s: %s\n", statCode, mazTypeName, ApiErrorMsg(resp))
	}
	return nil
}

// Creates directory object of given type in Azure, with a confirmation prompt.
func CreateDirObject(force bool, obj AzureObject, mazType string, z *Config) AzureObject {
	// Present confirmation prompt if force isn't set
	mazTypeName := MazTypeNames[mazType]
	fmt.Printf("Creating new %s with below attributes:\n", utl.Yel(mazTypeName))
	utl.PrintYamlColor(obj)
	if !force {
		msg := fmt.Sprintf("%s %s ? y/n ", utl.Yel("Create"), mazTypeName)
		if utl.PromptMsg(msg) != 'y' {
			utl.Die("Operation aborted by user.\n")
		}
	}

	// Create the object in Azure, and return result
	azObj := CreateDirObjectInAzure(mazType, obj, z)

	return azObj
}

// Creates directory object of given type in Azure, and updates local cache.
func CreateDirObjectInAzure(mazType string, obj AzureObject, z *Config) AzureObject {
	azObj := AzureObject{}
	mazTypeName := MazTypeNames[mazType]

	// Creates object in Azure using obj as payload
	apiUrl := ConstMgUrl + ApiEndpoint[mazType]
	payload := obj
	resp, statCode, _ := ApiPost(apiUrl, z, payload, nil)
	if statCode != 201 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 201 {
		azObj = AzureObject(resp) // Cast newly created object to our standard type
		id := utl.Str(azObj["id"])
		fmt.Printf("Successfully %s %s with new ID %s\n", utl.Gre("CREATED"), mazTypeName, id)

		// Upsert object in local cache also
		cache, err := GetCache(mazType, z)
		if err != nil {
			Logf("Failed to get cache for %s: %w\n", mazTypeName, err)
		}
		err = cache.Upsert(azObj.TrimForCache(mazType))
		if err != nil {
			Logf("Failed to upsert object with ID %s: %w\n", id, err)
		}
		if err := cache.Save(); err != nil {
			Logf("Failed to save cache: %v", err)
		}
	} else {
		fmt.Printf("HTTP %d: Error creating %s: %s\n", statCode, mazTypeName, ApiErrorMsg(resp))
	}
	return azObj
}

// Updates directory object of given type in Azure, with a confirmation prompt.
func UpdateDirObject(force bool, id string, obj AzureObject, mazType string, z *Config) {
	mazTypeName := MazTypeNames[mazType]

	// Present confirmation prompt if force isn't set
	fmt.Printf("Update exiting %s with below attributes:\n", utl.Yel(mazTypeName))
	utl.PrintYamlColor(obj)
	if !force {
		msg := fmt.Sprintf("%s %s ? y/n ", utl.Yel("Update"), mazTypeName)
		if utl.PromptMsg(msg) != 'y' {
			utl.Die("Operation aborted by user.\n")
		}
	}

	// Update the object in Azure
	UpdateDirObjectInAzure(mazType, id, obj, z)
}

// Updates directory object of given type in Azure, and updates local cache.
func UpdateDirObjectInAzure(mazType, id string, obj AzureObject, z *Config) error {
	mazTypeName := MazTypeNames[mazType]
	apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "/" + id
	payload := obj
	resp, statCode, _ := ApiPatch(apiUrl, z, payload, nil)
	if statCode != 204 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 204 {
		fmt.Printf("Successfully %s %s!\n", utl.Gre("UPDATED"), mazTypeName)

		// Above API patch call does NOT return the updated object, so to update
		// the local cache we have to re-use our original item.
		obj["id"] = id // Ensure it has the id, so local cache update works

		// Upsert object in local cache also
		cache, err := GetCache(mazType, z)
		if err != nil {
			Logf("Failed to get cache for %s: %w\n", mazTypeName, err)
		}
		err = cache.Upsert(obj.TrimForCache(mazType))
		if err != nil {
			Logf("Failed to upsert object with ID %s: %w\n", id, err)
		}
		if err := cache.Save(); err != nil {
			Logf("Failed to save cache: %v", err)
		}
	} else {
		fmt.Printf("HTTP %d: Error updating %s: %s\n", statCode, mazTypeName, ApiErrorMsg(resp))
	}
	return nil
}

// Renames directory object of given type in Azure.
func RenameDirObject(force bool, mazType, from, newName string, z *Config) {
	// Note that 'from' can be ID or displayName

	mazTypeName := MazTypeNames[mazType]

	// Only supports renaming DirectoryGroup and DirRoleDefinition
	// Renaming App/SP is a special case has special function RenameAppSp()
	if mazType != DirectoryGroup && mazType != DirRoleDefinition {
		utl.Die("Rename not supported for %s object types\n", utl.Yel(mazTypeName))
	}

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
