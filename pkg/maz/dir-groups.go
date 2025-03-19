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
	fmt.Printf("%s\n", utl.Gra("# Directory Group"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(utl.Str(x["displayName"])))
	if x["description"] != nil {
		fmt.Printf("%s: %s\n", utl.Blu("description"), utl.Gre(utl.Str(x["description"])))
	}
	if x["isAssignableToRole"] != nil {
		fmt.Printf("%s: %s\n", utl.Blu("isAssignableToRole"), utl.Mag(x["isAssignableToRole"].(bool)))
	}

	// Print owners of this group
	apiUrl := ConstMgUrl + "/v1.0/groups/" + id + "/owners"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode == 200 && resp != nil && resp["value"] != nil {
		owners := resp["value"].([]interface{}) // Assert as JSON array type
		if len(owners) > 0 {
			fmt.Printf("%s:\n", utl.Blu("owners"))
			for _, i := range owners {
				if mapObj, ok := i.(map[string]interface{}); ok {
					o := AzureObject(mapObj) // Convert map[string]interface{} to AzureObject
					fmt.Printf("  %-50s %s\n", utl.Gre(utl.Str(o["userPrincipalName"])),
						utl.Gre(utl.Str(o["id"])))
				}
			}
		}
	}

	// Print app role assignment members and the specific role assigned
	apiUrl = ConstMgUrl + "/v1.0/groups/" + id + "/appRoleAssignments"
	appRoleAssignments := GetAzAllPages(apiUrl, z)
	PrintAppRoleAssignmentsOthers(appRoleAssignments, z)

	// Print all groups and roles it is a member of
	apiUrl = ConstMgUrl + "/v1.0/groups/" + id + "/transitiveMemberOf"
	resp, statCode, _ = ApiGet(apiUrl, z, nil)
	if statCode == 200 && resp != nil && resp["value"] != nil {
		if memberOf, ok := resp["value"].([]interface{}); ok {
			PrintMemberOfs(memberOf)
		}
	}

	// Print members of this group
	apiUrl = ConstMgUrl + "/v1.0/groups/" + id + "/members" // beta works
	resp, statCode, _ = ApiGet(apiUrl, z, nil)
	if statCode == 200 && resp != nil && resp["value"] != nil {
		members := resp["value"].([]interface{})
		if len(members) > 0 {
			fmt.Printf("%s:\n", utl.Blu("members"))
			for _, item := range members {
				if member := utl.Map(item); member != nil {
					azObj := AzureObject(member) // Cast map[string]interface{}
					Type, Name := "-", "-"
					Type = utl.LastElem(utl.Str(azObj["@odata.type"]), ".")
					switch Type {
					case "group", "servicePrincipal":
						Name = utl.Str(azObj["displayName"])
					default:
						Name = utl.Str(azObj["userPrincipalName"])
					}
					fmt.Printf("  %-50s %s (%s)\n", utl.Gre(Name),
						utl.Gre(utl.Str(azObj["id"])), utl.Gre(Type))
				}
			}
		}
	}
}

// Lists all cached Privileged Access Groups (PAGs)
func PrintPags(z *Config) {
	groups := GetMatchingDirObjects("g", "", false, z) // false = get from cache, not Azure
	for _, x := range groups {
		if utl.Bool(x["isAssignableToRole"]) {
			PrintTersely("g", x)
		}
	}
}

func PrintCountStatusGroups(z *Config) {
	c1Width := 44 // Column 1 width
	c2Width := 10 // Column 2 width
	c3Width := 10 // Column 3 width
	fmt.Print(utl.Whi2(utl.PostSpc("OBJECTS", c1Width)+
		utl.PreSpc("LOCAL", c2Width)+
		utl.PreSpc("AZURE", c3Width)) + "\n")
	status := utl.Blu(utl.PostSpc("Azure Directory Groups", c1Width))
	localCount := utl.Int2StrWithCommas(ObjectCountLocal("g", z))
	azureCount := utl.Int2StrWithCommas(ObjectCountAzure("g", z))
	status += utl.Gre(utl.PreSpc(localCount, c2Width))
	status += utl.Gre(utl.PreSpc(azureCount, c3Width)) + "\n"
	fmt.Print(status)
}

// Creates or updates an Azure directory group from given command-line arguments.
func UpsertGroupFromArgs(force, isAssignableToRole bool, id, description string, z *Config) {
	// Note that id may be a UUID or a displayName

	// Initialize the obj, and add any user-supplied attributes
	obj := make(AzureObject)
	obj["description"] = description
	obj["isAssignableToRole"] = isAssignableToRole

	x := PreFetchAzureObject(DirectoryGroup, id, z)
	if x != nil {
		// It exists, let's update
		// Note: At the moment, via CLI args, the *only* UPDATEable field is 'description'
		UpdateDirObject(force, utl.Str(x["id"]), obj, DirectoryGroup, z)
	} else {
		// Doesn't exist, let's create
		// Initialize the object with the minimum required attributes for creation.
		obj["displayName"] = id // So id is actually the displayName
		obj["mailEnabled"] = false
		obj["mailNickname"] = "NotSet"
		obj["securityEnabled"] = true
		CreateDirObject(force, obj, DirectoryGroup, z)
	}
}

// Creates or updates an Azure directory group from given specfile.
func UpsertGroupFromFile(force bool, specfile string, z *Config) {
	// Abort if specfile is not YAML
	formatType, mazType, mapObj := GetObjectFromFile(specfile)
	obj := AzureObject(mapObj)
	if formatType != YamlFormat {
		utl.Die("File is not YAML\n")
	}
	// Abort if specfile object isn't valid
	if obj == nil {
		utl.Die("Specfile does not contain a valid directory group definition.\n")
	}
	if mazType != DirectoryGroup {
		utl.Die("Object defined in specfile is not a directory group.\n")
	}

	// Cannot continue without at least a displayName from that specfile
	displayName := utl.Str(obj["displayName"])
	msg := "Specfile object is missing"
	if displayName == "" {
		utl.Die("%s %s\n", msg, utl.Red("displayName"))
	}

	x := PreFetchAzureObject(DirectoryGroup, displayName, z)
	if x != nil {
		// Update if group exists
		UpdateDirObject(force, utl.Str(x["id"]), obj, DirectoryGroup, z)
	} else {
		// Create if group does not exist
		// Set up obj with the minimally required attributes to create a group
		if obj["mailEnabled"] == nil {
			utl.Die("%s %s\n", msg, utl.Red("mailEnabled"))
		}
		if obj["mailNickname"] == nil {
			utl.Die("%s %s\n", msg, utl.Red("mailNickname"))
		}
		if obj["securityEnabled"] == nil {
			utl.Die("%s %s\n", msg, utl.Red("securityEnabled"))
		}
		CreateDirObject(force, obj, DirectoryGroup, z)
	}
}

// Helper function to check if the object is a directory group
func IsDirectoryGroup(obj AzureObject) bool {
	displayName := utl.Str(obj["displayName"])
	mailEnabled := utl.Str(obj["mailEnabled"])
	mailNickname := utl.Str(obj["mailNickname"])
	securityEnabled := utl.Str(obj["securityEnabled"])
	return displayName != "" && mailEnabled != "" && mailNickname != "" && securityEnabled != ""
}
