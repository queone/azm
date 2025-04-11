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
func PrintResRoleDefinition(obj AzureObject, z *Config) {
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
			subIdMap := GetIdNameMap(Subscription, z) // Get all subscription id:name pairs
			for _, item := range assignableScopes {
				if scope := utl.Str(item); scope != "" {
					if strings.HasPrefix(scope, "/subscriptions") {
						// Print subscription name as a comment at the end of line
						comment := "# " + subIdMap[path.Base(scope)]
						fmt.Printf("    - %s  %s\n", utl.Gre(scope), utl.Gra(comment))
					} else {
						fmt.Printf("    - %s\n", utl.Gre(scope))
					}
				}
			}
		}
	}

	// Print role permissions
	permissions := utl.Slice(props["permissions"])
	permCount := len(permissions)
	if permCount == 0 {
		fmt.Printf("%s\n", utl.Red("    <Error. Role 'permissions' has no entries.>"))
	} else if permCount > 1 {
		msg := fmt.Sprintf("    <Error. Role 'permissions' has %d entries. There can only be one.>",
			permCount)
		fmt.Printf("%s\n", utl.Red(msg))
	} else {
		fmt.Printf("  %s:\n", utl.Blu("permissions"))
		// Select and focus on the one expected single permission set
		perms := utl.Map(permissions[0])
		if perms == nil {
			utl.Die("%s\n", utl.Red("    <Error. The expected single permission set is empty.>"))
		}

		// Print the 4 sets of permissions type
		fmt.Printf("    - %s: ", utl.Blu("actions"))
		// NOTE, that 'actions' is printed differently, it starts with the YAML array dash '-'
		actions := utl.Slice(perms["actions"])
		if len(actions) > 0 {
			fmt.Println() // Note this newline ends the header Printf() above
			for _, value := range actions {
				// Function StrSingleQuote() wraps any potential leading '*' in single-quotes
				permission := utl.StrSingleQuote(value)
				fmt.Printf("        - %s\n", utl.Gre(permission))
			}
		} else {
			fmt.Println("[]") // Note this newline ends the header Printf() above
		}

		fmt.Printf("      %s: ", utl.Blu("notActions"))
		notActions := utl.Slice(perms["notActions"])
		if len(notActions) > 0 {
			fmt.Println()
			for _, value := range notActions {
				permission := utl.StrSingleQuote(value)
				fmt.Printf("        - %s\n", utl.Gre(permission))
			}
		} else {
			fmt.Println("[]")
		}

		fmt.Printf("      %s: ", utl.Blu("dataActions"))
		dataActions := utl.Slice(perms["dataActions"])
		if len(dataActions) > 0 {
			fmt.Println()
			for _, value := range dataActions {
				permission := utl.StrSingleQuote(value)
				fmt.Printf("        - %s\n", utl.Gre(permission))
			}
		} else {
			fmt.Println("[]")
		}

		fmt.Printf("      %s: ", utl.Blu("notDataActions"))
		notDataActions := utl.Slice(perms["notDataActions"])
		if len(notDataActions) > 0 {
			fmt.Println()
			for _, value := range notDataActions {
				permission := utl.StrSingleQuote(value)
				fmt.Printf("        - %s\n", utl.Gre(permission))
			}
		} else {
			fmt.Println("[]")
		}
	}
}

// Helper function to check if the object is a resource role definition
func IsResRoleDefinition(obj AzureObject) bool {
	// Check if 'properties' exists and is a map
	props := utl.Map(obj["properties"])
	if props == nil {
		return false
	}

	// Check if 'roleName' exists and is non-empty
	if roleName := utl.Str(props["roleName"]); roleName == "" {
		return false
	}

	// Check if 'assignableScopes' exists, is a slice, and has at least one entry
	assignableScopes := utl.Slice(props["assignableScopes"])
	if len(assignableScopes) == 0 {
		return false
	}

	// Check if 'permissions' exists, is a slice, and has at least one entry
	permissions := utl.Slice(props["permissions"])
	if len(permissions) == 0 {
		return false
	}

	// Check that each entry in 'permissions' is a valid map
	for _, perm := range permissions {
		if utl.Map(perm) == nil {
			return false
		}
	}

	// If all checks pass, it's a valid resource role definition
	return true
}

