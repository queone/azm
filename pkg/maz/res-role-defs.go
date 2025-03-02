package maz

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/queone/utl"
)

// Prints role definition object in a YAML-like format
func PrintRoleDefinition(x map[string]interface{}, z *Config) {
	fmt.Printf("%s\n", utl.Gra("# RBAC Role Definition"))
	if x == nil {
		return
	}
	if x["name"] != nil {
		fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(utl.Str(x["name"])))
	}
	if x["properties"] != nil {
		fmt.Println(utl.Blu("properties") + ":")
	} else {
		fmt.Println(utl.Red("  <Missing properties??>"))
	}

	xProp := x["properties"].(map[string]interface{})

	list := []string{"roleName", "description"}
	for _, i := range list {
		fmt.Printf("  %s: %s\n", utl.Blu(i), utl.Gre(utl.Str(xProp[i])))
	}

	fmt.Printf("  %s: ", utl.Blu("assignableScopes"))
	if xProp["assignableScopes"] == nil {
		fmt.Printf("[]\n")
	} else {
		fmt.Printf("\n")
		scopes := xProp["assignableScopes"].([]interface{})
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
	if xProp["permissions"] == nil {
		fmt.Println(utl.Red("    < No permissions?? >\n"))
	} else {
		permsSet := xProp["permissions"].([]interface{})
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

// Creates or updates an RBAC role definition as defined by give x object
func UpsertAzRoleDefinition(force bool, x map[string]interface{}, z *Config) {
	if x == nil {
		return
	}
	xProp := x["properties"].(map[string]interface{})
	xRoleName := utl.Str(xProp["roleName"])
	// Below two are required in the API body call, but we don't need to burden
	// the user with this requirement, and just update the values for them here.
	if xProp["type"] == nil {
		xProp["type"] = "CustomRole"
	}
	if xProp["description"] == nil {
		xProp["description"] = ""
	}
	xScopes := xProp["assignableScopes"].([]interface{})
	xScope1 := utl.Str(xScopes[0]) // For deployment, we'll use 1st scope
	var permSet []interface{} = nil
	if xProp["permissions"] != nil {
		permSet = xProp["permissions"].([]interface{})
	}
	if xProp == nil || xScopes == nil || xRoleName == "" || xScope1 == "" ||
		permSet == nil || len(permSet) < 1 {
		utl.Die("Specfile is missing required attributes. The bare minimum is:\n\n" +
			"properties:\n" +
			"  roleName: \"My Role Name\"\n" +
			"  assignableScopes:\n" +
			"    - /providers/Microsoft.Management/managementGroups/3f550b9f-8888-7777-ad61-111199992222\n" +
			"  permissions:\n" +
			"    - actions:\n\n" +
			"See script '-k*' options to create properly formatted sample files.\n")
	}

	roleId := ""
	existing := GetAzRoleDefinitionByName(xRoleName, z)
	if existing == nil {
		// Role definition doesn't exist, so we're creating a new one
		roleId = uuid.New().String() // Generate a new global UUID in string format
	} else {
		// Role exists, we'll prompt for update choice
		PrintRoleDefinition(existing, z)
		if !force {
			msg := utl.Yel("Role already exists! UPDATE it? y/n ")
			if utl.PromptMsg(msg) != 'y' {
				utl.Die("Aborted.\n")
			}
		}
		fmt.Println("Updating role ...")
		roleId = utl.Str(existing["name"])
	}

	payload := x                                             // Obviously using x object as the payload
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	apiUrl := ConstAzUrl + xScope1 + "/providers/Microsoft.Authorization/roleDefinitions/" + roleId
	r, statusCode, _ := ApiPut(apiUrl, z, payload, params)
	if statusCode == 201 {
		PrintRoleDefinition(r, z) // Print the newly updated object
	} else {
		e := r["error"].(map[string]interface{})
		fmt.Println(e["message"].(string))
	}
}

// Deletes an RBAC role definition object by its fully qualified object Id
// Example of a fully qualified Id string:
//
//	"/providers/Microsoft.Authorization/roleDefinitions/50a6ff7c-3ac5-4acc-b4f4-9a43aee0c80f"
func DeleteAzRoleDefinitionByFqid(fqid string, z *Config) map[string]interface{} {
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	apiUrl := ConstAzUrl + fqid
	r, statusCode, _ := ApiDelete(apiUrl, z, params)
	//ApiErrorCheck("DELETE", apiUrl, utl.Trace(), r)
	if statusCode != 200 {
		if statusCode == 204 {
			fmt.Println("Role definition already deleted or does not exist. Give Azure a minute to flush it out.")
		} else {
			e := r["error"].(map[string]interface{})
			fmt.Println(e["message"].(string))
		}
	}
	return nil
}

// Returns id:name map of all RBAC role definitions
func GetIdMapRoleDefs(z *Config) (nameMap map[string]string) {
	nameMap = make(map[string]string)
	roleDefs := GetMatchingAzRoleDefinitions("", false, z) // false = don't force going to Azure
	// By not forcing an Azure call we're opting for cache speed over id:name map accuracy
	for _, x := range roleDefs {
		//x := i.(AzureObjectList)
		if x["name"] != nil {
			xProp := x["properties"].(map[string]interface{})
			if xProp["roleName"] != nil {
				nameMap[utl.Str(x["name"])] = utl.Str(xProp["roleName"])
			}
		}
	}
	return nameMap
}

// Dedicated role definition local cache counter able to discern if role is custom to native tenant or it's an Azure BuilIn role
func RoleDefinitionCountLocal(z *Config) (builtin, custom int64) {
	var customList []interface{} = nil
	var builtinList []interface{} = nil
	cacheFile := filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions."+ConstCacheFileExtension)
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile, true) // Read compressed file
		if rawList != nil {
			definitions := rawList.([]interface{})
			for _, i := range definitions {
				x := i.(map[string]interface{}) // Assert as JSON object type
				xProp := x["properties"].(map[string]interface{})
				if utl.Str(xProp["type"]) == "CustomRole" {
					customList = append(customList, x)
				} else {
					builtinList = append(builtinList, x)
				}
			}
			return int64(len(builtinList)), int64(len(customList))
		}
	}
	return 0, 0
}

