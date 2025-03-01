package maz

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/queone/utl"
)

// Creates or updates an Azure object based on given specfile
func UpsertAzObject(force bool, filePath string, z *Config) {
	if utl.FileNotExist(filePath) || utl.FileSize(filePath) < 1 {
		utl.Die("File does not exist, or it is zero size\n")
	}
	formatType, t, x := GetObjectFromFile(filePath)
	if formatType != "JSON" && formatType != "YAML" {
		utl.Die("File is not in JSON nor YAML format\n")
	}
	if t != "d" && t != "a" {
		utl.Die("File is not a role definition nor an assignment specfile\n")
	}
	switch t {
	case "d":
		UpsertAzRoleDefinition(force, x, z)
	case "a":
		CreateAzRoleAssignment(x, z)
	}
	os.Exit(0)
}

// Deletes object based on string specifier (currently only supports roleDefinitions or Assignments)
// String specifier can be either of 3: UUID, specfile, or displaName (only for roleDefinition)
// 1) Search Azure by given identifier; 2) Grab object's Fully Qualified Id string;
// 3) Print and prompt for confirmation; 4) Delete or abort
func DeleteAzObject(force bool, specifier string, z *Config) {
	if utl.ValidUuid(specifier) {
		list := FindAzObjectsById(specifier, z) // Get all objects that may match this UUID, hopefully just one
		if len(list) > 1 {
			utl.Die("%s\n", utl.Red("UUID collision? Run utility with UUID argument to see the list."))
		}
		if len(list) < 1 {
			utl.Die("Object does not exist.\n")
		}
		y := list[0].(map[string]interface{}) // Single out the only object
		if y != nil && y["mazType"] != nil {
			t := utl.Str(y["mazType"])
			fqid := utl.Str(y["id"]) // Grab fully qualified object Id
			PrintObject(t, y, z)
			if !force {
				if utl.PromptMsg("DELETE above? y/n ") != 'y' {
					utl.Die("Aborted.\n")
				}
			}
			switch t {
			case "d":
				DeleteAzRoleDefinitionByFqid(fqid, z)
			case "a":
				DeleteAzRoleAssignmentByFqid(fqid, z)
			}
		}
	} else if utl.FileExist(specifier) {
		// Delete object defined in specfile
		formatType, t, x := GetObjectFromFile(specifier) // x is for the object in Specfile
		if formatType != "JSON" && formatType != "YAML" {
			utl.Die("File is not in JSON nor YAML format\n")
		}
		var y map[string]interface{} = nil
		switch t {
		case "d":
			y = GetAzRoleDefinitionByObject(x, z) // y is for the object from Azure
			fqid := utl.Str(y["id"])              // Grab fully qualified object Id
			if y == nil {
				utl.Die("Role definition does not exist.\n")
			} else {
				PrintRoleDefinition(y, z) // Use specific role def print function
				if !force {
					if utl.PromptMsg("DELETE above? y/n ") != 'y' {
						utl.Die("Aborted.\n")
					}
				}
				DeleteAzRoleDefinitionByFqid(fqid, z)
			}
		case "a":
			y = GetAzRoleAssignmentByObject(x, z)
			fqid := utl.Str(y["id"]) // Grab fully qualified object Id
			if y == nil {
				utl.Die("Role assignment does not exist.\n")
			} else {
				PrintRoleAssignment(y, z) // Use specific role assgmnt print function
				if !force {
					if utl.PromptMsg("DELETE above? y/n ") != 'y' {
						utl.Die("Aborted.\n")
					}
				}
				DeleteAzRoleAssignmentByFqid(fqid, z)
			}
		default:
			utl.Die("%s specfile is neither an RBAC role definition or assignment.\n", formatType)
		}
	} else {
		// Delete role definition by its displayName, if it exists. This only applies to definitions
		// since assignments do not have a displayName attribute. Also, other objects are not supported.
		y := GetAzRoleDefinitionByName(specifier, z)
		if y == nil {
			utl.Die("Role definition does not exist.\n")
		}
		fqid := utl.Str(y["id"]) // Grab fully qualified object Id
		PrintRoleDefinition(y, z)
		if !force {
			if utl.PromptMsg("DELETE above? y/n ") != 'y' {
				utl.Die("Aborted.\n")
			}
		}
		DeleteAzRoleDefinitionByFqid(fqid, z)
	}
}

