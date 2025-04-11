package maz

import (
	"fmt"
	"strings"

	"github.com/queone/utl"
)

// Consolidate -a/-d/-s/-m into single generic functions:
//     One GetMatchingResourceObjects(mazType...) in favor of below 4:
//       GetMatchingAzureMgmtGroups()
//       GetMatchingAzureSubscriptions()
//       GetMatchingResRoleAssignments()
//       GetMatchingResRoleDefinitions()
//     One CacheResourceObjects(mazType...) in favor of below 4:
//       CacheAzureMgmtGroups()
//       CacheAzureSubscriptions()
//       CacheAzureResRoleAssignments()
//       CacheAzureResRoleDefinitions()

func CacheResourceObjects(mazType string) {

}

// Retrieves an Azure resource object by its unique ID
func GetAzureResObjectById(mazType, targetId string, z *Config) AzureObject {
	// We were previously using the ARM API directly for these types of query, but
	// now gradually switching to the more performant Azure Resource Graph API way.
	// https://learn.microsoft.com/en-us/azure/governance/resource-graph/overview

	// Build payload query string
	var query string
	switch mazType {
	case ResRoleDefinition:
		query = fmt.Sprintf(`
		    AuthorizationResources
			| where type =~ "Microsoft.Authorization/roleDefinitions"
			| where name =~ "%s"`, targetId)
	case ResRoleAssignment:
		query = fmt.Sprintf(`
			AuthorizationResources
			| where type =~ "Microsoft.Authorization/roleAssignments"
			| where name =~ "%s"`, targetId)
	case Subscription:
		query = fmt.Sprintf(`
			ResourceContainers
			| where type =~ "Microsoft.Resources/subscriptions"
			| where subscriptionId =~ "%s"`,
			targetId)
		// For subscriptions one has to use subscriptionId for the GUID,
		// because 'name' holds the 'displayName'.
	case ManagementGroup:
		query = fmt.Sprintf(`
			ResourceContainers
			| where type =~ "Microsoft.Management/managementGroups"
			| where name =~ "%s"`, targetId)
	}
	payload := map[string]interface{}{"query": query}

	// Post the query to the Resource Graph API call
	params := map[string]string{"api-version": "2024-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.ResourceGraph/resources"
	resp, statCode, _ := ApiPost(apiUrl, z, payload, params)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 200 {
		if data := utl.Slice(resp["data"]); len(data) > 0 {
			if obj := utl.Map(data[0]); obj != nil {
				// Found matching object
				obj["maz_from_azure"] = true

				// Normalize subscription object's displayName
				if strings.ToLower(utl.Str(obj["type"])) == "microsoft.resources/subscriptions" {
					obj["displayName"] = utl.Str(obj["name"])
				}
				// It's a bit frustrating that Resource Graph doesn't use 'displayName'
				// for this attribute, like the ARM API does
				return AzureObject(obj)
			}
		}
	}

	return nil // Nothing found, return empty object
}

// Retrieves an Azure resource object by its display name or role name
func GetAzureResObjectByName(mazType, targetName string, z *Config) AzureObject {
	// Build payload query string
	var query string
	switch mazType {
	case ResRoleDefinition:
		query = fmt.Sprintf(`
            AuthorizationResources
			| where type =~ "Microsoft.Authorization/roleDefinitions"
            | where properties.roleName =~ "%s"`,
			targetName)
	case Subscription:
		query = fmt.Sprintf(`
			ResourceContainers
			| where type =~ "Microsoft.Resources/subscriptions"
			| where name =~ "%s"`,
			targetName)
		// For subscriptions 'name' actually holds 'displayName'
		// Took a lot of wasted time to find that out!
	case ManagementGroup:
		query = fmt.Sprintf(`
			ResourceContainers
			| where type =~ "Microsoft.Management/managementGroups"
			| where properties.displayName =~ "%s"`,
			targetName)
	}
	payload := map[string]interface{}{"query": query}

	// Post the query to the Resource Graph API call
	params := map[string]string{"api-version": "2024-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.ResourceGraph/resources"
	resp, statCode, _ := ApiPost(apiUrl, z, payload, params)
	if statCode != 200 {
		Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
	}
	if statCode == 200 {
		if data := utl.Slice(resp["data"]); len(data) > 0 {
			if obj := utl.Map(data[0]); obj != nil {
				// Found matching object
				obj["maz_from_azure"] = true

				// Normalize subscription object's displayName
				if strings.ToLower(utl.Str(obj["type"])) == "microsoft.resources/subscriptions" {
					obj["displayName"] = utl.Str(obj["name"])
				}
				// It's a bit frustrating that Resource Graph doesn't follow the same
				// pattern that ARM API does, using 'displayName' attribute.

				return AzureObject(obj)
			}
		}
	}
	return nil // Nothing found, return empty object
}