// Validates given object to ensure if conforms to the format of an Azure resource
// role definition. If it is valid, return the roleName and the firstScope.
func ValidateResRoleDefinitionObject(obj AzureObject, z *Config) (string, string) {
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

// Creates or updates an Azure resource role definition as defined by given object
func UpsertAzureResRoleDefinition(force bool, obj AzureObject, z *Config) {
	roleName, firstScope := ValidateResRoleDefinitionObject(obj, z)

	// Add the required 'type' to the object. Below assertion works because we have
	// already validated that 'properties' is indeed part of the object's structure.
	obj["properties"].(map[string]interface{})["type"] = "CustomRole"

	// Check if role definition already exists
	id, azureObj := GetAzureResRoleDefinitionByScopeAndName(firstScope, roleName, z)
	promptType := "CREATE"
	deployType := "CREATED"
	if utl.ValidUuid(id) { // A valid UUID means the role already exists
		promptType = "UPDATE"
		deployType = "UPDATED"
	} else {
		id = uuid.New().String() // Does not exist, so we will create anew
	}
	obj["name"] = id // Update inputted object with corresponding ID

	// Prompt to create/update
	if promptType == "UPDATE" {
		DiffRoleDefinitionSpecfileVsAzure(obj, azureObj)
	} else {
		PrintResRoleDefinition(obj, z)
	}

	if !force {
		msg := fmt.Sprintf("%s above role definition? y/n", promptType)
		if utl.PromptMsg(utl.Yel(msg)) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	mazType := ResRoleDefinition

	// Call API to create or update definition
	payload := obj // Obviously using the inputed object as the payload
	params := map[string]string{"api-version": "2022-04-01"}
	apiUrl := ConstAzUrl + firstScope + ApiEndpoint[mazType] + "/" + id
	resp, statCode, _ := ApiPut(apiUrl, z, payload, params)
	if statCode != 201 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 201 {
		msg := fmt.Sprintf("Successfully %s %s!", deployType, MazTypeNames[mazType])
		fmt.Printf("%s\n", utl.Gre(msg))

		// Upsert object in local cache also
		cache, err := GetCache(mazType, z)
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
		err = cache.Upsert(obj.TrimForCache(mazType))
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
		if err := cache.Save(); err != nil {
			Logf("Failed to save cache: %v", err)
		}
	} else {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		utl.Die("%s\n", utl.Red(msg))
	}
}

// Deletes a role definition as defined by given object
func DeleteResRoleDefinition(force bool, obj AzureObject, z *Config) {
	roleName, firstScope := ValidateResRoleDefinitionObject(obj, z)

	// Check if role definition exists
	id, _ := GetAzureResRoleDefinitionByScopeAndName(firstScope, roleName, z)
	if !utl.ValidUuid(id) {
		utl.Die("Role definition %s doesn't exist\n", utl.Red(roleName))
	} else {
		obj["name"] = id
	}

	// Display the role definition and prompt for delete confirmation
	PrintResRoleDefinition(obj, z)
	if !force {
		msg := "Delete above role definition? y/n"
		if utl.PromptMsg(utl.Yel(msg)) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	mazType := ResRoleDefinition

	// Delete the object
	params := map[string]string{"api-version": "2022-04-01"}
	apiUrl := ConstAzUrl + firstScope + ApiEndpoint[mazType] + "/" + id
	resp, statCode, _ := ApiDelete(apiUrl, z, params)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 200 {
		msg := fmt.Sprintf("Successfully DELETED %s!", MazTypeNames[mazType])
		fmt.Printf("%s\n", utl.Gre(msg))

		// Also remove from local cache
		cache, err := GetCache(mazType, z)
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
		err = cache.Delete(id)
		if err == nil { // Only save if deletion succeeded
			err = cache.Save()
		}
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

// Counts all role definitions. If fromAzure is true, the definitions are sourced
// directly from Azure; otherwise, they are read from the local cache. It returns
// separate counts for custom and built-in roles.
func CountResRoleDefinitions(fromAzure bool, z *Config) (customCount, builtinCount int64) {
	customCount, builtinCount = 0, 0
	definitions := GetMatchingResRoleDefinitions("", fromAzure, z)
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
func GetMatchingResRoleDefinitions(filter string, force bool, z *Config) (list AzureObjectList) {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		singleRole := GetAzureResRoleDefinitionById(filter, z)
		if singleRole != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{singleRole}
		}
	}

	// Get current cache, or initialize a new cache for this type
	cache, err := GetCache(ResRoleDefinition, z)
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
		CacheAzureResRoleDefinitions(cache, true, z) // true = be verbose
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.StringSet{}            // Keep track of unique IDs to eliminate duplicates

	for i := range cache.data {
		role := cache.data[i]       // Index-based loop seem a bit faster
		id := utl.Str(role["name"]) // Resource role definitions use 'name' as the unique ID
		if id == "" || ids.Exists(id) {
			continue // Skip if the ID is empty () or already seen
		}
		// Check if the object contains the filter string
		if role.HasString(filter) {
			matchingList = append(matchingList, role) // Add matching object to the list
			ids.Add(id)                               // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all Azure resource role definition objects in current tenant and saves them
// to local cache. Note that we are updating the cache via its pointer, so no return values.
func CacheAzureResRoleDefinitions(cache *Cache, verbose bool, z *Config) {
	list := AzureObjectList{} // List of role definitions to cache
	ids := utl.StringSet{}    // Keep track of unique IDs to eliminate duplicate objects
	callCount := 1            // Track number of API calls for verbose output

	// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions-list

	// Set up these maps for more informative verbose output
	var mgroupIdMap, subIdMap map[string]string
	if verbose {
		mgroupIdMap = GetIdNameMap(ManagementGroup, z)
		subIdMap = GetIdNameMap(Subscription, z)
	}

	// Search in each resource scope
	scopes := GetAzureResRoleScopes(z)

	// Collate every unique role definition in each scope
	params := map[string]string{"api-version": "2022-04-01"}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		resp, statCode, _ := ApiGet(apiUrl, z, params)
		if statCode != 200 {
			Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
		}
		roles := utl.Slice(resp["value"])
		count := 0
		for i := range roles {
			obj := roles[i]
			if role := utl.Map(obj); role != nil {
				// Root out potential duplicates
				id := utl.Str(role["name"])
				if ids.Exists(id) {
					continue
					// Skip this repeated one. This can happen because of the way Azure resource
					// hierarchy inheritance works, and the same role is seen from multiple places.
				}
				list = append(list, role) // Add object to the list
				ids.Add(id)               // Mark this id as seen
				count++
			}
		}
		if verbose && count > 0 {
			scopeName := scope
			scopeType := "Subscription"
			if strings.HasPrefix(scope, "/providers") {
				scopeName = mgroupIdMap[scope]
				scopeType = "Management Group"
			} else if strings.HasPrefix(scope, "/subscriptions") {
				scopeName = subIdMap[path.Base(scope)]
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
		list[i] = list[i].TrimForCache(ResRoleDefinition)
	}

	// Update the cache with the entire list of definitions
	cache.data = list

	// Save the cache
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated resource role definitions cache: %v\n", err.Error())
	}
}

// Retrieves resource role definition by scope and name
func GetAzureResRoleDefinitionByScopeAndName(scope, roleName string, z *Config) (string, AzureObject) {
	params := map[string]string{
		"api-version": "2022-04-01",
		"$filter":     "roleName eq '" + roleName + "'",
	}
	apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
	resp, statCode, _ := ApiGet(apiUrl, z, params)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	roles := utl.Slice(resp["value"]) // Cast to a slice
	if len(roles) == 1 {
		// rolenNames are all unique within each scope, so a single entry means we found it
		if role := utl.Map(roles[0]); role != nil {
			id := utl.Str(role["name"])
			return id, role
		}
	}
	return "", nil
}

// Retrieves all role definitions with given roleName from the Azure resource hierarchy.
func GetAzureResRoleDefinitionsByName(roleName string, z *Config) AzureObjectList {
	// Get all the scopes in the tenant hierarchy
	scopes := GetAzureResRoleScopes(z)

	// NOTE: It is possible for a role with the same roleName to exist in multiple scopes
	// within the Azure ARM API. That is the reason why a list of matchingRoles is required.
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/custom-roles
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions

	matchingRoles := AzureObjectList{} // Initialize an empty AzureObjectList
	ids := utl.StringSet{}             // Keep track of unique IDs to eliminate duplicates

	// Search each of the tenant scopes
	params := map[string]string{
		"api-version": "2022-04-01",
		"$filter":     "roleName eq '" + roleName + "'",
	}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		resp, statCode, _ := ApiGet(apiUrl, z, params)
		if statCode != 200 {
			Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
		}
		roles := utl.Slice(resp["value"])
		for i := range roles {
			obj := roles[i]
			if role := utl.Map(obj); role != nil {
				id := utl.Str(role["name"])
				if ids.Exists(id) {
					continue
					// Skip this repeated one. This can happen because of the way Azure resource
					// hierarchy inherantance works, and the same role is seen from multiple places.
				}
				matchingRoles = append(matchingRoles, AzureObject(role)) // Append unique role
				ids.Add(id)                                              // Mark this id as seen
			}
		}
	}
	return matchingRoles
}

// Retrieves a role definition by its unique ID from the Azure resource hierarchy.
func GetAzureResRoleDefinitionById(targetId string, z *Config) AzureObject {
	// 1st try with new function that calls Azure Resource Graph API
	if role := GetAzureResObjectById(ResRoleDefinition, targetId, z); role != nil {
		return role // Return immediately if we found it
	}

	// Fallback to using the ARM API way if above returns nothing.

	// https://learn.microsoft.com/en-us/azure/role-based-access-control/custom-roles
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions

	// Create a list of API URLs to check
	apiUrls := []string{
		// The 1st is the standard roleDefinitions endpoint
		ConstAzUrl + "/providers/Microsoft.Authorization/roleDefinitions/" + targetId,
	}
	for _, scope := range GetAzureResRoleScopes(z) {
		// The others are all other scopes in the tenant resource hierarchy
		apiUrls = append(apiUrls, ConstAzUrl+scope+"/providers/Microsoft.Authorization/roleDefinitions/"+targetId)
	}

	// Check each API URL in the list
	params := map[string]string{"api-version": "2022-04-01"}
	for _, apiUrl := range apiUrls {
		resp, statCode, _ := ApiGet(apiUrl, z, params)
		if statCode != 200 {
			Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
		}
		if statCode == 200 {
			if role := utl.Map(resp); role != nil {
				if id := utl.Str(role["name"]); id == targetId {
					role["maz_from_azure"] = true
					return AzureObject(role) // Return immediately on 1st match
				}
			}
		}
	}

	return nil // Nothing found, return empty object
}
