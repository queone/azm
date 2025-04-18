package maz

import (
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

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
	// See learn.microsoft.com/en-us/azure/governance/resource-graph/overview

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

// Fetch Azure resources across all role scopes concurrently using parallel goroutines,
// with optional verbose logging.
func fetchAzureObjectsAcrossScopes(
	endpointSuffix string,
	z *Config,
	params map[string]string,
	verbose bool,
	mgroupIdMap, subIdMap map[string]string,
) AzureObjectList {
	var (
		list      = AzureObjectList{}
		ids       = utl.StringSet{} // Tracks unique object names to prevent duplicates
		callCount = 1               // For tracking and printing API call counts
		wg        sync.WaitGroup    // WaitGroup for synchronizing goroutines
		mu        sync.Mutex        // Mutex to safely update shared state across goroutines
		results   = make(chan AzureObjectList, 10)
		scopes    = GetAzureResRoleScopes(z) // All scopes to search across
	)

	// Launch a goroutine for each scope
	for _, scope := range scopes {
		wg.Add(1)
		go func(scope string) {
			defer wg.Done()

			apiUrl := ConstAzUrl + scope + endpointSuffix
			resp, statCode, _ := ApiGet(apiUrl, z, params)
			if statCode != 200 {
				Logf("%s\n", utl.Red2(fmt.Sprintf("HTTP %d: %s", statCode, ApiErrorMsg(resp))))
				return
			}

			items := utl.Slice(resp["value"])
			if items == nil {
				return // Skip if no items found
			}

			scopeList := AzureObjectList{}
			count := 0

			// Process each item in the response
			for _, obj := range items {
				objMap := utl.Map(obj)
				if objMap == nil {
					continue // Skip if object isn't a map
				}
				id := utl.Str(objMap["name"])

				// Use mutex to safely check/update deduplication set
				mu.Lock()
				if ids.Exists(id) {
					mu.Unlock()
					continue // Skip duplicates
				}
				ids.Add(id)
				mu.Unlock()

				scopeList = append(scopeList, objMap)
				count++
			}

			// Verbose output for progress tracking
			if verbose && count > 0 {
				scopeName := scope
				scopeType := "Subscription"
				if strings.HasPrefix(scope, "/providers") {
					if name, ok := mgroupIdMap[scope]; ok {
						scopeName = name
					}
					scopeType = "Management Group"
				} else if strings.HasPrefix(scope, "/subscriptions") {
					if name, ok := subIdMap[path.Base(scope)]; ok {
						scopeName = name
					}
				}
				fmt.Printf("%sCall %05d: %05d items under %s %s", clrLine, callCount, count, scopeType, scopeName)
			}
			callCount++

			// Send collected items from this scope to the result channel
			results <- scopeList
		}(scope)
	}

	// Close the results channel once all goroutines finish
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results from goroutines into the final list
	for partial := range results {
		list = append(list, partial...)
	}

	if verbose {
		fmt.Print(clrLine) // Clear last verbose line
	}

	return list
}

// Generate a password expiry report for all Apps and Service Principals in the tenant.
func PrintPasswordExpiryReport(csvMode bool, daysStr string, z *Config) {
	var combinedList AzureObjectList

	// Normalize and parse days
	if daysStr == "" {
		daysStr = "-1"
	}
	daysInt, err := utl.StringToInt64(daysStr)
	if err != nil {
		Logf("Invalid 'days' value: %s. Defaulting to -1 (show all).\n", utl.Mag(daysStr))
		daysInt = -1
	}

	// OLD SEQUENTIAL METHOD
	// apps := GetMatchingDirObjects(Application, "", true, z)
	// Logf("Apps count %5s\n", utl.Mag(utl.ToStr(len(apps))))
	// for _, app := range apps {
	// 	app["maz_type"] = Application
	// 	combinedList = append(combinedList, app)
	// }

	// sps := GetMatchingDirObjects(ServicePrincipal, "", true, z)
	// Logf("SPs count  %5s\n", utl.Mag(utl.ToStr(len(sps))))
	// for _, sp := range sps {
	// 	sp["maz_type"] = ServicePrincipal
	// 	combinedList = append(combinedList, sp)
	// }

	// WHAT EXACTLY ARE WE PARALLELIZING?: For each MazType, Apps and SPs, we're querying
	// cache and Azure in parallel get the list of those object types.
	// ====
	var mu sync.Mutex     // Used to safely append to 'list' from multiple goroutines
	var wg sync.WaitGroup // WaitGroup to wait for all goroutines to complete

	for _, mazType := range []string{Application, ServicePrincipal} {
		mazType := mazType // Capture loop variable to avoid race condition inside goroutine
		wg.Add(1)          // Register one more goroutine with the WaitGroup

		// Start a goroutine to query cache/Azure for this type in parallel
		go func() {
			defer wg.Done()
			objs := fetchAndTagDirObjects(mazType, z)
			// Above helper function avoids the performance hit of locking/unlocking
			// each if we were to do that here
			Logf("%-3s count %5s\n", mazType, utl.Mag(utl.ToStr(len(objs))))
			mu.Lock()
			combinedList = append(combinedList, objs...)
			mu.Unlock()
		}()
	}

	wg.Wait() // Wait for all goroutines to finish
	// ====

	Logf("Combined count  %5s\n", utl.Mag(utl.ToStr(len(combinedList))))

	// Print header
	if csvMode {
		fmt.Printf("\"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\"\n",
			"TYPE", "NAME", "CLIENT_ID", "SECRET_ID", "SECRET_NAME", "EXPIRY")
	} else {
		fmt.Printf("%-6s %-40s %-38s %-38s %-38s %s\n",
			"TYPE", "NAME", "CLIENT_ID", "SECRET_ID", "SECRET_NAME", "EXPIRY")
	}

	for _, obj := range combinedList {
		if secrets := utl.Slice(obj["passwordCredentials"]); len(secrets) > 0 {
			PrintExpiringSecrets(csvMode,
				utl.Str(obj["maz_type"]),
				utl.Str(obj["displayName"]),
				utl.Str(obj["id"]),
				utl.Str(obj["appId"]),
				daysInt,
				secrets,
			)
		}
	}
}

// Fetch and tag directory objects for the given type in parallel-safe format.
func fetchAndTagDirObjects(mazType string, z *Config) AzureObjectList {
	list := GetMatchingDirObjects(mazType, "", true, z)
	for _, obj := range list {
		obj["maz_type"] = mazType
	}
	return list
}

// Print details of secrets that are expiring within the specified number of days.
func PrintExpiringSecrets(csvMode bool, mazType, name, id, appId string, days int64, secrets []interface{}) {
	now := time.Now().Unix()

	for _, item := range secrets {
		sec := utl.Map(item)
		if sec == nil {
			Logf("Improperly formed secret entry. Skipping.\n")
			continue
		}

		// Extract the relevant fields from the secret
		secretId := utl.Str(sec["keyId"])
		secretName := utl.Str(sec["displayName"])
		if secretName == "" {
			secretName = "<blank>"
		}
		expiryRaw := utl.Str(sec["endDateTime"])

		// Convert the expiry date to other formats for processing and printing
		expiryTime, err := time.Parse(time.RFC3339, expiryRaw)
		if err != nil {
			Logf("Error parsing endDateTime: %s\n", expiryRaw)
			continue
		}
		expiryInt := expiryTime.Unix()
		expiryStr := UnixDateTimeString(expiryInt)
		Logf("endDateTime raw | int64 | formatted : [ %s | %d | %s ]\n",
			expiryRaw, expiryInt, expiryStr)

		daysDiff := (expiryInt - now) / 86400
		cExpiryStr := expiryStr
		if daysDiff <= 0 {
			cExpiryStr = utl.Red(expiryStr)
		}

		if days == -1 || daysDiff <= days {
			if csvMode {
				fmt.Printf("\"%s\", \"%s\", \"%s\", \"%s\", \"%s\", \"%s\"\n",
					mazType, name, appId, secretId, secretName, expiryStr)
			} else {
				fmt.Printf("%-6s %-40s %-38s %-38s %-38s %s\n",
					mazType, name, appId, secretId, secretName, cExpiryStr)
			}
		}
	}
}
