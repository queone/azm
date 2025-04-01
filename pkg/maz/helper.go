package maz

import (
	"fmt"
	"os"
	"path"

	"github.com/queone/utl"
)

// Creates or updates an Azure object by given specfile
func ApplyObjectBySpecfile(force bool, specfile string, z *Config) {
	_, mazType, obj := GetObjectFromFile(specfile)
	switch mazType {
	case ResRoleDefinition:
		UpsertAzureResRoleDefinition(force, obj, z)
	case ResRoleAssignment:
		CreateAzureResRoleAssignment(force, obj, z)
	case Application, ServicePrincipal:
		UpsertAppSp(force, obj, z)
	case DirectoryGroup:
		UpsertGroup(force, obj, z)
	default:
		utl.Die("Option only available for resource role definitions and" +
			" assignments, directory groups, and directory AppSPs. This specfile" +
			" doesn't contain any of those types of objects.\n")
	}
	os.Exit(0)
}

// Deletes an Azure object by given specfile
func DeleteObjectBySpecfile(force bool, specfile string, z *Config) {
	_, mazType, obj := GetObjectFromFile(specfile)
	switch mazType {
	case ResRoleDefinition:
		DeleteResRoleDefinition(force, obj, z)
	case ResRoleAssignment:
		DeleteAzureResRoleAssignment(force, obj, z)
	case Application, ServicePrincipal:
		displayName := utl.Str(obj["displayName"])
		DeleteAppSp(force, displayName, z)
	case DirectoryGroup, DirRoleDefinition, DirRoleAssignment:
		displayName := utl.Str(obj["displayName"])
		DeleteDirObject(force, displayName, mazType, z)
	default:
		utl.Die("Option only available for resource role definitions and" +
			" assignments, directory groups, and directory AppSPs. This specfile" +
			" doesn't contain any of those types of objects.\n")
	}
	os.Exit(0)
}

// Deletes an Azure object by given Id
func DeleteObjectById(force bool, targetId string, z *Config) {
	list, _ := FindAzureObjectsById(targetId, z)

	// Handle the rare case of multiple objects sharing the same ID
	if len(list) > 1 {
		fmt.Println("Found multiple objects with this ID:")
		for _, item := range list {
			id := utl.Str(item["id"])
			mazType := utl.Str(item["maz_type"])
			fmt.Printf("  %-30s  %s\n", MazTypeNames[mazType], id)
		}
		utl.Die("%s\n", utl.Red("Cannot delete by ID. Try deleting by name or specfile instead."))
	}

	if len(list) < 1 {
		utl.Die("%s\n", utl.Red("Cannot find an object with this ID"))
	}

	// Single out the object
	targetObj := AzureObject(list[0])
	mazType := utl.Str(targetObj["maz_type"])
	switch mazType {
	case ResRoleDefinition:
		DeleteResRoleDefinition(force, targetObj, z)
	case ResRoleAssignment:
		DeleteAzureResRoleAssignment(force, targetObj, z)
	case Application, ServicePrincipal:
		DeleteAppSp(force, targetId, z)
	case DirectoryGroup, DirRoleDefinition, DirRoleAssignment:
		DeleteDirObject(force, targetId, mazType, z)
	default:
		msg := fmt.Sprintf("Deleting %s objects by ID is not supported.", MazTypeNames[mazType])
		utl.Die("%s\n", utl.Red(msg))
	}
}

// Deletes an Azure object by name. Only 4 types of objects are supported: resource
// role definitions, App & SP pairs, directory groups and role definitions.
func DeleteObjectByName(force bool, name string, z *Config) {
	idMap := FindAzureObjectsByName(name, z)

	// Handle the case of multiple objects sharing the same name
	if len(idMap) > 1 {
		fmt.Printf("Found multiple objects named %s:\n", utl.Yel(name))
		for id, mazType := range idMap {
			fmt.Printf("  %-38s  %s\n", id, MazTypeNames[mazType])
		}
		utl.Die("%s\n", utl.Red("Cannot delete by name. Try deleting by ID or specfile instead."))
	}

	if len(idMap) < 1 {
		utl.Die("Could not find an object named %s\n", utl.Red(name))
	}

	// Process for the single object with this name
	for targetId, mazType := range idMap {
		switch mazType {
		case ResRoleDefinition:
			targetObj := GetAzureResRoleDefinitionById(targetId, z)
			DeleteResRoleDefinition(force, targetObj, z)
		case Application, ServicePrincipal:
			DeleteAppSp(force, targetId, z)
		case DirectoryGroup, DirRoleDefinition:
			DeleteDirObject(force, targetId, mazType, z)
		default:
			msg := fmt.Sprintf("Utility does not support deleting %s objects by name.",
				MazTypeNames[mazType])
			utl.Die("%s\n", utl.Red(msg))
		}
	}
}

