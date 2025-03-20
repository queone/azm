package maz

import (
	"fmt"
	"path"

	"github.com/queone/utl"
)

// Prints Azure management group object in YAML-like format
func PrintMgmtGroup(x AzureObject) {
	id := utl.Str(x["name"])
	if id == "" {
		return
	}

	fmt.Printf("%s\n", utl.Gra("# Management Group"))
	displayName := utl.Str(x["displayName"])
	tenantId := utl.Str(x["tenantId"])
	// REVIEW
	// Is below really needed? We are normalizing these properties values to root of object cache
	if x["properties"] != nil {
		xProp := x["properties"].(map[string]interface{})
		displayName = utl.Str(xProp["displayName"])
		tenantId = utl.Str(xProp["tenantId"])
	}
	fmt.Printf("%-12s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%-12s: %s\n", utl.Blu("display_name"), utl.Gre(displayName))
	fmt.Printf("%-12s: %s\n", utl.Blu("tenant_id"), utl.Gre(tenantId))
}

// Gets the full ID of all management groups currently in cache.
// Full IDs are commonly used when handling resource role definitions and assignments.
func GetAzureMgmtGroupsIds(z *Config) (mgmtGroupIds []string) {
	mgmtGroupIds = nil
	mgmtGroups := GetMatchingAzureMgmtGroups("", false, z) // false = get from cache, not Azure
	for _, item := range mgmtGroups {
		mgmtGroupIds = append(mgmtGroupIds, utl.Str(item["id"]))
	}
	return mgmtGroupIds
}

// Returns an id:name map of all Azure management groups
func GetIdMapMgmtGroups(z *Config) map[string]string {
	nameMap := make(map[string]string)
	mgmtGroups := GetMatchingAzureMgmtGroups("", false, z) // false = get from cache, not Azure

	// Memory-walk the slice to gather these values more efficiently
	for i := range mgmtGroups {
		groupPtr := &mgmtGroups[i]   // Use a pointer to avoid copying the element
		group := *groupPtr           // Dereference the pointer for easier access
		id := utl.Str(group["name"]) // Accessing the field directly
		if id == "" {
			continue // Skip if "name" is missing or not a string
		}
		name := utl.Str(group["displayName"])
		if name == "" {
			continue // Skip if "displayName" is missing or not a string
		}
		nameMap[id] = name
	}

	return nameMap
}

// Gets all Azure management groups matching on 'filter'. Returns entire list if filter is empty ""
func GetMatchingAzureMgmtGroups(filter string, force bool, z *Config) AzureObjectList {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		x := GetAzureMgmtGroupById(filter, z)
		if x != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{x}
		}
		// If not found, then filter will be used below in obj.HasString(filter)
	}

	// Get current cache, or initialize a new cache for this type
	cache, err := GetCache(ManagementGroup, z) // Get subscriptions type cache
	if err != nil {
		utl.Die("Error: %v\n", err)
	}

	// Return an empty list if cache is nil and internet is not available
	internetIsAvailable := utl.IsInternetAvailable()
	if cache == nil && !internetIsAvailable {
		return AzureObjectList{} // Return empty list
	}

	// Determine if cache is empty or outdated and needs to be refreshed from Azure
	cacheNeedsRefreshing := force || cache.Age() == 0 || cache.Age() > ConstMgCacheFileAgePeriod
	if internetIsAvailable && cacheNeedsRefreshing {
		CacheAzureMgmtGroups(cache, z, true)
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.StringSet{}            // Keep track of unique IDs to eliminate duplicates

	for i := range cache.data {
		obj := &cache.data[i] // Access the element directly via pointer (memory walk)

		// Extract the ID: use the last part of the "id" path or fall back to the "name" field
		id := utl.Str((*obj)["id"])
		name := utl.Str((*obj)["name"])
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
			matchingList.Add(*obj) // Add matching object to the list
			ids.Add(id)            // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all Azure management groups objects in current tenant and saves them to
// local cache. Note that we are updating the cache via its pointer, so no return value.
// Old function = func GetAzMgGroups(z *Config) (list []interface{}) {
func CacheAzureMgmtGroups(cache *Cache, z *Config, verbose bool) {
	params := map[string]string{"api-version": "2023-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.Management/managementGroups"
	r, _, _ := ApiGet(apiUrl, z, params)
	if r["value"] != nil {
		rawMgmtGroups, ok := r["value"].([]interface{})
		if !ok {
			utl.Die("unexpected type for management groups")
		}
		for _, raw := range rawMgmtGroups {
			azObj, ok := raw.(map[string]interface{})
			if !ok {
				fmt.Printf("WARNING: Unexpected type for management group object: %v\n", raw)
				continue
			}
			trimmedObj := AzureObject(azObj).TrimForCache("m")
			if err := cache.Upsert(trimmedObj); err != nil {
				fmt.Printf("WARNING: Failed to upsert cache for management group object with ID '%s': %v\n",
					azObj["id"], err)
			}
		}
	}
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

	for _, item := range children {
		child := item.(map[string]interface{})
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
		if child["children"] != nil {
			descendants := child["children"].([]interface{})
			PrintMgmtGroupChildren(indent+4, descendants)
			// Using recursion here to print additional children
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
	r, _, _ := ApiGet(apiUrl, z, params)
	if r["properties"] != nil {
		// Print everything under the hierarchy
		Prop := r["properties"].(map[string]interface{})
		name := utl.Blu(utl.PostSpc(utl.Str(Prop["displayName"]), 44))
		tenantId := utl.Blu(utl.PostSpc(utl.Str(Prop["tenantId"]), 38))
		fmt.Printf("%s%s%s\n", name, tenantId, utl.Blu("(Tenant)"))
		if Prop["children"] != nil {
			children := Prop["children"].([]interface{})
			PrintMgmtGroupChildren(4, children)
		}
	}
}

// Gets a specific Azure management group by its stand-alone object UUID or name
func GetAzureMgmtGroupById(id string, z *Config) AzureObject {
	params := map[string]string{"api-version": "2023-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.Management/managementGroups/" + id
	r, _, _ := ApiGet(apiUrl, z, params)
	azObj := AzureObject(r)
	azObj["maz_from_azure"] = true
	return azObj
}

// Returns count of all subscriptions in current Azure tenant
func CountAzureMgmtGroups(z *Config) int64 {
	params := map[string]string{"api-version": "2023-04-01"}
	apiUrl := ConstAzUrl + "/providers/Microsoft.Management/managementGroups"
	resp, _, _ := ApiGet(apiUrl, z, params)
	if resp["value"] != nil {
		rawList, ok := resp["value"].([]interface{})
		if ok {
			count := len(rawList)
			return int64(count)
		}
	}
	return 0
}
