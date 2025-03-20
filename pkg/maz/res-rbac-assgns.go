package maz

import (
	"fmt"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/queone/utl"
)

// Prints RBAC role definition object in YAML-like format
func PrintRbacAssignment(obj AzureObject, z *Config) {
	id := utl.Str(obj["name"])
	if id == "" {
		return
	}
	fmt.Printf("%s\n", utl.Gra("# Resource role assignment"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	props := utl.Map(obj["properties"])
	if props == nil {
		utl.Die("%s\n", utl.Red("  <Missing properties?>"))
	}
	fmt.Println(utl.Blu("properties") + ":")

	// Get all role definition id:name pairs to print their names as comments
	roleNameMap := GetIdMapRoleDefs(z)
	roleDefinitionId := path.Base(utl.Str(props["roleDefinitionId"]))
	comment := "# Role '" + roleNameMap[roleDefinitionId] + "'"
	fmt.Printf("  %s: %s  %s\n", utl.Blu("roleDefinitionId"), utl.Gre(roleDefinitionId), utl.Gra(comment))

	// Get all id:name pairs for the principal type, to print their names as comments
	var principalNameMap map[string]string = nil
	pType := utl.Str(props["principalType"])
	switch pType {
	case "Group":
		principalNameMap = GetIdMapDirObjects(DirectoryGroup, z) // Get all group id:name pairs
	case "User":
		principalNameMap = GetIdMapDirObjects(DirectoryUser, z) // Get all users id:name pairs
	case "ServicePrincipal":
		principalNameMap = GetIdMapDirObjects(ServicePrincipal, z) // Get all SPs id:name pairs
	default:
		pType = "UnknownPrincipalType"
	}
	principalId := utl.Str(props["principalId"])
	pName := principalNameMap[principalId]
	if pName == "" {
		pName = "???"
	}
	comment = "# " + pType + " '" + pName + "'"
	fmt.Printf("  %s: %s  %s\n", utl.Blu("principalId"), utl.Gre(principalId), utl.Gra(comment))

	// Get all subscription id:name pairs, to print their names as comments
	subNameMap := GetIdMapSubscriptions(z)
	scope := utl.Str(props["scope"])
	colorKey := utl.Blu("scope")
	colorValue := utl.Gre(scope)
	if strings.HasPrefix(scope, "/subscriptions") {
		split := strings.Split(scope, "/")
		subName := subNameMap[split[2]]
		fmt.Printf("  %s: %s  %s\n", colorKey, colorValue, utl.Gra("# Subscription = "+subName))
	} else if scope == "/" {
		fmt.Printf("  %s: %s  %s\n", colorKey, colorValue, utl.Gra("# Tenant-wide assignment!"))
	} else {
		fmt.Printf("  %s: %s\n", colorKey, colorValue)
	}
}

// Prints a human-readable report of all Azure resource role assignments in the tenant
func PrintRbacAssignmentReport(z *Config) {
	roleNameMap := GetIdMapRoleDefs(z)                    // Get all role definition id:name pairs
	subNameMap := GetIdMapSubscriptions(z)                // Get all subscription id:name pairs
	groupNameMap := GetIdMapDirObjects(DirectoryGroup, z) // Get all groups id:name pairs
	userNameMap := GetIdMapDirObjects(DirectoryUser, z)   // Get all users id:name pairs
	spNameMap := GetIdMapDirObjects(ServicePrincipal, z)  // Get all SPs id:name pairs

	assignments := GetMatchingRbacAssignments("", false, z) // Get all the assignments. false = quietly

	// Memory-walk the slice to process them more efficiently
	for i := range assignments {
		assignmentPtr := &assignments[i] // Use a pointer to avoid copying the element
		assignment := *assignmentPtr     // Dereference the pointer for easier access

		props := utl.Map(assignment["properties"])
		if props == nil {
			continue // Skip if "properties" is missing or not a map
		}

		roleDefinitionId := path.Base(utl.Str(props["roleDefinitionId"]))
		principalId := utl.Str(props["principalId"])
		principalType := utl.Str(props["principalType"])
		principalName := "ID-Not-Found"
		switch principalType {
		case "Group":
			principalName = groupNameMap[principalId]
		case "User":
			principalName = userNameMap[principalId]
		case "ServicePrincipal":
			principalName = spNameMap[principalId]
		}

		scope := utl.Str(props["scope"])
		if strings.HasPrefix(scope, "/subscriptions") {
			// Replace subscription ID with its name, but keep the rest of the resource path
			split := strings.Split(scope, "/")
			scope = subNameMap[split[2]] + " " + strings.Join(split[3:], "/")
		}
		scope = strings.TrimSpace(scope)

		fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\"\n", roleNameMap[roleDefinitionId], principalName, principalType, scope)
	}
}

// Creates a role assignment as defined by give object
func CreateRbacAssignment(x map[string]interface{}, z *Config) {
	if x == nil {
		return
	}
	props := x["properties"].(map[string]interface{})
	roleDefinitionId := path.Base(utl.Str(props["roleDefinitionId"])) // Note we only care about the UUID
	principalId := utl.Str(props["principalId"])
	scope := utl.Str(props["scope"])
	if scope == "" {
		scope = utl.Str(props["Scope"]) // Account for possibly capitalized key
	}
	if roleDefinitionId == "" || principalId == "" || scope == "" {
		utl.Die("Specfile is missing required attributes. Need at least:\n\n" +
			"properties:\n" +
			"    roleDefinitionId: <UUID or fully_qualified_roleDefinitionId>\n" +
			"    principalId: <UUID>\n" +
			"    scope: <resource_path_scope>\n\n" +
			"See script '-k*' options to create properly formatted sample files.\n")
	}

	// Note, there is no need to pre-check if assignment exists, since call will simply let us know
	newUuid := uuid.New().String() // Generate a new global UUID in string format
	payload := map[string]interface{}{
		"properties": map[string]string{
			"roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/" + roleDefinitionId,
			"principalId":      principalId,
		},
	}
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments/" + newUuid
	resp, statCode, _ := ApiPut(apiUrl, z, payload, params)
	if statCode == 200 || statCode == 201 {
		utl.PrintYaml(resp)
	} else {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		fmt.Printf("%s\n", utl.Red(msg))
	}
}

// Deletes an Azure resource role assignment as defined by given object
func DeleteRbacAssignment(force bool, obj AzureObject, z *Config) {
}

// Deletes an Azure resource role assignment by its fully qualified object Id
// Example of a fully qualified Id string (note it's one long line):
//
//	/providers/Microsoft.Management/managementGroups/33550b0b-2929-4b4b-adad-cccc66664444 \
//	  /providers/Microsoft.Authorization/roleAssignments/5d586a7b-3f4b-4b5c-844a-3fa8efe49ab3
func DeleteRbacAssignmentByFqid(fqid string, z *Config) map[string]interface{} {
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	apiUrl := ConstAzUrl + fqid
	resp, statCode, _ := ApiDelete(apiUrl, z, params)
	if statCode != 200 {
		if statCode == 204 {
			fmt.Println("Role assignment already deleted or does not exist. Give Azure a minute to flush it out.")
		} else {
			msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
			fmt.Printf("%s\n", utl.Red(msg))
		}
	}
	return nil
}

// Calculates count of all role assignment objects in Azure
func RoleAssignmentsCountAzure(z *Config) int64 {
	list := GetMatchingRbacAssignments("", false, z) // false = quiet
	return int64(len(list))
}

// Gets all RBAC role assignments matching on 'filter'. Return entire list if filter is empty ""
func GetMatchingRbacAssignments(filter string, force bool, z *Config) (list AzureObjectList) {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		singleAssignment := GetAzureRbacAssignmentById(filter, z)
		if singleAssignment != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{singleAssignment}
		}
	}

	// Get current cache, or initialize a new cache for this type
	cache, err := GetCache(RbacAssignment, z)
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
		CacheAzureRbacAssignments(cache, true, z) // true = be verbose
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.StringSet{}            // Keep track of unique IDs to eliminate duplicates
	for i := range cache.data {
		rawPtr := &cache.data[i]         // Access the element directly via pointer (memory walk)
		rawObj := *rawPtr                // Dereference the pointer manually
		assignmentMap := utl.Map(rawObj) // Try asserting as a map type
		if assignmentMap == nil {
			continue // Skip this entry if not a map
		}
		assignment := AzureObject(assignmentMap) // Cast as our standard AzureObject type

		// Extract the ID: use the last part of the "id" path or fall back to the "name" field
		id := utl.Str(assignment["id"])
		name := utl.Str(assignment["name"])
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
		if assignment.HasString(filter) {
			matchingList = append(matchingList, assignment) // Add matching object to the list
			ids.Add(id)                                     // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all Azure resource RBAC assignments in current tenant and saves them
// to local cache. Note that we are updating the cache via its pointer, so no return values.
func CacheAzureRbacAssignments(cache *Cache, verbose bool, z *Config) {
	list := AzureObjectList{} // List of role assignments to cache
	ids := utl.StringSet{}    // Keep track of unique resourceIds (API SPs)
	callCount := 1            // Track number of API calls for verbose output

	// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments-list-rest

	// Set up these maps for more informative verbose output
	var mgGroupNameMap, subNameMap map[string]string
	if verbose {
		mgGroupNameMap = GetIdMapMgmtGroups(z)
		subNameMap = GetIdMapSubscriptions(z)
	}

	// Search in each resource RBAC scope
	scopes := GetAzureRbacScopes(z)

	// Collate every unique role assignment in each scope
	params := map[string]string{"api-version": "2022-04-01"}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments"
		resp, statCode, _ := ApiGet(apiUrl, z, params)
		if statCode != 200 {
			// For now, I don't think we care about any errors
			continue // If any issues retrieving items for this scope, go to next one
		}
		assignments := utl.Slice(resp["value"]) // Try asserting value as an object of slice type
		if assignments == nil {
			continue // If its's not a slice with values, process next scope
		}

		count := 0
		for i := range assignments {
			rawPtr := &assignments[i]     // Get a pointer to the current item in the slice
			rawObj := *rawPtr             // Dereference the pointer manually
			assignment := utl.Map(rawObj) // Try asserting as a map type
			if assignment == nil {
				continue // Skip this entry if not a map
			}
			// Root out potential duplicates
			id := utl.Str(assignment["name"])
			if ids.Exists(id) {
				continue // Skip this entry if it's a repeat
				// Skip this repeated one. This can happen because of the way Azure RBAC
				// hierarchy inheritance works, and the same role is seen from multiple places.
			}
			ids.Add(id) // Mark this id as seen
			list = append(list, assignment)
			count++
		}

		if verbose && count > 0 {
			scopeName := scope
			scopeType := "subscription"
			if strings.HasPrefix(scope, "/providers") {
				scopeName = mgGroupNameMap[scope]
				scopeType = "Management Group"
			} else if strings.HasPrefix(scope, "/subscriptions") {
				scopeName = subNameMap[path.Base(scope)]
			}
			fmt.Printf("%sCall %05d: %05d assignments under %s %s", rUp, callCount, count, scopeType, scopeName)
		}
		callCount++
	}
	if verbose {
		fmt.Print(rUp) // Go up to overwrite progress line
	}

	// Trim and prepare all objects for caching
	for i := range list {
		// Directly modify the object in the original list
		list[i] = list[i].TrimForCache(RbacAssignment)
	}

	// Update the cache with the entire list of definitions
	cache.data = list

	// Save the cache
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated resource role assignment cache: %v\n", err.Error())
	}
}

