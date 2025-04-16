package maz

import (
	"fmt"

	"github.com/queone/utl"
)

// Prints Azure directory role definition object in YAML-like format
func PrintDirRoleDefinition(x AzureObject, z *Config) {
	id := utl.Str(x["id"])
	if id == "" {
		return
	}

	// Print the most important attributes first
	fmt.Printf("%s\n", utl.Gra("# Directory role definition"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(utl.Str(x["displayName"])))
	fmt.Printf("%s: %s\n", utl.Blu("description"), utl.Gre(utl.Str(x["description"])))

	// List permissions
	rolePermissions := utl.Slice(x["rolePermissions"])
	if len(rolePermissions) > 0 {
		if perms := utl.Map(rolePermissions[0]); perms != nil {
			allowedResourceActions := utl.Slice(perms["allowedResourceActions"])
			count := len(allowedResourceActions)
			if count > 0 {
				fmt.Printf("%s:\n", utl.Blu("permissions"))
				limit := 10 // Print maximum of 10 permissions
				for i, v := range allowedResourceActions {
					if i >= limit {
						break
					}
					fmt.Printf("  %s\n", utl.Gre(utl.Str(v)))
				}
				if count > 10 {
					msg := fmt.Sprintf("# Shorten to 10 entries -- to see all %d entries, view the JSON object with '-drj'", count)
					fmt.Printf("  %s\n", utl.Gra(msg))
				}
			}
		}
	}

	// Print assignments
	// learn.microsoft.com/en-us/azure/active-directory/roles/view-assignments
	// github.com/microsoftgraph/microsoft-graph-docs-contrib/blob/main/api-reference/v1.0/api/rbacapplication-list-roleassignments.md
	params := map[string]string{
		"$filter": "roleDefinitionId eq '" + utl.Str(x["templateId"]) + "'",
		"$expand": "principal",
	}
	apiUrl := ConstMgUrl + "/v1.0/roleManagement/directory/roleAssignments"
	resp, statCode, _ := ApiGet(apiUrl, z, params)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	assignments := utl.Slice(resp["value"])
	if len(assignments) > 0 {
		fmt.Printf("%s:\n", utl.Blu("assignments"))
		for _, item := range assignments {
			if asgn := utl.Map(item); asgn != nil {
				principalId := utl.Str(asgn["principalId"])
				scope := utl.Str(asgn["directoryScopeId"])
				// TODO: Find out how to get/print the scope displayName?
				if mPrinc := utl.Map(asgn["principal"]); mPrinc != nil {
					pName := utl.Str(mPrinc["displayName"])
					pType := utl.LastElemByDot(utl.Str(mPrinc["@odata.type"]))
					fmt.Printf("  %-36s  %-50s  %-36s (%s)\n", utl.Gre(scope), utl.Gre(pName),
						utl.Gre(principalId), utl.Gre(pType))
				}
			}
		}
	}
}

// Prints Azure directory role assignment object in YAML-like format
func PrintDirRoleAssignment(x AzureObject, z *Config) {
	id := utl.Str(x["id"])
	if id == "" {
		return
	}

	// Print the most important attributes first
	fmt.Printf("%s\n", utl.Gra("# Directory role assignment"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("directoryScopeId"), utl.Gre(utl.Str(x["directoryScopeId"])))
	fmt.Printf("%s: %s\n", utl.Blu("principalId"), utl.Gre(utl.Str(x["principalId"])))
	fmt.Printf("%s: %s\n", utl.Blu("roleDefinitionId"), utl.Gre(utl.Str(x["roleDefinitionId"])))
}

// Returns count of Azure AD directory role entries in current tenant
func AdRolesCountAzure(z *Config) int64 {
	// Note that endpoint "/v1.0/directoryRoles" is for Activated AD roles, so it wont give us
	// the full count of all AD roles. Also, the actual role definitions, with what permissions
	// each has, is at endpoint "/v1.0/roleManagement/directory/roleDefinitions", but because
	// we only care about their count it is easier to just call end point
	// "/v1.0/directoryRoleTemplates" which is a quicker API call and has the accurate count.
	// It's not clear why this has been made this confusing.
	apiUrl := ConstMgUrl + "/v1.0/directoryRoleTemplates"
	resp, statCode, _ := ApiGet(apiUrl, z, nil)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	dirRoles := utl.Slice(resp["value"])
	return int64(len(dirRoles))
}
