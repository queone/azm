package maz

import (
	"fmt"
	"time"

	"github.com/queone/utl"
)

// Returns the number of object entries in the local cache file for the given type.
func ObjectCountLocal(t string, z *Config) int64 {
	// Initialize cache
	cache, err := GetCache(t, z)
	if err != nil {
		return 0 // If the cache cannot be loaded, return 0
	}
	return cache.Count() // Return the count of entries in the cache
}

// Returns the number of objects of given type in the Azure tenant.
func ObjectCountAzure(t string, z *Config) int64 {
	z.AddMgHeader("ConsistencyLevel", "eventual")
	apiUrl := ConstMgUrl + ApiEndpoint[t] + "/$count"
	r, _, _ := ApiGet(apiUrl, z, nil)
	if value, ok := r["value"]; ok {
		if count, valid := value.(int64); valid {
			return count
		}
		fmt.Printf("Unexpected value type in response: %T\n", value)
	} else {
		fmt.Println("Response does not contain 'value' field.")
	}
	return 0
}

// Returns an id:name map of objects of the given type.
func GetDirObjectIdMap(t string, z *Config) map[string]string {
	nameMap := make(map[string]string)
	// Fetch objects of the given type, using the cache for speed
	objects := GetMatchingObjects(t, "", false, z) // false = don't go to Azure
	for _, x := range objects {
		// Safely extract "id" and "displayName" with type assertions
		id, okID := x["id"].(string)
		displayName, okName := x["displayName"].(string)
		if okID && okName {
			nameMap[id] = displayName
		} else {
			// Log or handle entries with missing or invalid fields
			//fmt.Printf("Skipping object with invalid id or displayName: %+v\n", x) // DEBUG
		}
	}
	return nameMap
}

// Gets object of given type from Azure by id. Updates entry in local cache.
func GetObjectFromAzureById(t, id string, z *Config) AzureObject {
	baseUrl := ConstMgUrl + ApiEndpoint[t]
	apiUrl := baseUrl + "/" + id
	obj, _, _ := ApiGet(apiUrl, z, nil)
	if obj == nil || obj["id"] == nil {
		if t == "ap" || t == "sp" {
			// If 1st search doesn't find the object, then for Apps and SPS,
			// do a 2nd search based on their Client Id.
			apiUrl := baseUrl
			params := map[string]string{"$filter": "appId eq '" + id + "'"}
			r, _, _ := ApiGet(apiUrl, z, params)
			if r != nil {
				// Check if "value" key exists and is a list
				if value, ok := r["value"].([]interface{}); ok {
					count := len(value)
					switch {
					case count == 1:
						obj = value[0].(map[string]interface{}) // Assign single object
					case count > 1:
						msg := fmt.Sprintf("Warning! Found %d entries with this appId", count)
						fmt.Println(utl.Yel(msg))
					}
				}
			}
		}
	}

	if obj == nil || obj["id"] == nil {
		return nil // No valid object found after all attempts
	}

	x := AzureObject(obj) // Cast the result to AzureObject

	// Update the object in the local cache
	cache, err := GetCache(t, z)
	if err != nil {
		fmt.Printf("Warning: Failed to load cache for type '%s': %v\n", t, err)
		return x // Return the fetched object even if cache update fails
	}

	cache.Upsert(x.TrimForCache(t)) // Add or update the object in the cache
	if err := cache.Save(); err != nil {
		fmt.Printf("Warning: Failed to save updated cache for type '%s': %v\n", t, err)
	}

	return x // Return the found object or nil
}

// Fetches objects of the given type from Azure by displayName. It returns a list of
// matching objects, accounting for the possibility of multiple objects with the
// same displayName.
func GetObjectFromAzureByName(t, displayName string, z *Config) AzureObjectList {
	apiUrl := ConstMgUrl + ApiEndpoint[t] + "?$filter=displayName eq '" + displayName + "'"
	r, statusCode, err := ApiGet(apiUrl, z, nil)
	if err != nil {
		fmt.Printf("Error: Failed to fetch objects by name '%s' for type '%s': %v\n",
			displayName, t, err)
		return nil
	}

	// Check for a successful response
	if statusCode == 200 && r != nil && r["value"] != nil {
		result := AzureObjectList{} // Initialize the result list

		// Safely iterate over the returned objects
		if items, ok := r["value"].([]interface{}); ok {
			for _, item := range items {
				if mapObj, mapOk := item.(map[string]interface{}); mapOk {
					result.Add(AzureObject(mapObj))
				}
			}
		} else {
			fmt.Printf("Warning: Unexpected data format for 'value' in response for name '%s'.\n",
				displayName)
		}
		return result
	}

	// Log a warning if the request was unsuccessful
	fmt.Printf("Warning: Failed to fetch objects by name '%s' for type '%s'. Status code: %d\n",
		displayName, t, statusCode)
	return nil
}

