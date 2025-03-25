package maz

import (
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/queone/utl"
)

// Returns a file name and object name based on the given mazType and name.
func generateName(mazType string, names ...string) (fileName, objName string) {
	// Support function signature variants: if a single name parameter is provided normally,
	// when calling generateName from the calling code in your switch, you can call:
	//    fileName, objName := generateName(mazType, name)
	// where name may be an empty string.

	// If name is empty, it returns the commented-out defaults:
	//
	//	ResRoleDefinition:   file: "rd_specfile.yaml",  obj: "Azure resource role definition"
	//	ResRoleAssignment:   file: "ra_specfile.yaml",  obj: "Azure resource role assignment"
	//	DirectoryGroup:      file: "dg_specfile.yaml",  obj: "Azure directory group"
	//	Application:         file: "ap_specfile.yaml",  obj: "Azure application definition"
	//
	// If name is non-empty, for the file name it prefixes it with "rd_", "ra_", "dg_", or
	// "ap_" based on the type, sanitizes the name by converting spaces and other
	// non-printable characters to underscores, and appends ".yaml". The object name is set
	// to the input name (using its original casing) after validating that
	// all characters are printable and truncating it to 256 characters.

	var name string
	if len(names) > 0 {
		name = names[0]
	}

	// Map the mazType to default file and object names, as well as the file name prefix.
	var defaultFileName, defaultObjName, prefix string
	switch mazType {
	case ResRoleDefinition:
		defaultFileName = "rd_specfile.yaml"
		defaultObjName = "Azure resource role definition"
		prefix = "rd_"
	case ResRoleAssignment:
		defaultFileName = "ra_specfile.yaml"
		defaultObjName = "Azure resource role assignment"
		prefix = "ra_"
	case DirectoryGroup:
		defaultFileName = "dg_specfile.yaml"
		defaultObjName = "Azure directory group"
		prefix = "dg_"
	case Application:
		defaultFileName = "ap_specfile.yaml"
		defaultObjName = "Azure AppSP definition"
		prefix = "ap_"
	default:
		// Unknown mazType, choose a neutral default.
		defaultFileName = "init_specfile.yaml"
		defaultObjName = "Azure object definition"
		prefix = "init_"
	}

	// If name is empty, return the defaults.
	if strings.TrimSpace(name) == "" {
		return defaultFileName, defaultObjName
	}

	// Validate that the input name for objName contains only printable characters.
	// If any non-printable characters are found, exit with an error.
	for _, r := range name {
		if !unicode.IsPrint(r) {
			die("Error: name contains non-printable character: %q\n", r)
		}
	}

	// Prepare fileName by sanitizing the name: convert spaces and any non-printable characters
	// (or characters we do not consider safe in a file name) to underscores.
	safeName := sanitizeFileName(name)

	// Build fileName: prefix + safeName + .yaml
	fileName = prefix + safeName + ".yaml"

	// The object name is the input name with its casing intact, but truncate it if longer than 256 characters.
	if len(name) > 256 {
		objName = name[:256]
	} else {
		objName = name
	}

	return fileName, objName
}

// Replaces spaces and any characters that are not letters, digits, or typical punctuation
// with an underscore. Adjust the allowed character set as needed.
func sanitizeFileName(s string) string {
	var sb strings.Builder
	prevHyphen := false

	for _, r := range s {
		var c rune
		// Allow letters, digits, and a few safe punctuation characters.
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' || r == '.' {
			c = r
		} else if unicode.IsSpace(r) || !unicode.IsPrint(r) {
			// Map spaces and non-printable characters to hyphen
			c = '-'
		} else {
			// For all other characters, convert to hyphen as well.
			c = '-'
		}

		// If we're about to write a hyphen and the previous rune was a hyphen, skip writing it.
		if c == '-' {
			if prevHyphen {
				continue
			}
			prevHyphen = true
		} else {
			prevHyphen = false
		}

		// Write the rune in lower case.
		sb.WriteRune(unicode.ToLower(c))
	}

	return sb.String()
}

