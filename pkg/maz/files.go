package maz

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"fmt"
	"os"
	"time"
)

// Saves a single AzureObject as a gob binary file with specified permissions.
func SaveFileBinaryObject(filePath string, data AzureObject, perm os.FileMode) error {
	// Step 1: Encode data into a buffer
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data to binary: %w", err)
	}

	// Replace the original file atomically by writing to a temporary file first
	// and renaming it. This ensures the original file remains intact if an error occurs.

	// Step 2: Write to a temporary file
	tempFilePath := filePath + ".tmp"
	if err := os.WriteFile(tempFilePath, buf.Bytes(), perm); err != nil {
		return fmt.Errorf("failed to write data to temporary file: %w", err)
	}

	// Step 3: Replace the original file atomically
	if err := os.Rename(tempFilePath, filePath); err != nil {
		return fmt.Errorf("failed to replace original file with new data: %w", err)
	}

	return nil
}

// Reads a gob binary file and decodes it into an AzureObject.
func LoadFileBinaryObject(filePath string) (AzureObject, error) {
	// Step 1: Check if the file exists and is usable
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, err // Preserve os.IsNotExist compatibility
		}
		return nil, fmt.Errorf("file access error: %w", err)
	}
	if info.Size() == 0 {
		return nil, fmt.Errorf("file exists but is zero size: %s", filePath)
	}

	// Step 2: Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Step 3: Decode the file content
	var data AzureObject
	decoder := gob.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode binary file: %w", err)
	}

	return data, nil
}

// Serializes a slice of AzureObject objects into a gob binary file.
// The file is saved with the specified permissions, optionally compressed using Gzip.
func SaveFileBinaryList(filePath string, data AzureObjectList, perm os.FileMode, compress bool) error {
	// Step 1: Encode data into a buffer
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)

	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode data to binary: %w", err)
	}

	// Step 2: Handle compression if required
	var outputData []byte
	if compress {
		var compressedBuf bytes.Buffer
		gzipWriter := gzip.NewWriter(&compressedBuf)

		// Write encoded data to gzip writer
		if _, err := gzipWriter.Write(buf.Bytes()); err != nil {
			gzipWriter.Close() // Explicitly close even on error
			return fmt.Errorf("failed to compress binary data: %w", err)
		}

		// Ensure gzip writer closes cleanly
		if err := gzipWriter.Close(); err != nil {
			return fmt.Errorf("failed to close gzip writer: %w", err)
		}

		outputData = compressedBuf.Bytes()
	} else {
		outputData = buf.Bytes()
	}

	// Replace the original file atomically by writing to a temporary file first
	// and renaming it. This ensures the original file remains intact if an error occurs.

	// Step 3: Write to the file
	tempFilePath := filePath + ".tmp"
	if err := os.WriteFile(tempFilePath, outputData, perm); err != nil {
		return fmt.Errorf("failed to write data to temporary file: %w", err)
	}

	// // Step 4: Atomic file replacement
	// if err := os.Rename(tempFilePath, filePath); err != nil {
	// 	return fmt.Errorf("failed to replace old file with new data: %w", err)
	// }
	// Step 4: Atomic file replacement with retry
	const maxRetries = 5
	for i := 0; i < maxRetries; i++ {
		if err := os.Rename(tempFilePath, filePath); err != nil {
			if i < maxRetries-1 {
				time.Sleep(time.Duration(i+1) * time.Second) // Exponential backoff
				continue
			}
			return fmt.Errorf("failed to replace old file with new data: %w", err)
		}
		break // Success, exit the loop
	}

	return nil
}

// Reads a gob binary file and decodes it into a slice of AzureObject.
// If the file is compressed, it decompresses it using Gzip before decoding.
func LoadFileBinaryList(filePath string, compressed bool) (AzureObjectList, error) {
	// Check if the file exists
	info, err := os.Stat(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return the error without wrapping so os.IsNotExist(err) works
			return nil, err
		}
		return nil, fmt.Errorf("file access error: %w", err)
	}

	// Check if the file is empty
	if info.Size() == 0 {
		return nil, fmt.Errorf("file exists but is zero size: %s", filePath)
	}

	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var data AzureObjectList
	if compressed {
		// Use Gzip reader to decompress
		gzipReader, err := gzip.NewReader(file)
		if err != nil {
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer gzipReader.Close()

		decoder := gob.NewDecoder(gzipReader)
		if err := decoder.Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to decode compressed binary file: %w", err)
		}
	} else {
		// Decode directly without compression
		decoder := gob.NewDecoder(file)
		if err := decoder.Decode(&data); err != nil {
			return nil, fmt.Errorf("failed to decode binary file: %w", err)
		}
	}

	return data, nil
}