// Returns a map of id:mazType objects sharing given name. Only 4 types of
// objects are supported: resource role definitions, App & SP pairs, directory
// groups and role definitions.
func FindAzureObjectsByName(name string, z *Config) map[string]string {
	// Set up the map to collect the set id:mazType objects that share this name
	idMap := make(map[string]string)

	// Get any resource role definitions with that name and add them to our growing list
	rbacDefinitions := GetAzureResRoleDefinitionsByName(name, z)
	if len(rbacDefinitions) > 0 {
		for i := range rbacDefinitions {
			obj := rbacDefinitions[i]
			id := utl.Str(obj["name"])
			if id != "" {
				idMap[id] = ResRoleDefinition
			}
		}
	}
	// Get any other supported object with that name and add them to our growing list
	for _, mazType := range []string{Application, ServicePrincipal, DirectoryGroup, DirRoleDefinition} {
		matchingSet := GetObjectFromAzureByName(mazType, name, z)
		if len(matchingSet) > 0 {
			for i := range matchingSet {
				obj := matchingSet[i]
				id := utl.Str(obj["id"])
				if id != "" {
					idMap[id] = mazType
				}
			}
		}
	}
	return idMap
}

// Returns a list of Azure objects that match the given ID. Only object types that are
// supported by this maz package are searched.
func FindAzureObjectsById(id string, z *Config) (AzureObjectList, error) {
	// Note that multiple objects may be returned because: 1) A single appId can be shared by
	// both an App and an SP, 2) Although unlikely, UUID collisions can occur, resulting in
	// multiple objects with the same UUID.

	// Focus on the last element, in case it's a fully-qualified long ID
	uuid := path.Base(id)
	if !utl.ValidUuid(uuid) {
		return nil, fmt.Errorf("invalid id %s", id)
	}

	list := AzureObjectList{}
	for _, mazType := range MazTypes {
		obj := GetAzureObjectById(mazType, id, z)
		if obj != nil && obj["id"] != nil { // Valid objects have an 'id' attribute
			// Found one of these types with this ID
			obj["maz_type"] = mazType // Extend object with maz_type as an ADDITIONAL field
			list = append(list, obj)
		}
	}

	return list, nil
}

// Retrieves Azure object by mazType and object ID
func GetAzureObjectById(mazType, id string, z *Config) AzureObject {
	switch mazType {
	case ResRoleDefinition:
		return GetAzureResRoleDefinitionById(id, z)
	case ResRoleAssignment:
		return GetAzureResRoleAssignmentById(id, z)
	case Subscription:
		return GetAzureSubscriptionById(id, z)
	case ManagementGroup:
		return GetAzureMgmtGroupById(id, z)
	case DirectoryUser, DirectoryGroup, Application, ServicePrincipal,
		DirRoleDefinition, DirRoleAssignment:
		return GetObjectFromAzureById(mazType, id, z)
	default:
		return nil
	}
}

// Gets all Azure resource scopes in the tenant's resource hierarchy, starting with the
// Tenant Root Group, then all management groups, and finally all subscription scopes.
func GetAzureResRoleScopes(z *Config) (scopes []string) {
	// Collect all resource management groups and subscription resource scopes
	scopes = nil
	mgmtGroupIds := GetAzureMgmtGroupsIds(z) // Includes the Tenant Root Group
	scopes = append(scopes, mgmtGroupIds...)
	subIds := GetAzureSubscriptionsIds(z)
	scopes = append(scopes, subIds...)

	// Retrieving scopes that are each subscription do not appear neccessary. Most
	// API list search functions appear to be pulling all objects in lower scopes
	// also. If in the future we discover that we drilling down further into those
	// other scopes, then we will need to add that here.

	return scopes
}

// Generic querying function to get Azure objects of any mazType, whose attributes
// match on given filter string. If the filter is the "" empty string, return ALL
// of the objects of this particular type. Works accross MS Graph and ARM objects.
func GetMatchingObjects(mazType, filter string, force bool, z *Config) AzureObjectList {
	switch mazType {
	case ResRoleDefinition:
		return GetMatchingResRoleDefinitions(filter, force, z)
	case ResRoleAssignment:
		return GetMatchingResRoleAssignments(filter, force, z)
	case Subscription:
		return GetMatchingAzureSubscriptions(filter, force, z)
	case ManagementGroup:
		return GetMatchingAzureMgmtGroups(filter, force, z)
	case DirectoryUser, DirectoryGroup, Application,
		ServicePrincipal, DirRoleDefinition, DirRoleAssignment:
		return GetMatchingDirObjects(mazType, filter, force, z)
	}
	return nil
}