// Returns a list of Azure objects that match the given UUID. Note that multiple
// objects may be returned because:
// 1. A single appId can be shared by both an application and a service principal.
// 2. Although unlikely, UUID collisions can occur, resulting in multiple objects
// with the same UUID.
// This function only searches for objects of the Azure types supported by the maz package.
func FindAzObjectsById(id string, z *Config) (list []interface{}) {
	list = nil
	for _, t := range mazTypes {
		x := GetAzObjectById(t, id, z)
		if x != nil && x["id"] != nil { // Valid objects have an 'id' attribute
			// Found one of these types with this UUID
			x["mazType"] = t // Extend object with mazType as an ADDITIONAL field
			list = append(list, x)
		}
	}
	return list
}

// Retrieves Azure object by Object UUID
func GetAzObjectById(t, id string, z *Config) (x map[string]interface{}) {
	switch t {
	case "d":
		return GetAzRoleDefinitionById(id, z)
	case "a":
		return GetAzRoleAssignmentById(id, z)
	case "s":
		return GetAzSubscriptionById(id, z)
	case "u":
		return GetObjectFromAzureById(t, id, z)
	case "g":
		return GetObjectFromAzureById(t, id, z)
	case "ap":
		return GetObjectFromAzureById(t, id, z)
	case "sp":
		return GetObjectFromAzureById(t, id, z)
	case "dr":
		return GetObjectFromAzureById(t, id, z)
	}
	return nil
}

// Gets all scopes in the Azure tenant RBAC hierarchy: Tenant Root Group and all
// management groups, plus all subscription scopes
func GetAzRbacScopes(z *Config) (scopes []string) {
	scopes = nil
	managementGroups := GetAzMgGroups(z) // Start by adding all the managementGroups scopes
	for _, i := range managementGroups {
		x := i.(map[string]interface{})
		scopes = append(scopes, utl.Str(x["id"]))
	}
	subIds := GetAzSubscriptionsIds(z) // Now add all the subscription scopes
	scopes = append(scopes, subIds...)

	// SCOPES below subscriptions do not appear to be REALLY NEEDED. Most list
	// search functions pull all objects in lower scopes. If there is a future
	// need to keep drilling down, next level being Resource Group scopes, then
	// they can be acquired with something like below:

	// params := map[string]string{"api-version": "2021-04-01"} // resourceGroups
	// for subId := range subIds {
	// 	apiUrl := ConstAzUrl + subId + "/resourcegroups"
	// 	r, _, _ := ApiGet(apiUrl, z, params)
	// 	if r != nil && r["value"] != nil {
	// 		resourceGroups := r["value"].([]interface{})
	// 		for _, j := range resourceGroups {
	// 			y := j.(map[string]interface{})
	// 			rgId := utl.Str(y["id"])
	// 			scopes = append(scopes, rgId)
	// 		}
	// 	}
	// }
	// // Then repeat for next leval scope ...

	return scopes
}

// Retrieves locally cached list of objects in given cache file
func GetCachedObjects(cacheFile string) (cachedList []interface{}) {
	cachedList = nil
	if utl.FileUsable(cacheFile) {
		rawList, _ := utl.LoadFileJson(cacheFile, false)
		if rawList != nil {
			cachedList = rawList.([]interface{})
		}
	}
	return cachedList
}

