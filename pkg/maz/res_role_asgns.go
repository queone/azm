package maz

import (
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/queone/utl"
)

// Prints resource role assignment object in YAML-like format
func PrintResRoleAssignment(obj AzureObject, z *Config) {
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
	roleIdMap := GetIdNameMap(ResRoleDefinition, z)
	roleDefinitionId := path.Base(utl.Str(props["roleDefinitionId"]))
	comment := "# Role '" + roleIdMap[roleDefinitionId] + "'"
	fmt.Printf("  %s: %s  %s\n", utl.Blu("roleDefinitionId"), utl.Gre(roleDefinitionId), utl.Gra(comment))

	// Get all id:name pairs for the principal type, to print their names as comments
	var principalIdMap map[string]string = nil
	pType := utl.Str(props["principalType"])
	switch pType {
	case "Group":
		principalIdMap = GetIdNameMap(DirectoryGroup, z) // Get all group id:name pairs
	case "User":
		principalIdMap = GetIdNameMap(DirectoryUser, z) // Get all users id:name pairs
	case "ServicePrincipal":
		principalIdMap = GetIdNameMap(ServicePrincipal, z) // Get all SPs id:name pairs
	default:
		pType = "UnknownPrincipalType"
	}
	principalId := utl.Str(props["principalId"])
	pName := principalIdMap[principalId]
	if pName == "" {
		pName = "???"
	}
	comment = "# " + pType + " '" + pName + "'"
	fmt.Printf("  %s: %s  %s\n", utl.Blu("principalId"), utl.Gre(principalId), utl.Gra(comment))

	// Get all subscription id:name pairs, to print their names as comments
	subIdMap := GetIdNameMap(Subscription, z)
	scope := utl.Str(props["scope"])
	colorKey := utl.Blu("scope")
	colorValue := utl.Gre(scope)
	if strings.HasPrefix(scope, "/subscriptions") {
		split := strings.Split(scope, "/")
		subName := subIdMap[split[2]]
		fmt.Printf("  %s: %s  %s\n", colorKey, colorValue, utl.Gra("# Subscription = "+subName))
	} else if scope == "/" {
		fmt.Printf("  %s: %s  %s\n", colorKey, colorValue, utl.Gra("# Tenant-wide assignment!"))
	} else {
		fmt.Printf("  %s: %s\n", colorKey, colorValue)
	}
}

// Helper function to check if the object is a resource role assignment
func IsResRoleAssignment(obj AzureObject) bool {
	// Check if 'properties' exists and is a map
	props := utl.Map(obj["properties"])
	if props == nil {
		return false
	}

	// Check if 'roleDefinitionId' exists and is a non-empty string
	if roleDefinitionId := utl.Str(props["roleDefinitionId"]); roleDefinitionId == "" {
		return false
	}

	// Check if 'principalId' exists and is a non-empty string
	if principalId := utl.Str(props["principalId"]); principalId == "" {
		return false
	}

	// Check if 'scope' exists and is a non-empty string
	if scope := utl.Str(props["scope"]); scope == "" {
		return false
	}

	// If all checks pass, it's a valid resource role assignment
	return true
}

// Prints a human-readable report of all Azure resource role assignments in the tenant
func PrintResRoleAssignmentReport(z *Config) {
	totalStart := time.Now()

	start := time.Now()
	roleIdMap := GetIdNameMap(ResRoleDefinition, z)
	Logf("Fetched role definition ID map      in %s ms\n", utl.Cya(fmt.Sprintf("%6d", time.Since(start).Milliseconds())))

	start = time.Now()
	subIdMap := GetIdNameMap(Subscription, z)
	Logf("Fetched subscription ID map         in %s ms\n", utl.Cya(fmt.Sprintf("%6d", time.Since(start).Milliseconds())))

	start = time.Now()
	groupIdMap := GetIdNameMap(DirectoryGroup, z)
	Logf("Fetched group ID map                in %s ms\n", utl.Cya(fmt.Sprintf("%6d", time.Since(start).Milliseconds())))

	start = time.Now()
	userIdMap := GetIdNameMap(DirectoryUser, z)
	Logf("Fetched user ID map                 in %s ms\n", utl.Cya(fmt.Sprintf("%6d", time.Since(start).Milliseconds())))

	start = time.Now()
	spIdMap := GetIdNameMap(ServicePrincipal, z)
	Logf("Fetched service principal ID map    in %s ms\n", utl.Cya(fmt.Sprintf("%6d", time.Since(start).Milliseconds())))

	Logf("Total ID map fetch time             in %s ms\n", utl.Cya(fmt.Sprintf("%6d", time.Since(totalStart).Milliseconds())))

	assignments := GetMatchingResRoleAssignments("", false, z)

	for i := range assignments {
		assignment := assignments[i]
		props := utl.Map(assignment["properties"])
		if assignment == nil || props == nil {
			continue
		}

		roleDefinitionId := path.Base(utl.Str(props["roleDefinitionId"]))
		principalId := utl.Str(props["principalId"])
		principalType := utl.Str(props["principalType"])

		principalName := "ID-Not-Found"
		switch principalType {
		case "Group":
			principalName = groupIdMap[principalId]
		case "User":
			principalName = userIdMap[principalId]
		case "ServicePrincipal":
			principalName = spIdMap[principalId]
		}

		scope := utl.Str(props["scope"])
		if strings.HasPrefix(scope, "/subscriptions") {
			split := strings.Split(scope, "/")
			scope = subIdMap[split[2]] + " " + strings.Join(split[3:], "/")
		}
		scope = strings.TrimSpace(scope)

		fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\"\n", roleIdMap[roleDefinitionId],
			principalName, principalType, scope)
	}
}

