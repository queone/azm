package maz

import (
	"fmt"
	"time"

	"github.com/queone/utl"
)

// Prints a status count of all AZ and MG objects that are in Azure, and the local files.
func PrintCountStatus(z *Config) {
	c1Width := 50 // Column 1 width
	c2Width := 10 // Column 2 width
	c3Width := 10 // Column 3 width
	fmt.Printf("%s\n", utl.Gra("# Please note that enumerating some Azure resources can be slow"))
	fmt.Print(utl.Whi2(utl.PostSpc("OBJECTS", c1Width)+
		utl.PreSpc("LOCAL", c2Width)+
		utl.PreSpc("AZURE", c3Width)) + "\n")
	status := utl.Blu(utl.PostSpc("Directory Users", c1Width))
	status += utl.Gre(utl.PreSpc(UsersCountLocal(z), c2Width))
	status += utl.Gre(utl.PreSpc(UsersCountAzure(z), c3Width)) + "\n"
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
	status += utl.Blu(utl.PostSpc("Directory Role Definitions", c1Width))
	status += utl.Gre(utl.PreSpc(AdRolesCountLocal(z), c2Width))
	status += utl.Gre(utl.PreSpc(AdRolesCountAzure(z), c3Width)) + "\n"

	// To be developed
	// status += utl.Blu(utl.PostSpc("Directory Role Assignments", c1Width))
	// status += utl.Gre(utl.PreSpc(AdRolesCountLocal(z), c2Width))
	// status += utl.Gre(utl.PreSpc(AdRolesCountAzure(z), c3Width)) + "\n"

	status += utl.Blu(utl.PostSpc("Resource Management Groups", c1Width))
	status += utl.Gre(utl.PreSpc(MgGroupCountLocal(z), c2Width))
	status += utl.Gre(utl.PreSpc(MgGroupCountAzure(z), c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Resource Subscriptions", c1Width))
	status += utl.Gre(utl.PreSpc(SubsCountLocal(z), c2Width))
	status += utl.Gre(utl.PreSpc(SubsCountAzure(z), c3Width)) + "\n"
	builtinLocal, customLocal := RoleDefinitionCountLocal(z)
	builtinAzure, customAzure := RoleDefinitionCountAzure(z)
	status += utl.Blu(utl.PostSpc("Resource Role Definitions (built-in)", c1Width))
	status += utl.Gre(utl.PreSpc(builtinLocal, c2Width))
	status += utl.Gre(utl.PreSpc(builtinAzure, c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Resource Role Definitions (custom)", c1Width))
	status += utl.Gre(utl.PreSpc(customLocal, c2Width))
	status += utl.Gre(utl.PreSpc(customAzure, c3Width)) + "\n"
	status += utl.Blu(utl.PostSpc("Resource Role Assignments", c1Width))
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
func PrintTersely(t string, object interface{}) {
	switch t {
	case "d":
		x := object.(map[string]interface{}) // Assert as JSON object
		xProp := x["properties"].(map[string]interface{})
		fmt.Printf("%s  %-60s  %s\n", utl.Str(x["name"]), utl.Str(xProp["roleName"]), utl.Str(xProp["type"]))
	case "a":
		x := object.(map[string]interface{}) // Assert as JSON object
		xProp := x["properties"].(map[string]interface{})
		rdId := utl.LastElem(utl.Str(xProp["roleDefinitionId"]), "/")
		principalId := utl.Str(xProp["principalId"])
		principalType := utl.Str(xProp["principalType"])
		scope := utl.Str(xProp["scope"])
		fmt.Printf("%s  %s  %s %-20s %s\n", utl.Str(x["name"]), rdId, principalId, "("+principalType+")", scope)
	case "s":
		x := object.(map[string]interface{}) // Assert as JSON object
		fmt.Printf("%s  %-10s  %s\n", utl.Str(x["subscriptionId"]), utl.Str(x["state"]), utl.Str(x["displayName"]))
	case "m":
		x := object.(map[string]interface{}) // Assert as JSON object
		xProp := x["properties"].(map[string]interface{})
		fmt.Printf("%-38s  %-20s  %s\n", utl.Str(x["name"]), utl.Str(xProp["displayName"]), MgType(utl.Str(x["type"])))
	case "u":
		x := object.(map[string]interface{}) // Assert as JSON object
		upn := utl.Str(x["userPrincipalName"])
		onPremisesSamAccountName := utl.Str(x["onPremisesSamAccountName"])
		fmt.Printf("%s  %-50s %-18s %s\n", utl.Str(x["id"]), upn, onPremisesSamAccountName, utl.Str(x["displayName"]))
	case "g":
		x := object.(AzureObject)
		fmt.Printf("%s  %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]))
	case "sp", "ap":
		x := object.(AzureObject) // Assert as JSON object
		fmt.Printf("%s  %-66s %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]), utl.Str(x["appId"]))
	case "ad":
		x := object.(map[string]interface{}) // Assert as JSON object
		builtIn := "Custom"
		if utl.Str(x["isBuiltIn"]) == "true" {
			builtIn = "BuiltIn"
		}
		enabled := "Disabled"
		if utl.Str(x["isEnabled"]) == "true" {
			enabled = "Enabled"
		}
		fmt.Printf("%s  %-60s  %s  %s\n", utl.Str(x["id"]), utl.Str(x["displayName"]), builtIn, enabled)
	}
}

// Prints object by given UUID
func PrintObjectById(id string, z *Config) {
	list := FindAzObjectsById(id, z) // Search for this UUID under all maz objects types

	for _, obj := range list {
		x := obj.(map[string]interface{})
		mazType := utl.Str(x["mazType"]) // Function FindAzObjectsById() should have added this field
		if mazType != "" {
			PrintObject(mazType, x, z)
		} else {
			fmt.Println(utl.Gra("# Unknown object type, but dumping it anyway:"))
			utl.PrintYamlColor(x)
		}
	}

	// Multiple objects shared this ID. Print additional comments
	if len(list) > 1 {
		x0 := list[0].(map[string]interface{})
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
func PrintObject(t string, x map[string]interface{}, z *Config) {
	switch t {
	case "d":
		PrintRoleDefinition(x, z)
	case "a":
		PrintRoleAssignment(x, z)
	case "s":
		PrintSubscription(x)
	case "m":
		PrintMgGroup(x)
	case "u":
		PrintUser(x, z)
	case "g":
		PrintGroup(x, z)
	case "sp":
		PrintSp(x, z)
	case "ap":
		PrintApp(x, z)
	case "ad":
		PrintAdRole(x, z)
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
func PrintMemberOfs(t string, memberOf []interface{}) {
	if len(memberOf) < 1 {
		return
	}
	fmt.Printf("%s :\n", utl.Blu("member_of"))
	for _, i := range memberOf {
		x := i.(map[string]interface{}) // Assert as JSON object type
		Type := utl.LastElem(utl.Str(x["@odata.type"]), ".")
		Type = utl.Gre(Type)
		iId := utl.Gre(utl.Str(x["id"]))
		name := utl.Gre(utl.Str(x["displayName"]))
		fmt.Printf("  %-50s %s (%s)\n", name, iId, Type)
	}
}

// Prints secret list stanza for App and SP objects
func PrintSecretList(secretsList []interface{}) {
	if len(secretsList) < 1 {
		return
	}
	fmt.Println(utl.Blu("secrets") + ":")
	for _, i := range secretsList {
		pw := i.(map[string]interface{})
		cId := utl.Str(pw["keyId"])
		cName := utl.Str(pw["displayName"])
		cHint := utl.Str(pw["hint"]) + "********"

		// Reformat date strings for better readability
		cStart, err := utl.ConvertDateFormat(utl.Str(pw["startDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			utl.Die("%s %s\n", utl.Trace(), err.Error())
		}
		cExpiry, err := utl.ConvertDateFormat(utl.Str(pw["endDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			utl.Die("%s %s\n", utl.Trace(), err.Error())
		}

		// Check if expiring soon
		now := time.Now().Unix()
		expiry, err := utl.DateStringToEpocInt64(utl.Str(pw["endDateTime"]), time.RFC3339Nano)
		if err != nil {
			utl.Die("%s %s\n", utl.Trace(), err.Error())
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
			utl.Die("%s %s\n", utl.Trace(), err.Error())
		}
		cExpiry, err := utl.ConvertDateFormat(utl.Str(a["endDateTime"]), time.RFC3339Nano, "2006-01-02 15:04")
		if err != nil {
			utl.Die("%s %s\n", utl.Trace(), err.Error())
		}
		// Check if expiring soon
		now := time.Now().Unix()
		expiry, err := utl.DateStringToEpocInt64(utl.Str(a["endDateTime"]), time.RFC3339Nano)
		if err != nil {
			utl.Die("%s %s\n", utl.Trace(), err.Error())
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

// Print owners stanza for Apps and Sps
func PrintOwners(owners []interface{}) {
	if len(owners) < 1 {
		return
	}
	fmt.Printf("%s :\n", utl.Blu("owners"))
	for _, i := range owners {
		o := i.(map[string]interface{})
		Type, Name := "?UnknownType?", "?UnknownName?"
		Type = utl.LastElem(utl.Str(o["@odata.type"]), ".")
		switch Type {
		case "user":
			Name = utl.Str(o["userPrincipalName"])
		case "group":
			Name = utl.Str(o["displayName"])
		case "servicePrincipal":
			Name = utl.Str(o["displayName"])
			if utl.Str(o["servicePrincipalType"]) == "ManagedIdentity" {
				Type = "ManagedIdentity"
			}
		}
		fmt.Printf("  %-50s %s (%s)\n", utl.Gre(Name), utl.Gre(utl.Str(o["id"])), utl.Gre(Type))
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
func PrintMatching(printFormat, t, specifier string, z *Config) {
	if utl.ValidUuid(specifier) {
		// If valid UUID string, get object direct from Azure
		x := GetAzObjectById(t, specifier, z)
		if x != nil {
			if printFormat == "json" {
				utl.PrintJsonColor(x)
			} else if printFormat == "reg" {
				PrintObject(t, x, z)
			}
			return
		}
	}
	matchingObjects := GetObjects(t, specifier, false, z)
	if len(matchingObjects) == 1 {
		// If it's only one object, try getting it direct from Azure instead of using the local cache
		x := matchingObjects[0].(map[string]interface{})
		id := utl.Str(x["id"])
		if utl.ValidUuid(id) {
			x = GetAzObjectById(t, id, z) // Replace object with version directly in Azure
		}
		if printFormat == "json" {
			utl.PrintJsonColor(x)
		} else if printFormat == "reg" {
			PrintObject(t, x, z)
		}
	} else if len(matchingObjects) > 1 {
		if printFormat == "json" {
			utl.PrintJsonColor(matchingObjects) // Print all matching objects in JSON
		} else if printFormat == "reg" {
			for _, i := range matchingObjects { // Print all matching object teresely
				x := i.(map[string]interface{})
				PrintTersely(t, x)
			}
		}
	}
}
