package repo

import (
	"fmt"
	"os"
	"path/filepath"
)

var tempDir = filepath.Join(".", "temp")

func TempDataDirectory() string {
	return tempDir
}

func CreateTempDataDirectory() error {
	if err := os.MkdirAll(tempDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create temp directory: %v", err)
	}

	return nil
}

func RemoveTempDataDirectory() error {
	if err := os.RemoveAll(tempDir); err != nil {
		return fmt.Errorf("failed to remove temp directory: %v", err)
	}

	return nil
}

func CreateTempDataDirectoryForRepo(repoName string) (string, error) {
	tempRepoDir := filepath.Join(TempDataDirectory(), repoName)

	if err := os.MkdirAll(tempRepoDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create temp repository directory: %v", err)
	}

	return tempRepoDir, nil
}

func RemoveTempDataDirectoryForRepo(repoName string) error {
	tempRepoDir := filepath.Join(TempDataDirectory(), repoName)

	if err := os.RemoveAll(tempRepoDir); err != nil {
		return fmt.Errorf("failed to remove temp repository directory: %v", err)
	}

	return nil
}