// Counts all role definition in Azure. Returns 2 lists: one of native custom roles, the other of built-in role
func RoleDefinitionCountAzure(z *Config) (builtin, custom int64) {
	var customList []interface{} = nil
	var builtinList []interface{} = nil
	definitions := GetAzRoleDefinitions(z, false) // false = be silent
	for _, i := range definitions {
		x := i.(map[string]interface{}) // Assert as JSON object type
		xProp := x["properties"].(map[string]interface{})
		if utl.Str(xProp["type"]) == "CustomRole" {
			customList = append(customList, x)
		} else {
			builtinList = append(builtinList, x)
		}
	}
	return int64(len(builtinList)), int64(len(customList))
}

// Gets all role definitions matching on 'filter'. Returns entire list if filter is empty ""
func GetMatchingAzRoleDefinitions(filter string, force bool, z *Config) (list AzureObjectList) {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		x := GetResRoleDefinitionById(filter, z)
		if x != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{x}
		}
		// If not found, then filter will be used below in obj.HasString(filter)
	}

	// Get current cache, or initialize a new cache for this type
	cache, err := GetCache("d", z)
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
		SyncAzRoleDefinitionsToLocalCache(cache, z, true)
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

// OLD, being replaced by below
func GetAzRoleDefinitions(z *Config, verbose bool) map[string]interface{} {
	return nil
}

