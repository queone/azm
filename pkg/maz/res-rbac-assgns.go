package maz

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/queone/utl"
)

// Prints RBAC role definition object in YAML-like format
func PrintRbacAssignment(x map[string]interface{}, z *Config) {
	fmt.Printf("%s\n", utl.Gra("# Resource RBAC Assignment"))
	if x == nil {
		return
	}
	if x["name"] != nil {
		fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(utl.Str(x["name"])))
	}
	if x["properties"] != nil {
		fmt.Println(utl.Blu("properties") + ":")
	} else {
		fmt.Println("  < Missing properties? What's going? >")
	}

	props := x["properties"].(map[string]interface{})

	roleNameMap := GetIdMapRoleDefs(z) // Get all role definition id:name pairs
	roleId := utl.LastElem(utl.Str(props["roleDefinitionId"]), "/")
	comment := "# Role \"" + roleNameMap[roleId] + "\""
	fmt.Printf("  %s: %s  %s\n", utl.Blu("roleDefinitionId"), utl.Gre(roleId), utl.Gra(comment))

	var principalNameMap map[string]string = nil
	pType := utl.Str(props["principalType"])
	switch pType {
	case "Group":
		principalNameMap = GetDirObjectIdMap("g", z) // Get all group id:name pairs
	case "User":
		principalNameMap = GetDirObjectIdMap("u", z) // Get all users id:name pairs
	case "ServicePrincipal":
		principalNameMap = GetDirObjectIdMap("sp", z) // Get all SPs id:name pairs
	default:
		pType = "SomeObject"
	}
	principalId := utl.Str(props["principalId"])
	pName := principalNameMap[principalId]
	if pName == "" {
		pName = "???"
	}
	comment = "# " + pType + " \"" + pName + "\""
	fmt.Printf("  %s: %s  %s\n", utl.Blu("principalId"), utl.Gre(principalId), utl.Gra(comment))

	subNameMap := GetAzureSubscriptionsIdMap(z) // Get all subscription id:name pairs
	scope := utl.Str(props["scope"])
	if scope == "" {
		scope = utl.Str(props["Scope"])
	} // Account for possibly capitalized key
	cScope := utl.Blu("scope")
	if strings.HasPrefix(scope, "/subscriptions") {
		split := strings.Split(scope, "/")
		subName := subNameMap[split[2]]
		comment = "# Sub = " + subName
		fmt.Printf("  %s: %s  %s\n", cScope, utl.Gre(scope), utl.Gra(comment))
	} else if scope == "/" {
		comment = "# Entire tenant"
		fmt.Printf("  %s: %s  %s\n", cScope, utl.Gre(scope), utl.Gra(comment))
	} else {
		fmt.Printf("  %s: %s\n", cScope, utl.Gre(scope))
	}
}

// Prints a human-readable report of all Azure resource RBAC assignments in the tenant
func PrintRbacAssignmentReport(z *Config) {
	roleNameMap := GetIdMapRoleDefs(z)          // Get all role definition id:name pairs
	subNameMap := GetAzureSubscriptionsIdMap(z) // Get all subscription id:name pairs
	groupNameMap := GetDirObjectIdMap("g", z)   // Get all groups id:name pairs
	userNameMap := GetDirObjectIdMap("u", z)    // Get all users id:name pairs
	spNameMap := GetDirObjectIdMap("sp", z)     // Get all SPs id:name pairs

	assignments := GetRbacAssignments(z, false)
	for _, i := range assignments {
		x := i.(map[string]interface{})
		props := x["properties"].(map[string]interface{})
		Rid := utl.LastElem(utl.Str(props["roleDefinitionId"]), "/")
		principalId := utl.Str(props["principalId"])
		Type := utl.Str(props["principalType"])
		pName := "ID-Not-Found"
		switch Type {
		case "Group":
			pName = groupNameMap[principalId]
		case "User":
			pName = userNameMap[principalId]
		case "ServicePrincipal":
			pName = spNameMap[principalId]
		}

		Scope := utl.Str(props["scope"])
		if strings.HasPrefix(Scope, "/subscriptions") {
			// Replace sub ID to name
			split := strings.Split(Scope, "/")
			// Map subscription Id to its name + the rest of the resource path
			Scope = subNameMap[split[2]] + " " + strings.Join(split[3:], "/")
		}
		Scope = strings.TrimSpace(Scope)

		fmt.Printf("\"%s\",\"%s\",\"%s\",\"%s\"\n", roleNameMap[Rid], pName, Type, Scope)
	}
}

