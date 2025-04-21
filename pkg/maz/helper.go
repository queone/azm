package maz

import (
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/queone/utl"
)

// Creates or updates an Azure object by given specfile
func ApplyObjectBySpecfile(force bool, specfile string, z *Config) {
	if !utl.FileUsable(specfile) {
		utl.Die("Specfile %s is missing or empty\n", utl.Yel(specfile))
	}

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
		onlyFor := fmt.Sprintf("%s, %s, %s, and %s/%s combos",
			utl.Red(ResRoleDefinition), utl.Red(ResRoleAssignment), utl.Red(DirectoryGroup),
			utl.Red(Application), utl.Red(ServicePrincipal))
		utl.Die("The current implementation is only for objects %s, but none of these were "+
			"found in the specfile.\n", onlyFor)
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
		utl.Die("This option is only available for the following object types:\n"+
			"  %s  %s\n  %s  %s\n  %s  %s\n  %s  %s\n  %s  %s\n  %s  %s\n  %s  %s\n",
			utl.Yel(fmt.Sprintf("%-2s", ResRoleDefinition)), MazTypeNames[ResRoleDefinition],
			utl.Yel(fmt.Sprintf("%-2s", ResRoleAssignment)), MazTypeNames[ResRoleAssignment],
			utl.Yel(fmt.Sprintf("%-2s", DirectoryGroup)), MazTypeNames[DirectoryGroup],
			utl.Yel(fmt.Sprintf("%-2s", Application)), MazTypeNames[Application],
			utl.Yel(fmt.Sprintf("%-2s", ServicePrincipal)), MazTypeNames[ServicePrincipal],
			utl.Yel(fmt.Sprintf("%-2s", DirRoleDefinition)), MazTypeNames[DirRoleDefinition],
			utl.Yel(fmt.Sprintf("%-2s", DirRoleAssignment)), MazTypeNames[DirRoleAssignment])
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

// Returns a map of id:mazType objects sharing given name. Only 5 types of
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

// Returns a list of locally cached objects that match the given ID
func FindCachedObjectsById(id string, z *Config) AzureObjectList {
	Logf("Searching cache for object with ID: %s\n", utl.Mag(id))
	start := time.Now()

	// WHAT EXACTLY ARE WE PARALLELIZING?: For each MazType, we're loading its cache and
	// searching for a matching object ID concurrently, to speed up the overall lookup.

	var mu sync.Mutex         // Used to safely append to 'list' from multiple goroutines
	list := AzureObjectList{} // Final list of matching objects
	var wg sync.WaitGroup     // WaitGroup to wait for all goroutines to complete

	for _, mazType := range MazTypes {
		// Capture loop variable locally to avoid closure issues in goroutines.
		// Without this, all goroutines might see the same (last) value of mazType.
		mazType := mazType
		wg.Add(1) // Register one more goroutine with the WaitGroup

		// Start a goroutine to search this type's cache in parallel
		go func() {
			defer wg.Done() // Signal completion of this goroutine

			mazTypeName := MazTypeNames[mazType]

			// Load the cached data for this type
			cache, err := GetCache(mazType, z)
			if err != nil {
				Logf("Error. Could not load %s local cache\n", utl.Mag(mazTypeName))
				return
			}

			count := cache.Count()
			Logf("%s cached object count: %d\n", utl.Mag(mazTypeName), count)

			// If there's anything cached, search for the matching ID
			if count > 0 {
				if existingObj := cache.data.FindById(id); existingObj != nil {
					obj := *existingObj
					Logf("Found object with ID %s of type: %s\n", id, utl.Mag(mazTypeName))
					obj["maz_type"] = mazType // Add the type as an extra field

					// Concurrent writes to shared 'list' require synchronization.
					// The mutex ensures only one goroutine appends at a time,
					// preventing data races or corruption.
					mu.Lock()
					list = append(list, obj)
					mu.Unlock()
				}
			}
		}()
	}

	wg.Wait() // Wait for all goroutines to finish

	microseconds := fmt.Sprintf("%15s", utl.Commafy(time.Since(start).Microseconds()))
	Logf("ID search took %s µs\n", utl.Cya(microseconds))

	return list
}

// Returns a list of Azure objects that match the given ID. Only object types that are
// supported by this maz package are searched.
func FindAzureObjectsById(id string, z *Config) (AzureObjectList, error) {
	// Note that multiple objects may be returned because: 1) A single appId can be shared by
	// both an App and an SP, and 2) although unlikely, UUID collisions can occur, resulting
	// in multiple objects with the same UUID.

	// WHAT EXACTLY ARE WE PARALLELIZING?: For each MazType, we're querying Azure in parallel
	// to find an object with the given ID, since the same ID might exist under different types.

	// Look in the local cache first
	list := FindCachedObjectsById(id, z)
	count := len(list)
	Logf("Found %s cache object with ID %s\n", utl.Mag(count), id)
	if count > 0 {
		return list, nil
	}

	// Fallback to searching directly in Azure
	Logf("Searching Azure for object with ID: %s\n", utl.Mag(id))
	start := time.Now()

	var mu sync.Mutex     // Used to safely append to 'list' from multiple goroutines
	var wg sync.WaitGroup // WaitGroup to wait for all goroutines to complete

	for _, mazType := range MazTypes {
		mazType := mazType // Capture loop variable to avoid race condition inside goroutine
		wg.Add(1)          // Register one more goroutine with the WaitGroup

		// Start a goroutine to query Azure for this type in parallel
		go func() {
			defer wg.Done() // Signal completion of this goroutine

			mazObjectType := utl.Mag(MazTypeNames[mazType])

			// Get object of this type from Azure by ID
			obj := GetAzureObjectById(mazType, id, z)
			if obj == nil {
				Logf("There are no %s objects with this ID\n", mazObjectType)
				return // Exit the goroutine early if no object found
			}

			if objId := ExtractID(obj); objId != "" {
				Logf("ID %s associated with object type: %s\n", objId, mazObjectType)
				obj["maz_type"] = mazType // Extend object with maz_type as an ADDITIONAL field

				// Protect shared slice with a mutex while appending
				mu.Lock()
				list = append(list, obj)
				mu.Unlock()
			}
		}()
	}

	wg.Wait() // Wait for all goroutines to finish

	microseconds := fmt.Sprintf("%15s", utl.Commafy(time.Since(start).Microseconds()))
	Logf("ID search took %s µs\n", utl.Cya(microseconds))

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
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	for {
		if statCode != 200 {
			Logf("HTTP %d: Continuing to try...\n", statCode)
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
			Logf("%v\n", err)
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
		resp, statCode, _ := ApiGet(apiUrl, z, params)
		if statCode != 200 {
			Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
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
	// Resource (ResRoleAssignment) nor directory (DirRoleAssignment) role
	// assignments have name, so this doesn't apply to them

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

// Renames Azure object
func RenameAzureObject(force bool, mazType, currentName, newName string, z *Config) {
	// Missing mazTypes are deliberately unsupported because one, they don't have
	// display names, or simply because renaming them brings on too many complexities.
	switch mazType {
	case ResRoleDefinition:
		RenameResRoleDefinition(force, currentName, newName, z)
	case Application, ServicePrincipal:
		// This renaming is special becase of the relationship between the App and the SP
		RenameAppSp(force, currentName, newName, z)
	case DirectoryGroup, DirRoleDefinition:
		RenameDirObject(force, mazType, currentName, newName, z)
	}
}
