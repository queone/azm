package maz

import (
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/queone/utl"
)

// Prints Azure resource RBAC definition object in a YAML-like format
func PrintRbacDefinition(obj AzureObject, z *Config) {
	id := utl.Str(obj["name"])
	if id == "" {
		return
	}

	fmt.Printf("%s\n", utl.Gra("# Resource RBAC Definition"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	props, ok := obj["properties"].(map[string]interface{})
	if !ok {
		utl.Die("%s\n", utl.Red("  <Missing properties??>"))
	}
	fmt.Println(utl.Blu("properties") + ":")

	list := []string{"type", "roleName", "description"}
	for _, i := range list {
		fmt.Printf("  %s: %s\n", utl.Blu(i), utl.Gre(utl.Str(props[i])))
	}

	fmt.Printf("  %s: ", utl.Blu("assignableScopes"))
	if props["assignableScopes"] == nil {
		fmt.Printf("[]\n")
	} else {
		fmt.Printf("\n")
		scopes := props["assignableScopes"].([]interface{})
		if len(scopes) > 0 {
			subNameMap := GetAzureSubscriptionsIdMap(z) // Get all subscription id:name pairs
			for _, i := range scopes {
				if strings.HasPrefix(i.(string), "/subscriptions") {
					// Print subscription name as a comment at end of line
					subId := utl.LastElem(i.(string), "/")
					comment := "# " + subNameMap[subId]
					fmt.Printf("    - %s  %s\n", utl.Gre(utl.Str(i)), utl.Gra(comment))
				} else {
					fmt.Printf("    - %s\n", utl.Gre(utl.Str(i)))
				}
			}
		} else {
			fmt.Println(utl.Red("    <Not an arrays??>\n"))
		}
	}

	fmt.Printf("  %s:\n", utl.Blu("permissions"))
	if props["permissions"] == nil {
		fmt.Println(utl.Red("    < No permissions?? >\n"))
	} else {
		permsSet := props["permissions"].([]interface{})
		if len(permsSet) == 1 {
			perms := permsSet[0].(map[string]interface{}) // Select the 1 expected single permission set

			// Note that this one is different, as it starts the YAML array with the dash '-'
			fmt.Printf("    - %s:\n", utl.Blu("actions"))
			if perms["actions"] != nil {
				permsA := perms["actions"].([]interface{})
				if utl.GetType(permsA)[0] != '[' { // Open bracket character means it's an array list
					fmt.Println(utl.Red("        <Not an array??>\n"))
				} else {
					for _, i := range permsA {
						// Special function to lookout for leading '*' which must be single-quoted
						s := utl.StrSingleQuote(i)
						fmt.Printf("        - %s\n", utl.Gre(s))
					}
				}
			}

			fmt.Printf("      %s:\n", utl.Blu("notActions"))
			if perms["notActions"] != nil {
				permsNA := perms["notActions"].([]interface{})
				if utl.GetType(permsNA)[0] != '[' {
					fmt.Println(utl.Red("        <Not an array??>\n"))
				} else {
					for _, i := range permsNA {
						s := utl.StrSingleQuote(i)
						fmt.Printf("        - %s\n", utl.Gre(s))
					}
				}
			}

			fmt.Printf("      %s:\n", utl.Blu("dataActions"))
			if perms["dataActions"] != nil {
				permsDA := perms["dataActions"].([]interface{})
				if utl.GetType(permsDA)[0] != '[' {
					fmt.Println(utl.Red("        <Not an array??>\n"))
				} else {
					for _, i := range permsDA {
						s := utl.StrSingleQuote(i)
						fmt.Printf("        - %s\n", utl.Gre(s))
					}
				}
			}

			fmt.Printf("      %s:\n", utl.Blu("notDataActions"))
			if perms["notDataActions"] != nil {
				permsNDA := perms["notDataActions"].([]interface{})
				if utl.GetType(permsNDA)[0] != '[' {
					fmt.Println(utl.Red("        <Not an array??>\n"))
				} else {
					for _, i := range permsNDA {
						s := utl.StrSingleQuote(i)
						fmt.Printf("        - %s\n", utl.Gre(s))
					}
				}
			}

		} else {
			fmt.Println(utl.Red("    <More than one set??>\n"))
		}
	}
}

// Validates given object to ensure if conforms to the format of a resource role
// definition, and if valid, returns the roleName and the firstScope.
func ValidateRbacDefinition(obj AzureObject, z *Config) (string, string) {
	props, ok := obj["properties"].(map[string]interface{})
	if !ok {
		utl.Die("Expected 'properties' to be a map, but got %T\n", obj["properties"])
	}

	// Check if the object is a definition
	if _, exists := props["roleName"]; !exists {
		utl.Die("Error. Object is not a role definition. Missing %s in properties\n",
			utl.Red("roleName"))
	}

	// Validate DEFINITION
	requiredKeys := []string{"roleName", "description", "assignableScopes"}
	for _, key := range requiredKeys {
		if _, exists := props[key]; !exists {
			utl.Die("Missing required key: properties.%s\n", utl.Red(key))
		}
	}

	roleName := props["roleName"].(string)
	scopes, _ := props["assignableScopes"].([]interface{})
	if len(scopes) < 1 {
		utl.Die("Error. Object properties.%s has no entries\n", utl.Red("assignableScopes"))
	}

	firstScope := scopes[0].(string)
	if !strings.HasPrefix(firstScope, "/") {
		utl.Die("Error. Object properties.assignableScopes entry 0 does not start with '/'\n")
	}

	isMgmtGroupScope := strings.HasPrefix(firstScope, "/providers/Microsoft.Management/managementGroups")
	isTenantMismatch := filepath.Base(firstScope) != z.TenantId
	if isMgmtGroupScope && isTenantMismatch {
		utl.Die("Error. Scope %s does not match with target tenant ID %s\n",
			utl.Red(firstScope), utl.Red(z.TenantId))
	}
	return roleName, firstScope
}

// Creates or updates a role definition as defined by given object
func UpsertRbacDefinition(force bool, obj AzureObject, z *Config) {
	roleName, firstScope := ValidateRbacDefinition(obj, z)

	// Add the required 'type' to the object. Below assertion works because we have
	// already validated that 'properties' is indeed part of the object's structure.
	obj["properties"].(map[string]interface{})["type"] = "CustomRole"

	// Check if role definition already exists
	id, _, _ := GetAzureRbacDefinitionByNameAndScope(roleName, firstScope, z)
	promptType := "Create"
	deployType := "created"
	if utl.ValidUuid(id) { // A valid UUID means the role already exists
		promptType = "Update"
		deployType = "updated"
	} else {
		id = uuid.New().String() // Does not exist, so we will create anew
	}
	obj["name"] = id

	// Prompt to create/update
	PrintRbacDefinition(obj, z)
	if !force {
		msg := fmt.Sprintf("%s above role definition? y/n", promptType)
		if utl.PromptMsg(utl.Yel(msg)) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Call API
	payload := JsonObject(obj) // Obviously using x object as the payload
	params := map[string]string{"api-version": "2022-04-01"}
	apiUrl := ConstAzUrl + firstScope + "/providers/Microsoft.Authorization/roleDefinitions/" + id
	resp, statCode, _ := ApiPut(apiUrl, z, payload, params)
	if statCode == 201 {
		msg := fmt.Sprintf("Successfully %s role definition!", deployType)
		fmt.Printf("%s\n", utl.Gre(msg))

		// Upsert object in local cache also
		cache, err := GetCache(RbacDefinition, z)
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
		err = cache.Upsert(obj.TrimForCache(RbacDefinition))
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
	} else {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		utl.Die("%s\n", utl.Red(msg))
	}
}

// Deletes a role definition as defined by given object
func DeleteRbacDefinition(force bool, obj AzureObject, z *Config) {
	roleName, firstScope := ValidateRbacDefinition(obj, z)

	// Check if role definition already exists
	id, _, _ := GetAzureRbacDefinitionByNameAndScope(roleName, firstScope, z)
	if !utl.ValidUuid(id) {
		utl.Die("Role definition %s doesn't exist at scope %s\n",
			utl.Red(roleName), utl.Red(firstScope))
	} else {
		obj["name"] = id
	}

	// Prompt to delete
	PrintRbacDefinition(obj, z)
	if !force {
		msg := "Delete above role definition? y/n"
		if utl.PromptMsg(utl.Yel(msg)) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Call API
	params := map[string]string{"api-version": "2022-04-01"}
	apiUrl := ConstAzUrl + firstScope + "/providers/Microsoft.Authorization/roleDefinitions/" + id
	resp, statCode, _ := ApiDelete(apiUrl, z, params)
	if statCode == 200 {
		msg := "Successfully deleted role definition!"
		fmt.Printf("%s\n", utl.Gre(msg))

		// Also remove from local cache
		cache, err := GetCache(RbacDefinition, z)
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
		err = cache.Delete(id)
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
	} else if statCode == 204 {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		utl.Die("%s\n", utl.Yel(msg))
	} else {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		utl.Die("%s\n", utl.Red(msg))
	}
}

// DELETE
// Deletes a role definition by its ID
func DeleteRbacDefinitionById(id string, z *Config) error {
	params := map[string]string{"api-version": "2022-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.Authorization/roleDefinitions/" + id
	resp, statCode, _ := ApiDelete(apiUrl, z, params)
	if statCode == 200 {
		msg := "Successfully deleted role definition!"
		fmt.Printf("%s\n", utl.Gre(msg))

		// Also remove from local cache
		cache, err := GetCache(RbacDefinition, z)
		if err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
		err = cache.Delete(id)
		if err != nil {
			utl.Die("Warning: %v\n", err)
		}
	} else if statCode == 204 {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		fmt.Printf("%s\n", utl.Yel(msg))
	} else {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		fmt.Printf("%s\n", utl.Red(msg))
	}
	return nil
}

// Returns id:name map of all role definitions
func GetIdMapRoleDefs(z *Config) (nameMap map[string]string) {
	nameMap = make(map[string]string)
	roleDefs := GetMatchingRbacDefinitions("", false, z) // false = get from cache, not Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, x := range roleDefs {
		//x := i.(AzureObjectList)
		if x["name"] != nil {
			props := x["properties"].(map[string]interface{})
			if props["roleName"] != nil {
				nameMap[utl.Str(x["name"])] = utl.Str(props["roleName"])
			}
		}
	}
	return nameMap
}

// Counts all role definitions. If fromAzure is true, the definitions are sourced
// directly from Azure; otherwise, they are read from the local cache. It returns
// separate counts for custom and built-in roles.
func CountRbacDefinitions(fromAzure bool, z *Config) (customCount, builtinCount int64) {
	customCount, builtinCount = 0, 0
	definitions := GetMatchingRbacDefinitions("", fromAzure, z)
	for _, def := range definitions {
		// Ensure that the "properties" field exists and is a map.
		props, ok := def["properties"].(map[string]interface{})
		if !ok {
			fmt.Printf("WARNING: Unexpected or missing 'properties' field for RBAC definition: %v\n", def)
			continue
		}
		// Determine the role type.
		roleType := utl.Str(props["type"])
		if roleType == "CustomRole" {
			customCount++
		} else {
			builtinCount++
		}
	}
	return customCount, builtinCount
}

// Gets all role definitions matching on 'filter'. Returns entire list if filter is empty ""
func GetMatchingRbacDefinitions(filter string, force bool, z *Config) (list AzureObjectList) {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		x := GetAzureRbacDefinitionById(filter, z)
		if x != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{x}
		}
		// If not found, then filter will be used below in obj.HasString(filter)
	}

	// Get current cache, or initialize a new cache for this type
	cache, err := GetCache(RbacDefinition, z)
	if err != nil {
		utl.Die("Error: %v\n", err)
	}

	// Return an empty list if cache is nil and internet is not available
	internetIsAvailable := utl.IsInternetAvailable()
	if cache == nil && !internetIsAvailable {
		return AzureObjectList{} // Return empty list
	}

	// Determine if cache is empty or outdated and needs to be refreshed from Azure
	cacheNeedsRefreshing := force || cache.Age() == 0 || cache.Age() > ConstMgCacheFileAgePeriod
	if internetIsAvailable && cacheNeedsRefreshing {
		CacheAzureRbacDefinitions(cache, z, true)
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.NewStringSet()         // Keep track of unique IDs to eliminate duplicates

	for i := range cache.data {
		obj := &cache.data[i] // Access the element directly via pointer (memory walk)

		// Extract the ID: use the last part of the "id" path or fall back to the "name" field
		id := ""
		if idVal, ok := (*obj)["id"].(string); ok && idVal != "" {
			id = path.Base(idVal) // Extract the last part of the path (UUID)
		} else if nameVal, ok := (*obj)["name"].(string); ok && nameVal != "" {
			id = nameVal // Fall back to the "name" field if "id" is empty
		}

		// Skip if the ID is empty or already seen
		if id == "" || ids.Exists(id) {
			continue
		}

		// Check if the object matches the filter
		if obj.HasString(filter) {
			matchingList.Add(*obj) // Add matching object to the list
			ids.Add(id)            // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all Azure resource RBAC definition objects in current tenant and saves them
// to local cache. Note that we are updating the cache via its pointer, so no return values.
func CacheAzureRbacDefinitions(cache *Cache, z *Config, verbose bool) {
	// Keep track of unique IDs to eliminate duplicate objects
	list := NewList()
	ids := utl.NewStringSet()
	k := 1 // Track number of API calls for verbose output

	// Set up these maps for more informative verbose output
	var mgmtGroupIdMap, subscriptionIdMap map[string]string
	if verbose {
		mgmtGroupIdMap = GetAzureMgmtGroupsIdMap(z)
		subscriptionIdMap = GetAzureSubscriptionsIdMap(z)
	}

	// Search in all resource RBAC scopes
	scopes := GetAzureRbacScopes(z)
	params := map[string]string{"api-version": "2022-04-01"}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(apiUrl, z, params)
		if r != nil && r["value"] != nil {
			count := 0
			rawDefinitions, ok := r["value"].([]interface{})
			if !ok {
				utl.Die("unexpected type for Azure RBAC definitions")
			}

			// Collect all unique definitions under this scope
			for _, raw := range rawDefinitions {
				azObj, ok := raw.(map[string]interface{})
				if !ok {
					fmt.Printf("WARNING: Unexpected type for RBAC definition object: %v\n", raw)
					continue
				}
				// Root out potential duplicates
				id := utl.Str(azObj["name"])
				if ids.Exists(id) {
					continue
					// Skip this repeated one. This can happen because of the way Azure RBAC
					// hierarchy inherantance works, and the same role is seen from multiple places.
				}
				ids.Add(id) // Mark this id as seen
				list = append(list, azObj)
				count++

				// This is unique one, so add it to the cache
				trimmedObj := AzureObject(azObj).TrimForCache("d")
				if err := cache.Upsert(trimmedObj); err != nil {
					fmt.Printf("WARNING: Failed to upsert cache for RBAC definition object with ID '%s': %v\n",
						azObj["id"], err)
				}
			}
			if verbose && count > 0 {
				scopeName := scope
				scopeType := "Subscription"
				if strings.HasPrefix(scope, "/providers") {
					scopeName = mgmtGroupIdMap[scope]
					scopeType = "Management Group"
				} else if strings.HasPrefix(scope, "/subscriptions") {
					scopeName = subscriptionIdMap[utl.LastElem(scope, "/")]
				}
				fmt.Printf("%sCall %05d: %05d definitions under %s %s", rUp, k, count, scopeType, scopeName)
			}
		}
		k++
	}
	if verbose {
		fmt.Print(rUp) // Go up to overwrite progress line
	}
	// Save updated cache
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated RBAC definitions cache: %s\n", err.Error())
	}
}

// Gets an RBAC definition object if it exists exactly with given roleName at given scope.
func GetRbacDefinitionByObject(obj AzureObject, z *Config) (AzureObject, error) {
	// Validate input object
	if obj == nil {
		return nil, errors.New("input object is nil")
	}

	// Extract properties
	props, ok := obj["properties"].(map[string]interface{})
	if !ok || props == nil {
		return nil, errors.New("input object is missing or has invalid properties")
	}

	// Extract and validate roleName
	roleName, ok := props["roleName"].(string)
	if !ok || roleName == "" {
		return nil, errors.New("input object is missing or has an invalid roleName")
	}

	// Extract and validate assignableScopes
	scopes, ok := props["assignableScopes"].([]interface{})
	if !ok || len(scopes) == 0 {
		return nil, errors.New("input object is missing or has invalid assignableScopes")
	}

	// Search for the role definition under each scope
	for _, item := range scopes {
		scope, ok := item.(string)
		if !ok || scope == "" {
			continue // Skip invalid scopes
		}

		// Normalize scope
		if scope == "/" {
			scope = ""
		}

		// Build API URL and parameters and make API call
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		params := map[string]string{
			"api-version": "2022-04-01",
			"$filter":     "roleName eq '" + roleName + "'",
		}
		resp, _, err := ApiGet(apiUrl, z, params)
		if err != nil {
			fmt.Printf("DEBUG: API call failed for scope %s: %v", utl.Yel(scope), err)
			continue // TODO: Fix this so it is not too noisy
		}

		// Process API response
		if resp != nil && resp["value"] != nil {
			results, ok := resp["value"].([]interface{})
			if !ok || len(results) != 1 {
				continue // Skip if no unique result is found
			}

			// Return the matching role definition
			if roleDef, ok := results[0].(AzureObject); ok {
				return roleDef, nil
			}
		}
	}

	return nil, errors.New("no matching role definition found")
}

// Retrieves role definition with given name at given scope, and
// returns the ID, the object, and any error.
func GetAzureRbacDefinitionByNameAndScope(roleName, scope string, z *Config) (string, AzureObject, error) {
	params := map[string]string{
		"api-version": "2022-04-01",
		"$filter":     "roleName eq '" + roleName + "'",
	}
	apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
	resp, statCode, _ := ApiGet(apiUrl, z, params)
	if statCode == 200 {
		if value, exists := resp["value"].([]interface{}); exists {
			if len(value) == 1 {
				roleDef := value[0].(map[string]interface{})
				azObj := AzureObject(roleDef)
				id := utl.Str(azObj["name"])
				return id, azObj, nil // Found it
			}
		}
	} else {
		return "", nil, fmt.Errorf("http %d: %s", statCode, ApiErrorMsg(resp))
	}
	return "", nil, fmt.Errorf("role '%s' not found at scope '%s'", roleName, scope)
}

// Gets an Azure resource RBAC definition by roleName
func GetAzureRbacDefinitionByName(roleName string, z *Config) AzureObject {
	scopes := GetAzureRbacScopes(z)
	params := map[string]string{
		"api-version": "2022-04-01",
		"$filter":     "roleName eq '" + roleName + "'",
	}
	list := NewList()         // Start with a new empty list
	ids := utl.NewStringSet() // Keep track of unique IDs to eliminate duplicates
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(apiUrl, z, params)
		if r != nil && r["value"] != nil {
			if rawList, ok := r["value"].([]interface{}); ok {
				// Because the response is normally a JSON object with a "value" field that is a slice
				for _, itemRaw := range rawList {
					// We assume that each item is a map[string]interface{}
					if item, ok := itemRaw.(map[string]interface{}); ok {
						azObj := AzureObject(item) // We assume that each item is an AzureObject
						id := utl.Str(azObj["name"])
						if ids.Exists(id) {
							continue
							// Skip this repeated one. This can happen because of the way Azure RBAC
							// hierarchy inherantance works, and the same role is seen from multiple places.
						}
						ids.Add(id) // Mark this id as seen
						list = append(list, azObj)
					}
				}
				if len(list) == 1 {
					utl.PrintJsonColor(list[0])
					return list[0] // Return the on and only entry
				} else if len(list) > 1 {
					// Hopefully this is never surfaces?
					utl.PrintJsonColor(list)
					utl.Die("%s\n", utl.Red("Found multiple RBAC definitions with same roleName!"))
				}
			}
		}
	}
	return nil
}

// Gets a role definition by its Id. Unfortunately we have to iterate through the entire
// tenant scope hierarchy, which can be slow.
func GetAzureRbacDefinitionById(id string, z *Config) AzureObject {
	scopes := GetAzureRbacScopes(z)
	params := map[string]string{"api-version": "2022-04-01"}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions/" + id
		resp, _, _ := ApiGet(apiUrl, z, params)
		if resp != nil && resp["id"] != nil {
			azObj := AzureObject(resp)
			azObj["maz_from_azure"] = true
			return azObj // Return as soon as we find a match
		}
	}
	return nil
}