// Retrieves existing object from Azure by its ID or displayName. This is
// typically used as preprocessing for operations like renaming, deleting,
// or updating a group.
func PreFetchAzureObject(t, identifier string, z *Config) (x AzureObject) {
	if utl.ValidUuid(identifier) {
		return GetObjectFromAzureById(t, identifier, z)
	}

	matchingObjects := GetObjectFromAzureByName(t, identifier, z)
	if len(matchingObjects) == 0 {
		return nil
	}

	if len(matchingObjects) > 1 {
		fmt.Printf("Found multiple '%s' objects with same name '%s'\n", t, utl.Red(identifier))
		for _, x := range matchingObjects {
			fmt.Printf("  %s  %s\n", x["id"], x["displayName"])
		}
		utl.Die("%s. Please try processing by id instead of name.\n", utl.Red("Aborting"))
	}

	return matchingObjects[0]
}

// Gets all objects of given type, matching on 'filter'. Returns the entire list if filter is empty "".
func GetMatchingObjects(t, filter string, force bool, z *Config) AzureObjectList {
	// Get current cache data, or initialize a new cache for this type
	cache, err := GetCache(t, z)
	if err != nil {
		utl.Die("Error: %s\n", err.Error())
	}

	// Return an empty list if cache is nil and internet is not available
	internetIsAvailable := utl.IsInternetAvailable()
	if cache == nil && !internetIsAvailable {
		return AzureObjectList{} // Return empty list
	}

	// Determine if cache is empty or outdated and needs to be refreshed from Azure
	cacheNeedsRefreshing := force || cache.Age() == 0 || cache.Age() > ConstMgCacheFileAgePeriod
	if internetIsAvailable && cacheNeedsRefreshing {
		SyncDirObjectsWithAzure(t, cache, z, true) // Call Azure to refresh cache
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.NewStringSet()         // Keep track of unique IDs to eliminate duplicates
	for _, obj := range cache.data {
		id := obj["id"].(string)
		if ids.Exists(id) {
			continue // Skip repeated entries
		}

		if obj.HasString(filter) {
			matchingList.Add(obj) // Add matching object to the list
			ids.Add(id)           // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all directory objects of given type from Azure and syncs them to local cache.
// Shows progress if verbose = true.
func SyncDirObjectsWithAzure(t string, cache *Cache, z *Config, verbose bool) {
	// Setup REST API URL for the specific type
	apiUrl := ConstMgUrl + ApiEndpoint[t] // e.g., 'https://graph.microsoft.com/v1.0/groups'

	// Setup select criteria for each object type: what fields will trigger delta updates upon changing
	switch t {
	case "u":
		apiUrl += "/delta?$select=displayName,userPrincipalName,onPremisesSamAccountName&$top=999"
	case "g":
		apiUrl += "/delta?$select=displayName,description,isAssignableToRole,createdDateTime&$top=999"
	case "ap":
		apiUrl += "?$select=displayName,appId,requiredResourceAccess,passwordCredentials&$top=999"
	case "sp":
		apiUrl += "?$select=displayName,appId,accountEnabled,appOwnerOrganizationId,passwordCredentials&$top=999"
	case "dr":
		//No additional adjustment required
	case "da":
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
		if deltaLink, ok := deltaLinkMap["@odata.deltaLink"].(string); ok {
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
	cache.Normalize(t, deltaSet)

	// Save the updated cache back to file
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated cache: %s\n", err.Error())
	}
}

// Retrieves Azure directory object deltas. Returns the set of new or updated items, and
// a deltaLink for running the next future Azure query. Implements the code logic pattern
// described at https://docs.microsoft.com/en-us/graph/delta-query-overview
func FetchDirObjectsDelta(apiUrl string, z *Config, verbose bool) (deltaSet AzureObjectList, deltaLinkMap AzureObject) {
	k := 1 // Track number of API calls
	r, _, _ := ApiGet(apiUrl, z, nil)
	for {
		// Infinite for-loop until deltaLink appears (meaning we're done getting current delta set)
		var thisBatch []interface{} = nil // Assume zero entries in this batch
		var objCount int = 0
		if r["value"] != nil {
			thisBatch = r["value"].([]interface{})
			objCount = len(thisBatch)

			// Convert thisBatch from []interface{} to []map[string]interface{} to AzureObject
			for _, item := range thisBatch {
				if obj, ok := item.(map[string]interface{}); ok {
					deltaSet.Add(AzureObject(obj)) // Add converted object to deltaSet
				} else {
					fmt.Printf("Warning: Skipping invalid object type: %v\n", item)
				}
			}
		}
		if verbose {
			// Progress count indicator. Using global var rUp to overwrite last line. Defer newline until done
			fmt.Printf("%sCall %05d : count %05d", rUp, k, objCount)
		}

		// Return immediately when deltaLink appears
		if r["@odata.deltaLink"] != nil {
			deltaLinkMap := map[string]interface{}{
				"@odata.deltaLink": r["@odata.deltaLink"].(string),
			}
			if verbose {
				fmt.Print(rUp) // Go up to overwrite progress line
			}
			return deltaSet, deltaLinkMap
		}
		// Check if nextLink is nil before attempting to use it
		if r["@odata.nextLink"] != nil {
			nextLink := r["@odata.nextLink"].(string) // Safe to assert as string now
			r, _, _ = ApiGet(nextLink, z, nil)        // Get next batch
			k++
		} else {
			break // If nextLink is nil, we can break out of the loop
		}
	}
	return deltaSet, deltaLinkMap
}

// Deletes directory object of given type in Azure, and updates local cache.
func DeleteDirObjectInAzure(t, id string, z *Config) error {
	apiUrl := ConstMgUrl + ApiEndpoint[t] + "/" + id
	r, statusCode, _ := ApiDelete(apiUrl, z, nil)
	if statusCode == 204 {
		// Also remove from local cache using Cache.Delete
		cache, err := GetCache(t, z)
		if err == nil {
			if err := cache.Delete(id); err != nil {
				return fmt.Errorf("failed to delete %s from local cache ", MazObjName[t])
			}
		} else {
			return fmt.Errorf("failed to load cache: %s", err)
		}
		return nil
	} else {
		if errDetails, ok := r["error"].(map[string]interface{}); ok {
			return fmt.Errorf("error: %s", errDetails["message"].(string))
		}
		return fmt.Errorf("failed to delete %s", MazObjName[t])
	}
}

// Deletes directory object of given type in Azure, with a confirmation prompt.
func DeleteDirObject(opts *Options, z *Config) error {
	force, _ := opts.GetBool("force")
	id, _ := opts.GetString("id") // Note that id may be a UUID or a displayName
	t, _ := opts.GetString("t")

	x := PreFetchAzureObject(t, id, z)
	if x == nil {
		return fmt.Errorf("no such %s", MazObjName[t])
	}

	// Confirmation prompt
	PrintObject(t, x, z)
	if !force {
		msg := utl.Yel("Delete " + MazObjName[t] + "? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			return fmt.Errorf("operation aborted by user")
		}
	}

	// Delete object in Azure
	id = x["id"].(string)
	if err := DeleteDirObjectInAzure(t, id, z); err != nil {
		return fmt.Errorf("%s", err.Error())
	}
	return nil
}

// Creates directory object of given type in Azure, and updates local cache.
func CreateDirObjectInAzure(t string, obj AzureObject, z *Config) (AzureObject, error) {
	// Creates object in Azure using obj as payload
	apiUrl := ConstMgUrl + ApiEndpoint[t]
	r, statusCode, _ := ApiPost(apiUrl, z, jsonT(obj), nil)
	if statusCode == 201 {
		azObj := AzureObject(r) // Newly created object

		// Add object to local cache
		cache, err := GetCache(t, z)
		if err != nil {
			fmt.Printf("Warning: Failed to load cache: %v\n", err)
			// TODO: Should we panic here instead of warn?
		}
		if cache != nil {
			cache.Upsert(azObj.TrimForCache(t))
			if err := cache.Save(); err != nil {
				fmt.Printf("Warning: Failed to save updated cache: %v\n", err)
				// TODO: Should we panic here instead of warn?
			}
		}
		return azObj, nil
	} else {
		if errDetails, ok := r["error"].(map[string]interface{}); ok {
			return nil, fmt.Errorf("error: %s", errDetails["message"].(string))
		}
		return nil, fmt.Errorf("error: failed to create %s", MazObjName[t])
	}
}

// Creates directory object of given type in Azure, with a confirmation prompt.
func CreateDirObject(force bool, obj AzureObject, t string, z *Config) (AzureObject, error) {
	// Present confirmation prompt if force isn't set
	fmt.Printf("%s\n", utl.Yel("Creating new "+MazObjName[t]+" with below attributes:"))
	utl.PrintYamlColor(obj)
	if !force {
		msg := utl.Yel("Create " + MazObjName[t] + "? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			return nil, fmt.Errorf("operation aborted by user")
		}
	}

	// Create the object in Azure
	var azObj AzureObject
	var err error
	if azObj, err = CreateDirObjectInAzure(t, obj, z); err != nil {
		return nil, fmt.Errorf("%s", err.Error())
	}

	return azObj, nil
}

// Updates directory object of given type in Azure, and updates local cache.
func UpdateDirObjectInAzure(t, id string, obj AzureObject, z *Config) error {
	apiUrl := ConstMgUrl + ApiEndpoint[t] + "/" + id
	r, statusCode, _ := ApiPatch(apiUrl, z, jsonT(obj), nil)
	if statusCode != 204 {
		if err, ok := r["error"].(map[string]interface{}); ok {
			return fmt.Errorf("error: %s", err["message"].(string))
		}
		return fmt.Errorf("error: failed to update %s %s in Azure", MazObjName[t], id)
	}

	// Retrieve recently updated object
	r, statusCode, err := ApiGet(apiUrl, z, nil)
	if r == nil || r["id"] == nil {
		return fmt.Errorf("http %d error: failed to retrieve newly created %s %s from Azure: %s",
			statusCode, MazObjName[t], id, err.Error())
	}

	// Also update the local cache
	azObj := AzureObject(r) // Cast into standard AzureObject type
	cache, err := GetCache(t, z)
	if err != nil {
		fmt.Printf("Warning: Failed to load cache: %v\n", err) // TODO: Panic instead of warn here?
	}
	if err := cache.Upsert(azObj.TrimForCache(t)); err != nil {
		fmt.Printf("Warning: Failed to upsert object in cache: %v\n", err) // TODO: Panic instead of warn here?
	}

	return nil
}

// Updates directory object of given type in Azure, with a confirmation prompt.
func UpdateDirObject(force bool, id string, obj AzureObject, t string, z *Config) {
	// Present confirmation prompt if force isn't set
	fmt.Printf("%s\n", utl.Yel("Update "+MazObjName[t]+" with below attributes:"))
	utl.PrintYamlColor(obj)
	if !force {
		msg := utl.Yel("Update " + MazObjName[t] + "? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Update the object in Azure
	if err := UpdateDirObjectInAzure(t, id, obj, z); err != nil {
		utl.Die("%s", err.Error())
	}
}

// Renames directory object of given type in Azure.
func RenameDirObject(opts *Options, z *Config) {
	force, _ := opts.GetBool("force")
	from, _ := opts.GetString("from") // Can be ID or displayName
	newName, _ := opts.GetString("newName")
	t, _ := opts.GetString("t")

	x := PreFetchAzureObject(t, from, z)
	if x == nil {
		utl.Die("No such %s\n", MazObjName[t])
	}

	id := x["id"].(string)

	// Confirmation prompt
	if !force {
		oldName := x["displayName"].(string)
		msg := utl.Yel("Rename "+MazObjName[t]+" "+id+"\n  from \"") + utl.Blu(oldName) +
			utl.Yel("\"\n    to \"") + utl.Blu(newName) + utl.Yel("\"\n? y/n ")
		if utl.PromptMsg(msg) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Update the object in Azure
	obj := AzureObject{"displayName": newName}
	// The obj payload only requires the displayName
	if err := UpdateDirObjectInAzure(t, id, obj, z); err != nil {
		utl.Die("%s", err.Error())
	}
}

// Adds a new secret to the given App or SP
func AddAppSpSecret(t, id, displayName, expiry string, z *Config) {
	if t != "ap" && t != "sp" {
		utl.Die("Error: Secrets can only be added to an App or SP object.\n")
	}
	x := GetObjectFromAzureById(t, id, z)
	if x == nil {
		utl.Die("No %s with that ID.\n", MazObjName[t])
	}

	// Check if a password with the same displayName already exists
	object_id := utl.Str(x["id"]) // NOTE: We call Azure with the OBJECT ID
	apiUrl := ConstMgUrl + ApiEndpoint[t] + "/" + object_id + "/passwordCredentials"
	r, statusCode, _ := ApiGet(apiUrl, z, nil)
	if statusCode == 200 {
		passwordCredentials := r["value"].([]interface{})
		for _, credential := range passwordCredentials {
			credentialMap := credential.(map[string]interface{})
			if credentialMap["displayName"].(string) == displayName {
				utl.Die("A password named %s already exists.\n", utl.Yel(displayName))
			}
		}
	}

	// Setup expiry for endDateType payload variable
	var endDateTime string
	if expiry != "" {
		if utl.ValidDate(expiry, "2006-01-02") {
			// If user-supplied expiry is a valid date, reformat and use for our purpose
			var err error
			endDateTime, err = utl.ConvertDateFormat(expiry, "2006-01-02", time.RFC3339Nano)
			if err != nil {
				utl.Die("Error converting %s Expiry to RFC3339Nano/ISO8601 format.\n", utl.Yel(expiry))
			}
		} else if days, err := utl.StringToInt64(expiry); err == nil {
			// If expiry not a valid date, see if it's a valid integer number
			expiryTime := utl.GetDateInDays(utl.Int64ToString(days)) // Set expiryTime to 'days' from now
			endDateTime = expiryTime.Format(time.RFC3339Nano)        // Convert to RFC3339Nano/ISO8601 format
		} else {
			utl.Die("Invalid expiry format. Please use YYYY-MM-DD or number of days.\n")
		}
	} else {
		// If expiry is blank, default to 365 days from now
		endDateTime = time.Now().AddDate(0, 0, 365).Format(time.RFC3339Nano)
	}

	// Call Azure to create the new secret
	payload := AzureObject{
		"passwordCredential": map[string]string{
			"displayName": displayName,
			"endDateTime": endDateTime,
		},
	}
	apiUrl = ConstMgUrl + ApiEndpoint[t] + "/" + object_id + "/addPassword"
	r, statusCode, _ = ApiPost(apiUrl, z, jsonT(payload), nil)
	if statusCode == 200 {
		if t == "ap" {
			fmt.Printf("%s: %s\n", utl.Blu("app_object_id"), utl.Gre(object_id))
		} else {
			fmt.Printf("%s: %s\n", utl.Blu("sp_object_id"), utl.Gre(object_id))
		}
		fmt.Printf("%s: %s\n", utl.Blu("new_secret_id"), utl.Gre(utl.Str(r["keyId"])))
		fmt.Printf("%s: %s\n", utl.Blu("new_secret_name"), utl.Gre(displayName))
		fmt.Printf("%s: %s\n", utl.Blu("new_secret_expiry"), utl.Gre(expiry))
		fmt.Printf("%s: %s\n", utl.Blu("new_secret_text"), utl.Gre(utl.Str(r["secretText"])))
	} else {
		e := r["error"].(map[string]interface{})
		utl.Die("%s\n", e["message"].(string))
	}
}

// Removes a secret from the given App or SP object
func RemoveAppSpSecret(t, id, keyId string, force bool, z *Config) {
	// TODO: Needs a prompt/force option
	if t != "ap" && t != "sp" {
		utl.Die("Error: Secrets can only be removed from an App or SP object.\n")
	}
	x := GetObjectFromAzureById(t, id, z)
	if x == nil {
		utl.Die("No %s with that ID.\n", MazObjName[t])
	}
	if !utl.ValidUuid(keyId) {
		utl.Die("Secret ID is not a valid UUID.\n")
	}

	// Display object secret details, and prompt for delete confirmation
	pwdCreds := x["passwordCredentials"].([]interface{})
	if len(pwdCreds) < 1 {
		utl.Die("App object has no secrets.\n")
	}
	var a AzureObject = nil // Target keyId, Secret ID to be deleted
	for _, i := range pwdCreds {
		targetKeyId := i.(map[string]interface{})
		if utl.Str(targetKeyId["keyId"]) == keyId {
			a = targetKeyId
			break
		}
	}
	if a == nil {
		utl.Die("App object does not have this Secret ID.\n")
	}
	cId := utl.Str(a["keyId"])
	cName := utl.Str(a["displayName"])
	cHint := utl.Str(a["hint"]) + "********"
	cStart, err := utl.ConvertDateFormat(utl.Str(a["startDateTime"]), time.RFC3339Nano, "2006-01-02")
	if err != nil {
		utl.Die("%s %s\n", utl.Trace(), err.Error())
	}
	cExpiry, err := utl.ConvertDateFormat(utl.Str(a["endDateTime"]), time.RFC3339Nano, "2006-01-02")
	if err != nil {
		utl.Die("%s %s\n", utl.Trace(), err.Error())
	}

	// Prompt
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(utl.Str(x["id"])))
	fmt.Printf("%s: %s\n", utl.Blu("appId"), utl.Gre(utl.Str(x["appId"])))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(utl.Str(x["displayName"])))
	fmt.Printf("%s:\n", utl.Yel("secret_to_be_deleted"))
	fmt.Printf("  %-36s  %-30s  %-16s  %-16s  %s\n", utl.Yel(cId), utl.Yel(cName),
		utl.Yel(cHint), utl.Yel(cStart), utl.Yel(cExpiry))
	if utl.PromptMsg(utl.Yel("DELETE above? y/n ")) == 'y' {
		payload := AzureObject{"keyId": keyId}
		object_id := utl.Str(x["id"]) // NOTE: We call Azure with the OBJECT ID
		apiUrl := ConstMgUrl + ApiEndpoint[t] + "/" + object_id + "/removePassword"
		r, statusCode, _ := ApiPost(apiUrl, z, jsonT(payload), nil)
		if statusCode == 204 {
			utl.Die("Successfully deleted secret.\n")
		} else {
			e := r["error"].(map[string]interface{})
			utl.Die("%s\n", e["message"].(string))
		}
	} else {
		utl.Die("Aborted.\n")
	}
}

// Find JSON object with given ID in slice
func FindObjectOld(objSet []interface{}, id string) map[string]interface{} {
	for _, obj := range objSet {
		if x, ok := obj.(map[string]interface{}); ok { // Inline type assertion and check
			if utl.Str(x["id"]) == id { // Compare directly
				return x
			}
		}
	}
	return nil
}

// Builds JSON mergeSet from deltaSet, and builds and returns the list of deleted IDs
func NormalizeCache(baseSet, deltaSet []interface{}) (list []interface{}) {
	// OLD: To gradually be replaced by NormalizeDirObjectCache()

	// 1. Process deltaSet to build mergeSet and track deleted IDs
	deletedIds := utl.NewStringSet()
	uniqueIds := utl.NewStringSet()
	var mergeSet []interface{} = nil
	for _, i := range deltaSet {
		x := i.(map[string]interface{})
		id := utl.Str(x["id"])
		if x["@removed"] == nil && x["members@delta"] == nil {
			// Only add to mergeSet if '@remove' and 'members@delta' are missing
			if !uniqueIds.Exists(id) {
				// Only add if it's unique
				mergeSet = append(mergeSet, x)
				uniqueIds.Add(id) // Track unique IDs
			}
		} else {
			deletedIds.Add(id)
		}
	}

	// 2. Remove recently deleted entries (deletedIs) from baseSet
	list = nil
	baseIds := utl.NewStringSet() // Track all the IDs in the base cache set
	for _, i := range baseSet {
		x := i.(map[string]interface{})
		id := utl.Str(x["id"])
		if deletedIds.Exists(id) {
			continue
		}
		list = append(list, x)
		baseIds.Add(id)
	}

	// 3. Merge new entries in deltaSet into baseSet
	var duplicates []interface{} = nil
	duplicateIds := utl.NewStringSet()
	for _, obj := range mergeSet {
		x := obj.(map[string]interface{})
		id := utl.Str(x["id"])
		if baseIds.Exists(id) {
			duplicates = append(duplicates, x)
			duplicateIds.Add(id)
			continue // Skip duplicates (these are updates)
		}
		list = append(list, x) // Merge all others (these are new entries)
	}

	// 4. Merge updated entries in deltaSet into baseSet
	list2 := list
	list = nil
	for _, obj := range list2 {
		x := obj.(map[string]interface{})
		id := utl.Str(x["id"])
		if !duplicateIds.Exists(id) {
			// If this object is not a duplicate, add it to our growing list
			list = append(list, x)
		} else {
			// Merge object updates, then add it to our growing list
			y := FindObjectOld(duplicates, id)
			x = utl.MergeJsonObjects(y, x)
			list = append(list, x)
		}
	}
	return list
}
