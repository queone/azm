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
	case ResRoleDefinition:
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
	case ResRoleAssignment:
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
		// Most objects use 'id' for their unique key, but resource role
		// definitions and assignments, and Subscriptions use different keys.
		if (id != "" && (*obj)["id"] == id) ||
			(name != "" && (*obj)["name"] == name) ||
			(subscriptionId != "" && (*obj)["subscriptionId"] == subscriptionId) {
			(*list)[j] = newObj // Replace the existing object with the new one
			return true         // Return true if the replacement was successful
		}
	}
	return false // Return false if no object with a matching key was found
}

// Optimized AzureObjectList methods
func (list *AzureObjectList) DeleteById(targetId string) bool {
	if targetId == "" {
		return false
	}

	for i := 0; i < len(*list); i++ {
		obj := (*list)[i] // Access the element directly via pointer
		if matchId(obj, targetId) {
			// Modify the slice in-place by removing the matched element
			*list = append((*list)[:i], (*list)[i+1:]...)
			return true // Return true after successful deletion
		}
	}
	return false
}

// Helper function
func matchId(obj AzureObject, targetId string) bool {
	// Most objects use 'id' for their unique key, but resource role
	// definitions and assignments, and Subscriptions use different keys.
	return obj["id"] == targetId ||
		obj["name"] == targetId ||
		obj["subscriptionId"] == targetId
}

// Helper function
func matchAnyId(obj AzureObject, ids utl.StringSet) bool {
	if _, exists := ids[utl.Str(obj["id"])]; exists {
		return true
	}
	if _, exists := ids[utl.Str(obj["name"])]; exists {
		return true
	}
	if _, exists := ids[utl.Str(obj["subscriptionId"])]; exists {
		return true
	}
	return false
}

// New batch deletion method
func (list *AzureObjectList) BatchDeleteByIds(ids utl.StringSet) int {
	if len(ids) == 0 {
		return 0
	}

	count := 0
	newList := make(AzureObjectList, 0, len(*list))

	for _, obj := range *list {
		if !matchAnyId(obj, ids) {
			newList = append(newList, obj)
		} else {
			count++
		}
	}

	*list = newList
	return count
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
		// Most objects use 'id' for their unique key, but resource role
		// definitions and assignments, and Subscriptions use different keys.
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
		// Most objects use 'id' for their unique key, but resource role
		// definitions and assignments, and Subscriptions use different keys.
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
