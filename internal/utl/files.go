package utl

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Saves given JSON object as a text file, with compression as an option
// which can sometimes help to speed things up for some very large objects.
func SaveFileJson(jsonObject any, filePath string, compress bool) {
	jsonData, err := json.Marshal(jsonObject)
	if err != nil {
		panic(err.Error())
	}

	if compress {
		file, err := os.Create(filePath)
		if err != nil {
			panic(err.Error())
		}
		defer file.Close()

		gzipWriter := gzip.NewWriter(file)
		defer gzipWriter.Close()

		_, err = gzipWriter.Write(jsonData)
		if err != nil {
			panic(err.Error())
		}
	} else {
		err = os.WriteFile(filePath, jsonData, 0600)
		if err != nil {
			panic(err.Error())
		}
	}
}

// Reads, load, and decode given filePath as a JSON object text file. Returns
// JSON object and err if any. The compression option can sometimes help speed
// things up for some very large objects.

// Saves given byte slice as text file.
// Returns error is any.

// Read and recode given filePath as text byte slice.
// Returns the byte slice and error if any.

// Save given YAML object to given filePath

// Tries to read, load, and decode given filePath as some YAML object.
// Returns YAML object and error if any.
func LoadFileYaml(filePath string) (yamlObject any, err error) {
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(fileContent, &yamlObject)
	if err != nil {
		return nil, err
	}
	return yamlObject, nil
}

// Tries to read, load, and recode given filePath as some YAML object as
// byte slice. Returns YAML object as byte slice. and error if any.

// Load YAML file into byte slice, including comments
// Can also JSON file into byte slice!

// Check YAML formatting compliancy using "github.com/goccy/go-yaml"
// which provides errors with line numbers

// We only care about returning the byte slice

// Removes the given filepath if it exists.
func RemoveFile(filePath string) error {
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			return nil // Ignore if the file does not exist
		}
		return err // Return the error
	}
	return os.Remove(filePath)
}

// Returns true if filepath exists and has some content. False otherwise.
func FileUsable(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false // File does not exist or cannot be accessed
	}
	return info.Size() > 0 // Check if file has content
}

// Returns true if given filePath exists. False otherwise.
func FileExist(filePath string) (e bool) {
	if _, err := os.Stat(filePath); err == nil || os.IsExist(err) {
		return true
	}
	return false
}

// Returns size of given filePath as int64

// Returns true is given filePath does not exist. False otherwise.

// Returns given filePath modified time in Unix epoch int

// Returns the age of the given filePath in seconds as int64.
// If the file does not exist or is empty, it returns -1.
func FileAge(filePath string) int64 {
	info, err := os.Stat(filePath)
	if err != nil || info.Size() == 0 {
		return -1 // Return -1 to signify an error or empty file
	}
	return int64(time.Since(info.ModTime()).Seconds())
}

// Checks if a given file contains valid JSON

// Use a decoder to validate JSON while reading

// Decode each JSON value in the file

// File is valid and reached the end

// Checks if a given file contains valid YAML

// Decode each YAML document in the file

// File is valid and reached the end

// LoadFileAuto reads a file, detects compression (gzip), and decodes it as JSON,
// YAML, or plain text. Returns raw object, format string ('json', 'yaml', or
// 'text' ), and error.
func LoadFileAuto(filePath string) (any, string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to open file: %v", err)
	}
	defer f.Close()

	// Read the first 512 bytes to detect compression
	buffer := make([]byte, 512)
	n, err := f.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, "", fmt.Errorf("failed to read file header: %v", err)
	}

	// Check if the file is gzip compressed (magic number: 0x1F 0x8B)
	var reader io.Reader
	if n >= 2 && buffer[0] == 0x1F && buffer[1] == 0x8B {
		// Reset the file pointer to the beginning
		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			return nil, "", fmt.Errorf("failed to reset file pointer: %v", err)
		}

		gzipReader, err := gzip.NewReader(f)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create gzip reader: %v", err)
		}
		defer gzipReader.Close()
		reader = gzipReader
	} else {
		// Reset the file pointer to the beginning
		_, err = f.Seek(0, io.SeekStart)
		if err != nil {
			return nil, "", fmt.Errorf("failed to reset file pointer: %v", err)
		}
		reader = f
	}

	// Read the entire file content
	byteValue, err := io.ReadAll(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read file content: %v", err)
	}

	// Try JSON first
	var jsonObj any
	err = json.Unmarshal(byteValue, &jsonObj)
	if err == nil {
		return jsonObj, "json", nil
	}

	// Fallback to YAML
	var yamlObj any
	err = yaml.Unmarshal(byteValue, &yamlObj)
	if err == nil {
		return yamlObj, "yaml", nil
	}

	// Fallback to plain text
	textContent := string(byteValue)
	return textContent, "text", nil
}

// SaveFileAuto writes a file in the specified format (json/yaml/text), optionally compressed,
// with the given permissions (default 0600 if perm is 0).
func SaveFileAuto(filePath, format string, rawObj any, compress bool, perm os.FileMode) error {
	// Examples
	// err := SaveFileAuto("output.json", "json", data, false, 0)       // JSON with default permissions
	// err := SaveFileAuto("output.yaml.gz", "yaml", data, true, 0640)  // compressed YAML with custom permissions
	// err := SaveFileAuto("output.txt", "text", "some text", false, 0) // plain text

	// Set default permissions if not specified
	if perm == 0 {
		perm = 0600
	}

	// Create the file with the specified permissions
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer f.Close()

	var writer io.Writer = f

	// Add gzip compression if requested
	if compress {
		gzipWriter := gzip.NewWriter(f)
		defer gzipWriter.Close()
		writer = gzipWriter
	}

	// Convert the object to bytes based on format
	var byteValue []byte
	var marshalErr error

	switch strings.ToLower(format) {
	case "json":
		byteValue, marshalErr = json.MarshalIndent(rawObj, "", "  ")
	case "yaml":
		byteValue, marshalErr = yaml.Marshal(rawObj)
	case "text":
		switch v := rawObj.(type) {
		case string:
			byteValue = []byte(v)
		case []byte:
			byteValue = v
		default:
			byteValue = []byte(fmt.Sprintf("%v", v))
		}
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}

	if marshalErr != nil {
		return fmt.Errorf("failed to marshal object: %v", marshalErr)
	}

	// Write the content
	_, err = writer.Write(byteValue)
	if err != nil {
		return fmt.Errorf("failed to write file content: %v", err)
	}

	return nil
}
