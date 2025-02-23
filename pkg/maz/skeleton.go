package maz

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/queone/utl"
)

// Creates specfile skeleton/scaffold files
func CreateSkeletonFile(t string) {
	pwd, err := os.Getwd()
	if err != nil {
		utl.Die("%s Error getting current working directory.\n", utl.Trace())
	}
	fileName, fileContent := "init-file-name.extension", []byte("init-file-content\n")
	switch t {
	case "d":
		fileName = "resource-role-definition.yaml"
		fileContent = []byte("properties:\n" +
			"  roleName: My RBAC Role\n" +
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
	case "a":
		fileName = "resource-role-assignment.yaml"
		fileContent = []byte("properties:\n" +
			"  principalId: 65c6427a-1111-5555-7777-274d26531314  # Group = \"My Special Group\"\n" +
			"  roleDefinitionId: 2489dfa4-3333-4444-9999-b04b7a1e4ea6  # Role = \"My Special Role\"\n" +
			"  scope: /providers/Microsoft.Management/managementGroups/3f550b9f-8888-7777-ad61-111199992222\n")
	case "g":
		fileName = "directory-group.yaml"
		fileContent = []byte("# Minimal specfile to create an Azure directory group\n" +
			"displayName: My Group\n" +
			"mailEnabled: false\n" +
			"mailNickname: NotSet\n" +
			"securityEnabled: true\n" +
			"# First 4 are mandatory, but there are many other options.\n" +
			"# See https://learn.microsoft.com/en-us/graph/api/resources/group?view=graph-rest-1.0#properties\n" +
			"description: My Group description\n" +
			"isAssignableToRole: false\n")
	case "ap":
		fileName = "appsp.yaml"
		fileContent = []byte("# Azure AppSP Pair SpecFile\n" +
			"#\n" +
			"# This file defines the configuration for creating an Azure App registration and a\n" +
			"# corresponding Service Principal.\n" +
			"#\n" +
			"# Required Parameters:\n" +
			"#   - displayName: The display name of the application.\n" +
			"#   - signInAudience: Accounts this App/SP pair is limited to. See MS Graph documentation.\n" +
			"#\n" +
			"# Optional Parameters:\n" +
			"#   For a full list of available properties, please refer to the Microsoft Graph API\n" +
			"#   documentation:\n" +
			"#   - Application: https://learn.microsoft.com/en-us/graph/api/resources/application?view=graph-rest-1.0#properties\n" +
			"#   - Service Principal: https://learn.microsoft.com/en-us/graph/api/resources/servicePrincipal?view=graph-rest-1.0#properties\n" +
			"#\n" +
			"displayName: My App           # Define the display name of the application (required)\n" +
			"signInAudience: AzureADMyOrg  # Determines which users can sign in to an application (required)\n" +
			"#\n" +
			"# Additional properties can be added as needed, such as:\n" +
			"# identifierUri: The URI used to identify the application\n" +
			"# replyUrls: A list of URLs that the application can redirect to after authentication\n")
	}
	filePath := filepath.Join(pwd, fileName)
	if utl.FileExist(filePath) {
		utl.Die("Error: File %s already exists.\n", fileName)
	}
	f, err := os.Create(filePath) // Create the file
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()
	f.Write(fileContent) // Write the content
	os.Exit(0)
}
