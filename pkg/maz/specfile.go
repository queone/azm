package maz

import (
	"fmt"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/queone/utl"
)

// Generates and prints a sanitized specfile name from given specfile or ID
func GenerateAndPrintSpecfileName(specifier string, z *Config) {
	var mazType string
	var obj AzureObject

	// Determine the specifier type
	if utl.FileUsable(specifier) {
		// If it's a specfile, try to get the mazType and object
		_, mazType, obj = GetObjectFromFile(specifier)
	} else if utl.ValidUuid(specifier) {
		// If it's an ID, get the mazType and object of all matching objects
		list, _ := FindAzureObjectsById(specifier, z)
		if len(list) == 0 {
			utl.Die("There's no object with that ID\n")
		} else if len(list) > 1 {
			utl.Die("Too many objects with that ID. This is not supported.\n")
		}
		obj = list[0] // Isolate the single object
		mazType = utl.Str(obj["maz_type"])
	} else {
		utl.Die("Invalid specfile or ID\n")
	}

	var specfileName string
	var err error

	switch mazType {
	case ResRoleDefinition:
		props := utl.Map(obj["properties"])
		roleName := utl.Str(props["roleName"])
		part2 := sanitizePart(roleName)
		specfileName = fmt.Sprintf("%s_%s.yaml", mazType, part2)

	case ResRoleAssignment:
		props := utl.Map(obj["properties"])

		// Get the name of the role, and sanitize as part3
		roleDefinitionId := utl.Str(props["roleDefinitionId"])
		baseId := path.Base(roleDefinitionId) // We only care about the UUID part
		roleName := GetObjectNameFromId(ResRoleDefinition, baseId, z)
		part3 := sanitizePart(roleName)
		if part3 == "" {
			part3 = "error"
		}

		// Get the name of principal, and sanitize as part2
		principalId := utl.Str(props["principalId"])
		principalName := GetObjectNameFromId(DirectoryGroup, principalId, z)
		if principalName == "" {
			principalName = GetObjectNameFromId(DirectoryUser, principalId, z)
		}
		if principalName == "" {
			principalName = GetObjectNameFromId(ServicePrincipal, principalId, z)
		}
		part2 := sanitizePart(principalName)
		if part2 == "" {
			part2 = "error"
		}

		// Sanitize the scope as part1
		scope := utl.Str(props["scope"])
		part1 := getScopeName(scope, z)
		part1 = sanitizePart(part1)
		if part1 == "" {
			part1 = "error"
		}

		specfileName = fmt.Sprintf("%s_%s_%s.yaml", part1, part2, part3)

	case DirectoryGroup, Application, ServicePrincipal,
		DirRoleDefinition, DirRoleAssignment:
		displayName := utl.Str(obj["displayName"])
		part2 := sanitizePart(displayName)
		specfileName = fmt.Sprintf("%s_%s.yaml", mazType, part2)

	default:
		utl.Die("Can't determine object type for this specfile\n")
	}

	// Print the generated file name
	fmt.Printf("Recommended specfile name = %s\n", utl.Yel(specfileName))

	// Save the object in new recommended specfile name
	if utl.FileUsable(specfileName) {
		fmt.Println("Above file already exists. Content not overwritten.")
	} else {
		if err = utl.SaveFileAuto(specfileName, "yaml", obj, false, 0); err != nil {
			utl.Die("Error saving specfile: %v\n", err)
		}
		fmt.Println("Above file has been created with the object's content.")
	}
}

func sanitizePart(s string) string {
	s = strings.TrimSpace(s)
	var sb strings.Builder
	prevHyphen := false

	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			sb.WriteRune(unicode.ToLower(r))
			prevHyphen = false
		case unicode.IsSpace(r), !unicode.IsPrint(r):
			if !prevHyphen {
				sb.WriteRune('-')
				prevHyphen = true
			}
		default:
			if !prevHyphen {
				sb.WriteRune('-')
				prevHyphen = true
			}
		}
	}

	return strings.Trim(sb.String(), "-")
}

// getScopeName determines the appropriate name part based on the scope
func getScopeName(scope string, z *Config) string {
	// Check if it's a management group
	if name := getManagementGroupName(scope); name != "" {
		return name
	}

	// Check if it's a subscription
	if strings.HasPrefix(scope, "/subscriptions/") {
		subId := path.Base(scope)
		subName := GetObjectNameFromId(Subscription, subId, z)
		return sanitizePart(subName)
	}

	// Default case
	return sanitizePart(scope)
}

// getManagementGroupName determines the appropriate name part based on the scope
func getManagementGroupName(scope string) string {
	if strings.HasPrefix(scope, "/providers/Microsoft.Management/managementGroups/") {
		// Check if it's a root management group
		if regexp.MustCompile(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`).MatchString(scope) {
			return "mg-root"
		}
		// If it's not the root, return the base name
		return sanitizePart(path.Base(scope))
	}
	return ""
}
