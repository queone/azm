package maz

import (
	"fmt"
	"strings"

	"github.com/queone/utl"
)

// Prints service principal object in YAML-like format
func PrintSp(x AzureObject, z *Config) {
	id := utl.Str(x["id"])
	if id == "" {
		return
	}

	// Print the most important attributes first
	displayName := utl.Str(x["displayName"])
	fmt.Printf("%s\n", utl.Gra("# Service principal"))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(displayName))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("appId"), utl.Gre(utl.Str(x["appId"])))

	// Print certificates details
	apiUrl := ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/keyCredentials"
	resp, _, _ := ApiGet(apiUrl, z, nil)
	keyCredentials := utl.Slice(resp["value"])
	PrintCertificateList(keyCredentials)

	// Print secrets details
	apiUrl = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/passwordCredentials"
	resp, _, _ = ApiGet(apiUrl, z, nil)
	passwordCredentials := utl.Slice(resp["value"])
	PrintSecretList(passwordCredentials)

	// Print owners
	apiUrl = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/owners"
	resp, _, _ = ApiGet(apiUrl, z, nil)
	owners := utl.Slice(resp["value"])
	PrintOwners(owners)

	// Below loop does 2 things:
	// 1. Prints the SP's all app roles
	// 2. Creates an role:name map to use later when calling PrintAppRoleAssignments()
	roleNameMap := make(map[string]string)
	roleNameMap["00000000-0000-0000-0000-000000000000"] = "Default" // Include default app permissions role
	appRoles := utl.Slice(x["appRoles"])
	if len(appRoles) > 0 {
		fmt.Printf("%s:\n", utl.Blu("app_roles"))
		for _, item := range appRoles {
			if appRole := utl.Map(item); appRole != nil {
				rId := utl.Str(appRole["id"])
				roleName := utl.Str(appRole["displayName"])
				roleNameMap[rId] = roleName // Update growing list of roleNameMap
				if len(roleName) >= 60 {
					roleName = roleName[:57] + "..." // Shorten roleName for nicer printout
				}
				fmt.Printf("  %s  %-50s  %-60s\n", utl.Gre(rId),
					utl.Gre(utl.Str(appRole["value"])), utl.Gre(roleName))
			}
		}
	}

	// Print app role assignment members and the specific role assigned
	apiUrl = ConstMgUrl + "/beta/servicePrincipals/" + id + "/appRoleAssignedTo"
	appRoleAssignedTo := GetAzureAllPages(apiUrl, z)
	PrintAppRoleAssignmentsSp(roleNameMap, appRoleAssignedTo) // roleNameMap is used here

	// Prints groups and roles it is a member of
	apiUrl = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/transitiveMemberOf"
	resp, _, _ = ApiGet(apiUrl, z, nil)
	memberOf := utl.Slice(resp["value"])
	PrintMemberOfs(memberOf)

	// Print API permissions that have been granted admin consent
	// ======================================================================
	// - https://learn.microsoft.com/en-us/entra/identity-platform/app-objects-and-service-principals?tabs=browser
	// - https://learn.microsoft.com/en-us/entra/identity-platform/permissions-consent-overview
	// This is a bit long-winded and requires to major subsections for gathering both Delegated and
	// Application type admin consents ...
	var apiPerms [][]string = [][]string{}

	// 1st, let us gather any 'Delegated' type permission admin grants
	params := map[string]string{"$filter": "clientId eq '" + id + "'"}
	apiUrl = ConstMgUrl + "/v1.0/oauth2PermissionGrants"
	resp, _, _ = ApiGet(apiUrl, z, params)
	oauth2PermissionGrants := utl.Slice(resp["value"])

	// IMPORTANT: Please read this carefully -- not as obvious as it seems -- if no admin grants
	// have been done for any assigned 'Delegated' type permission for this clientId, then above
	// call will return nothing. Again, above is looking for DELEGATED type grants! Note also
	// that 'clientId' refers to the 'Object ID' of the SP in question. Moreover, the call is
	// for ALL Delegated permissions in the ENTIRE tenant.

	if len(oauth2PermissionGrants) > 0 {
		// Collate OAuth 2.0 scope permission admin grants
		for _, item := range oauth2PermissionGrants {
			if api := utl.Map(item); api != nil {
				oauthId := utl.Str(api["id"])
				resourceId := utl.Str(api["resourceId"]) // Get API's SP to get its displayName and claim values
				apiUrl2 := ConstMgUrl + "/v1.0/servicePrincipals/" + resourceId
				r2, _, _ := ApiGet(apiUrl2, z, nil)

				apiName := "Unknown"
				if r2["displayName"] != nil {
					apiName = utl.Str(r2["displayName"])
				}
				// Collect each Delegated claim value for this permission
				scope := strings.TrimSpace(utl.Str(api["scope"]))
				scopeValues := strings.Split(scope, " ")
				for _, claim := range scopeValues {
					// Keep growing the list of api permission grants
					apiPerms = append(apiPerms, []string{oauthId, apiName, "Delegated", claim})
				}
			}
		}
	}

	// 2nd, let us gather any 'Application' type permission admin grants
	apiUrl = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "/appRoleAssignments"
	resp, _, _ = ApiGet(apiUrl, z, nil)
	appRoleAssignments := utl.Slice(resp["value"])

	// IMPORTANT: Again, read this carefully -- not as obvious as it seems -- if no admin grants
	// have been done for any assigned 'Application' type permission for this SP, then above API
	// call will return nothing. And again, above is looking *only* for APPLICATION type grants.

	if len(appRoleAssignments) > 0 {
		// Create temporary map of role Ids to role values
		roleIdValueMap := make(map[string]string)
		uniqueResourceIds := utl.StringSet{} // Unique resourceIds (API SPs)
		for _, item := range appRoleAssignments {
			if api := utl.Map(item); api != nil {
				resourceId := utl.Str(api["resourceId"]) // Get API's SP, to then fetch the role's claim value

				// Skip processing if this resourceId (this API SP) has already been seen
				if !uniqueResourceIds.Exists(resourceId) {
					continue
				}

				// Map each role ID to its claim value
				apiUrl2 := ConstMgUrl + "/v1.0/servicePrincipals/" + resourceId
				resp2, _, _ := ApiGet(apiUrl2, z, nil)
				appRoles := utl.Slice(resp2["appRoles"])
				for item := range appRoles {
					if role := utl.Map(item); role != nil {
						roleId := utl.Str(role["id"])
						claim := utl.Str(role["value"])
						if claim == "" {
							claim = "<unknown>"
						}
						roleIdValueMap[roleId] = claim // Keep growing roleId:value map
					}
				}
				// QUESTION: Aside from roles under r2["appRoles"] is there ever a need to also parse
				// r2["resourceSpecificApplicationPermissions"]? It doesn't appear those are grantable,
				// but it's unclear exactly what that attribute parameter block is used for.

				uniqueResourceIds.Add(resourceId) // Mark resourceId as seen
			}
		}

		// Collate OAuth 2.0 role permission admin grants
		for _, item := range appRoleAssignments {
			if api := utl.Map(item); api != nil {
				oauthId := utl.Str(api["id"])
				apiName := utl.Str(api["resourceDisplayName"])
				appRoleId := utl.Str(api["appRoleId"])
				claim := roleIdValueMap[appRoleId]

				// Keep growing the list of api permission grants
				apiPerms = append(apiPerms, []string{oauthId, apiName, "Application", claim})
			}
		}
	}

	// Now print the list of api_permissions_consented
	if len(apiPerms) > 0 {
		fmt.Printf("%s:\n", utl.Blu("api_permissions_consented"))
		for _, v := range apiPerms {
			oauth_id := v[0]
			api_name := v[1]
			perm_type := v[2]
			value := v[3]

			fmt.Printf("  %s%s  %s%s  %s%s  %s\n",
				utl.Gre(oauth_id), utl.PadSpaces(40, len(oauth_id)),
				utl.Gre(api_name), utl.PadSpaces(40, len(api_name)),
				utl.Gre(perm_type), utl.PadSpaces(14, len(perm_type)),
				utl.Gre(value))
		}
	}

	// Print published permission scopes
	publishedPermissionScopes := utl.Slice(x["publishedPermissionScopes"])
	if len(publishedPermissionScopes) > 0 {
		fmt.Printf("%s:\n", utl.Blu("published_permission_scopes"))
		for _, item := range publishedPermissionScopes {
			if scope := utl.Map(item); scope != nil {
				scopeId := utl.Str(scope["id"])
				enabledStat := "Disabled"
				if utl.Str(scope["isEnabled"]) == "true" {
					enabledStat = "Enabled"
				}
				apiName := displayName
				scopeType := "Delegated"
				scopeValue := utl.Str(scope["value"])
				fmt.Printf("  %s%s  %s%s  %s%s  %s%s  %s\n",
					utl.Gre(scopeId), utl.PadSpaces(38, len(scopeId)),
					utl.Gre(enabledStat), utl.PadSpaces(10, len(enabledStat)),
					utl.Gre(apiName), utl.PadSpaces(50, len(apiName)),
					utl.Gre(scopeType), utl.PadSpaces(12, len(scopeType)),
					utl.Gre(scopeValue))
			}
		}
	}

	// Print all Custom Security Attributes for this SP
	apiUrl = ConstMgUrl + "/v1.0/servicePrincipals/" + id + "?$select=customSecurityAttributes"
	resp, _, _ = ApiGet(apiUrl, z, nil)
	customSecurityAttributes := utl.Map(resp["customSecurityAttributes"])
	if customSecurityAttributes != nil {
		fmt.Printf("%s:\n", utl.Blu("custom_security_attributes"))
		var csa_list []map[string]string = nil
		for attr_set, value := range customSecurityAttributes {
			attr_map := utl.Map(value)
			if attr_map == nil {
				continue // Skip if not a map
			}
			for attr_name, v2 := range attr_map {
				// Skip '*@odata.type' entries. Hey Microsoft, this is a terrible design!
				// The value of each of these CSAs should have been a list insted of a map.
				// Adding an additional type entry, for example 'Project@odata.type' for
				// the 'Project' entry is ugly programming.
				if strings.HasSuffix(attr_name, "@odata.type") {
					continue
				}

				var attr_type string
				var attr_value string

				// Use type assertion to determine the type of v2
				switch val := v2.(type) {
				case []interface{}:
					if len(val) > 0 {
						switch val[0].(type) {
						case string:
							attr_type = "[]string"
							attr_value = ""
							for _, i := range val {
								attr_value += " '" + i.(string) + "'"
							}
						case float64:
							attr_type = "[]int"
							attr_value = ""
							for _, i := range val {
								attr_value += " '" + fmt.Sprintf("%d", int(i.(float64))) + "'"
							}
						default:
							attr_type = "[]unknown"
							attr_value = "unsupported type"
						}
					} else {
						attr_type = "[]int or []string, unclear"
						attr_value = "empty"
					}
				case string:
					attr_type = "string"
					attr_value = val
				case float64:
					attr_type = "int"
					attr_value = fmt.Sprintf("%d", int(val))
				default:
					attr_type = "unknown"
					attr_value = "unsupported type"
				}

				attr_value = strings.TrimSpace(attr_value)
				csa_list = append(csa_list, map[string]string{
					"attr_set":   attr_set,
					"attr_name":  attr_name,
					"attr_value": attr_value,
					"attr_type":  attr_type,
				})
			}
		}
		for _, csa := range csa_list {
			col1 := csa["attr_set"]
			col2 := csa["attr_name"]
			col3 := csa["attr_value"] + " (" + csa["attr_type"] + ")"
			fmt.Printf("  %s%s  %s%s  %s\n",
				utl.Gre(col1), utl.PadSpaces(26, len(col1)),
				utl.Gre(col2), utl.PadSpaces(26, len(col2)),
				utl.Gre(col3))
		}
	}
}

