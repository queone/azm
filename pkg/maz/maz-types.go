package maz

import (
	"encoding/gob"

	"github.com/queone/utl"
)

// Basic types for this package
type AzureObject map[string]interface{} // Represents a single Azure JSON object
type AzureObjectList []AzureObject      // Represents a list of Azure JSON objects

// Register the types for Gob encoding.
func init() {
	gob.Register(AzureObject{})     // For single object
	gob.Register(AzureObjectList{}) // For list of objects
	// Register all concrete types that might appear in AzureObject or AzureObjectList
	gob.Register(map[string]interface{}{})
	gob.Register([]interface{}{})
}

// ==== AzureObject methods and functions

// Checks if the filter string is found anywhere within the AzureObject.
func (obj AzureObject) HasString(filter string) bool {
	for i := range obj {
		field := obj[i]
		switch v := field.(type) {
		case string:
			// Below does as case-insensitive strings.Contains()
			if utl.SubString(v, filter) {
				return true
			}
		case []interface{}:
			// Drill into other maps and string fields in the slice
			for i := range v {
				element := v[i]
				if nestedMap := utl.Map(element); nestedMap != nil {
					if AzureObject(nestedMap).HasString(filter) {
						return true
					}
				} else if itemStr := utl.Str(element); itemStr != "" {
					if utl.SubString(itemStr, filter) {
						return true
					}
				}
			}
		case map[string]interface{}:
			// Recursively call HasString on nested maps
			if AzureObject(v).HasString(filter) {
				return true
			}
		}
	}
	return false
}

// Trims the AzureObject to retain only the fields needed for caching based on the type code.
func (obj AzureObject) TrimForCache(mazType string) (trimmed AzureObject) {
	// We restrict caching to only certain fields to make this library more performant.
	switch mazType {
	case RbacDefinition:
		// Trim the AzureObject to retain only specific fields for role definitions.
		if props := utl.Map(obj["properties"]); props != nil {
			trimmed = AzureObject{
				"id":   obj["id"],
				"name": obj["name"],
				"properties": map[string]interface{}{
					"assignableScopes": props["assignableScopes"],
					"description":      props["description"],
					"permissions":      props["permissions"],
					"roleName":         props["roleName"],
					"type":             props["type"],
				},
			}
		} else {
			// Fallback: if properties is missing or not a map, just include the top-level fields.
			trimmed = AzureObject{
				"id":   obj["id"],
				"name": obj["name"],
			}
		}
	case RbacAssignment:
		// Trim the AzureObject to retain only specific fields for role definitions.
		if props := utl.Map(obj["properties"]); props != nil {
			trimmed = AzureObject{
				"id":   obj["id"],
				"name": obj["name"],
				"properties": map[string]interface{}{
					"roleDefinitionId": props["roleDefinitionId"],
					"description":      props["description"],
					"principalId":      props["principalId"],
					"principalType":    props["principalType"],
					"scope":            props["scope"],
				},
			}
		} else {
			// Fallback: if properties is missing or not a map, just include the top-level fields.
			trimmed = AzureObject{
				"id":   obj["id"],
				"name": obj["name"],
			}
		}
	case Subscription:
		trimmed = AzureObject{
			"id":             obj["id"],
			"subscriptionId": obj["subscriptionId"],
			"displayName":    obj["displayName"],
			"state":          obj["state"],
		}
	case ManagementGroup:
		// Trim the AzureObject to retain only specific fields for management group.
		if props := utl.Map(obj["properties"]); props != nil {
			trimmed = AzureObject{
				"id":   obj["id"],
				"name": obj["name"],
				"properties": map[string]interface{}{
					"displayName": props["displayName"],
					"tenantId":    props["tenantId"],
				},
			}
		} else {
			// Fallback: if properties is missing or not a map, just include the top-level fields.
			trimmed = AzureObject{
				"id":   obj["id"],
				"name": obj["name"],
			}
		}
	case DirectoryUser:
		trimmed = AzureObject{
			"id":                obj["id"],
			"displayName":       obj["displayName"],
			"userPrincipalName": obj["userPrincipalName"],
		}
	case DirectoryGroup:
		trimmed = AzureObject{
			"id":                 obj["id"],
			"displayName":        obj["displayName"],
			"description":        obj["description"],
			"createdDateTime":    obj["createdDateTime"],
			"isAssignableToRole": obj["isAssignableToRole"],
		}
	case Application:
		trimmed = AzureObject{
			"id":          obj["id"],
			"displayName": obj["displayName"],
			"appId":       obj["appId"],
		}
	case ServicePrincipal:
		trimmed = AzureObject{
			"id":                     obj["id"],
			"displayName":            obj["displayName"],
			"appId":                  obj["appId"],
			"appOwnerOrganizationId": obj["appOwnerOrganizationId"],
		}
	case DirRoleDefinition:
		trimmed = AzureObject{
			"id":          obj["id"],
			"displayName": obj["displayName"],
			"description": obj["description"],
			"isBuiltIn":   obj["isBuiltIn"],
			"isEnabled":   obj["isEnabled"],
			"templateId":  obj["templateId"],
			//"rolePermissions": obj["rolePermissions"],
		}
	case DirRoleAssignment:
		trimmed = AzureObject{
			"id":               obj["id"],
			"directoryScopeId": obj["directoryScopeId"],
			"principalId":      obj["principalId"],
			"roleDefinitionId": obj["roleDefinitionId"],
		}
	default:
		// For unknown type just include the ID field.
		trimmed = AzureObject{
			"id": obj["id"],
		}
	}
	return trimmed
}