// Creates specfile skeleton/scaffold files
func CreateSkeletonFile(mazType, name string) {
	pwd, err := os.Getwd()
	if err != nil {
		die("%s Error getting current working directory.\n", utl.Trace())
	}
	fileName, objName, fileContent := "", "", []byte("")
	switch mazType {
	case ResRoleDefinition:
		fileName, objName = generateName(mazType, name)
		fileContent = []byte("#\n" +
			"# Example Azure resource role definition specfile object definition\n" +
			"#\n" +
			"properties:\n" +
			"  roleName: " + objName + "\n" +
			"  description: Description of what this role does.\n" +
			"  assignableScopes:\n" +
			"    # Recommendation: Always define at highest point in hierarchy, the Tenant Root Group.\n" +
			"    - /providers/Microsoft.Management/managementGroups/3f550b9f-8888-7777-ad61-111199992222\n" +
			"  permissions:\n" +
			"    - actions:\n" +
			"        - Microsoft.DevCenter/projects/*/read\n" +
			"        - '*/read'       # Wrap leading asterik entries in single-quotes\n" +
			"      notActions:\n" +
			"        - Microsoft.DevCenter/projects/pools/read\n" +
			"      dataActions:\n" +
			"        - Microsoft.KeyVault/vaults/secrets/*\n" +
			"      notDataActions:\n" +
			"        - Microsoft.CognitiveServices/accounts/LUIS/apps/delete\n")
	case ResRoleAssignment:
		fileName, _ = generateName(mazType, name)
		fileContent = []byte("#\n" +
			"# Example Azure resource role assignment specfile object definition\n" +
			"#\n" +
			"# Only three parameters are mandatory: The principal ID for the group, user, or SP that needs\n" +
			"# the access; the roleDefinitionId for the privilege being granted; and the scope within the\n" +
			"# Azure resource hierarchy where the access is being granted:\n" +
			"#\n" +
			"properties:\n" +
			"  principalId: 65c6427a-1111-5555-7777-274d26531314  # Group = \"My Special Group\"\n" +
			"  roleDefinitionId: 2489dfa4-3333-4444-9999-b04b7a1e4ea6  # Role = \"My Special Role\"\n" +
			"  scope: /providers/Microsoft.Management/managementGroups/3f550b9f-8888-7777-ad61-111199992222\n")
	case DirectoryGroup:
		fileName, objName = generateName(mazType, name)
		fileContent = []byte("#\n" +
			"# Example Azure directory group specfile object definition\n" +
			"#\n" +
			"# First four parameters are mandatory, but there are many other options:\n" +
			"# See https://learn.microsoft.com/en-us/graph/api/resources/group?view=graph-rest-1.0#properties\n" +
			"#\n" +
			"displayName: " + objName + "\n" +
			"mailEnabled: false\n" +
			"mailNickname: NotSet\n" +
			"securityEnabled: true\n" +
			"description: Group description\n" +
			"isAssignableToRole: false\n")
	case Application:
		fileName, objName = generateName(mazType, name)
		fileContent = []byte("#\n" +
			"# Example Azure App registration & corresponding Service Principal (SP) specfile objects definition\n" +
			"#\n" +
			"# Mandatory Parameters:\n" +
			"#   - displayName: The display name of the App/SP pair\n" +
			"#   - signInAudience: Accounts these App/SP objects are limited to\n" +
			"#\n" +
			"# Optional Parameters:\n" +
			"#   For a full list of available parameters see respective Microsoft Graph API pages:\n" +
			"#   - Application: https://learn.microsoft.com/en-us/graph/api/resources/application?view=graph-rest-1.0#properties\n" +
			"#   - Service Principal: https://learn.microsoft.com/en-us/graph/api/resources/servicePrincipal?view=graph-rest-1.0#properties\n" +
			"#\n" +
			"displayName: " + objName + "\n" +
			"signInAudience: AzureADMyOrg\n")
	}
	specfile := filepath.Join(pwd, fileName)
	if utl.FileExist(specfile) {
		die("Error: File %s already exists.\n", fileName)
	}
	f, err := os.Create(specfile) // Create the file
	if err != nil {
		die("Error creating file: %s", err)
	}
	defer f.Close()
	f.Write(fileContent) // Write the content
	os.Exit(0)
}