// Generic function to get objects of type t whose attributes match on filter.
// If filter is the "" empty string return ALL of the objects of this type.
func GetObjects(t, filter string, force bool, z *Config) (list []interface{}) {
	switch t {
	case "d":
		return GetMatchingRoleDefinitions(filter, force, z)
	case "a":
		return GetMatchingRoleAssignments(filter, force, z)
	case "m":
		return GetMatchingMgGroups(filter, force, z)
	case "s":
		return GetMatchingSubscriptions(filter, force, z)
	}
	return nil
}

// Returns all Azure pages for given API URL call
func GetAzAllPages(apiUrl string, z *Config) (list []interface{}) {
	list = nil
	r, _, _ := ApiGet(apiUrl, z, nil)
	for {
		// Forever loop until there are no more pages
		var thisBatch []interface{} = nil // Assume zero entries in this batch
		if r["value"] != nil {
			thisBatch = r["value"].([]interface{})
			if len(thisBatch) > 0 {
				list = append(list, thisBatch...) // Continue growing list
			}
		}
		nextLink := utl.Str(r["@odata.nextLink"])
		if nextLink == "" {
			break // Break once there is no more pages
		}
		r, _, _ = ApiGet(nextLink, z, nil) // Get next batch
	}
	return list
}

func GetAzObjects(apiUrl string, z *Config, verbose bool) (deltaSet []interface{}, deltaLinkMap map[string]interface{}) {
	// To be replaced by FetchAzureObjectsDelta()
	k := 1 // Track number of API calls
	r, _, _ := ApiGet(apiUrl, z, nil)
	for {
		// Infinite for-loop until deltaLink appears (meaning we're done getting current delta set)
		var thisBatch []interface{} = nil // Assume zero entries in this batch
		var objCount int = 0
		if r["value"] != nil {
			thisBatch = r["value"].([]interface{})
			objCount = len(thisBatch)
			if objCount > 0 {
				deltaSet = append(deltaSet, thisBatch...) // Continue growing deltaSet
			}
		}
		if verbose {
			// Progress count indicator. Using global var rUp to overwrite last line. Defer newline until done
			fmt.Printf("%sCall %05d : count %05d", rUp, k, objCount)
		}
		if r["@odata.deltaLink"] != nil {
			deltaLinkMap := map[string]interface{}{
				"@odata.deltaLink": utl.Str(r["@odata.deltaLink"]),
			}
			if verbose {
				fmt.Print(rUp) // Go up to overwrite progress line
			}
			return deltaSet, deltaLinkMap // Return immediately after deltaLink appears
		}
		r, _, _ = ApiGet(utl.Str(r["@odata.nextLink"]), z, nil) // Get next batch
		k++
	}
}

// Removes specified cache file
func RemoveCacheFile(t string, z *Config) {
	// Takes global pointer z
	switch t {
	case "id":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.CredsFile))
	case "t":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TokenFile))
	case "d":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_roleDefinitions."+ConstCacheFileExtension))
	case "a":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_roleAssignments."+ConstCacheFileExtension))
	case "s":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_subscriptions."+ConstCacheFileExtension))
	case "m":
		utl.RemoveFile(filepath.Join(z.ConfDir, z.TenantId+"_managementGroups."+ConstCacheFileExtension))
	case "all":
		// See https://stackoverflow.com/questions/48072236/remove-files-with-wildcard
		fileList, err := filepath.Glob(filepath.Join(z.ConfDir, z.TenantId+"_*."+ConstCacheFileExtension))
		if err != nil {
			panic(err)
		}
		for _, filePath := range fileList {
			utl.RemoveFile(filePath)
		}
	}
}

