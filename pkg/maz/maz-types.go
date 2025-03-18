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
// This method performs a recursive search.
func (obj AzureObject) HasString(filter string) bool {
	// TODO: Consider optimizing further
	for _, value := range obj {
		switch v := value.(type) {
		case string:
			if utl.SubString(v, filter) { // Directly check if the substring exists
				return true
			}
		case []interface{}:
			for _, item := range v {
				if nestedMap, ok := item.(map[string]interface{}); ok {
					nestedObj := AzureObject(nestedMap) // Convert to AzureObject
					if nestedObj.HasString(filter) {
						return true
					}
				} else if itemStr, ok := item.(string); ok {
					// Check string elements in the slice
					if utl.SubString(itemStr, filter) {
						return true
					}
				}
			}
		case map[string]interface{}:
			// Recursively call HasString on nested maps
			nestedObj := AzureObject(v)
			if nestedObj.HasString(filter) {
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
		if properties, ok := obj["properties"].(map[string]interface{}); ok {
			trimmed = AzureObject{
				"id":   obj["id"],
				"name": obj["name"],
				"properties": map[string]interface{}{
					"assignableScopes": properties["assignableScopes"],
					"description":      properties["description"],
					"permissions":      properties["permissions"],
					"roleName":         properties["roleName"],
					"type":             properties["type"],
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
		if properties, ok := obj["properties"].(map[string]interface{}); ok {
			trimmed = AzureObject{
				"id":   obj["id"],
				"name": obj["name"],
				"properties": map[string]interface{}{
					"roleDefinitionId": properties["roleDefinitionId"],
					"description":      properties["description"],
					"principalId":      properties["principalId"],
					"principalType":    properties["principalType"],
					"scope":            properties["scope"],
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
		trimmed = AzureObject{
			"id":   obj["id"],
			"name": obj["name"],
		}
		// Normalize object by extracting fields from the nested "properties" object
		if properties, ok := obj["properties"].(map[string]interface{}); ok {
			trimmed["displayName"] = properties["displayName"]
			trimmed["tenantId"] = properties["tenantId"]
			// OPTIONAL: Keep the same struct as Azure does
			// trimmed["properties"] = map[string]interface{}{
			// 	"displayName": properties["displayName"],
			// 	"tenantId":    properties["tenantId"],
			// }
		} else {
			// Fallback if "properties" is missing or not a map
			trimmed["displayName"] = nil
			trimmed["tenantId"] = nil
			// OPTIONAL: Keep the same struct as Azure does
			// trimmed["properties"] = map[string]interface{}{}
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

// Initializes a new list of objects.
func NewList() AzureObjectList {
	return make(AzureObjectList, 0)
}

// Add appends an AzureObject to the AzureObjectList.
func (list *AzureObjectList) Add(obj AzureObject) {
	*list = append(*list, obj) // Append the new object to the list
}

// EFFICIENCY NOTE: Below functions use 'for i := range list' and access each
// element via '&list[i]' to avoid copying each item in the list. This is more
// efficient than 'for _, obj := range list', which creates a copy of each
// element during iteration. This is especially beneficial for large lists or
// large objects, as it reduces memory overhead and improves performance.

// Replaces an object in an AzureObjectList by matching on id, name, or subscriptionId.
func (list *AzureObjectList) Replace(newObj AzureObject) bool {
	id, idOk := newObj["id"].(string)
	name, nameOk := newObj["name"].(string)
	subscriptionId, subscriptionIdOk := newObj["subscriptionId"].(string)

	// The new object must have at least one of the unique keys
	if !idOk && !nameOk && !subscriptionIdOk {
		return false
	}

	// Iterate through the list to find an object with a matching key
	for j := range *list {
		obj := &(*list)[j] // Access the element directly via pointer
		// Most objects use 'id' for their unique key, but RBAC role definitions
		// and assignments, and Subscriptions use different keys.
		if (idOk && (*obj)["id"] == id) ||
			(nameOk && (*obj)["name"] == name) ||
			(subscriptionIdOk && (*obj)["subscriptionId"] == subscriptionId) {
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
	// See EFFICIENCY NOTE above
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
	// See EFFICIENCY NOTE above
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
	// See EFFICIENCY NOTE above
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
	// See EFFICIENCY NOTE above
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
	// See EFFICIENCY NOTE above
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
	// See EFFICIENCY NOTE above
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
	// See EFFICIENCY NOTE above
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
	// See EFFICIENCY NOTE above
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
