package maz

import (
	"fmt"
	"path"

	"github.com/queone/utl"
)

// Prints Azure subscription object in YAML-like format
func PrintSubscription(x AzureObject) {
	id := utl.Str(x["subscriptionId"])
	if id == "" {
		return
	}
	fmt.Printf("%s\n", utl.Gra("# Subscription"))
	fmt.Printf("%s: %s\n", utl.Blu("id"), utl.Gre(id))
	fmt.Printf("%s: %s\n", utl.Blu("displayName"), utl.Gre(utl.Str(x["displayName"])))
	fmt.Printf("%s: %s\n", utl.Blu("state"), utl.Gre(utl.Str(x["state"])))
	fmt.Printf("%s: %s\n", utl.Blu("tenantId"), utl.Gre(utl.Str(x["tenantId"])))
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
func GetIdMapSubscriptions(z *Config) map[string]string {
	nameMap := make(map[string]string)
	subscriptions := GetMatchingAzureSubscriptions("", false, z) // false = get from cache, not Azure

	for i := range subscriptions {
		sub := subscriptions[i]
		id := utl.Str(sub["subscriptionId"]) // Accessing the field directly
		if id == "" {
			continue // Skip if "subscriptionId" is missing or not a string
		}
		name := utl.Str(sub["displayName"])
		if name == "" {
			continue // Skip if "displayName" is missing or not a string
		}
		nameMap[id] = name
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
	cache, err := GetCache(Subscription, z) // Get subscriptions type cache
	if err != nil {
		utl.Die("Error: %s\n", err.Error())
	}

	// Return an empty list if cache is nil and internet is not available
	internetIsAvailable := utl.IsInternetAvailable()
	if cache == nil && !internetIsAvailable {
		return AzureObjectList{} // Return empty list
	}

	// Determine if cache is empty or outdated and needs to be refreshed from Azure
	cacheNeedsRefreshing := force || cache.Count() < 1 || cache.Age() == 0 || cache.Age() > ConstMgCacheFileAgePeriod
	if internetIsAvailable && cacheNeedsRefreshing {
		CacheAzureSubscriptions(cache, z, true)
	}

	// Filter the objects based on the provided filter
	if filter == "" {
		return cache.data // Return all data if no filter is specified
	}

	matchingList := AzureObjectList{} // Initialize an empty list for matching items
	ids := utl.StringSet{}            // Keep track of unique IDs to eliminate duplicates

	for i := range cache.data {
		sub := cache.data[i]

		// Extract the ID: use the last part of the "id" path or fall back to the "name" field
		id := ""
		if id = utl.Str(sub["id"]); id != "" {
			id = path.Base(id) // Extract the last part of the path (UUID)
		} else if subscriptionId := utl.Str(sub["subscriptionId"]); subscriptionId != "" {
			id = subscriptionId // Fall back to the "subscriptionId" field if "id" is empty
		}

		// Skip if the ID is empty or already seen
		if id == "" || ids.Exists(id) {
			continue
		}

		// Check if the object matches the filter
		if sub.HasString(filter) {
			matchingList = append(matchingList, sub) // Add matching object to the list
			ids.Add(id)                              // Mark this ID as seen
		}
	}
	return matchingList
}

// Retrieves all Azure subscription objects in current tenant and saves them to local
// cache. Note that we are updating the cache via its pointer, so no return value.
func CacheAzureSubscriptions(cache *Cache, z *Config, verbose bool) {
	list := AzureObjectList{} // List of subscription objects to cache

	params := map[string]string{"api-version": "2024-11-01"}
	apiUrl := ConstAzUrl + "/subscriptions"
	resp, _, _ := ApiGet(apiUrl, z, params)
	subscriptions := utl.Slice(resp["value"])
	for i := range subscriptions {
		obj := subscriptions[i]
		if subscription := utl.Map(obj); subscription != nil {
			list = append(list, subscription)
		}
	}

	// Trim and prepare all objects for caching
	for i := range list {
		// Directly modify the object in the original list
		list[i] = list[i].TrimForCache(Subscription)
	}

	// Update the cache with the entire list of definitions
	cache.data = list

	// Save updated cache
	if err := cache.Save(); err != nil {
		utl.Die("Error saving updated cache: %s\n", err.Error())
	}
}

// Gets a specific Azure subscription object by its nme
func GetAzureSubscriptionByName(targetName string, z *Config) AzureObject {
	params := map[string]string{"api-version": "2024-11-01"}
	apiUrl := ConstAzUrl + "/subscriptions"
	resp, _, _ := ApiGet(apiUrl, z, params)
	subscriptions := utl.Slice(resp["value"])
	for i := range subscriptions {
		obj := subscriptions[i]
		if subscription := utl.Map(obj); subscription != nil {
			if utl.Str(subscription["displayName"]) == targetName {
				return AzureObject(subscription)
			}
		}
	}
	return nil
}

// Gets a specific Azure subscription by its stand-alone object UUID
func GetAzureSubscriptionById(id string, z *Config) AzureObject {
	params := map[string]string{"api-version": "2024-11-01"}
	apiUrl := ConstAzUrl + "/subscriptions/" + id
	resp, _, _ := ApiGet(apiUrl, z, params)
	azObj := AzureObject(resp)
	azObj["maz_from_azure"] = true
	return azObj
}

// Returns count of all subscriptions in current Azure tenant
func CountAzureSubscriptions(z *Config) int64 {
	params := map[string]string{"api-version": "2024-11-01"}
	apiUrl := ConstAzUrl + "/subscriptions"
	resp, _, _ := ApiGet(apiUrl, z, params)
	if rawCount := utl.Map(resp["count"]); rawCount != nil {
		count := utl.Int64(rawCount["value"]) // Get int64 value
		return count
	}
	return 0
}