// Processes given specfile and returns the specfile format type, the maz object letter
// string type, and the actual object.
func GetObjectFromFile(filePath string) (formatType, t string, specfileObj map[string]interface{}) {
	var objRaw interface{}

	// Validate JSON first
	err := utl.ValidateJson(filePath)
	if err == nil {
		objRaw, _ = utl.LoadFileJson(filePath, false)
		formatType = "JSON"
	} else {
		// Fallback to YAML if JSON validation fails
		err = utl.ValidateYaml(filePath)
		if err == nil {
			objRaw, _ = utl.LoadFileYaml(filePath)
			formatType = "YAML"
		} else {
			// Neither JSON nor YAML
			return "", "", nil
		}
	}

	// Continue unpacking the object to see what it is
	specfileObj, ok := objRaw.(map[string]interface{})
	if !ok {
		return formatType, "", nil // Not a valid map object
	}

	// Check if it's an RBAC role definition or assignment
	xProp, hasProperties := specfileObj["properties"].(map[string]interface{})
	if hasProperties {
		roleName := utl.Str(xProp["roleName"])       // Assert and assume it's a definition
		roleId := utl.Str(xProp["roleDefinitionId"]) // Assert and assume it's an assignment

		if roleName != "" {
			return formatType, "d", specfileObj // Role definition
		} else if roleId != "" {
			return formatType, "a", specfileObj // Role assignment
		}
	}

	// Check if it's an Azure directory group
	if specfileObj["displayName"] != nil && specfileObj["mailEnabled"] != nil &&
		specfileObj["mailNickname"] != nil && specfileObj["securityEnabled"] != nil {
		return formatType, "g", specfileObj // Directory group
	}

	// Check if it's an Azure AppSP pair
	if specfileObj["displayName"] != nil && specfileObj["signInAudience"] != nil {
		return formatType, "ap", specfileObj // AppSP pair
	}

	// Additional object type checks (extend here as needed) ...

	return formatType, "", specfileObj // Return unknown object type
}

// Compares object in specfile to what is in Azure
func CompareSpecfileToAzure(filePath string, z *Config) {
	if utl.FileNotExist(filePath) || utl.FileSize(filePath) < 1 {
		utl.Die("File does not exist, or is zero size\n")
	}
	formatType, t, specfileObj := GetObjectFromFile(filePath)
	if formatType != "JSON" && formatType != "YAML" {
		utl.Die("File is neither JSON nor YAML\n")
	}
	if specfileObj == nil {
		utl.Die("Invalid map object found in %s specfile.\n", formatType)
	}

	switch t {
	case "d":
		azureObj := GetAzRoleDefinitionByObject(specfileObj, z)
		if azureObj == nil {
			fileProp := specfileObj["properties"].(map[string]interface{})
			fileRoleName := utl.Str(fileProp["roleName"])
			fmt.Printf("Role '%s', as defined in specfile, does %s exist in Azure.\n", utl.Mag(fileRoleName), utl.Red("not"))
		} else {
			fmt.Printf("Role definition in specfile %s exists in Azure:\n", utl.Gre("already"))
			DiffRoleDefinitionSpecfileVsAzure(specfileObj, azureObj, z)
		}
	case "a":
		azureObj := GetAzRoleAssignmentByObject(specfileObj, z)
		if azureObj == nil {
			fmt.Printf("Role assignment defined in specfile does %s exist in Azure.\n", utl.Red("not"))
		} else {
			fmt.Printf("Role assignment defined in specfile %s exists in Azure:\n", utl.Gre("already"))
			PrintRoleAssignment(azureObj, z)
		}
	case "g":
		displayName := utl.Str(specfileObj["displayName"])
		if displayName == "" {
			utl.Die("Specfile object is missing %s. Cannot continue.\n", utl.Red("displayName"))
		}
		azureObj := GetObjectFromAzureByName("g", displayName, z)
		// Note that there could be more than one group with same name
		count := len(azureObj)
		if count < 1 {
			fmt.Printf("Group defined in specfile does %s exist in Azure.\n", utl.Red("not"))
		} else if count > 1 {
			fmt.Printf("Found multiple Azure directory groups named %s. Cannot continue.\n", utl.Red(displayName))
		} else {
			fmt.Printf("Group defined in specfile %s exists in Azure:\n", utl.Gre("already"))
			PrintGroup(azureObj[0], z)
		}
	default:
		utl.Die("Unsupported %s object type found in %s specfile.\n", utl.Red(t), formatType)
	}
	os.Exit(0)
}
