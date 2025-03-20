package maz

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/queone/utl"
)

// Prints resource role definition object in a YAML-like format
func PrintRbacDefinition(obj AzureObject, z *Config) {
	id := utl.Str(obj["name"])
	if id == "" {
		return
	}

	fmt.Printf("%s\n", utl.Gra("# Resource role definition"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	props := utl.Map(obj["properties"])
	if props == nil {
		utl.Die("%s\n", utl.Red("  <Missing properties??>"))
	}
	fmt.Println(utl.Blu("properties") + ":")

	for _, item := range []string{"type", "roleName", "description"} {
		fmt.Printf("  %s: %s\n", utl.Blu(item), utl.Gre(utl.Str(props[item])))
	}

	// Assignable scopes
	fmt.Printf("  %s: ", utl.Blu("assignableScopes"))
	assignableScopes := utl.Slice(props["assignableScopes"])
	if assignableScopes == nil {
		fmt.Println("[]")
	} else {
		fmt.Println()
		if len(assignableScopes) < 1 {
			fmt.Println(utl.Red("    <Error: Role 'assignableScopes' slice has no entries?>\n"))
		} else {
			subNameMap := GetIdMapSubscriptions(z) // Get all subscription id:name pairs
			for _, item := range assignableScopes {
				if scope := utl.Str(item); scope != "" {
					if strings.HasPrefix(scope, "/subscriptions") {
						// Print subscription name as a comment at the end of line
						subId := path.Base(scope)
						comment := "# " + subNameMap[subId]
						fmt.Printf("    - %s  %s\n", utl.Gre(scope), utl.Gra(comment))
					} else {
						fmt.Printf("    - %s\n", utl.Gre(scope))
					}
				}
			}
		}
	}

	// Print role permissions
	permissionsSlice := utl.Slice(props["permissions"])
	if permissionsSlice == nil {
		fmt.Printf("%s\n", utl.Red("    <Error: Role 'permissions' is not a slice.>"))
	} else {
		fmt.Printf("  %s:\n", utl.Blu("permissions"))

		permissionsCount := len(permissionsSlice)
		if permissionsCount == 0 {
			fmt.Printf("%s\n", utl.Red("    <Error. Role 'permissions' has no entries.>"))
		} else if permissionsCount > 1 {
			msg := fmt.Sprintf("    <Error. Role 'permissions' slice has more than one entry (%d)?.>",
				permissionsCount)
			fmt.Printf("%s\n", utl.Red(msg))
		} else {
			// Select and focus on the one expected single permission set
			perms := utl.Map(permissionsSlice[0])
			if perms == nil {
				utl.Die("%s\n", utl.Red("    <Error. The expected single permission set is nil.>"))
			} else {
				// Print the 4 sets of permissions type
				fmt.Printf("    - %s: ", utl.Blu("actions"))
				// NOTE, that the first one is different, it starts with the YAML array dash '-'
				if actions := utl.Slice(perms["actions"]); actions == nil {
					fmt.Println("[]") // Note this newline, to finish the last Printf()
				} else {
					fmt.Println() // Note this newline, to finish the last Printf()
					for _, item := range actions {
						// StrSingleQuote() wraps any potential leading '*' in single-quotes
						fmt.Printf("        - %s\n", utl.Gre(utl.StrSingleQuote(item)))
					}
				}
				fmt.Printf("      %s: ", utl.Blu("notActions"))
				if notActions := utl.Slice(perms["notActions"]); notActions == nil {
					fmt.Println("[]")
				} else {
					fmt.Println()
					for _, item := range notActions {
						fmt.Printf("        - %s\n", utl.Gre(utl.StrSingleQuote(item)))
					}
				}
				fmt.Printf("      %s: ", utl.Blu("dataActions"))
				if dataActions := utl.Slice(perms["dataActions"]); dataActions == nil {
					fmt.Println("[]")
				} else {
					fmt.Println()
					for _, item := range dataActions {
						fmt.Printf("        - %s\n", utl.Gre(utl.StrSingleQuote(item)))
					}
				}
				fmt.Printf("      %s: ", utl.Blu("notDataActions"))
				if notDataActions := utl.Slice(perms["notDataActions"]); notDataActions == nil {
					fmt.Println("[]")
				} else {
					fmt.Println()
					for _, item := range notDataActions {
						fmt.Printf("        - %s\n", utl.Gre(utl.StrSingleQuote(item)))
					}
				}
			}
		}
	}
}

// Validates given object to ensure if conforms to the format of a Azure resource
// RBAC role definition, and if valid, returns the roleName and the firstScope.
func ValidateRbacDefinition(obj AzureObject, z *Config) (string, string) {
	props := utl.Map(obj["properties"])
	if props == nil {
		utl.Die("Error. Object 'properties' is not a map, but a %T\n", obj["properties"])
	}

	// Check if the object is a definition
	roleName := utl.Str(props["roleName"])
	if roleName == "" {
		utl.Die("Error. Object is not a role definition. Missing %s in properties\n",
			utl.Red("roleName"))
	}

	// Validate DEFINITION
	for _, key := range []string{"description", "assignableScopes"} {
		if _, exists := props[key]; !exists {
			utl.Die("Missing required key: properties.%s\n", utl.Red(key))
		}
	}

	scopes := utl.Slice(props["assignableScopes"])
	if scopes == nil {
		utl.Die("Error. Object properties.%s is not a slice\n", utl.Red("assignableScopes"))
	}

	if len(scopes) < 1 {
		utl.Die("Error. Object properties.%s has no entries\n", utl.Red("assignableScopes"))
	}

	firstScope := utl.Str(scopes[0])
	if !strings.HasPrefix(firstScope, "/") {
		utl.Die("Error. Object properties.%s entry 0 does not start with '/'\n",
			utl.Red("assignableScopes"))
	}

	isMgmtGroupScope := strings.HasPrefix(firstScope, "/providers/Microsoft.Management/managementGroups")
	isTenantMismatch := filepath.Base(firstScope) != z.TenantId
	if isMgmtGroupScope && isTenantMismatch {
		utl.Die("Error. Object assignableScopes entry %s does not match with target tenant ID %s\n",
			utl.Red(firstScope), utl.Red(z.TenantId))
	}
	return roleName, firstScope
}

// Creates or updates an Azure resource RBAC role definition as defined by given object
func UpsertRbacDefinition(force bool, obj AzureObject, z *Config) {
	roleName, firstScope := ValidateRbacDefinition(obj, z)

	// Add the required 'type' to the object. Below assertion works because we have
	// already validated that 'properties' is indeed part of the object's structure.
	obj["properties"].(map[string]interface{})["type"] = "CustomRole"

	// Check if role definition already exists
	id, _, _ := GetAzureRbacDefinitionByNameAndScope(roleName, firstScope, z)
	promptType := "CREATE"
	deployType := "CREATED"
	if utl.ValidUuid(id) { // A valid UUID means the role already exists
		promptType = "UPDATE"
		deployType = "UPDATED"
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
	payload := obj // Obviously using x object as the payload
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
		utl.Die("Role definition %s doesn't exist\n", utl.Red(roleName))
	} else {
		obj["name"] = id
	}

	// Display the role definition and prompt for delete confirmation
	PrintRbacDefinition(obj, z)
	if !force {
		msg := "Delete above role definition? y/n"
		if utl.PromptMsg(utl.Yel(msg)) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Delete the object
	params := map[string]string{"api-version": "2022-04-01"}
	apiUrl := ConstAzUrl + firstScope + "/providers/Microsoft.Authorization/roleDefinitions/" + id
	resp, statCode, _ := ApiDelete(apiUrl, z, params)
	if statCode == 200 {
		msg := "Successfully DELETED role definition!"
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

// Returns id:name map of all Azure RBAC role definitions
func GetIdMapRoleDefs(z *Config) (nameMap map[string]string) {
	nameMap = make(map[string]string)
	definitions := GetMatchingRbacDefinitions("", false, z) // false = get from cache, not Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy

	// Memory-walk the slice to gather these values more efficiently
	for i := range definitions {
		rolePtr := &definitions[i]  // Use a pointer to avoid copying the element
		role := *rolePtr            // Dereference the pointer for easier access
		id := utl.Str(role["name"]) // Accessing the field directly
		if id == "" {
			continue // Skip if "name" is missing or not a string
		}
		props := utl.Map(role["properties"])
		if props == nil {
			continue // Skip if "properties" is missing or not a map
		}
		name := utl.Str(props["roleName"])
		if name == "" {
			continue // Skip if "roleName" is missing or not a string
		}
		nameMap[id] = name
	}

	return nameMap
}

// Counts all role definitions. If fromAzure is true, the definitions are sourced
// directly from Azure; otherwise, they are read from the local cache. It returns
// separate counts for custom and built-in roles.
func CountRbacDefinitions(fromAzure bool, z *Config) (customCount, builtinCount int64) {
	customCount, builtinCount = 0, 0
	definitions := GetMatchingRbacDefinitions("", fromAzure, z)
	for _, role := range definitions {
		if props := utl.Map(role["properties"]); props != nil {
			if roleType := utl.Str(props["type"]); roleType != "" {
				if roleType == "CustomRole" {
					customCount++
				} else {
					builtinCount++
				}
			}
		}
	}
	return customCount, builtinCount
}

// Gets all role definitions matching on 'filter'. Returns entire list if filter is empty ""
func GetMatchingRbacDefinitions(filter string, force bool, z *Config) (list AzureObjectList) {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		singleRole := GetAzureRbacDefinitionById(filter, z)
		if singleRole != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{singleRole}
		}
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
		CacheAzureRbacDefinitions(cache, true, z) // true = be verbose
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.StringSet{}            // Keep track of unique IDs to eliminate duplicates
	for i := range cache.data {
		rawPtr := &cache.data[i]   // Access the element directly via pointer (memory walk)
		rawObj := *rawPtr          // Dereference the pointer manually
		roleMap := utl.Map(rawObj) // Try asserting as a map type
		if roleMap == nil {
			continue // Skip this entry if not a map
		}
		role := AzureObject(roleMap) // Cast as our standard AzureObject type

		// Extract the ID: use the last part of the "id" path or fall back to the "name" field
		id := utl.Str(role["id"])
		name := utl.Str(role["name"])
		if id != "" {
			id = path.Base(id) // Extract the last part of the path (UUID)
		} else if name != "" {
			id = name // Fall back to the "name" field if "id" is empty
		}

		// Skip if the ID is empty or already seen
		if id == "" || ids.Exists(id) {
			continue
		}

		// Check if the object contains the filter string
		if role.HasString(filter) {
			matchingList = append(matchingList, role) // Add matching object to the list
			ids.Add(id)                               // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all Azure resource RBAC definition objects in current tenant and saves them
// to local cache. Note that we are updating the cache via its pointer, so no return values.
func CacheAzureRbacDefinitions(cache *Cache, verbose bool, z *Config) {
	list := AzureObjectList{} // List of role definitions to cache
	ids := utl.StringSet{}    // Keep track of unique IDs to eliminate duplicate objects
	callCount := 1            // Track number of API calls for verbose output

	// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions-list

	// Set up these maps for more informative verbose output
	var mgmtGroupIdMap, subscriptionIdMap map[string]string
	if verbose {
		mgmtGroupIdMap = GetIdMapMgmtGroups(z)
		subscriptionIdMap = GetIdMapSubscriptions(z)
	}

	// Search in each resource RBAC scope
	scopes := GetAzureRbacScopes(z)

	// Collate every unique role definition in each scope
	params := map[string]string{"api-version": "2022-04-01"}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		resp, statCode, _ := ApiGet(apiUrl, z, params)
		if statCode != 200 {
			// For now, I don't think we care about any errors
			continue // If any issues retrieving items for this scope, go to next one
		}
		definitions := utl.Slice(resp["value"]) // Try asserting value as an object of slice type
		if definitions == nil {
			continue // If its's not a slice with values, process next scope
		}

		count := 0
		for i := range definitions {
			rawPtr := &definitions[i] // Access the element directly via pointer (memory walk)
			rawObj := *rawPtr         // Dereference the pointer manually
			role := utl.Map(rawObj)   // Try asserting as a map type
			if role == nil {
				continue // Skip this entry if not a map
			}
			// Root out potential duplicates
			id := utl.Str(role["name"])
			if ids.Exists(id) {
				continue // Skip this entry if it's a repeat
				// Skip this repeated one. This can happen because of the way Azure RBAC
				// hierarchy inheritance works, and the same role is seen from multiple places.
			}
			list = append(list, role) // Add object to the list
			ids.Add(id)               // Mark this id as seen
			count++
		}

		if verbose && count > 0 {
			scopeName := scope
			scopeType := "Subscription"
			if strings.HasPrefix(scope, "/providers") {
				scopeName = mgmtGroupIdMap[scope]
				scopeType = "Management Group"
			} else if strings.HasPrefix(scope, "/subscriptions") {
				scopeName = subscriptionIdMap[path.Base(scope)]
			}
			fmt.Printf("%sCall %05d: %05d definitions under %s %s", rUp, callCount, count, scopeType, scopeName)
		}
		callCount++
	}
	if verbose {
		fmt.Print(rUp) // Go up to overwrite progress line
	}

	// Trim and prepare all objects for caching
	for i := range list {
		// Directly modify the object in the original list
		list[i] = list[i].TrimForCache(RbacDefinition)
	}

	// Update the cache with the entire list of definitions
	cache.data = list

	// Save the cache
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated resource role definitions cache: %v\n", err.Error())
	}
}

// Retrieves role definition with given name at given scope, and returns
// the ID, the object, and any error.
func GetAzureRbacDefinitionByNameAndScope(roleName, scope string, z *Config) (string, AzureObject, error) {
	params := map[string]string{
		"api-version": "2022-04-01",
		"$filter":     "roleName eq '" + roleName + "'",
	}
	apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
	resp, statCode, _ := ApiGet(apiUrl, z, params)
	if statCode != 200 {
		return "", nil, fmt.Errorf("http %d: %s", statCode, ApiErrorMsg(resp))
	} else {
		sliceArray := utl.Slice(resp["value"]) // The response is a 'value' slice array
		if sliceArray == nil {
			return "", nil, fmt.Errorf("error asserting response value to slice type")
		} else {
			if len(sliceArray) == 1 {
				obj := utl.Map(sliceArray[0]) // Assert single object entry as a map
				if obj == nil {
					return "", nil, fmt.Errorf("error asserting array entry 0 to map type")
				} else {
					role := AzureObject(obj)
					id := utl.Str(role["name"])
					return id, role, nil
				}
			}
		}
	}
	return "", nil, fmt.Errorf("role '%s' not found at scope '%s'", roleName, scope)
}

// Retrieves all role definitions with given roleName from the Azure resource RBAC hierarchy.
func GetAzureRbacDefinitionsByName(roleName string, z *Config) AzureObjectList {
	scopes := GetAzureRbacScopes(z)    // Get all the scopes in the tenant hierarchy
	matchingRoles := AzureObjectList{} // Initialize an empty AzureObjectList
	ids := utl.StringSet{}             // Keep track of unique IDs to eliminate duplicates

	// Search each of the tenant scopes
	params := map[string]string{
		"api-version": "2022-04-01",
		"$filter":     "roleName eq '" + roleName + "'",
	}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		resp, _, _ := ApiGet(apiUrl, z, params)
		sliceArray := utl.Slice(resp["value"]) // The response is a 'value' slice array
		if resp == nil || sliceArray == nil {
			continue // Skip this scope if a role with that roleName was not found
		}
		// NOTE: It is possible for a role with the same roleName to exist in multiple scopes
		// within the Azure ARM API. That is the reason why a list of matchingRoles is required.
		// https://learn.microsoft.com/en-us/azure/role-based-access-control/custom-roles
		// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions
		for _, obj := range sliceArray {
			if role := utl.Map(obj); role != nil {
				id := utl.Str(role["name"])
				if ids.Exists(id) {
					continue
					// Skip this repeated one. This can happen because of the way Azure RBAC
					// hierarchy inherantance works, and the same role is seen from multiple places.
				}
				ids.Add(id)                                              // Mark this id as seen
				matchingRoles = append(matchingRoles, AzureObject(role)) // Append unique role
			}
		}
	}
	return matchingRoles
}

// Retrieves a role definition by its unique ID from the Azure resource RBAC hierarchy.
func GetAzureRbacDefinitionById(id string, z *Config) AzureObject {
	// Get all the scopes in the tenant hierarchy
	scopes := GetAzureRbacScopes(z)

	// NOTE: Microsoft documentation explicitly states that a role definition UUID
	// cannot be repeated across different scopes in the hierarchy. This is why we
	// return immediately upon a successful match.
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/custom-roles
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions

	// Search each of the tenant scopes
	params := map[string]string{"api-version": "2022-04-01"}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions/" + id
		resp, statCode, _ := ApiGet(apiUrl, z, params)
		if statCode == 200 {
			roleObj := utl.Map(resp) // Try asserting the response as a single object of map type
			if roleObj == nil {
				continue // Invalid response/object type - proceed to next scope
			}
			roleObj["maz_from_azure"] = true
			return AzureObject(roleObj) // Return immediately when a match is found
		}
	}
	return nil
}