// Gets Azure resource RBAC role assignment object by matching given objects: roleId, principalId,
// and scope (the 3 parameters which make a role assignment unique)
func GetRbacAssignmentByObject(x map[string]interface{}, z *Config) (y map[string]interface{}) {
	// First, make sure x is a searchable role assignment object
	if x == nil {
		return nil
	}

	props := x["properties"].(map[string]interface{})
	if props == nil {
		return nil
	}

	xRoleDefinitionId := path.Base(utl.Str(props["roleDefinitionId"]))
	xPrincipalId := utl.Str(props["principalId"])
	xScope := utl.Str(props["scope"])
	if xScope == "" {
		xScope = utl.Str(props["Scope"]) // Account for possibly capitalized key
	}
	if xScope == "" || xPrincipalId == "" || xRoleDefinitionId == "" {
		return nil
	}

	// Get all role assignments for xPrincipalId under xScope
	params := map[string]string{
		"api-version": "2022-04-01", // roleAssignments
		"$filter":     "principalId eq '" + xPrincipalId + "'",
	}
	apiUrl := ConstAzUrl + xScope + "/providers/Microsoft.Authorization/roleAssignments"
	resp, _, _ := ApiGet(apiUrl, z, params)
	if resp != nil && resp["value"] != nil {
		results := resp["value"].([]interface{})
		//fmt.Println(len(results))
		for _, i := range results {
			y = i.(map[string]interface{})
			yProp := y["properties"].(map[string]interface{})
			yScope := utl.Str(yProp["scope"])
			yRoleDefinitionId := path.Base(utl.Str(yProp["roleDefinitionId"]))
			if yScope == xScope && yRoleDefinitionId == xRoleDefinitionId {
				return y // As soon as we find it
			}
		}
	}
	return nil // If we get here, we didn't fine it, so return nil
}

