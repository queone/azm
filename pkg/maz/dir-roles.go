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
	fmt.Printf("%s\n", utl.Gra("# Directory Role Definition"))
	fmt.Printf("%s: %s\n", utl.Blu("object_id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("display_name"), utl.Gre(utl.Str(x["displayName"])))
	fmt.Printf("%s: %s\n", utl.Blu("description"), utl.Gre(utl.Str(x["description"])))

	// List permissions
	if x["rolePermissions"] != nil {
		rolePerms := x["rolePermissions"].([]interface{})
		if len(rolePerms) > 0 {
			perms := rolePerms[0].(map[string]interface{})
			allowedResourceActions, ok := perms["allowedResourceActions"].([]interface{})
			count := len(allowedResourceActions)
			if ok && count > 0 {
				fmt.Printf("%s:\n", utl.Blu("permissions"))
				limit := 10 // Let's at max 10 perms
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
	// https://learn.microsoft.com/en-us/azure/active-directory/roles/view-assignments
	// https://github.com/microsoftgraph/microsoft-graph-docs-contrib/blob/main/api-reference/v1.0/api/rbacapplication-list-roleassignments.md
	params := map[string]string{
		"$filter": "roleDefinitionId eq '" + utl.Str(x["templateId"]) + "'",
		"$expand": "principal",
	}
	apiUrl := ConstMgUrl + "/v1.0/roleManagement/directory/roleAssignments"
	r, statusCode, _ := ApiGet(apiUrl, z, params)
	if statusCode == 200 && r != nil && r["value"] != nil {
		assignments := r["value"].([]interface{})
		if len(assignments) > 0 {
			fmt.Printf("%s:\n", utl.Blu("assignments"))
			//utl.PrintJsonColor(assignments) // DEBUG
			for _, i := range assignments {
				m := i.(map[string]interface{})
				principalId := utl.Str(m["principalId"])
				scope := utl.Str(m["directoryScopeId"])
				// TODO: Find out how to get/print the scope displayName?
				mPrinc := m["principal"].(map[string]interface{})
				pName := utl.Str(mPrinc["displayName"])
				pType := utl.LastElemByDot(utl.Str(mPrinc["@odata.type"]))
				fmt.Printf("  %-36s  %-50s  %-36s (%s)\n", utl.Gre(scope), utl.Gre(pName),
					utl.Gre(principalId), utl.Gre(pType))
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
	fmt.Printf("%s\n", utl.Gra("# Directory Role Assignment"))
	fmt.Printf("%s: %s\n", utl.Blu("object_id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("directory_scope_id"), utl.Gre(utl.Str(x["directoryScopeId"])))
	fmt.Printf("%s: %s\n", utl.Blu("principal_id"), utl.Gre(utl.Str(x["principalId"])))
	fmt.Printf("%s: %s\n", utl.Blu("role_definition_d"), utl.Gre(utl.Str(x["roleDefinitionId"])))
}

// Returns count of Azure AD directory role entries in current tenant
func AdRolesCountAzure(z *Config) int64 {
	// Note that endpoint "/v1.0/directoryRoles" is for Activated AD roles, so it wont give us
	// the full count of all AD roles. Also, the actual role definitions, with what permissions
	// each has is at endpoint "/v1.0/roleManagement/directory/roleDefinitions", but because
	// we only care about their count it is easier to just call end point
	// "/v1.0/directoryRoleTemplates" which is a quicker API call and has the accurate count.
	// It's not clear why MSFT makes this so darn confusing.
	apiUrl := ConstMgUrl + "/v1.0/directoryRoleTemplates"
	r, _, _ := ApiGet(apiUrl, z, nil)
	if r["value"] != nil {
		return int64(len(r["value"].([]interface{})))
	}
	return 0
}