// ==== AzureObjectList methods and functions

/*
This package provides a set of methods and functions for managing lists of Azure objects, represented as `AzureObject` (a `map[string]interface{}`) and `AzureObjectList` (a slice of `AzureObject`). The methods are designed to efficiently manipulate and query these lists, with a focus on performance and memory optimization. A key optimization technique used throughout this package is the use of **pointers** to access and modify elements in the list directly, avoiding unnecessary copying of data.

The use of pointers is particularly important in methods like `Replace`, `DeleteById`, `DeleteByName`, and `Delete`, where elements in the list are modified or removed. By accessing elements via pointers (e.g., `obj := &(*list)[j]`), the code avoids creating temporary copies of the objects, which can be costly for large lists or deeply nested structures. Instead, it directly references the underlying data in the slice, enabling in-place modifications. This approach reduces memory overhead and improves performance, especially when dealing with large datasets.

For methods that query the list, such as `FindById`, `FindByName`, and `Find`, pointers are also used to return references to the matching objects rather than copying them. This allows callers to interact with the original data in the list without duplicating it. Similarly, methods like `ExistsById`, `ExistsByName`, and `Exists` leverage pointers to efficiently check for the presence of objects without unnecessary data copying.

While the use of pointers introduces some syntactic complexity (e.g., dereferencing with `(*obj)`), it provides significant performance benefits. The trade-off is justified in scenarios where the lists are large or the operations are performance-critical. This design ensures that the package remains efficient and scalable while providing a clean and consistent API for managing Azure objects.
*/

// Add appends an AzureObject to the AzureObjectList.
func (list *AzureObjectList) Add(obj AzureObject) {
	*list = append(*list, obj) // Append the new object to the list
}

// Replaces an object in an AzureObjectList by matching on id, name, or subscriptionId.
func (list *AzureObjectList) Replace(newObj AzureObject) bool {
	id := utl.Str(newObj["id"])
	name := utl.Str(newObj["name"])
	subscriptionId := utl.Str(newObj["subscriptionId"])

	// The new object must have at least one of the unique keys
	if id == "" && name == "" && subscriptionId == "" {
		return false
	}

	// Iterate through the list to find an object with a matching key
	for j := range *list {
		obj := &(*list)[j] // Access the element directly via pointer
		// Most objects use 'id' for their unique key, but RBAC role definitions
		// and assignments, and Subscriptions use different keys.
		if (id != "" && (*obj)["id"] == id) ||
			(name != "" && (*obj)["name"] == name) ||
			(subscriptionId != "" && (*obj)["subscriptionId"] == subscriptionId) {
			(*list)[j] = newObj // Replace the existing object with the new one
			return true         // Return true if the replacement was successful
		}
	}
	return false // Return false if no object with a matching key was found
}