// Checks if object conforms to an Azure resource role assignment format. If it's valid,
// return the three key values: roleDefinitionId, principalId, and scope.
func ValidateResRoleAssignmentObject(obj AzureObject, z *Config) (string, string, string) {
	props := utl.Map(obj["properties"])
	if props == nil {
		utl.Die("Error with object's %s map\n", utl.Red("properties"))
	}

	roleDefinitionId := utl.Str(props["roleDefinitionId"])
	roleDefinitionId = path.Base(roleDefinitionId) // Standardize to standalone UUID
	principalId := utl.Str(props["principalId"])
	scope := utl.Str(props["scope"])

	if roleDefinitionId == "" || principalId == "" || scope == "" {
		utl.Die("Specfile is missing required attributes. Need at least:\n\n" +
			"properties:\n" +
			"    roleDefinitionId: <UUID or fully_qualified_roleDefinitionId>\n" +
			"    principalId:      <UUID>\n" +
			"    scope:            <resource_path_scope>\n\n" +
			"See utility '-k*' options to create properly formatted sample files.\n")
	}

	return roleDefinitionId, principalId, scope
}

// Creates an Azure resource role assignment as defined by give object
func CreateAzureResRoleAssignment(force bool, obj AzureObject, z *Config) {
	roleDefinitionId, principalId, scope := ValidateResRoleAssignmentObject(obj, z)

	// Check if role assignment already exists
	id, _ := GetAzureResRoleAssignmentBy3Args(roleDefinitionId, principalId, scope, z)
	if id == "" {
		// Does not exist, let's generate a new UUID to try to create below
		id = uuid.New().String()
	} else {
		utl.Die("This role assignment %s exists with ID %s\n", utl.Yel("already"), utl.Yel(id))
	}
	obj["name"] = id // This, so that it's printable in below prompt

	// Prompt to create
	PrintResRoleAssignment(obj, z)
	if !force {
		msg := "CREATE above role assignment? y/n"
		if utl.PromptMsg(utl.Yel(msg)) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Call API to create assignment
	payload := map[string]interface{}{
		"properties": map[string]string{
			"roleDefinitionId": "/providers/Microsoft.Authorization/roleDefinitions/" + roleDefinitionId,
			"principalId":      principalId,
		},
	}
	params := map[string]string{"api-version": "2022-04-01"}
	apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments/" + id
	resp, statCode, _ := ApiPut(apiUrl, z, payload, params)
	if statCode != 200 && statCode != 201 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 200 || statCode == 201 {
		fmt.Printf("%s\n", utl.Gre("Successfully CREATED role assignment!"))
		azObj := AzureObject(resp) // Cast newly created assignment object to our standard type

		// Upsert object in local cache also
		cache, err := GetCache(ResRoleAssignment, z)
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
		err = cache.Upsert(azObj.TrimForCache(ResRoleAssignment))
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
		if err := cache.Save(); err != nil {
			Logf("Failed to save cache: %v", err)
		}
	} else {
		msg := fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))
		fmt.Printf("%s\n", utl.Red(msg))
	}
}