// Retrieves a role assignment by its unique ID from the Azure resource RBAC hierarchy.
func GetAzureRbacAssignmentById(id string, z *Config) AzureObject {
	// Get all the scopes in the tenant hierarchy
	scopes := GetAzureRbacScopes(z)

	// NOTE: Microsoft documentation explicitly states that a role assignment UUID
	// cannot be repeated across different scopes in the hierarchy. This is why we
	// return immediately upon a successful match in any of the scopes.
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/custom-roles
	// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions

	// Search each of the tenant scopes
	params := map[string]string{"api-version": "2022-04-01"}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments/" + id
		//apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments"
		resp, statCode, _ := ApiGet(apiUrl, z, params)
		if statCode == 200 {
			assignmentObj := utl.Map(resp) // Try asserting the response as a single object of map type
			if assignmentObj == nil {
				continue
			}
			assignmentObj["maz_from_azure"] = true
			return AzureObject(assignmentObj) // Return immediately on 1st match

			// Or is below rote scope search method still needed?? -- with out id in apiUrl
			// assignments := utl.Slice(resp["value"])
			// if assignments == nil {
			// 	continue
			// }
			// for i := range assignments {
			// 	assignmentPtr := &assignments[i]
			// 	assignmentRaw := *assignmentPtr
			// 	assignment := utl.Map(assignmentRaw)
			// 	if assignment == nil {
			// 		continue
			// 	}
			// 	name := utl.Str(assignment["name"])
			// 	if name == "" {
			// 		continue
			// 	}
			// 	if name == id {
			// 		assignment["maz_from_azure"] = true
			// 		return AzureObject(assignment) // Return immediately on 1st match
			// 	}
			// }
		}
	}
	return nil
}
