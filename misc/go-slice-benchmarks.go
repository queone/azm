package main

import (
	"fmt"
	"time"
)

// Safely casts obj to a map (map[string]interface{}); returns nil if not possible.
func Map(obj interface{}) map[string]interface{} {
	if objMap, ok := obj.(map[string]interface{}); ok {
		return objMap
	}
	return nil
}

// Safely casts obj to a string (string); returns empty string "" if not possible.
func Str(obj interface{}) string {
	if objString, ok := obj.(string); ok {
		return objString
	}
	return ""
}

// generateAssignments creates a slice of assignment objects as maps.
func generateAssignments(numElements int) []interface{} {
	assignments := make([]interface{}, numElements)
	for i := 0; i < numElements; i++ {
		assignments[i] = map[string]interface{}{
			"id":   fmt.Sprintf("id-%d", i),
			"name": fmt.Sprintf("name-%d", i),
			"properties": map[string]interface{}{
				"principalId":      fmt.Sprintf("principal-%d", i),
				"roleDefinitionId": fmt.Sprintf("role-%d", i),
				"scope":            fmt.Sprintf("scope-%d", i),
			},
		}
	}
	return assignments
}

// Method 1: Using value-based loop (original)
func method1(numElements int) time.Duration {
	assignments := generateAssignments(numElements)
	start := time.Now()
	var count int
	for _, obj := range assignments {

		assignment := Map(obj)

		if assignment != nil {
			// Access a field from properties to simulate work.
			props := Map(assignment["properties"])
			_ = Str(props["scope"])
			count++ // Prevent dead-code elimination.
		}
	}
	return time.Since(start)
}

// Method 2: Using index-based loop with pointer derefencing
func method2(numElements int) time.Duration {
	assignments := generateAssignments(numElements)
	start := time.Now()
	var count int
	for i := range assignments {

		ptr := &assignments[i]
		obj := *ptr
		assignment := Map(obj)

		if assignment != nil {
			// Access a field from properties to simulate work.
			props := Map(assignment["properties"])
			_ = Str(props["scope"])
			count++ // Prevent dead-code elimination.
		}
	}
	return time.Since(start)
}

// Method 3: Using index-based loop alone
func method3(numElements int) time.Duration {
	assignments := generateAssignments(numElements)
	start := time.Now()
	var count int
	for i := range assignments {

		assignment := Map(assignments[i])

		if assignment != nil {
			// Access a field from properties to simulate work.
			props := Map(assignment["properties"])
			_ = Str(props["scope"])
			count++ // Prevent dead-code elimination.
		}
	}
	return time.Since(start)
}

func main() {
	numElements := 4000
	duration1 := method1(numElements)
	fmt.Printf("Method 1: Using value-based loop (original)               %7d elements took : %v\n", numElements, duration1)
	duration2 := method2(numElements)
	fmt.Printf("Method 2: Using index-based loop with pointer derefencing %7d elements took : %v\n", numElements, duration2)
	duration3 := method3(numElements)
	fmt.Printf("Method 3: Using index-based loop alone                    %7d elements took : %v\n", numElements, duration3)
	fmt.Println()
	numElements = 4000
	duration1 = method1(numElements)
	fmt.Printf("Method 1: Using value-based loop (original)               %7d elements took : %v\n", numElements, duration1)
	duration2 = method2(numElements)
	fmt.Printf("Method 2: Using index-based loop with pointer derefencing %7d elements took : %v\n", numElements, duration2)
	duration3 = method3(numElements)
	fmt.Printf("Method 3: Using index-based loop alone                    %7d elements took : %v\n", numElements, duration3)

	// RESULTS
	// 1. These 2 methods yield essentially the same performace results.
	// 2. The index-based loops seems cleaner and easier to read, and it thefore recommended.
}

// Test with: go run THIS_GO_FILE