// Deletes an Azure resource role assignment as defined by given object
func DeleteAzureResRoleAssignment(force bool, obj AzureObject, z *Config) {
	roleDefinitionId, principalId, scope := ValidateResRoleAssignmentObject(obj, z)

	// Check if role assignment exists
	azureId, _ := GetAzureResRoleAssignmentBy3Args(roleDefinitionId, principalId, scope, z)
	if azureId == "" {
		utl.Die("This role assignment does %s exist in Azure\n", utl.Yel("not"))
	}
	obj["name"] = azureId // So Print function can print it and we can see it in below prompt

	// Prompt to delete
	PrintResRoleAssignment(obj, z)
	if !force {
		msg := "DELETE above role assignment? y/n"
		if utl.PromptMsg(utl.Yel(msg)) != 'y' {
			utl.Die("Aborted.\n")
		}
	}

	// Delete the assignment by scope and 'name' (stand-alone UUID)
	// See learn.microsoft.com/en-us/rest/api/authorization/role-assignments/delete
	params := map[string]string{"api-version": "2022-04-01"}
	apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments/" + azureId
	resp, statCode, _ := ApiDelete(apiUrl, z, params)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 200 {
		fmt.Printf("%s\n", utl.Gre("Successfully DELETED role assignment!"))

		// Also remove from local cache
		cache, err := GetCache(ResRoleAssignment, z)
		if err != nil {
			utl.Die("Error: %v\n", err)
		}
		err = cache.Delete(azureId)
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

// Calculates count of all role assignment objects in Azure
func RoleAssignmentsCountAzure(z *Config) int64 {
	list := GetMatchingResRoleAssignments("", false, z) // false = quiet
	return int64(len(list))
}

// Gets all resource role assignments matching on 'filter'. Return entire list if filter is empty ""
func GetMatchingResRoleAssignments(filter string, force bool, z *Config) (list AzureObjectList) {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		singleAssignment := GetAzureResRoleAssignmentById(filter, z)
		if singleAssignment != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{singleAssignment}
		}
	}

	// Get current cache, or initialize a new cache for this type
	cache, err := GetCache(ResRoleAssignment, z)
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
		CacheAzureResRoleAssignments(cache, z)
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.StringSet{}            // Keep track of unique IDs to eliminate duplicates
	for i := range cache.data {
		assignment := cache.data[i] // No need to cast; should already be AzureObject type
		if assignment == nil {
			continue // But skip if it is nil for whatever reason
		}

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

// Retrieves all Azure resource role assignments in current tenant and saves them
// to local cache. Note that we are updating the cache via its pointer, so no return values.
func CacheAzureResRoleAssignments(cache *Cache, z *Config) {
	params := map[string]string{"api-version": "2022-04-01"}

	// Prepare ID name maps for more informative logging
	mgroupIdMap := GetIdNameMap(ManagementGroup, z)
	subIdMap := GetIdNameMap(Subscription, z)

	// Fetch all assignments across scopes concurrently using parallel goroutines function
	allAssignments := fetchAzureObjectsAcrossScopes(
		"/providers/Microsoft.Authorization/roleAssignments",
		z,
		params,
		mgroupIdMap,
		subIdMap,
	)

	ids := utl.StringSet{}
	list := AzureObjectList{}

	// Deduplicate results by ID
	for _, assignment := range allAssignments {
		id := utl.Str(assignment["name"])
		if ids.Exists(id) {
			continue
		}
		list = append(list, assignment)
		ids.Add(id)
	}

	Logf("Fetched %d unique role assignments across all scopes\n", len(list))

	// Trim and cache results
	for i := range list {
		list[i] = list[i].TrimForCache(ResRoleAssignment)
	}

	cache.data = list

	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated resource role assignment cache: %v\n", err.Error())
	}
}

// Retrieves Azure resource role assignment by matching on the three values that
// make up a unique assignment: roleDefinitionId, principalId, and scope
func GetAzureResRoleAssignmentBy3Args(targetRoleDefinitionId, targetPrincipalId, targetScope string, z *Config) (string, AzureObject) {
	// Validate input
	if targetScope == "" || targetPrincipalId == "" || targetRoleDefinitionId == "" {
		return "", nil
	}

	// Get all role assignments for targetPrincipalId under targetScope
	params := map[string]string{
		"api-version": "2022-04-01",
		"$filter":     "principalId eq '" + targetPrincipalId + "'",
	}
	apiUrl := ConstAzUrl + targetScope + "/providers/Microsoft.Authorization/roleAssignments"
	resp, statCode, _ := ApiGet(apiUrl, z, params)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	assignments := utl.Slice(resp["value"])
	if len(assignments) > 0 { // Inspect all qualifying assignments for this principalId
		for i := range assignments {
			element := assignments[i]
			assignment := utl.Map(element)             // Try casting object to a map
			props := utl.Map(assignment["properties"]) // Try casting its properties to a map
			if assignment == nil || props == nil {
				continue // Skip this entry if neither is a valid map
			}
			// Compare this entry to the target we're looking for
			id := utl.Str(assignment["name"])
			roleDefinitionId := path.Base(utl.Str(props["roleDefinitionId"]))
			if roleDefinitionId == targetRoleDefinitionId {
				return id, AzureObject(assignment) // If they match, return immediately
			}
		}
	}
	return "", nil
}

// Retrieves a role assignment by its unique ID from the Azure resource hierarchy.
func GetAzureResRoleAssignmentById(targetId string, z *Config) AzureObject {
	// First try using Azure Resource Graph API
	if assignment := GetAzureResObjectById(ResRoleAssignment, targetId, z); assignment != nil {
		return assignment // Return immediately if found
	}

	// Fallback to ARM API using the generic fetcher
	params := map[string]string{"api-version": "2022-04-01"}

	// Construct the suffix once to be reused across all scopes
	suffix := "/providers/Microsoft.Authorization/roleAssignments/" + targetId

	// Fetch all assignments across scopes concurrently using parallel goroutines function
	assignments := fetchAzureObjectsAcrossScopes(suffix, z, params, nil, nil)

	for _, assignment := range assignments {
		if id := utl.Str(assignment["name"]); id == targetId {
			assignment["maz_from_azure"] = true
			return AzureObject(assignment)
		}
	}

	return nil // Nothing found
}
