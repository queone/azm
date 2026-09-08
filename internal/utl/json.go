package utl

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Prints JSON object, flushing the output buffer

// Convert JSON interface object to byte slice, with option to indent spacing

// Convert JSON interface object to byte slice, with option to indent spacing
func JsonToBytesIndent(jsonObject any, indent int) (jsonBytes []byte, err error) {
	indentStr := strings.Repeat(" ", indent)
	jsonBytes, err = json.MarshalIndent(jsonObject, "", indentStr)
	if err != nil {
		return nil, err
	}
	return jsonBytes, nil
}

// Convert JSON interface object to byte slice, with default 2-space indentation
func jsonToBytes(jsonObject any) (jsonBytes []byte, err error) {
	indent := 2 // With default 2 space indent
	jsonBytes, err = JsonToBytesIndent(jsonObject, indent)
	return jsonBytes, err
}

// Convert JSON interface object to a colorized string representation

// Convert JSON object to bytes

// Create a pipe to capture the output

// Redirect standard output to the pipe

// Print the colorized JSON bytes

// Restore the standard output

// Read the output from the pipe

// Return the colorized string

// Convert JSON byte slice to JSON interface object, with default 2-space indentation

// NOTE: To be replaced by jsonToBytes()

// Print JSON object in color
func PrintJsonColor(jsonObject any) {
	jsonBytes, err := jsonToBytes(jsonObject)
	if err != nil {
		fmt.Println(err.Error())
		return
	}
	printJsonBytesColor(jsonBytes)
}

// Prints JSON byte slice in color. Just an alias of yaml.go:printYamlBytesColor().
func printJsonBytesColor(jsonBytes []byte) {
	printYamlBytesColor(jsonBytes)
}

// Combines two string-to-string maps, with keys from the second map overwriting those
// from the first if duplicates exist. Returns the merged map.

// Alias for  MergeObjects function.

// Recursively merges the keys from object a into object b. Existing object b attributes
// are overwritten if there's a conflict.

// If both values are maps, recursively merge them

// Otherwise, overwrite or add the value from a to b

// Recursive function returns True if filter string value is anywhere within jsonObject
