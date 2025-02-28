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
func (obj AzureObject) TrimForCache(t string) (trimmed AzureObject) {
	// We restrict caching to only certain fields to make this library more performant.
	switch t {
	case "d": // Resource Role Definitions
		trimmed = AzureObject{
			"id":                 obj["id"],
			"displayName":        obj["displayName"],
			"description":        obj["description"],
			"isAssignableToRole": obj["isAssignableToRole"],
		}
	case "a": // Resource Role Assignments
		trimmed = AzureObject{
			"id":               obj["id"],
			"principalId":      obj["principalId"],
			"roleDefinitionId": obj["roleDefinitionId"],
			"scope":            obj["scope"],
		}
	case "s": // Resource Subscriptions
		trimmed = AzureObject{
			"id":          obj["id"],
			"displayName": obj["displayName"],
			"state":       obj["state"],
		}
	case "m": // Resource Management Groups
		trimmed = AzureObject{
			"id":          obj["id"],
			"displayName": obj["displayName"],
		}
	case "u": // Directory Users
		trimmed = AzureObject{
			"id":                obj["id"],
			"displayName":       obj["displayName"],
			"userPrincipalName": obj["userPrincipalName"],
		}
	case "g": // Directory Groups
		trimmed = AzureObject{
			"id":                 obj["id"],
			"displayName":        obj["displayName"],
			"description":        obj["description"],
			"createdDateTime":    obj["createdDateTime"],
			"isAssignableToRole": obj["isAssignableToRole"],
		}
	case "ap": // Directory Applications
		trimmed = AzureObject{
			"id":          obj["id"],
			"displayName": obj["displayName"],
			"appId":       obj["appId"],
		}
	case "sp": // Directory Service Principals
		trimmed = AzureObject{
			"id":                     obj["id"],
			"displayName":            obj["displayName"],
			"appId":                  obj["appId"],
			"appOwnerOrganizationId": obj["appOwnerOrganizationId"],
		}
	case "dr": // Directory Role Definitions
		trimmed = AzureObject{
			"id":          obj["id"],
			"displayName": obj["displayName"],
			"description": obj["description"],
			"isBuiltIn":   obj["isBuiltIn"],
			"isEnabled":   obj["isEnabled"],
			"templateId":  obj["templateId"],
			//"rolePermissions": obj["rolePermissions"],
		}
	case "da": // Directory Role Assignments
		trimmed = AzureObject{
			"id":               obj["id"],
			"directoryScopeId": obj["directoryScopeId"],
			"principalId":      obj["principalId"],
			"roleDefinitionId": obj["roleDefinitionId"],
		}
	default:
		// If type is unknown, just include the ID field.
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

// Replaces an object in an AzureObjectList list by matching on id.
func (list *AzureObjectList) Replace(newObj AzureObject) bool {
	id, idOk := newObj["id"].(string)
	if !idOk {
		return false // The new object must have an id field
	}

	// Iterate through the list to find an object with a matching id
	for j, obj := range *list {
		existingId, idOk := obj["id"].(string)
		if idOk && existingId == id {
			(*list)[j] = newObj // Replace the existing object with the new one
			return true         // Return true if the replacement was successful
		}
	}
	return false // Return false if no object with the matching id was found
}

// Deletes an object from the list by matching on its ID.
func (list *AzureObjectList) DeleteById(targetId string) bool {
	if !utl.ValidUuid(targetId) {
		return false
	}
	// Use Delete with ID criteria for deletion
	return list.Delete(AzureObject{"id": targetId})
}

// Deletes an object from the list by matching on its displayName.
func (list *AzureObjectList) DeleteByName(targetName string) bool {
	if targetName == "" {
		return false
	}
	// Use Delete with displayName criteria for deletion
	return list.Delete(AzureObject{"displayName": targetName})
}

// Deletes an object from the list based on one or more field matches.
func (list *AzureObjectList) Delete(criteria AzureObject) bool {
	for j, obj := range *list {
		matches := true
		// Check if the current object matches all criteria
		for key, targetValue := range criteria {
			if value, ok := obj[key]; !ok || value != targetValue {
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
	if !utl.ValidUuid(targetId) {
		return nil
	}
	return list.Find(AzureObject{"id": targetId})
}

// Finds an object in the list by its displayName and returns a pointer to it.
func (list AzureObjectList) FindByName(targetName string) *AzureObject {
	if targetName == "" {
		return nil
	}
	return list.Find(AzureObject{"displayName": targetName})
}

// Finds an object in the list based on one or more field matches and returns a pointer to it.
func (list AzureObjectList) Find(criteria AzureObject) *AzureObject {
	for i := range list {
		obj := &list[i] // Get a pointer to the current object
		matches := true
		for key, targetValue := range criteria {
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
	if !utl.ValidUuid(targetId) {
		return false
	}
	return list.Exists(AzureObject{"id": targetId})
}

// Checks if an object exists in the list by its displayName.
func (list AzureObjectList) ExistsByName(targetName string) bool {
	if targetName == "" {
		return false
	}
	return list.Exists(AzureObject{"displayName": targetName})
}

// Checks if an object exists in the list based on one or more field matches.
func (list AzureObjectList) Exists(criteria AzureObject) bool {
	for _, obj := range list {
		matches := true
		for key, targetValue := range criteria {
			if value, ok := obj[key]; !ok || value != targetValue {
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
