package maz

import (
	"fmt"
	"path"

	"github.com/queone/utl"
)

// Prints Azure management group object in YAML-like format
func PrintMgmtGroup(group AzureObject) {
	id := utl.Str(group["name"])
	if id == "" {
		return
	}

	fmt.Printf("%s\n", utl.Gra("# Management group"))
	if props := utl.Map(group["properties"]); props != nil {
		displayName := utl.Str(props["displayName"])
		tenantId := utl.Str(props["tenantId"])
		fmt.Printf("%-12s: %s\n", utl.Blu("id"), utl.Gre(id))
		fmt.Printf("%-12s: %s\n", utl.Blu("displayName"), utl.Gre(displayName))
		fmt.Printf("%-12s: %s\n", utl.Blu("tenantId"), utl.Gre(tenantId))
	}
}

// Gets the full ID of all management groups currently in cache.
// Full IDs are commonly used when handling resource role definitions and assignments.
func GetAzureMgmtGroupsIds(z *Config) (mgmtGroupIds []string) {
	mgmtGroupIds = nil

	// Optimize performance by using cached management groups; 'false' avoids querying Azure
	mgmtGroups := GetMatchingAzureMgmtGroups("", false, z)

	for i := range mgmtGroups {
		group := mgmtGroups[i]
		id := utl.Str(group["id"])
		mgmtGroupIds = append(mgmtGroupIds, id)
	}
	return mgmtGroupIds
}

// Gets all Azure management groups matching on 'filter'. Returns entire list if filter is empty ""
func GetMatchingAzureMgmtGroups(filter string, force bool, z *Config) AzureObjectList {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		obj := GetAzureMgmtGroupById(filter, z)
		if obj != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{obj}
		}
	}

	// Get current cache, or initialize a new cache for this type
	cache, err := GetCache(ManagementGroup, z)
	if err != nil {
		utl.Die("Error: %v\n", err)
	}

	// Return an empty list if cache is nil and internet is not available
	internetIsAvailable := utl.IsInternetAvailable()
	if cache == nil && !internetIsAvailable {
		return AzureObjectList{} // Return empty list
	}

	// Determine if cache is empty or outdated and needs to be refreshed from Azure
	cacheNeedsRefreshing := force || cache.Count() < 1 || cache.Age() == 0 || cache.Age() > ConstMgCacheFileAgePeriod
	if internetIsAvailable && cacheNeedsRefreshing {
		CacheAzureMgmtGroups(cache, z)
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.StringSet{}            // Keep track of unique IDs to eliminate duplicates
	for i := range cache.data {
		obj := cache.data[i]
		// Extract the ID: use the last part of the "id" path or fall back to the "name" field
		id := utl.Str(obj["id"])
		name := utl.Str(obj["name"])
		if id != "" {
			id = path.Base(id) // Extract the last part of the path (UUID)
		} else if name != "" {
			id = name // Fall back to the "name" field if "id" is empty
		}

		// Skip if the ID is empty or already seen
		if id == "" || ids.Exists(id) {
			continue
		}

		// Check if the object matches the filter
		if obj.HasString(filter) {
			matchingList.Add(obj) // Add matching object to the list
			ids.Add(id)           // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all Azure management groups objects in current tenant and saves them to
// local cache. Note that we are updating the cache via its pointer, so no return value.
func CacheAzureMgmtGroups(cache *Cache, z *Config) {
	list := AzureObjectList{} // List of management group objects to cache

	// Get all managements groups from Azure
	params := map[string]string{"api-version": "2023-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.Management/managementGroups"
	var err error
	resp, _, err := ApiGet(apiUrl, z, params)
	if err != nil {
		Logf("%v\n", err)
	}
	mgmtGroups := utl.Slice(resp["value"])
	for i := range mgmtGroups {
		obj := mgmtGroups[i]
		if group := utl.Map(obj); group != nil {
			list = append(list, group)
		}
	}

	// Trim and prepare all objects for caching
	for i := range list {
		// Directly modify the object in the original list
		list[i] = list[i].TrimForCache(ManagementGroup)
	}

	// Update the cache with the entire list of definitions
	cache.data = list

	// Save updated cache
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated cache: %s\n", err.Error())
	}
}

// Recursively prints children management groups subscriptions
func PrintMgmtGroupChildren(indent int, children []interface{}) {
	mgmtType := map[string]string{
		"Microsoft.Management/managementGroups":               "(Management Group)",
		"Microsoft.Management/managementGroups/subscriptions": "(Subscription)",
		"/subscriptions": "(Subscription)",
	}

	for i := range children {
		obj := children[i]
		if child := utl.Map(obj); child != nil {
			displayName := utl.Str(child["displayName"])
			Type := mgmtType[utl.Str(child["type"])]
			if displayName == "Access to Azure Active Directory" && Type == "(Subscription)" {
				continue // Ignore legacy subscriptions
			}
			fmt.Printf("%*s", indent, " ") // Space padded indent
			padding := 44 - indent
			if padding < 12 {
				padding = 12
			}
			cDisplayName := utl.Blu(utl.PostSpc(displayName, padding))
			cName := utl.Gre(utl.PostSpc(utl.Str(child["name"]), 38))
			fmt.Printf("%s%s%s\n", cDisplayName, cName, utl.Gre(Type))

			// Use recursion to print additional children
			descendants := utl.Slice(child["children"])
			PrintMgmtGroupChildren(indent+4, descendants)
		}
	}
}

// Prints the current Azure tenant management group tree.
func PrintAzureMgmtGroupTree(z *Config) {
	apiUrl := ConstAzUrl + "/providers/Microsoft.Management/managementGroups/" + z.TenantId
	params := map[string]string{
		"api-version": "2023-04-01",
		"$expand":     "children",
		"$recurse":    "true",
	}
	var err error
	resp, _, err := ApiGet(apiUrl, z, params)
	if err != nil {
		Logf("%v\n", err)
	}
	if props := utl.Map(resp["properties"]); props != nil {
		// Print top line of hierarchy in blue
		name := utl.Blu(utl.PostSpc(utl.Str(props["displayName"]), 44))
		tenantId := utl.Blu(utl.PostSpc(utl.Str(props["tenantId"]), 38))
		fmt.Printf("%s%s%s\n", name, tenantId, utl.Blu("(Tenant)"))

		// Recursively print tree of additional children
		children := utl.Slice(props["children"])
		PrintMgmtGroupChildren(4, children)
	}
}

// Gets a specific Azure management group by its stand-alone object UUID or name
func GetAzureMgmtGroupById(targetId string, z *Config) AzureObject {
	// 1st try with new function that calls Azure Resource Graph API
	if group := GetAzureResObjectById(ManagementGroup, targetId, z); group != nil {
		return group // Return immediately if we found it
	}

	// Fallback to using the ARM API way if above returns nothing

	params := map[string]string{"api-version": "2023-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.Management/managementGroups/" + targetId
	var err error
	resp, _, err := ApiGet(apiUrl, z, params)
	if err != nil {
		Logf("%v\n", err)
	}
	group := AzureObject(resp)
	group["maz_from_azure"] = true
	return group
}

// Returns count of all subscriptions in current Azure tenant
func CountAzureMgmtGroups(z *Config) int64 {
	params := map[string]string{"api-version": "2023-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.Management/managementGroups"
	var err error
	resp, _, err := ApiGet(apiUrl, z, params)
	if err != nil {
		Logf("%v\n", err)
	}
	mgmtGroups := utl.Slice(resp["value"])
	count := len(mgmtGroups)
	return int64(count)
}