// Returns all Azure pages for given API URL call
func GetAzureAllPages(apiUrl string, z *Config) (list []interface{}) {
	list = nil
	var err error
	resp, statCode, err := ApiGet(apiUrl, z, nil)
	for {
		if err != nil {
			Log("%v\n", err)
		}
		if statCode != 200 {
			msg := fmt.Sprintf("%sHTTP %d: Continuing to try...", rUp, statCode)
			fmt.Printf("%s", utl.Yel(msg))
		}
		// Forever loop until there are no more pages
		thisBatch := utl.Slice(resp["value"])
		if len(thisBatch) > 0 {
			list = append(list, thisBatch...) // Continue growing list
		}
		nextLink := utl.Str(resp["@odata.nextLink"])
		if nextLink == "" {
			break // Break once there is no more pages
		}
		resp, statCode, err = ApiGet(nextLink, z, nil) // Get next batch
	}
	return list
}

// Processes given specfile and returns the format type, the mazType, and the object.
func GetObjectFromFile(specfile string) (format, mazType string, obj AzureObject) {
	// Load specfile and capture the raw object, the format, and any error
	rawObj, format, err := utl.LoadFileAuto(specfile)
	if err != nil {
		utl.Die("Error loading specfile %s: %v\n", utl.Yel(specfile), err)
	}
	if format != YamlFormat && format != JsonFormat {
		utl.Die("Error. File %s is not in YAML format\n", utl.Yel(specfile))
	}

	// Attempt to unpack the object
	specfileObj := utl.Map(rawObj)
	if specfileObj == nil {
		utl.Die("Error unpacking the object in specfile %s\n", utl.Yel(specfile))
	}

	obj = AzureObject(specfileObj) // Cast to our standard AzureObject type

	// Determine object type
	if IsResRoleDefinition(obj) {
		return format, ResRoleDefinition, obj
	}
	if IsResRoleAssignment(obj) {
		return format, ResRoleAssignment, obj
	}
	if IsDirGroup(obj) {
		return format, DirectoryGroup, obj
	}
	if IsDirAppSp(obj) {
		return format, Application, obj
	}
	return format, UnknownObject, obj
}

// Compares object in specfile to what is in Azure. This is only for certain mazType objects.
func CompareSpecfileToAzure(specfile string, z *Config) {
	if !utl.FileUsable(specfile) {
		utl.Die("That specfile doesn't exist, or is zero size\n")
	}
	format, mazType, obj := GetObjectFromFile(specfile)
	if format != YamlFormat {
		utl.Die("Specfile is not in YAML format\n")
	}
	if obj == nil {
		utl.Die("Object in specfile is not valid\n")
	}

	switch mazType {
	case ResRoleDefinition:
		roleName, firstScope := ValidateResRoleDefinitionObject(obj, z)
		_, azureObj := GetAzureResRoleDefinitionByScopeAndName(firstScope, roleName, z)
		if azureObj == nil {
			fmt.Printf("Role %s, as defined in specfile, does %s exist in Azure.\n", utl.Mag(roleName), utl.Red("not"))
		} else {
			fmt.Printf("Role definition in specfile %s exists in Azure:\n", utl.Gre("already"))
			DiffRoleDefinitionSpecfileVsAzure(obj, azureObj)
		}
	case ResRoleAssignment:
		roleDefinitionId, principalId, scope := ValidateResRoleAssignmentObject(obj, z)
		_, azureObj := GetAzureResRoleAssignmentBy3Args(roleDefinitionId, principalId, scope, z)
		if azureObj == nil {
			fmt.Printf("Role assignment defined in specfile does %s exist in Azure.\n", utl.Red("not"))
		} else {
			fmt.Printf("Role assignment defined in specfile %s exists in Azure:\n", utl.Gre("already"))
			PrintResRoleAssignment(azureObj, z)
		}
	case DirectoryGroup, Application, ServicePrincipal:
		// Above call to GetObjectFromFile() guarantees below exists
		displayName := utl.Str(obj["displayName"])

		azureObj := GetObjectFromAzureByName(mazType, displayName, z)
		// Note that there could be more than one object with same name
		count := len(azureObj)
		if count < 1 {
			fmt.Printf("The %s defined in specfile does %s exist in Azure.\n",
				MazTypeNames[mazType], utl.Red("not"))
		} else if count > 1 {
			fmt.Printf("Found multiple %s named %s. Cannot continue.\n",
				MazTypeNames[mazType], utl.Red(displayName))
		} else {
			fmt.Printf("The %s defined in specfile %s exists in Azure:\n",
				MazTypeNames[mazType], utl.Gre("already"))
			PrintObject(mazType, azureObj[0], z)
		}
	default:
		utl.Die("This is a %s (%s) specfile, which is not currently supported.\n",
			MazTypeNames[mazType], utl.Red(mazType))
	}
	os.Exit(0)
}

