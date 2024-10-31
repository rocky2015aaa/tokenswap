package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func CreateFile(data []byte, path string, permissions os.FileMode) error {
	// Check if directory exists
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		// Create directory with specified permissions
		err = os.MkdirAll(filepath.Dir(path), permissions)
		if err != nil {
			return fmt.Errorf("error creating directory structure: %w", err)
		}
	}

	// Create the file with specified permissions
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write data to the file
	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

func ReadFile(path string, data interface{}) error {
	// Open the file for reading
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close() // Ensure file is closed regardless of errors

	// Read the entire file content into a byte slice
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return fmt.Errorf("error reading file: %w", err)
	}

	// Unmarshal the JSON data from the byte slice
	err = json.Unmarshal(fileBytes, data)
	if err != nil {
		return fmt.Errorf("error unmarshalling JSON data: %w", err)
	}

	return nil
}

func UpdateFile(data interface{}, path string) error {
	// Marshal data to JSON with indentation
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshalling data to JSON: %w", err)
	}

	// Open the file for writing with appropriate permissions
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("error opening file for writing: %w", err)
	}
	defer file.Close() // Ensure file is closed regardless of errors

	// Write the JSON data to the file
	_, err = file.Write(jsonData)
	if err != nil {
		return fmt.Errorf("error writing data to file: %w", err)
	}

	return nil
}