// Retrieves counts of all SPs in local cache, 2 values: Native ones to this tenant, and all others.
func SpsCountLocal(z *Config) (native, others int64) {
	// Load all service principals from the cache
	cache, err := GetCache(ServicePrincipal, z)
	if err != nil {
		return 0, 0 // If the cache cannot be loaded, return 0
	}

	// Iterate through the cached service principals and classify them
	for _, obj := range cache.data {
		if utl.Str(obj["appOwnerOrganizationId"]) == z.TenantId { // If owned by current tenant
			native++
		} else {
			others++
		}
	}

	return native, others
}

// Retrieves counts of SPs native to this Azure tenant, and all others.
func SpsCountAzure(z *Config) (native, others int64) {
	// First, get total number of SPs in native tenant
	z.AddMgHeader("ConsistencyLevel", "eventual")
	apiUrl := ConstMgUrl + ApiEndpoint[ServicePrincipal] + "/$count"
	resp, _, _ := ApiGet(apiUrl, z, nil)
	all := utl.Int64(resp["value"])

	// Now get count of SPs registered and native to only this tenant
	params := map[string]string{
		"$filter": "appOwnerOrganizationId eq " + z.TenantId,
		"$count":  "true",
	}
	apiUrl = ConstMgUrl + ApiEndpoint[ServicePrincipal]
	resp, _, _ = ApiGet(apiUrl, z, params)
	native = utl.Int64(resp["@odata.count"])

	others = all - native

	return native, others
}
