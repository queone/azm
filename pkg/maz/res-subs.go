package maz

import (
	"fmt"

	"github.com/queone/utl"
)

// Prints Azure subscription object in YAML-like format
func PrintSubscription(x AzureObject) {
	id := utl.Str(x["subscriptionId"])
	if id == "" {
		return
	}
	fmt.Printf("%s\n", utl.Gra("# Subscription"))
	fmt.Printf("%s: %s\n", utl.Blu("object_id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("display_name"), utl.Gre(utl.Str(x["displayName"])))
	fmt.Printf("%s: %s\n", utl.Blu("state"), utl.Gre(utl.Str(x["state"])))
	fmt.Printf("%s: %s\n", utl.Blu("tenant_id"), utl.Gre(utl.Str(x["tenantId"])))
}

// Gets the full ID, i.e. "/subscriptions/UUID", of all subscription currently in cache.
// Full IDs are commonly used when handling resource role definitions and assignments.
func GetAzureSubscriptionsIds(z *Config) (ids []string) {
	ids = nil
	subscriptions := GetMatchingAzureSubscriptions("", false, z) // false = get from cache, not Azure
	for _, item := range subscriptions {
		// Skip disabled and legacy subscriptions
		displayName := utl.Str(item["displayName"])
		state := utl.Str(item["state"])
		if state != "Enabled" || displayName == "Access to Azure Active Directory" {
			continue
		}
		ids = append(ids, utl.Str(item["id"]))
	}
	return ids
}

// Returns an id:name map of all Azure subscriptions
func GetAzureSubscriptionsIdMap(z *Config) map[string]string {
	nameMap := make(map[string]string)
	subscriptions := GetMatchingAzureSubscriptions("", false, z) // false = get from cache, not Azure
	for _, item := range subscriptions {
		// Safely extract "subscriptionId" and "displayName" with type assertions
		subscriptionId, okID := item["subscriptionId"].(string)
		displayName, okName := item["displayName"].(string)
		if okID && okName {
			nameMap[subscriptionId] = displayName
		} else {
			// Log or handle entries with missing or invalid fields
			//fmt.Printf("Skipping object with invalid id or displayName: %+v\n", x) // DEBUG
		}
	}
	return nameMap
}

// Gets all Azure subscriptions matching on 'filter'. Returns entire list if filter is empty ""
func GetMatchingAzureSubscriptions(filter string, force bool, z *Config) AzureObjectList {
	// If the filter is a UUID, we deliberately treat it as an ID and perform a
	// quick Azure lookup for the specific object.
	if utl.ValidUuid(filter) {
		x := GetAzureSubscriptionById(filter, z)
		if x != nil {
			// If found, return a list containing just this object.
			return AzureObjectList{x}
		}
		// If not found, then filter will be used below in obj.HasString(filter)
	}

	// Get current cache, or initialize a new cache for this type
	cache, err := GetCache("s", z) // Get subscriptions type cache
	if err != nil {
		utl.Die("Error: %s\n", err.Error())
	}

	// Return an empty list if cache is nil and internet is not available
	internetIsAvailable := utl.IsInternetAvailable()
	if cache == nil && !internetIsAvailable {
		return AzureObjectList{} // Return empty list
	}

	// Determine if cache is empty or outdated and needs to be refreshed from Azure
	cacheNeedsRefreshing := force || cache.Age() == 0 || cache.Age() > ConstMgCacheFileAgePeriod
	if internetIsAvailable && cacheNeedsRefreshing {
		CacheAzureSubscriptions(cache, z, true)
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}
	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.NewStringSet()         // Keep track of unique IDs to eliminate duplicates
	for _, obj := range cache.data {
		id := obj["id"].(string)
		if ids.Exists(id) {
			continue // Skip repeated entries
		}
		if obj.HasString(filter) {
			matchingList.Add(obj) // Add matching object to the list
			ids.Add(id)           // Mark this ID as seen
		}
	}

	return matchingList
}

// Retrieves all Azure subscription objects in current tenant and saves them to local
// cache. Note that we are updating the cache via its pointer, so no return value.
func CacheAzureSubscriptions(cache *Cache, z *Config, verbose bool) {
	params := map[string]string{"api-version": "2024-11-01"}
	apiUrl := ConstAzUrl + "/subscriptions"
	r, _, _ := ApiGet(apiUrl, z, params)
	if r["value"] != nil {
		rawSubscriptions, ok := r["value"].([]interface{})
		if !ok {
			utl.Die("unexpected type for subscriptions")
		}
		for _, raw := range rawSubscriptions {
			azObj, ok := raw.(map[string]interface{})
			if !ok {
				fmt.Printf("WARNING: Unexpected type for subscription object: %v\n", raw)
				continue
			}
			trimmedObj := AzureObject(azObj).TrimForCache("s")
			if err := cache.Upsert(trimmedObj); err != nil {
				fmt.Printf("WARNING: Failed to upsert cache for subscription object with ID '%s': %v\n",
					azObj["id"], err)
			}
		}
	}
	// Save updated cache
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated cache: %s\n", err.Error())
	}
}

// Gets a specific Azure subscription by its stand-alone object UUID
func GetAzureSubscriptionById(id string, z *Config) AzureObject {
	params := map[string]string{"api-version": "2024-11-01"}
	apiUrl := ConstAzUrl + "/subscriptions/" + id
	r, _, _ := ApiGet(apiUrl, z, params)
	azObj := AzureObject(r)
	azObj["maz_from_azure"] = true
	return azObj
}

// Returns count of all subscriptions in current Azure tenant
func CountAzureSubscriptions(z *Config) int64 {
	params := map[string]string{"api-version": "2024-11-01"}
	apiUrl := ConstAzUrl + "/subscriptions"
	r, _, _ := ApiGet(apiUrl, z, params)
	if r["count"] != nil {
		rawCount, ok := r["count"].(map[string]interface{})
		if ok {
			count := utl.Int64(rawCount["value"]) // Get int64 value
			return count
		}
	}
	return 0
}
