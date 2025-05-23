package maz

import (
	"fmt"

	"github.com/queone/utl"
)

// Print directory group object in YAML-like format
func PrintGroup(x AzureObject, z *Config) {
	id := utl.Str(x["id"])
	if id == "" {
		return
	}

	// Print the most important attributes first
	fmt.Printf("%s\n", utl.Gra("# Directory group"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(utl.Str(x["displayName"])))
	description := utl.Str(x["description"])
	if description != "" {
		fmt.Printf("%s: %s\n", utl.Blu("description"), utl.Gre(description))
	}
	isAssignableToRole := utl.Bool(x["isAssignableToRole"])
	if isAssignableToRole {
		fmt.Printf("%s: %s\n", utl.Blu("isAssignableToRole"), utl.Mag(isAssignableToRole))
	}

	// Print owners of this group
	apiUrl := ConstMgUrl + "/v1.0/groups/" + id + "/owners"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	owners := utl.Slice(resp["value"])
	if len(owners) > 0 {
		fmt.Printf("%s:\n", utl.Blu("owners"))
		for _, item := range owners {
			if owner := utl.Map(item); owner != nil {
				fmt.Printf("  %-50s %s\n", utl.Gre(utl.Str(owner["userPrincipalName"])),
					utl.Gre(utl.Str(owner["id"])))
			}
		}
	}

	// Print app role assignment members and the specific role assigned
	apiUrl = ConstMgUrl + "/v1.0/groups/" + id + "/appRoleAssignments"
	appRoleAssignments := GetAzureAllPages(apiUrl, z)
	PrintAppRoleAssignmentsOthers(appRoleAssignments, z)

	// Print all groups and roles it is a member of
	apiUrl = ConstMgUrl + "/v1.0/groups/" + id + "/transitiveMemberOf"
	resp, statCode, _ = ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	memberOfList := utl.Slice(resp["value"])
	if len(memberOfList) > 0 {
		PrintMemberOfs(memberOfList)
	}

	// Print members of this group
	//apiUrl = ConstMgUrl + "/v1.0/groups/" + id + "/members"
	apiUrl = ConstMgUrl + "/beta/groups/" + id + "/members"
	// API v1.0 does not currently work for SP members
	// See https://developer.microsoft.com/en-us/graph/known-issues/?search=25984
	resp, statCode, _ = ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	members := utl.Slice(resp["value"])
	if len(members) > 0 {
		fmt.Printf("%s:\n", utl.Blu("members"))
		for _, item := range members {
			if member := utl.Map(item); member != nil {
				Type, Name := "-", "-"
				Type = utl.LastElemByDot(utl.Str(member["@odata.type"]))
				switch Type {
				case "group", "servicePrincipal":
					Name = utl.Str(member["displayName"])
				default:
					Name = utl.Str(member["userPrincipalName"])
				}
				fmt.Printf("  %-50s %s (%s)\n", utl.Gre(Name),
					utl.Gre(utl.Str(member["id"])), utl.Gre(Type))
			}
		}
	}
}

// Lists all cached Privileged Access Groups (PAGs)
func PrintPags(z *Config) {
	groups := GetMatchingDirObjects(DirectoryGroup, "", false, z) // false = get from cache, not Azure
	for i := range groups {
		group := groups[i]
		isAssignableToRole := utl.Bool(group["isAssignableToRole"])
		if isAssignableToRole {
			PrintTersely(DirectoryGroup, group)
		}
	}
}

// Creates an Azure directory group from given command-line arguments.
func CreateDirGroupFromArgs(force, isAssignableToRole bool, name, description string, z *Config) {
	// Note that id may be a UUID or a displayName

	// Initialize obj variable, and add user-supplied attributes
	obj := make(AzureObject)
	obj["description"] = description
	obj["isAssignableToRole"] = isAssignableToRole
	// Add other required required attributes for creation.
	obj["displayName"] = name
	obj["mailEnabled"] = false
	obj["mailNickname"] = "NotSet"
	obj["securityEnabled"] = true

	CreateDirObject(force, obj, DirectoryGroup, z)
}

// Creates or updates an Azure directory group from given object
func UpsertGroup(force bool, obj AzureObject, z *Config) {
	// Cannot continue without at least a displayName from that object
	displayName := utl.Str(obj["displayName"])
	if displayName == "" {
		utl.Die("Object is missing %s\n", utl.Red("displayName"))
	}

	x := PreFetchAzureObject(DirectoryGroup, displayName, z)
	if x != nil {
		// Update if group exists
		UpdateDirObject(force, utl.Str(x["id"]), obj, DirectoryGroup, z)
		// TODO: Have above return obj and/or err or both?
		// if azObj, err := UpdateDirObject(force, utl.Str(x["id"]), obj, DirectoryGroup, z); err :=1 nil {
		// 	fmt.Printf("%s\n", err)
		// 	if azObj == nil {
		// 		fmt.Println("The object was still updated.")
		// 	}
		// }
	} else {
		// Create if group does not exist
		// Set up obj with the minimally required attributes to create a group
		if obj["mailEnabled"] == nil {
			utl.Die("Object is missing %s\n", utl.Red("mailEnabled"))
		}
		if obj["mailNickname"] == nil {
			utl.Die("Object is missing %s\n", utl.Red("mailNickname"))
		}
		if obj["securityEnabled"] == nil {
			utl.Die("Object is missing %s\n", utl.Red("securityEnabled"))
		}
		CreateDirObject(force, obj, DirectoryGroup, z)
	}
}

// Helper function to check if the object is a directory group
func IsDirGroup(obj AzureObject) bool {
	// Check if 'displayName' exists and is a non-empty string
	if utl.Str(obj["displayName"]) == "" {
		return false
	}

	// Check if 'mailEnabled' exists
	if obj["mailEnabled"] == nil {
		return false
	}

	// Check if 'mailNickname' exists and is a non-empty string
	if utl.Str(obj["mailNickname"]) == "" {
		return false
	}

	// Check if 'securityEnabled' exists
	if obj["securityEnabled"] == nil {
		return false
	}

	// If all checks pass, it's a valid directory group object
	return true
}