// Retrieves Azure object display name, given its mazType and ID
func GetObjectNameFromId(mazType, targetId string, z *Config) string {
	// This doesn't apply to ResRoleAssignment nor DirRoleAssignment

	switch mazType {
	case ResRoleDefinition:
		obj := GetAzureResObjectById(mazType, targetId, z)
		if obj != nil {
			if props := utl.Map(obj["properties"]); props != nil {
				return utl.Str(props["roleName"])
			}
		}
	case Subscription:
		obj := GetAzureResObjectById(mazType, targetId, z)
		if obj != nil {
			return utl.Str(obj["displayName"])
		}
	case ManagementGroup:
		obj := GetAzureResObjectById(mazType, targetId, z)
		if obj != nil {
			if props := utl.Map(obj["properties"]); props != nil {
				return utl.Str(props["displayName"])
			}
		}
	case DirectoryUser, DirectoryGroup, Application, ServicePrincipal, DirRoleDefinition:
		z.AddMgHeader("ConsistencyLevel", "eventual")
		apiUrl := ConstMgUrl + ApiEndpoint[mazType] + "/" + targetId
		var err error
		resp, _, err := ApiGet(apiUrl, z, nil)
		if err != nil {
			Log("%v\n", err)
		}
		if obj := utl.Map(resp); obj != nil {
			return utl.Str(obj["displayName"])
		}
	}
	return ""
}

// Retrieves Azure object ID, given its mazType and name
func GetObjectIdFromName(mazType, targetName string, z *Config) string {
	// This doesn't apply to ResRoleAssignment nor DirRoleAssignment

	switch mazType {
	case ResRoleDefinition, ManagementGroup:
		obj := GetAzureResObjectByName(mazType, targetName, z)
		if obj != nil {
			return utl.Str(obj["name"])
		}
	case Subscription:
		obj := GetAzureResObjectByName(mazType, targetName, z)
		if obj != nil {
			return utl.Str(obj["subscriptionId"])
		}
	case DirectoryUser, DirectoryGroup, Application, ServicePrincipal, DirRoleDefinition:
		z.AddMgHeader("ConsistencyLevel", "eventual")
		apiUrl := ConstMgUrl + ApiEndpoint[mazType]
		params := map[string]string{
			"$filter": fmt.Sprintf("displayName eq '%s'", targetName),
			"$top":    "1",
		}
		var err error
		resp, _, err := ApiGet(apiUrl, z, params)
		if err != nil {
			Log("%v\n", err)
		}
		if list := utl.Slice(resp["value"]); list != nil {
			if obj := utl.Map(list[0]); obj != nil {
				if id := utl.Str(obj["id"]); id != "" {
					return id
				}
			}
		}
	}
	return ""
}

// Returns an id:name map of the given object type.
func GetIdNameMap(mazType string, z *Config) map[string]string {
	// This doesn't apply to ResRoleAssignment nor DirRoleAssignment

	var dirObjects AzureObjectList
	idNameMap := make(map[string]string)

	// Note false = get from cache. We optimize speed for accuracy.
	switch mazType {
	case ResRoleDefinition:
		dirObjects = GetMatchingResRoleDefinitions("", false, z)
	case Subscription:
		dirObjects = GetMatchingAzureSubscriptions("", false, z)
	case ManagementGroup:
		dirObjects = GetMatchingAzureMgmtGroups("", false, z)
	case DirectoryUser, DirectoryGroup, Application, ServicePrincipal, DirRoleDefinition:
		dirObjects = GetMatchingDirObjects(mazType, "", false, z)
	default:
		return nil
	}
	for i := range dirObjects {
		obj := dirObjects[i]
		// We do Base() to cover subscriptions, mgmt groups, and role definitions
		id := path.Base(utl.Str(obj["id"]))
		if id != "" {
			var name string
			if displayName := utl.Str(obj["displayName"]); displayName != "" {
				name = displayName
			} else if props := utl.Map(obj["properties"]); props != nil {
				if displayName := utl.Str(props["displayName"]); displayName != "" {
					name = displayName
				} else if roleName := utl.Str(props["roleName"]); roleName != "" {
					name = roleName
				}
			}
			if name != "" {
				idNameMap[id] = name
			}
		}
	}
	return idNameMap
}
