package cargo

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type CargoToml struct {
    Package struct {
        Name    string `toml:"name"`
        Version string `toml:"version"`
    } `toml:"package"`
    Dependencies map[string]interface{} `toml:"dependencies"`
}

func LoadCargoTomlFromDir(dir string) (*CargoToml, error) {
	manifestPath := filepath.Join(dir, "Cargo.toml")

	if _, err := os.Stat(manifestPath); err != nil {
		return nil, fmt.Errorf("cargo toml not found at %s: %w", manifestPath, err)
	}

	cargoToml := CargoToml{}

	if _, err := toml.DecodeFile(manifestPath, &cargoToml); err != nil {
		return nil, fmt.Errorf("failed to decode cargo toml from %s: %w", manifestPath, err)
	}

	return &cargoToml, nil
}