// Creates an RBAC role assignment as defined by give x object
func CreateRbacAssignment(x map[string]interface{}, z *Config) {
	if x == nil {
		return
	}
	props := x["properties"].(map[string]interface{})
	roleDefinitionId := utl.LastElem(utl.Str(props["roleDefinitionId"]), "/") // Note we only care about the UUID
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
	resp, statusCode, _ := ApiPut(apiUrl, z, payload, params)
	if statusCode == 200 || statusCode == 201 {
		utl.PrintYaml(resp)
	} else {
		e := resp["error"].(map[string]interface{})
		fmt.Println(e["message"].(string))
	}
}

// Deletes an Azure resource RBAC role assignment as defined by given object
func DeleteRbacAssignment(force bool, obj AzureObject, z *Config) {
}

// Deletes an RBAC role assignment by its fully qualified object Id
// Example of a fully qualified Id string (note it's one long line):
//
//	/providers/Microsoft.Management/managementGroups/33550b0b-2929-4b4b-adad-cccc66664444 \
//	  /providers/Microsoft.Authorization/roleAssignments/5d586a7b-3f4b-4b5c-844a-3fa8efe49ab3
func DeleteRbacAssignmentByFqid(fqid string, z *Config) map[string]interface{} {
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	apiUrl := ConstAzUrl + fqid
	resp, statusCode, _ := ApiDelete(apiUrl, z, params)
	if statusCode != 200 {
		if statusCode == 204 {
			fmt.Println("Role assignment already deleted or does not exist. Give Azure a minute to flush it out.")
		} else {
			e := resp["error"].(map[string]interface{})
			fmt.Println(e["message"].(string))
		}
	}
	return nil
}

// Retrieves count of all role assignment objects in local cache file
func RoleAssignmentsCountLocal(z *Config) int64 {
	var cachedList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleAssignments."+ConstCacheFileExtension)
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile, true) // Load compressed file
		if rawList != nil {
			cachedList = rawList.([]interface{})
			return int64(len(cachedList))
		}
	}
	return 0
}

// Calculates count of all role assignment objects in Azure
func RoleAssignmentsCountAzure(z *Config) int64 {
	list := GetRbacAssignments(z, false) // false = quiet
	return int64(len(list))
}

// Gets all RBAC role assignments matching on 'filter'. Return entire list if filter is empty ""
func GetMatchingRoleAssignments(filter string, force bool, z *Config) (list []interface{}) {
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleAssignments."+ConstCacheFileExtension)
	cacheFileAge := utl.FileAge(cacheFile)
	if utl.IsInternetAvailable() && (force || cacheFileAge == 0 || cacheFileAge > ConstAzCacheFileAgePeriod) {
		// If Internet is available AND (force was requested OR cacheFileAge is zero (meaning does not exist)
		// OR it is older than ConstAzCacheFileAgePeriod) then query Azure directly to get all objects
		// and show progress while doing so (true = verbose below)
		list = GetRbacAssignments(z, true)
	} else {
		// Use local cache for all other conditions
		list = GetCachedObjects(cacheFile)
	}

	if filter == "" {
		return list
	}
	var matchingList []interface{} = nil
	roleNameMap := GetIdMapRoleDefs(z) // Get all role definition id:name pairs
	for _, i := range list {           // Parse every object
		x := i.(map[string]interface{})
		// Match against relevant strings within roleAssigment JSON object (Note: Not all attributes are maintained)
		props := x["properties"].(map[string]interface{})
		roleId := utl.Str(props["roleDefinitionId"])
		roleName := roleNameMap[utl.LastElem(roleId, "/")]
		if utl.SubString(roleName, filter) || utl.StringInJson(x, filter) {
			matchingList = append(matchingList, x)
		}
	}
	return matchingList
}

