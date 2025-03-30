package maz

import (
	"fmt"
	"path"
	"time"

	"github.com/queone/utl"
)

// Prints a status count of all AZ and MG objects that are in Azure, and the local files.
func PrintCountStatus(z *Config) {
	c1Width := 44 // Column 1 width
	c2Width := 10 // Column 2 width
	c3Width := 10 // Column 3 width
	fmt.Printf("%s\n", utl.Gra("# Please note that enumerating some Azure resources can be slow"))
	fmt.Print(utl.Whi2(utl.PostSpc("Objects", c1Width)+
		utl.PreSpc("Local", c2Width)+
		utl.PreSpc("Azure", c3Width)) + "\n")
	status := utl.Blu(utl.PostSpc("Directory users", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal(DirectoryUser, z), c2Width))
	status += utl.Gre(utl.PreSpc(ObjectCountAzure(DirectoryUser, z), c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Directory groups", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal(DirectoryGroup, z), c2Width))
	status += utl.Gre(utl.PreSpc(ObjectCountAzure(DirectoryGroup, z), c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Directory applications", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal(Application, z), c2Width))
	status += utl.Gre(utl.PreSpc(ObjectCountAzure(Application, z), c3Width)) + "\n"
	nativeSpsLocal, msSpsLocal := SpsCountLocal(z)
	nativeSpsAzure, msSpsAzure := SpsCountAzure(z)
	status += utl.Blu(utl.PostSpc("Directory service principals (this tenant)", c1Width))
	status += utl.Gre(utl.PreSpc(nativeSpsLocal, c2Width))
	status += utl.Gre(utl.PreSpc(nativeSpsAzure, c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Directory pervice principals (multi-tenant)", c1Width))
	status += utl.Gre(utl.PreSpc(msSpsLocal, c2Width))
	status += utl.Gre(utl.PreSpc(msSpsAzure, c3Width)) + "\n"

	// Note: ObjectCountAzure() doesn't support dr nor da objects so we just count
	// the ones in the local cache and print them for local *and* Azure
	status += utl.Blu(utl.PostSpc("Directory role definitions", c1Width))
	drCount := ObjectCountLocal(DirRoleDefinition, z)
	status += utl.Gre(utl.PreSpc(drCount, c2Width))
	status += utl.Gre(utl.PreSpc(drCount, c3Width)) + "\n"
	daCount := ObjectCountLocal(DirRoleAssignment, z)
	status += utl.Blu(utl.PostSpc("Directory role assignments", c1Width))
	status += utl.Gre(utl.PreSpc(daCount, c2Width))
	status += utl.Gre(utl.PreSpc(daCount, c3Width)) + "\n"

	status += utl.Blu(utl.PostSpc("Resource management groups", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal(ManagementGroup, z), c2Width))
	status += utl.Gre(utl.PreSpc(CountAzureMgmtGroups(z), c3Width)) + "\n"

	status += utl.Blu(utl.PostSpc("Resource subscriptions", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal(Subscription, z), c2Width))
	status += utl.Gre(utl.PreSpc(CountAzureSubscriptions(z), c3Width)) + "\n"

	customLocal, builtinLocal := CountResRoleDefinitions(false, z) // false = get from cache, not Azure
	customAzure, builtinAzure := CountResRoleDefinitions(true, z)  // true = get from Azure, not cache
	status += utl.Blu(utl.PostSpc("Resource role definitions (built-in)", c1Width))
	status += utl.Gre(utl.PreSpc(builtinLocal, c2Width))
	status += utl.Gre(utl.PreSpc(builtinAzure, c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Resource role definitions (custom)", c1Width))
	status += utl.Gre(utl.PreSpc(customLocal, c2Width))
	status += utl.Gre(utl.PreSpc(customAzure, c3Width)) + "\n"

	status += utl.Blu(utl.PostSpc("Resource role assignments", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal(ResRoleAssignment, z), c2Width))
	status += utl.Gre(utl.PreSpc(RoleAssignmentsCountAzure(z), c3Width)) + "\n"

	fmt.Print(status)
}

// Prints this single object of type mazType tersely, with minimal attributes
func PrintTersely(mazType string, obj AzureObject) {
	switch mazType {
	case ResRoleDefinition:
		if props := utl.Map(obj["properties"]); props != nil {
			fmt.Printf("%s  %-60s  %s\n", utl.Str(obj["name"]),
				utl.Str(props["roleName"]), utl.Str(props["type"]))
		}
	case ResRoleAssignment:
		if props := utl.Map(obj["properties"]); props != nil {
			rdId := path.Base(utl.Str(props["roleDefinitionId"]))
			principalId := utl.Str(props["principalId"])
			principalType := utl.Str(props["principalType"])
			scope := utl.Str(props["scope"])
			fmt.Printf("%s  %s  %s %-20s %s\n", utl.Str(obj["name"]),
				rdId, principalId, "("+principalType+")", scope)
		}
	case Subscription:
		fmt.Printf("%s  %-10s  %s\n", utl.Str(obj["subscriptionId"]),
			utl.Str(obj["state"]), utl.Str(obj["displayName"]))
	case ManagementGroup:
		id := utl.Str(obj["name"])
		if props := utl.Map(obj["properties"]); props != nil {
			displayName := utl.Str(props["displayName"])
			fmt.Printf("%-38s  %s\n", id, displayName)
		}
	case DirectoryUser:
		upn := utl.Str(obj["userPrincipalName"])
		onPremName := utl.Str(obj["onPremisesSamAccountName"])
		fmt.Printf("%s  %-50s %-18s %s\n", utl.Str(obj["id"]), upn,
			onPremName, utl.Str(obj["displayName"]))
	case DirectoryGroup:
		fmt.Printf("%s  %s\n", utl.Str(obj["id"]), utl.Str(obj["displayName"]))
	case Application, ServicePrincipal:
		fmt.Printf("%s  %-66s %s\n", utl.Str(obj["id"]), utl.Str(obj["displayName"]),
			utl.Str(obj["appId"]))
	case DirRoleDefinition:
		builtIn := "Custom"
		if utl.Str(obj["isBuiltIn"]) == "true" {
			builtIn = "BuiltIn"
		}
		enabled := "Disabled"
		if utl.Str(obj["isEnabled"]) == "true" {
			enabled = "Enabled"
		}
		fmt.Printf("%s  %-60s  %-10s  %s\n", utl.Str(obj["id"]),
			utl.Str(obj["displayName"]), builtIn, enabled)
	case DirRoleAssignment:
		scope := utl.Str(obj["directoryScopeId"])
		principalId := utl.Str(obj["principalId"])
		roleDefId := utl.Str(obj["roleDefinitionId"])
		fmt.Printf("%-66s  %-37s  %-36s  %s\n", utl.Str(obj["id"]), scope,
			principalId, roleDefId)
	}
}

// Prints object by given ID
func PrintObjectById(id string, z *Config) {
	list, err := FindAzureObjectsById(id, z) // Search for this ID under all maz objects types
	if err != nil {
		utl.Die("Error: %v\n", err)
	}

	for _, obj := range list {
		mazType := utl.Str(obj["maz_type"]) // Function FindAzureObjectsById() should have added this field
		if mazType != "" {
			PrintObject(mazType, obj, z)
		} else {
			fmt.Println(utl.Gra("# Unknown object type, but dumping it anyway:"))
			utl.PrintYamlColor(obj)
		}
	}

	// When multiple objects shared this ID, print below additional comments
	if len(list) > 1 {
		x0 := AzureObject(list[0]) // Cast single object
		appId := utl.Str(x0["appId"])
		if id == appId {
			msg := utl.Gra("# Given UUID is a ") + utl.Whi("Client Id") + utl.Gra(" shared by above App and SP(s)")
			fmt.Println(msg)
		} else {
			fmt.Println(utl.Red("# WARNING! Multiple objects share this Object Id! This is incredibly rare!"))
		}
	}
}

// Generic print object function
func PrintObject(mazType string, x AzureObject, z *Config) {
	switch mazType {
	case ResRoleDefinition:
		PrintResRoleDefinition(x, z)
	case ResRoleAssignment:
		PrintResRoleAssignment(x, z)
	case Subscription:
		PrintSubscription(x)
	case ManagementGroup:
		PrintMgmtGroup(x)
	case DirectoryUser:
		PrintUser(x, z)
	case DirectoryGroup:
		PrintGroup(x, z)
	case Application:
		PrintApp(x, z)
	case ServicePrincipal:
		PrintSp(x, z)
	case DirRoleDefinition:
		PrintDirRoleDefinition(x, z)
	case DirRoleAssignment:
		PrintDirRoleAssignment(x, z)
	}
}

// Prints appRoleAssignments for given service principal (SP)
func PrintAppRoleAssignmentsSp(roleNameMap map[string]string, appRoleAssignments []interface{}) {
	if len(appRoleAssignments) < 1 {
		return
	}
	fmt.Printf("%s:\n", utl.Blu("app_role_assignments"))
	for _, item := range appRoleAssignments {
		if ara := utl.Map(item); ara != nil {
			principalId := utl.Str(ara["principalId"])
			principalType := utl.Str(ara["principalType"])
			principalName := utl.Str(ara["principalDisplayName"])

			roleName := roleNameMap[utl.Str(ara["appRoleId"])] // Reference roleNameMap now
			if len(roleName) >= 40 {
				roleName = roleName[:37] + "..." // Shorten roleName for nicer printout
			}

			principalName = utl.Gre(principalName)
			roleName = utl.Gre(roleName)
			principalId = utl.Gre(principalId)
			principalType = utl.Gre(principalType)
			fmt.Printf("  %-50s %-50s %s (%s)\n", roleName, principalName, principalId, principalType)
		}
	}
}

// Prints appRoleAssignments for other types of objects (Users and Groups)
func PrintAppRoleAssignmentsOthers(appRoleAssignments []interface{}, z *Config) {
	if len(appRoleAssignments) < 1 {
		return
	}
	fmt.Printf("%s:\n", utl.Blu("app_role_assignments"))
	uniqueIds := utl.StringSet{} // Keep track of assignments
	for _, item := range appRoleAssignments {
		ara := utl.Map(item)
		if ara == nil {
			continue // Skip if not a map
		}
		appRoleId := utl.Str(ara["appRoleId"])
		resourceDisplayName := utl.Str(ara["resourceDisplayName"])
		resourceId := utl.Str(ara["resourceId"]) // SP where the appRole is defined

		// Only print unique assignments, skip over repeated ones
		conbinedId := resourceDisplayName + "_" + resourceId + "_" + appRoleId

		if uniqueIds.Exists(conbinedId) {
			continue // Skip this repeated one. This can happen due to inherited nesting
		}
		uniqueIds.Add(conbinedId) // Mark this id as seen

		// Now build roleNameMap and get roleName
		// We are forced to do this excessive processing for each appRole, because MG Graph does
		// not appear to have a global registry nor a call to get all SP app roles.
		roleNameMap := make(map[string]string)
		x := GetObjectFromAzureById(ServicePrincipal, resourceId, z)
		roleNameMap["00000000-0000-0000-0000-000000000000"] = "Default" // Include default app permissions role
		// But also get all other additional appRoles it may have defined
		appRoles := utl.Slice(x["appRoles"])
		if len(appRoles) > 0 {
			for _, item := range appRoles {
				if a := utl.Map(item); a != nil {
					rId := utl.Str(a["id"])
					displayName := utl.Str(a["displayName"])
					roleNameMap[rId] = displayName // Update growing list of roleNameMap
				}
			}
		}
		roleName := roleNameMap[appRoleId] // Reference roleNameMap now

		resourceDisplayName = utl.Gre(resourceDisplayName)
		resourceId = utl.Gre(resourceId)
		roleName = utl.Gre(roleName)
		fmt.Printf("  %-50s %-50s %s\n", roleName, resourceDisplayName, resourceId)
	}
}

// Prints all memberOf entries
func PrintMemberOfs(memberOf []interface{}) {
	if len(memberOf) < 1 {
		return
	}
	fmt.Printf("%s :\n", utl.Blu("member_of"))
	for i := range memberOf {
		obj := memberOf[i]
		if member := utl.Map(obj); member != nil {
			Type := utl.LastElemByDot(utl.Str(member["@odata.type"]))
			Type = utl.Gre(Type)
			id := utl.Gre(utl.Str(member["id"]))
			name := utl.Gre(utl.Str(member["displayName"]))
			fmt.Printf("  %-50s %s (%s)\n", name, id, Type)
		}
	}
}

func ColorizeExpiryDateTime(endDateTime string) string {
	cExpiry, err := utl.ConvertDateFormat(utl.Str(endDateTime), time.RFC3339Nano, "2006-01-02 15:04")
	if err != nil {
		return utl.Yel("DateFormatConversionError")
	}

	now := time.Now().Unix()
	expiry, err := utl.DateStringToEpocInt64(utl.Str(endDateTime), time.RFC3339Nano)
	if err != nil {
		return utl.Yel("DateFormatConversionError")
	}

	daysDiff := (expiry - now) / 86400
	if daysDiff <= 0 {
		return utl.Red(cExpiry) // If it's expired print in red
	} else if daysDiff < 7 {
		return utl.Yel(cExpiry) // If expiring within a week print in yellow
	} else {
		return utl.Gre(cExpiry)
	}
}

// Prints secret list stanza for App and SP objects
func PrintSecretList(secretsList []interface{}) {
	if len(secretsList) < 1 {
		return
	}
	fmt.Println(utl.Blu("secrets") + ":")
	for _, item := range secretsList {
		pwd := utl.Map(item)
		if pwd == nil {
			continue // Skip if not a map
		}
		cId := utl.Gre(utl.Str(pwd["keyId"]))
		cName := utl.Gre(utl.Str(pwd["displayName"]))
		cHint := utl.Gre(utl.Str(pwd["hint"]) + "***")

		// Reformat date strings for better readability
		cStart, err := utl.ConvertDateFormat(utl.Str(pwd["startDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			cStart = utl.Yel("DateFormatConversionError")
		}
		cExpiry := ColorizeExpiryDateTime(utl.Str(pwd["endDateTime"]))

		fmt.Printf("  - %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n",
			utl.Blu("keyId"), cId, utl.Blu("displayName"), cName, utl.Blu("hint"), cHint,
			utl.Blu("startDateTime"), utl.Gre(cStart), utl.Blu("expiry"), cExpiry)

		// Old way, all in one line
		// fmt.Printf("  %-36s  %-30s  %-16s  %-16s  %s\n", cId, cName,
		// 	cHint, utl.Gre(cStart), cExpiry)
	}
}

// Prints certificate list stanza for Apps and Sps
func PrintCertificateList(certificates []interface{}) {
	if len(certificates) < 1 {
		return
	}
	fmt.Println(utl.Blu("certificates") + ":")
	for _, item := range certificates {
		cert := utl.Map(item)
		if cert == nil {
			continue // Skip if not a map
		}
		cId := utl.Gre(utl.Str(cert["keyId"]))
		cName := utl.Gre(utl.Str(cert["displayName"]))
		cType := utl.Gre(utl.Str(cert["type"]))
		cCustomKeyIdentifier := utl.Gre(utl.Str(cert["customKeyIdentifier"]))
		// Reformat date strings for better readability
		cStart, err := utl.ConvertDateFormat(utl.Str(cert["startDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			cStart = utl.Yel("DateFormatConversionError")
		}
		cExpiry := ColorizeExpiryDateTime(utl.Str(cert["endDateTime"]))

		fmt.Printf("  - %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n    %s: %s\n",
			utl.Blu("keyId"), cId, utl.Blu("displayName"), cName, utl.Blu("type"), cType,
			utl.Blu("customKeyIdentifier"), cCustomKeyIdentifier,
			utl.Blu("startDateTime"), utl.Gre(cStart), utl.Blu("expiry"), cExpiry)

		// Old way, all in one line
		// fmt.Printf("  %-36s  %-30s  %-40s  %-10s  %s\n",
		// 	utl.Gre(cId), utl.Gre(cName), utl.Gre(cType), utl.Gre(cStart), cExpiry)
	}
}

// Print owners stanza for applications and service principals
func PrintOwners(owners []interface{}) {
	if len(owners) < 1 {
		return
	}
	fmt.Printf("%s :\n", utl.Blu("owners"))
	for _, item := range owners {
		if owner := utl.Map(item); owner != nil {
			Type, Name := "UnknownType", "UnknownName"
			Type = utl.LastElemByDot(utl.Str(owner["@odata.type"]))
			switch Type {
			case "user":
				Name = utl.Str(owner["userPrincipalName"])
			case "group":
				Name = utl.Str(owner["displayName"])
			case "servicePrincipal":
				Name = utl.Str(owner["displayName"])
				if utl.Str(owner["servicePrincipalType"]) == "ManagedIdentity" {
					Type = "ManagedIdentity"
				}
			}
			fmt.Printf("  %-50s %s (%s)\n", utl.Gre(Name), utl.Gre(utl.Str(owner["id"])), utl.Gre(Type))
		}
	}
}

// Prints string map in YAML-like format, sorted, and in color
func PrintStringMapColor(strMap map[string]string) {
	sortedKeys := utl.SortMapStringKeys(strMap)
	for _, k := range sortedKeys {
		v := strMap[k]
		cK := utl.Blu(utl.Str(k))                         // Key in blue
		fmt.Printf("  %s: %s\n", cK, utl.Gre(utl.Str(v))) // Value in green
	}
}

// Prints all objects that match on given specifier
func PrintMatchingObjects(specifier, filter string, z *Config) {
	// Range of possible specifiers: "d", "a", "s", "m", "u", "g", "ap", "sp", "dr",
	// "da", "dj", "aj", "sj", "mj", "uj", "gj", "apj", "spj", "drj", "daj"
	mazType := specifier
	printJson := mazType[len(mazType)-1] == 'j' // If last char is 'j', then JSON output is required
	if printJson {
		mazType = mazType[:len(mazType)-1] // Remove the 'j' from mazType
	}

	matchingObjects := GetMatchingObjects(mazType, filter, false, z) // false = get from cache, not Azure
	matchingCount := len(matchingObjects)

	if matchingCount > 1 {
		if printJson {
			utl.PrintJsonColor(matchingObjects) // Print macthing set in JSON format
		} else {
			for i := range matchingObjects { // Print matching set in terse format
				obj := matchingObjects[i]
				PrintTersely(mazType, obj)
			}
		}
	} else if matchingCount == 1 {
		singleObj := matchingObjects[0]
		isFromCache := !utl.Bool(singleObj["maz_from_azure"])
		if isFromCache {
			// If object is from cache, then get the full version from Azure
			id := utl.Str(singleObj["id"])
			if mazType == Subscription {
				// Subscriptions use 'subscriptionId' instead of the fully-qualified 'id'
				id = utl.Str(singleObj["subscriptionId"])
			}
			if mazType == ResRoleDefinition || mazType == ResRoleAssignment || mazType == ManagementGroup {
				// These 3 types use 'name' instead of the fully-qualified 'id'
				id = utl.Str(singleObj["name"])
			}
			singleObj = GetAzureObjectById(mazType, id, z)
		}
		if printJson {
			utl.PrintJsonColor(singleObj) // Print in JSON format
		} else {
			PrintObject(mazType, singleObj, z) // Print in regular format
		}
	}
}