// Retrieves all Azure resource role definitions in current tenant and save them to local cache.
// Note that we are updating the cache via its pointer. Shows progress if verbose == true.
// Reference:
// https://learn.microsoft.com/en-us/azure/role-based-access-control/role-definitions-list
// https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/list
func SyncAzRoleDefinitionsToLocalCache(cache *Cache, z *Config, verbose bool) {
	list := NewList()         // Start with a new empty list
	ids := utl.NewStringSet() // Keep track of unique IDs to eliminate duplicates
	k := 1                    // Track number of API calls to provide progress

	var mgGroupNameMap, subNameMap map[string]string
	if verbose {
		mgGroupNameMap = GetAzureMgmtGroupsIdMap(z)
		subNameMap = GetAzureSubscriptionsIdMap(z)
	}

	scopes := GetAzRbacScopes(z)                             // Get all scopes
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(apiUrl, z, params)
		if r != nil && r["value"] != nil {
			objectsUnderThisScope := r["value"].([]interface{})
			count := 0
			for _, i := range objectsUnderThisScope {
				x := i.(map[string]interface{})
				id := utl.Str(x["name"])
				if ids.Exists(id) {
					continue // Skip this repeated one. This can happen due to inherited nesting
				}
				ids.Add(id) // Mark this id as seen
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

				fmt.Printf("%sCall %05d: %05d definitions under %s %s", rUp, k, count, scopeType, scopeName)
			}
		}
		k++
	}

	if verbose {
		fmt.Print(rUp) // Go up to overwrite progress line
	}

	// Save the updated cache back to file
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated cache: %s\n", err.Error())
	}
}

// Gets role definition object if it exists exactly as x object (as per essential attributes).
// Matches on: displayName and assignableScopes
func GetAzRoleDefinitionByObject(x map[string]interface{}, z *Config) (y map[string]interface{}) {
	// First, make sure x is a searchable role definition object
	if x == nil { // Don't look for empty objects
		return nil
	}
	xProp := x["properties"].(map[string]interface{})
	if xProp == nil {
		return nil
	}

	xScopes := xProp["assignableScopes"].([]interface{})
	if utl.GetType(xScopes)[0] != '[' || len(xScopes) < 1 {
		return nil // Return nil if assignableScopes not an array, or it's empty
	}
	xRoleName := utl.Str(xProp["roleName"])
	if xRoleName == "" {
		return nil
	}

	// Look for x under all its scopes
	for _, i := range xScopes {
		scope := utl.Str(i)
		if scope == "/" {
			scope = ""
		} // Highly unlikely but just to avoid an err
		// Get all role assignments for xPrincipalId under xScope
		params := map[string]string{
			"api-version": "2022-04-01", // roleDefinitions
			"$filter":     "roleName eq '" + xRoleName + "'",
		}
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(apiUrl, z, params)
		if r != nil && r["value"] != nil {
			results := r["value"].([]interface{})
			if len(results) == 1 {
				y = results[0].(map[string]interface{}) // Select first index entry
				return y                                // We found it
			} else {
				return nil // If there's more than one entry we have other problems, so just return nil
			}
		}
	}
	return nil
}

// Gets role definition by displayName
// See https://learn.microsoft.com/en-us/rest/api/authorization/role-definitions/list
func GetAzRoleDefinitionByName(roleName string, z *Config) (y map[string]interface{}) {
	y = nil
	scopes := GetAzRbacScopes(z) // Get all scopes
	params := map[string]string{
		"api-version": "2022-04-01", // roleDefinitions
		"$filter":     "roleName eq '" + roleName + "'",
	}
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions"
		r, _, _ := ApiGet(apiUrl, z, params)
		if r != nil && r["value"] != nil {
			results := r["value"].([]interface{})
			if len(results) == 1 {
				y = results[0].(map[string]interface{}) // Select first, only index entry
				return y                                // We found it
			}
		}
	}
	// If above logic ever finds more than 1, then we have serious issuses, just nil below
	return nil
}

// Gets resource role definition by Object Id. Unfortunately we have to iterate
// through the entire tenant scope hierarchy, which can take time.
func GetResRoleDefinitionById(id string, z *Config) map[string]interface{} {
	scopes := GetAzRbacScopes(z)
	params := map[string]string{"api-version": "2022-04-01"} // roleDefinitions
	for _, scope := range scopes {
		apiUrl := ConstAzUrl + scope + "/providers/Microsoft.Authorization/roleDefinitions/" + id
		r, _, _ := ApiGet(apiUrl, z, params)
		if r != nil && r["id"] != nil {
			return r // Return as soon as we find a match
		}
	}
	return nil
}