// Gets all role assignments objects in current Azure tenant and save them to local cache file.
// Option to be verbose (true) or quiet (false), since it can take a while.
// References:
//
//	https://learn.microsoft.com/en-us/azure/role-based-access-control/role-assignments-list-rest
//	https://learn.microsoft.com/en-us/rest/api/authorization/role-assignments/list-for-subscription
func GetRbacAssignments(z *Config, verbose bool) (list []interface{}) {
	list = nil                      // We have to zero it out
	uniqueIds := utl.NewStringSet() // Unique resourceIds (API SPs)
	k := 1                          // Track number of API calls to provide progress

	var mgGroupNameMap, subNameMap map[string]string
	if verbose {
		mgGroupNameMap = GetAzureMgmtGroupsIdMap(z)
		subNameMap = GetAzureSubscriptionsIdMap(z)
	}

	scopes := GetAzureRbacScopes(z)                          // Get all scopes
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments"
		resp, _, _ := ApiGet(apiUrl, z, params)
		if resp != nil && resp["value"] != nil {
			objectsUnderThisScope := resp["value"].([]interface{})
			count := 0

			for _, i := range objectsUnderThisScope {
				x := i.(map[string]interface{})
				id := utl.Str(x["name"])
				if uniqueIds.Exists(id) {
					continue // Skip this repeated one. This can happen due to inherited nesting
				}
				uniqueIds.Add(id) // Mark this id as seen
				list = append(list, x)
				count++
			}
			if verbose && count > 0 {
				scopeName := scope
				scopeType := "subscription"
				if strings.HasPrefix(scope, "/providers") {
					scopeName = mgGroupNameMap[scope]
					scopeType = "magmnt group"
				} else if strings.HasPrefix(scope, "/subscriptions") {
					scopeName = subNameMap[utl.LastElem(scope, "/")]
				}

				fmt.Printf("%sCall %05d: %05d assignments under %s %s", rUp, k, count, scopeType, scopeName)
			}
		}
		k++
	}

	if verbose {
		fmt.Print(rUp) // Go up to overwrite progress line
	}

	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleAssignments."+ConstCacheFileExtension)
	utl.SaveFileJson(list, cacheFile, true) // Update the local cache, true = gzipped
	return list
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

	xRoleDefinitionId := utl.LastElem(utl.Str(props["roleDefinitionId"]), "/")
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
			yRoleDefinitionId := utl.LastElem(utl.Str(yProp["roleDefinitionId"]), "/")
			if yScope == xScope && yRoleDefinitionId == xRoleDefinitionId {
				return y // As soon as we find it
			}
		}
	}
	return nil // If we get here, we didn't fine it, so return nil
}

// Gets an Azure resource RBAC assignment by its object Id. Unfortunately we have to
// iterate through the entire tenant scope hierarchy, which can be slow.
func GetRbacAssignmentById(id string, z *Config) map[string]interface{} {
	scopes := GetAzureRbacScopes(z)
	params := map[string]string{"api-version": "2022-04-01"} // roleAssignments
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleAssignments"
		resp, _, _ := ApiGet(apiUrl, z, params)
		if resp != nil && resp["value"] != nil {
			assignmentsUnderThisScope := resp["value"].([]interface{})
			for _, i := range assignmentsUnderThisScope {
				x := i.(map[string]interface{})
				if utl.Str(x["name"]) == id {
					return x // Return as soon as we find a match
				}
			}
		}
	}
	return nil
}
