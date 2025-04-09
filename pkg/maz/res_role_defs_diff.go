package maz

import (
	"fmt"

	"github.com/queone/utl"
)

// Status indicators for DiffLists results
const (
	StaysSame   = "StaysSame"   // Item exists in both lists
	ToBeRemoved = "ToBeRemoved" // Item exists in list2 (Azure) but not in list1 (specfile)
	ToBeAdded   = "ToBeAdded"   // Item exists in list1 (specfile) but not in list2 (Azure)
)

// DiffLists compares two lists of strings and returns a map with the string entry and its status.
func DiffLists(list1, list2 []interface{}) map[string]string {
	result := make(map[string]string)

	// Create sets for quick lookup
	set1 := make(map[string]bool)
	for _, i := range list1 {
		set1[utl.Str(i)] = true
	}
	set2 := make(map[string]bool)
	for _, i := range list2 {
		set2[utl.Str(i)] = true
	}

	// Check items in list2 (Azure)
	for _, i := range list2 {
		key := utl.Str(i)
		if set1[key] {
			result[key] = StaysSame // Item exists in both lists
		} else {
			result[key] = ToBeRemoved // Item exists in list2 but not in list1
		}
	}

	// Check items in list1 (specfile)
	for _, i := range list1 {
		key := utl.Str(i)
		if !set2[key] {
			result[key] = ToBeAdded // Item exists in list1 but not in list2
		}
	}

	return result
}

// Prints differences between the two role definition objects. The caller
// should have already validated that all the respective fields exist within
// each object.
func DiffRoleDefinitionSpecfileVsAzure(obj, azureObj map[string]interface{}) {
	// Gather the SPECFILE object values
	objProps := utl.Map(obj["properties"])
	objDesc := utl.Str(objProps["description"])
	objScopes := utl.Slice(objProps["assignableScopes"])
	objPermSet := utl.Slice(objProps["permissions"])
	objPerms := utl.Map(objPermSet[0])
	//----
	objActions := utl.Slice(objPerms["actions"])
	objNotActions := utl.Slice(objPerms["notActions"])
	objDataActions := utl.Slice(objPerms["dataActions"])
	objNotDataActions := utl.Slice(objPerms["notDataActions"])

	// Gather the Azure object values, using casting functions
	azureId := utl.Str(azureObj["name"])
	azureProps := utl.Map(azureObj["properties"])
	azureRoleName := utl.Str(azureProps["roleName"])
	azureDesc := utl.Str(azureProps["description"])
	azureScopes := utl.Slice(azureProps["assignableScopes"])
	azurePermSet := utl.Slice(azureProps["permissions"])
	azurePerms := utl.Map(azurePermSet[0])
	//----
	azureActions := utl.Slice(azurePerms["actions"])
	azureNotActions := utl.Slice(azurePerms["notActions"])
	azureDataActions := utl.Slice(azurePerms["dataActions"])
	azureNotDataActions := utl.Slice(azurePerms["notDataActions"])

	// Display differences
	fmt.Println("Note the color coding below for what the changes in this specfile will do.")
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(azureId))
	fmt.Printf("%s: \n", utl.Blu("properties"))
	fmt.Printf("  %s: %s\n", utl.Blu("roleName"), utl.Gre(azureRoleName))

	fmt.Printf("  %s: %s\n", utl.Blu("description"), utl.Gre(azureDesc))
	if objDesc != azureDesc {
		fmt.Printf("  %s: %s\n", utl.Blu("description"), utl.Red(objDesc))
	}

	toBeRemoved := utl.Gra("# To be removed")
	toBeAdded := utl.Gra("# To be added")

	// scopes
	fmt.Printf("  %s:\n", utl.Blu("assignableScopes"))
	diff := DiffLists(objScopes, azureScopes)

	for key, status := range diff {
		switch status {
		case StaysSame:
			fmt.Printf("    - %s\n", utl.Gre(key))
		case ToBeRemoved:
			fmt.Printf("    - %-104s  %s\n", utl.Red(key), toBeRemoved)
		case ToBeAdded:
			fmt.Printf("    - %-104s  %s\n", utl.Mag(key), toBeAdded)
		}
	}

	// permissions
	fmt.Printf("  %s:\n", utl.Blu("permissions"))

	// actions
	if len(objActions) > 0 || len(azureActions) > 0 {
		fmt.Printf("    - %s:\n", utl.Blu("actions"))
		diff := DiffLists(objActions, azureActions)

		for key, status := range diff {
			s := utl.StrSingleQuote(key)
			switch status {
			case StaysSame:
				fmt.Printf("        - %-100s\n", utl.Gre(s))
			case ToBeRemoved:
				fmt.Printf("        - %-100s  %s\n", utl.Red(s), toBeRemoved)
			case ToBeAdded:
				fmt.Printf("        - %-100s  %s\n", utl.Mag(s), toBeAdded)
			}
		}
	}

	// notActions
	if len(objNotActions) > 0 || len(azureNotActions) > 0 {
		fmt.Printf("      %s:\n", utl.Blu("notActions"))
		diff := DiffLists(objNotActions, azureNotActions)

		for key, status := range diff {
			s := utl.StrSingleQuote(key)
			switch status {
			case StaysSame:
				fmt.Printf("        - %s\n", utl.Gre(s))
			case ToBeRemoved:
				fmt.Printf("        - %-100s  %s\n", utl.Red(s), toBeRemoved)
			case ToBeAdded:
				fmt.Printf("        - %-100s  %s\n", utl.Mag(s), toBeAdded)
			}
		}
	}

	// dataActions
	if len(objDataActions) > 0 || len(azureDataActions) > 0 {
		fmt.Printf("      %s:\n", utl.Blu("dataActions"))
		diff := DiffLists(objDataActions, azureDataActions)

		for key, status := range diff {
			s := utl.StrSingleQuote(key)
			switch status {
			case StaysSame:
				fmt.Printf("        - %s\n", utl.Gre(s))
			case ToBeRemoved:
				fmt.Printf("        - %-100s  %s\n", utl.Red(s), toBeRemoved)
			case ToBeAdded:
				fmt.Printf("        - %-100s  %s\n", utl.Mag(s), toBeAdded)
			}
		}
	}

	// notDataActions
	if len(objNotDataActions) > 0 || len(azureNotDataActions) > 0 {
		fmt.Printf("      %s:\n", utl.Blu("notDataActions"))
		diff := DiffLists(objNotDataActions, azureNotDataActions)

		for key, status := range diff {
			s := utl.StrSingleQuote(key)
			switch status {
			case StaysSame:
				fmt.Printf("        - %s\n", utl.Gre(s))
			case ToBeRemoved:
				fmt.Printf("        - %-100s  %s\n", utl.Red(s), toBeRemoved)
			case ToBeAdded:
				fmt.Printf("        - %-100s  %s\n", utl.Mag(s), toBeAdded)
			}
		}
	}
}
