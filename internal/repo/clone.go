package repo

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

func CloneRepositoryByTag(url string, tag string, repoName string) error {
	dir := filepath.Join(TempDataDirectory(), repoName)
	cmd := exec.Command("git", "clone", "--branch", tag, "--single-branch", "--depth", "1", url, dir)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to clone %s: %v", url, err)
	}

	return nil
}