// Deletes an object from the list by matching on its ID.
func (list *AzureObjectList) DeleteById(targetId string) bool {
	if targetId == "" {
		return false
	}
	for j := range *list {
		obj := &(*list)[j] // Access the element directly via pointer
		// Most objects use 'id' for their unique key, but RBAC role definitions
		// and assignments, and Subscriptions use different keys.
		if (*obj)["id"] == targetId || (*obj)["name"] == targetId || (*obj)["subscriptionId"] == targetId {
			// Modify the slice in-place by removing the matched element
			*list = append((*list)[:j], (*list)[j+1:]...)
			return true // Return true after successful deletion
		}
	}
	return false // Return false if no match is found
}

// Deletes an object from the list by matching on its displayName.
func (list *AzureObjectList) DeleteByName(targetName string) bool {
	if targetName == "" {
		return false
	}
	for j := range *list {
		obj := &(*list)[j] // Access the element directly via pointer
		if (*obj)["displayName"] == targetName {
			// Modify the slice in-place by removing the matched element
			*list = append((*list)[:j], (*list)[j+1:]...)
			return true // Return true after successful deletion
		}
	}
	return false // Return false if no match is found
}

// Deletes an object from the list based on one or more field matches.
func (list *AzureObjectList) Delete(targetObj AzureObject) bool {
	for j := range *list {
		obj := &(*list)[j] // Access the element directly via pointer
		matches := true
		// Check if the current object matches all criteria
		for key, targetValue := range targetObj {
			if value, ok := (*obj)[key]; !ok || value != targetValue {
				matches = false
				break
			}
		}
		if matches {
			// Modify the slice in-place by removing the matched element
			*list = append((*list)[:j], (*list)[j+1:]...)
			return true // Return true after successful deletion
		}
	}
	return false // Return false if no match is found
}

// Finds an object in the list by its ID and returns a pointer to it.
func (list AzureObjectList) FindById(targetId string) *AzureObject {
	if targetId == "" {
		return nil
	}
	for i := range list {
		obj := &list[i] // Access the element directly via pointer
		// Most objects use 'id' for their unique key, but RBAC role definitions
		// and assignments, and Subscriptions use different keys.
		if (*obj)["id"] == targetId || (*obj)["name"] == targetId || (*obj)["subscriptionId"] == targetId {
			return obj // Return pointer to the matching object
		}
	}
	return nil // Return nil if no match is found
}

// Finds an object in the list by its displayName and returns a pointer to it.
func (list AzureObjectList) FindByName(targetName string) *AzureObject {
	if targetName == "" {
		return nil
	}
	for i := range list {
		obj := &list[i] // Access the element directly via pointer
		if (*obj)["displayName"] == targetName {
			return obj // Return pointer to the matching object
		}
	}
	return nil // Return nil if no match is found
}

// Finds an object in the list based on one or more field matches and returns a pointer to it.
func (list AzureObjectList) Find(targetObj AzureObject) *AzureObject {
	for i := range list {
		obj := &list[i] // Access the element directly via pointer
		matches := true
		for key, targetValue := range targetObj {
			if value, ok := (*obj)[key]; !ok || value != targetValue {
				matches = false
				break
			}
		}
		if matches {
			return obj // Return pointer to the matching object
		}
	}
	return nil
}

// Checks if an object exists in the list by its ID.
func (list AzureObjectList) ExistsById(targetId string) bool {
	if targetId == "" {
		return false
	}
	for i := range list {
		obj := &list[i] // Access the element directly via pointer
		// Most objects use 'id' for their unique key, but RBAC role definitions
		// and assignments, and Subscriptions use different keys.
		if (*obj)["id"] == targetId || (*obj)["name"] == targetId || (*obj)["subscriptionId"] == targetId {
			return true
		}
	}
	return false
}

// Checks if an object exists in the list by its displayName.
func (list AzureObjectList) ExistsByName(targetName string) bool {
	if targetName == "" {
		return false
	}
	for i := range list {
		obj := &list[i] // Access the element directly via pointer
		if (*obj)["displayName"] == targetName {
			return true
		}
	}
	return false
}

// Checks if an object exists in the list based on one or more field matches.
func (list AzureObjectList) Exists(targetObj AzureObject) bool {
	for i := range list {
		obj := &list[i] // Access the element directly via pointer
		matches := true
		for key, targetValue := range targetObj {
			if value, ok := (*obj)[key]; !ok || value != targetValue {
				matches = false
				break
			}
		}
		if matches {
			return true
		}
	}
	return false
}
