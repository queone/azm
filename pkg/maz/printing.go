package maz

import (
	"fmt"
	"time"

	"github.com/queone/utl"
)

// Prints a status count of all AZ and MG objects that are in Azure, and the local files.
func PrintCountStatus(z *Config) {
	c1Width := 44 // Column 1 width
	c2Width := 10 // Column 2 width
	c3Width := 10 // Column 3 width
	fmt.Printf("%s\n", utl.Gra("# Please note that enumerating some Azure resources can be slow"))
	fmt.Print(utl.Whi2(utl.PostSpc("OBJECTS", c1Width)+
		utl.PreSpc("LOCAL", c2Width)+
		utl.PreSpc("AZURE", c3Width)) + "\n")
	status := utl.Blu(utl.PostSpc("Directory Users", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal("u", z), c2Width))
	status += utl.Gre(utl.PreSpc(ObjectCountAzure("u", z), c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Directory Groups", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal("g", z), c2Width))
	status += utl.Gre(utl.PreSpc(ObjectCountAzure("g", z), c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Directory Applications", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal("ap", z), c2Width))
	status += utl.Gre(utl.PreSpc(ObjectCountAzure("ap", z), c3Width)) + "\n"
	nativeSpsLocal, msSpsLocal := SpsCountLocal(z)
	nativeSpsAzure, msSpsAzure := SpsCountAzure(z)
	status += utl.Blu(utl.PostSpc("Directory Service Principals (this tenant)", c1Width))
	status += utl.Gre(utl.PreSpc(nativeSpsLocal, c2Width))
	status += utl.Gre(utl.PreSpc(nativeSpsAzure, c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Directory Service Principals (multi-tenant)", c1Width))
	status += utl.Gre(utl.PreSpc(msSpsLocal, c2Width))
	status += utl.Gre(utl.PreSpc(msSpsAzure, c3Width)) + "\n"

	// Note: ObjectCountAzure() doesn't support dr nor da objects so we just count
	// the ones in the local cache and print them for local *and* Azure
	status += utl.Blu(utl.PostSpc("Directory Role Definitions", c1Width))
	drCount := ObjectCountLocal("dr", z)
	status += utl.Gre(utl.PreSpc(drCount, c2Width))
	status += utl.Gre(utl.PreSpc(drCount, c3Width)) + "\n"
	daCount := ObjectCountLocal("da", z)
	status += utl.Blu(utl.PostSpc("Directory Role Assignments", c1Width))
	status += utl.Gre(utl.PreSpc(daCount, c2Width))
	status += utl.Gre(utl.PreSpc(daCount, c3Width)) + "\n"

	status += utl.Blu(utl.PostSpc("Resource Management Groups", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal("m", z), c2Width))
	status += utl.Gre(utl.PreSpc(CountAzureMgmtGroups(z), c3Width)) + "\n"

	status += utl.Blu(utl.PostSpc("Resource Subscriptions", c1Width))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal("s", z), c2Width))
	status += utl.Gre(utl.PreSpc(CountAzureSubscriptions(z), c3Width)) + "\n"

	customLocal, builtinLocal := CountRbacDefinitions(false, z) // false = get from cache, not Azure
	customAzure, builtinAzure := CountRbacDefinitions(true, z)  // true = get from Azure, not cache
	status += utl.Blu(utl.PostSpc("Resource RBAC Definitions (built-in)", c1Width))
	status += utl.Gre(utl.PreSpc(builtinLocal, c2Width))
	status += utl.Gre(utl.PreSpc(builtinAzure, c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Resource RBAC Definitions (custom)", c1Width))
	status += utl.Gre(utl.PreSpc(customLocal, c2Width))
	status += utl.Gre(utl.PreSpc(customAzure, c3Width)) + "\n"

	status += utl.Blu(utl.PostSpc("Resource RBAC Assignments", c1Width))
	status += utl.Gre(utl.PreSpc(RoleAssignmentsCountLocal(z), c2Width))
	status += utl.Gre(utl.PreSpc(RoleAssignmentsCountAzure(z), c3Width)) + "\n"

	fmt.Print(status)
}

func PrintCountStatusAppsAndSps(z *Config) {
	fmt.Printf("%-36s%16s%16s\n", "OBJECTS", "LOCAL", "AZURE")
	status := utl.Blu(utl.PostSpc("Azure App Registrations", 36))
	status += utl.Gre(utl.PreSpc(ObjectCountLocal("ap", z), 16))
	status += utl.Gre(utl.PreSpc(ObjectCountAzure("ap", z), 16)) + "\n"
	nativeSpsLocal, msSpsLocal := SpsCountLocal(z)
	nativeSpsAzure, msSpsAzure := SpsCountAzure(z)
	status += utl.Blu(utl.PostSpc("Azure SPs (native)", 36))
	status += utl.Gre(utl.PreSpc(nativeSpsLocal, 16))
	status += utl.Gre(utl.PreSpc(nativeSpsAzure, 16)) + "\n"
	status += utl.Blu(utl.PostSpc("Azure SPs (others)", 36))
	status += utl.Gre(utl.PreSpc(msSpsLocal, 16))
	status += utl.Gre(utl.PreSpc(msSpsAzure, 16)) + "\n"
	fmt.Print(status)
}

// Prints this single object of type 't' tersely, with minimal attributes.
func PrintTersely(mazType string, object interface{}) {
	switch mazType {
	case RbacDefinition:
		x := object.(AzureObject)
		xProp := x["properties"].(map[string]interface{})
		fmt.Printf("%s  %-60s  %s\n", utl.Str(x["name"]), utl.Str(xProp["roleName"]), utl.Str(xProp["type"]))
	case RbacAssignment:
		x := object.(map[string]interface{}) // Assert as JSON object
		xProp := x["properties"].(map[string]interface{})
		rdId := utl.LastElem(utl.Str(xProp["roleDefinitionId"]), "/")
		principalId := utl.Str(xProp["principalId"])
		principalType := utl.Str(xProp["principalType"])
		scope := utl.Str(xProp["scope"])
		fmt.Printf("%s  %s  %s %-20s %s\n", utl.Str(x["name"]), rdId, principalId, "("+principalType+")", scope)
	case Subscription:
		x := object.(AzureObject)
		fmt.Printf("%s  %-10s  %s\n", utl.Str(x["subscriptionId"]), utl.Str(x["state"]), utl.Str(x["displayName"]))
	case ManagementGroup:
		x := object.(AzureObject)
		displayName := utl.Str(x["displayName"])
		// REVIEW
		// Is below really needed? We are normalizing these properties values to root of object cache
		if x["properties"] != nil {
			xProp := x["properties"].(map[string]interface{})
			displayName = utl.Str(xProp["displayName"])
		}
		fmt.Printf("%-38s  %s\n", utl.Str(x["name"]), displayName)
	case DirectoryUser:
		x := object.(AzureObject)
		upn := utl.Str(x["userPrincipalName"])
		onPremName := utl.Str(x["onPremisesSamAccountName"])
		fmt.Printf("%s  %-50s %-18s %s\n", utl.Str(x["id"]), upn, onPremName, utl.Str(x["displayName"]))
	case DirectoryGroup:
		x := object.(AzureObject)
		fmt.Printf("%s  %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]))
	case Application, ServicePrincipal:
		x := object.(AzureObject)
		fmt.Printf("%s  %-66s %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]), utl.Str(x["appId"]))
	case DirRoleDefinition:
		x := object.(AzureObject)
		builtIn := "Custom"
		if utl.Str(x["isBuiltIn"]) == "true" {
			builtIn = "BuiltIn"
		}
		enabled := "Disabled"
		if utl.Str(x["isEnabled"]) == "true" {
			enabled = "Enabled"
		}
		fmt.Printf("%s  %-60s  %-10s  %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]), builtIn, enabled)
	case DirRoleAssignment:
		x := object.(AzureObject)
		scope := utl.Str(x["directoryScopeId"])
		principalId := utl.Str(x["principalId"])
		roleDefId := utl.Str(x["roleDefinitionId"])
		fmt.Printf("%-66s  %-37s  %-36s  %s\n", utl.Str(x["id"]), scope, principalId, roleDefId)
	}
}

// Prints object by given ID
func PrintObjectById(id string, z *Config) {
	list, err := FindAzureObjectsById(id, z) // Search for this ID under all maz objects types
	if err != nil {
		utl.Die("Error: %v\n", err)
	}

	for _, azObj := range list {
		mazType := utl.Str(azObj["maz_type"]) // Function FindAzureObjectsById() should have added this field
		if mazType != "" {
			PrintObject(mazType, azObj, z)
		} else {
			fmt.Println(utl.Gra("# Unknown object type, but dumping it anyway:"))
			utl.PrintYamlColor(azObj)
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
	case RbacDefinition:
		PrintRbacDefinition(x, z)
	case RbacAssignment:
		PrintRbacAssignment(x, z)
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
	for _, i := range appRoleAssignments {
		ara := i.(map[string]interface{}) // JSON object

		principalId := utl.Str(ara["principalId"])
		principalType := utl.Str(ara["principalType"])
		principalName := utl.Str(ara["principalDisplayName"])

		roleName := roleNameMap[utl.Str(ara["appRoleId"])] // Reference roleNameMap now
		if len(roleName) >= 40 {
			roleName = utl.FirstN(roleName, 37) + "..."
		}

		principalName = utl.Gre(principalName)
		roleName = utl.Gre(roleName)
		principalId = utl.Gre(principalId)
		principalType = utl.Gre(principalType)
		fmt.Printf("  %-50s %-50s %s (%s)\n", roleName, principalName, principalId, principalType)
	}
}

// Prints appRoleAssignments for other types of objects (Users and Groups)
func PrintAppRoleAssignmentsOthers(appRoleAssignments []interface{}, z *Config) {
	if len(appRoleAssignments) < 1 {
		return
	}

	fmt.Printf("%s:\n", utl.Blu("app_role_assignments"))
	uniqueIds := utl.NewStringSet() // Keep track of assignments
	for _, i := range appRoleAssignments {
		ara := i.(map[string]interface{}) // JSON object
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
		x := GetObjectFromAzureById("sp", resourceId, z)
		roleNameMap["00000000-0000-0000-0000-000000000000"] = "Default" // Include default app permissions role
		// But also get all other additional appRoles it may have defined
		appRoles := x["appRoles"].([]interface{})
		if len(appRoles) > 0 {
			for _, i := range appRoles {
				a := i.(map[string]interface{})
				rId := utl.Str(a["id"])
				displayName := utl.Str(a["displayName"])
				roleNameMap[rId] = displayName // Update growing list of roleNameMap
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
	for _, item := range memberOf {
		if obj, ok := item.(map[string]interface{}); ok {
			Type := utl.LastElem(utl.Str(obj["@odata.type"]), ".")
			Type = utl.Gre(Type)
			id := utl.Gre(utl.Str(obj["id"]))
			name := utl.Gre(utl.Str(obj["displayName"]))
			fmt.Printf("  %-50s %s (%s)\n", name, id, Type)
		}
	}
}

// Prints secret list stanza for App and SP objects
func PrintSecretList(secretsList []interface{}) {
	if len(secretsList) < 1 {
		return
	}
	fmt.Println(utl.Blu("secrets") + ":")
	for _, i := range secretsList {
		pw, ok := i.(map[string]interface{})
		if !ok {
			fmt.Printf("%s\n", utl.Yel("Error asserting this pwd map"))
			continue
		}
		cId := utl.Str(pw["keyId"])
		cName := utl.Str(pw["displayName"])
		cHint := utl.Str(pw["hint"]) + "********"

		// Reformat date strings for better readability
		cStart, err := utl.ConvertDateFormat(utl.Str(pw["startDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			fmt.Printf("%s\n", utl.Yel("Error converting startDateTime format"))
		}
		cExpiry, err := utl.ConvertDateFormat(utl.Str(pw["endDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			fmt.Printf("%s\n", utl.Yel("Error converting endDateTime format"))
		}

		// Check if expiring soon
		now := time.Now().Unix()
		expiry, err := utl.DateStringToEpocInt64(utl.Str(pw["endDateTime"]), time.RFC3339Nano)
		if err != nil {
			fmt.Printf("%s\n", utl.Yel("Error converting endDateTime epoc string"))
		}
		daysDiff := (expiry - now) / 86400
		if daysDiff <= 0 {
			cExpiry = utl.Red(cExpiry) // If it's expired print in red
		} else if daysDiff < 7 {
			cExpiry = utl.Yel(cExpiry) // If expiring within a week print in yellow
		} else {
			cExpiry = utl.Gre(cExpiry)
		}
		fmt.Printf("  %-36s  %-30s  %-16s  %-16s  %s\n", utl.Gre(cId), utl.Gre(cName),
			utl.Gre(cHint), utl.Gre(cStart), cExpiry)
	}
}

// Prints certificate list stanza for Apps and Sps
func PrintCertificateList(certificates []interface{}) {
	if len(certificates) < 1 {
		return
	}
	fmt.Println(utl.Blu("certificates") + ":")
	for _, i := range certificates {
		a := i.(map[string]interface{})
		cId := utl.Str(a["keyId"])
		cName := utl.Str(a["displayName"])
		cType := utl.Str(a["type"])
		// Reformat date strings for better readability
		cStart, err := utl.ConvertDateFormat(utl.Str(a["startDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			fmt.Printf("%s\n", utl.Yel("Error converting startDateTime format"))
		}
		cExpiry, err := utl.ConvertDateFormat(utl.Str(a["endDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			fmt.Printf("%s\n", utl.Yel("Error converting endDateTime format"))
		}
		// Check if expiring soon
		now := time.Now().Unix()
		expiry, err := utl.DateStringToEpocInt64(utl.Str(a["endDateTime"]), time.RFC3339Nano)
		if err != nil {
			fmt.Printf("%s\n", utl.Yel("Error converting endDateTime epoc string"))
		}
		daysDiff := (expiry - now) / 86400
		if daysDiff <= 0 {
			cExpiry = utl.Red(cExpiry) // If it's expired print in red
		} else if daysDiff < 7 {
			cExpiry = utl.Yel(cExpiry) // If expiring within a week print in yellow
		} else {
			cExpiry = utl.Gre(cExpiry)
		}
		// There's also:
		// 	"customKeyIdentifier": "09228573F93570D8113D90DA69D8DF6E2E396874",
		// 	"key": "<RSA_KEY>",
		// 	"usage": "Verify"
		fmt.Printf("  %-36s  %-30s  %-40s  %-10s  %s\n", utl.Gre(cId), utl.Gre(cName),
			utl.Gre(cType), utl.Gre(cStart), cExpiry)
	}
	// https://learn.microsoft.com/en-us/graph/api/application-addkey
}

// Print owners stanza for applications and service principals
func PrintOwners(owners []interface{}) {
	if len(owners) < 1 {
		return
	}
	fmt.Printf("%s :\n", utl.Blu("owners"))
	for _, item := range owners {
		owner, ok := item.(map[string]interface{})
		if !ok {
			continue // silently skip this owner?
		}
		Type, Name := "UnknownType", "UnknownName"
		Type = utl.LastElem(utl.Str(owner["@odata.type"]), ".")
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
		mazType = mazType[:len(mazType)-1] // Remove the 'j' from t
	}

	matchingObjects := GetMatchingObjects(mazType, filter, false, z) // false = get from cache, not Azure
	matchingCount := len(matchingObjects)
	if matchingCount > 1 {
		if printJson {
			utl.PrintJsonColor(matchingObjects) // Print macthing set in JSON format
		} else {
			for _, item := range matchingObjects { // Print matching set in terse format
				PrintTersely(mazType, item)
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
			if mazType == RbacDefinition || mazType == RbacAssignment || mazType == ManagementGroup {
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
