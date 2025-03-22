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
	case RbacDefinition:
		UpsertRbacDefinition(force, obj, z)

	case RbacAssignment:
		CreateRbacAssignment(obj, z)

	// Refocus below target functions, renaming 'FromFile' to simply
	// 'UpsertOBJECT_TYPE()' since above 'GetObjectFromFile' does the specfile
	// object extraction
	case Application, ServicePrincipal:
		//UpsertAppSpFromFile(force bool, specfile string, z *Config)
	case DirectoryGroup:
		//UpsertGroupFromFile(force bool, specfile string, z *Config)

	default:
		utl.Die("Unsupported specfile type. Only resource role definitions and assignment; groups; and AppSP specfiles are allowed.\n")
	}
	os.Exit(0)
}

// Deletes an Azure object by given specfile
func DeleteObjectBySpecfile(force bool, specfile string, z *Config) {
	_, mazType, obj := GetObjectFromFile(specfile)
	switch mazType {
	case RbacDefinition:
		DeleteRbacDefinition(force, obj, z)

	case RbacAssignment:
		DeleteRbacAssignment(force, obj, z)

	// Refocus below target functions, renaming 'ByIdentifier' & 'FromFile' to simply
	// 'DeleteOBJECT_TYPE()' since above 'GetObjectFromFile' does the specfile
	// object extraction
	case Application, ServicePrincipal:
		//DeleteAppSpByIdentifier(force bool, identifier string, z *Config) -- THIS MAY BE FINE
	case DirectoryGroup:
		//DeleteDirObject
	default:
		utl.Die("Unsupported specfile type. Only RBAC role definitions and assignment;" +
			" groups; and AppSP specfiles are allowed.\n")
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
	case RbacDefinition:
		DeleteRbacDefinition(force, targetObj, z)

	case RbacAssignment:
		DeleteRbacAssignmentByFqid(targetId, z)

	case Application, ServicePrincipal:
		DeleteAppSpByIdentifier(force, targetId, z)
	case DirectoryGroup, DirRoleDefinition, DirRoleAssignment:
		err := DeleteDirObject(force, targetId, mazType, z)
		msg := fmt.Sprintf("%v", err)
		utl.Die("%s\n", utl.Red(msg))
	default:
		msg := fmt.Sprintf("Utility does not support deleting %s objects.",
			MazTypeNames[mazType])
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
		case RbacDefinition:
			targetObj := GetAzureRbacDefinitionById(targetId, z)
			DeleteRbacDefinition(force, targetObj, z)
		case Application, ServicePrincipal:
			DeleteAppSpByIdentifier(force, targetId, z)
		case DirectoryGroup, DirRoleDefinition:
			err := DeleteDirObject(force, targetId, mazType, z)
			msg := fmt.Sprintf("%v", err)
			utl.Die("%s\n", utl.Red(msg))
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

	// Get any RBAC role definitions with that name and add them to our growing list
	rbacDefinitions := GetAzureRbacDefinitionsByName(name, z)
	if len(rbacDefinitions) > 0 {
		for i := range rbacDefinitions {
			obj := rbacDefinitions[i]
			id := utl.Str(obj["name"])
			if id != "" {
				idMap[id] = RbacDefinition
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
	case RbacDefinition:
		return GetAzureRbacDefinitionById(id, z)
	case RbacAssignment:
		return GetAzureRbacAssignmentById(id, z)
	case Subscription:
		return GetAzureSubscriptionById(id, z)
	case ManagementGroup:
		return GetAzureMgmtGroupById(id, z)
	case DirectoryUser, DirectoryGroup, Application, ServicePrincipal,
		DirRoleDefinition, DirRoleAssignment:
		return GetObjectFromAzureById(mazType, id, z)
	default:
		return nil
		//return GetObjectFromAzureById(t, id, z)
	}
}

// Gets all Azure RBAC scopes in the tenant's resource hierarchy, starting with the
// Tenant Root Group, then all management groups, and finally all subscription scopes.
func GetAzureRbacScopes(z *Config) (scopes []string) {
	// Collect all resource management groups and subscription RBAC scopes
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
	case RbacDefinition:
		return GetMatchingRbacDefinitions(filter, force, z)
	case RbacAssignment:
		return GetMatchingRbacAssignments(filter, force, z)
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
func GetAzAllPages(apiUrl string, z *Config) (list []interface{}) {
	list = nil
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	for {
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
		resp, statCode, _ = ApiGet(nextLink, z, nil) // Get next batch
	}
	return list
}

// Processes given specfile and returns the format type, the maz object type code, and the object.
func GetObjectFromFile(specfile string) (format, mazType string, obj AzureObject) {
	// Load specfile and capture the raw object, the format, and any error
	rawObj, format, err := utl.LoadFileAuto(specfile)
	if err != nil {
		utl.Die("Error loading specfile %s: %v\n", utl.Yel(specfile), err)
	}
	if format != YamlFormat {
		utl.Die("Error. File %s is not in YAML format\n", utl.Yel(specfile))
	}

	// Attempt to unpack the object
	specfileObj := utl.Map(rawObj)
	if specfileObj == nil {
		utl.Die("Error unpacking the object in specfile %s\n", utl.Yel(specfile))
	}

	obj = AzureObject(specfileObj) // Convert to AzureObject type

	// Determine object type based on properties
	// TODO: Move these to Is* functions like groups and AppSP below, with more checks
	if props := utl.Map(obj["properties"]); props != nil {
		roleName := utl.Str(props["roleName"])
		principalId := utl.Str(props["principalId"])
		if roleName != "" {
			return format, RbacDefinition, obj
		} else if principalId != "" {
			return format, RbacAssignment, obj
		}
	}

	// Check if it's a directory group
	if IsDirectoryGroup(obj) {
		return format, DirectoryGroup, obj
	}

	// Check if it's an App Service Principal
	if IsAppSp(obj) {
		return format, Application, obj
	}

	// If no known type is found, return unknown
	return format, UnknownObject, obj
}

// Compares object in specfile to what is in Azure. This is only for certain mazType objects.
func CompareSpecfileToAzure(specfile string, z *Config) {
	if !utl.FileUsable(specfile) {
		utl.Die("File does not exist, or is zero size\n")
	}
	format, mazType, obj := GetObjectFromFile(specfile)
	if format != JsonFormat && format != YamlFormat {
		utl.Die("File is neither JSON nor YAML\n")
	}
	if obj == nil {
		utl.Die("Invalid map object found in %s specfile.\n", format)
	}

	switch mazType {
	case RbacDefinition:
		roleName, firstScope := ValidateRbacDefinition(obj, z)
		_, azureObj, _ := GetAzureRbacDefinitionByNameAndScope(roleName, firstScope, z)
		if azureObj == nil {
			fmt.Printf("Role %s, as defined in specfile, does %s exist in Azure.\n", utl.Mag(roleName), utl.Red("not"))
		} else {
			fmt.Printf("Role definition in specfile %s exists in Azure:\n", utl.Gre("already"))
			DiffRoleDefinitionSpecfileVsAzure(obj, azureObj)
		}
	case RbacAssignment:
		azureObj := GetRbacAssignmentByObject(obj, z)
		if azureObj == nil {
			fmt.Printf("Role assignment defined in specfile does %s exist in Azure.\n", utl.Red("not"))
		} else {
			fmt.Printf("Role assignment defined in specfile %s exists in Azure:\n", utl.Gre("already"))
			PrintRbacAssignment(azureObj, z)
		}
	case DirectoryGroup:
		displayName := utl.Str(obj["displayName"])
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
		utl.Die("Unsupported %s object type found in %s specfile.\n", utl.Red(mazType), format)
	}
	os.Exit(0)
}
