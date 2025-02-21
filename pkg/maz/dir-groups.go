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
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(x["displayName"].(string)))
	if x["description"] != nil {
		fmt.Printf("%s: %s\n", utl.Blu("description"), utl.Gre(x["description"].(string)))
	}
	if x["isAssignableToRole"] != nil {
		fmt.Printf("%s: %s\n", utl.Blu("isAssignableToRole"), utl.Mag(x["isAssignableToRole"].(bool)))
	}

	// Print owners of this group
	apiUrl := ConstMgUrl + "/v1.0/groups/" + id + "/owners"
	r, statusCode, _ := ApiGet(apiUrl, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		owners := r["value"].([]interface{}) // Assert as JSON array type
		if len(owners) > 0 {
			fmt.Printf("%s:\n", utl.Blu("owners"))
			for _, i := range owners {
				if mapObj, ok := i.(map[string]interface{}); ok {
					o := AzureObject(mapObj) // Convert map[string]interface{} to AzureObject
					fmt.Printf("  %-50s %s\n", utl.Gre(o["userPrincipalName"].(string)), utl.Gre(o["id"].(string)))
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
	r, statusCode, _ = ApiGet(apiUrl, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		memberOf := r["value"].([]interface{})
		PrintMemberOfs("g", memberOf)
	}

	// Print members of this group
	apiUrl = ConstMgUrl + "/v1.0/groups/" + id + "/members" // beta works
	r, statusCode, _ = ApiGet(apiUrl, z, nil)
	if statusCode == 200 && r != nil && r["value"] != nil {
		members := r["value"].([]interface{})
		if len(members) > 0 {
			fmt.Printf("%s:\n", utl.Blu("members"))
			for _, i := range members {
				if mapObj, ok := i.(map[string]interface{}); ok {
					m := AzureObject(mapObj) // Convert map[string]interface{} to AzureObject
					Type, Name := "-", "-"
					Type = utl.LastElem(m["@odata.type"].(string), ".")
					switch Type {
					case "group", "servicePrincipal":
						Name = m["displayName"].(string)
					default:
						Name = m["userPrincipalName"].(string)
					}
					fmt.Printf("  %-50s %s (%s)\n", utl.Gre(Name), utl.Gre(m["id"].(string)), utl.Gre(Type))
				}
			}
		}
	}
}

// Lists all cached Privileged Access Groups (PAGs)
func PrintPags(z *Config) {
	groups := GetMatchingObjects("g", "", false, z) // Get all groups, false = don't hit Azure
	for _, x := range groups {
		if utl.Bool(x["isAssignableToRole"]) {
			PrintTersely("g", x)
		}
	}
}

func PrintCountStatusGroups(z *Config) {
	fmt.Printf("%-36s%16s%16s\n", "OBJECTS", "LOCAL", "AZURE")
	status := utl.Blu(utl.PostSpc("Azure Directory Groups", 36))
	localCount := utl.Int2StrWithCommas(ObjectCountLocal("g", z))
	azureCount := utl.Int2StrWithCommas(ObjectCountAzure("g", z))
	status += utl.Gre(utl.PreSpc(localCount, 16))
	status += utl.Gre(utl.PreSpc(azureCount, 16)) + "\n"
	fmt.Print(status)
}

// Creates or updates an Azure directory group from given command-line arguments.
func UpsertGroupFromArgs(opts *Options, z *Config) {
	force, _ := opts.GetBool("force")
	id, _ := opts.GetString("id") // Note that id may be a UUID or a displayName
	description, descriptionSet := opts.GetString("description")
	isAssignableToRole, isAssignableToRoleSet := opts.GetBool("isAssignableToRole")

	// Initialize the obj, and add any user-supplied attributes
	obj := make(AzureObject)
	if descriptionSet {
		obj["description"] = description
	}
	if isAssignableToRoleSet {
		obj["isAssignableToRole"] = isAssignableToRole
	}

	x := PreFetchAzureObject("g", id, z)
	if x != nil {
		// It exists, let's update
		// Note: At the moment, via CLI args, the *only* UPDATEable field is 'description'
		UpdateDirObject(force, x["id"].(string), obj, "g", z)
	} else {
		// Doesn't exist, let's create
		// Initialize the object with the minimum required attributes for creation.
		obj["displayName"] = id // So id is actually the displayName
		obj["mailEnabled"] = false
		obj["mailNickname"] = "NotSet"
		obj["securityEnabled"] = true
		CreateDirObject(force, obj, "g", z)
	}
}

// Creates or updates an Azure directory group from given specfile.
func UpsertGroupFromFile(opts *Options, z *Config) {
	force, _ := opts.GetBool("force")
	filePath, _ := opts.GetString("filePath")

	// Abort if specfile is not YAML
	formatType, t, mapObj := GetObjectFromFile(filePath)
	obj := AzureObject(mapObj)
	if formatType != "YAML" {
		utl.Die("File is not YAML\n")
	}
	// Abort if specfile object isn't valid
	if obj == nil {
		utl.Die("Specfile does not contain a valid directory group definition.\n")
	}
	if t != "g" {
		utl.Die("Object defined in specfile is not a directory group.\n")
	}

	// Cannot continue without at least a displayName from that specfile
	displayName := utl.Str(obj["displayName"])
	msg := "Specfile object is missing"
	if displayName == "" {
		utl.Die("%s %s\n", msg, utl.Red("displayName"))
	}

	x := PreFetchAzureObject("g", displayName, z)
	if x == nil {
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

		CreateDirObject(force, obj, "g", z)
	} else {
		// Update if group exists
		id := utl.Str(x["id"])
		UpdateDirObject(force, id, obj, "g", z)
	}
}
