package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// Config holds the user-defined skip-list for NmodCleaner.
type Config struct {
	// SkipPaths is a list of project paths that should never be cleaned.
	// Both exact matches and path prefixes are honoured.
	SkipPaths []string `json:"skipPaths"`
}

// Load reads the config file from the given path. If path is empty, it searches
// for .nmodcleanerrc in $HOME then the current working directory. If no config
// file is found, an empty Config is returned (not an error).
func Load(configPath string) (*Config, error) {
	path, err := resolve(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &Config{}, nil
		}
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ShouldSkip reports whether the given projectPath is on the skip-list.
// A project is skipped when its path is an exact match or a subpath of any
// entry in SkipPaths.
func (c *Config) ShouldSkip(projectPath string) bool {
	for _, skip := range c.SkipPaths {
		skip = filepath.Clean(skip)
		projectPath = filepath.Clean(projectPath)
		if projectPath == skip || strings.HasPrefix(projectPath, skip+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// resolve finds the config file path. Falls back to $HOME then $CWD.
func resolve(configPath string) (string, error) {
	if configPath != "" {
		if _, err := os.Stat(configPath); err != nil {
			return "", err
		}
		return configPath, nil
	}

	candidates := []string{}

	if home, err := os.UserHomeDir(); err == nil {
		candidates = append(candidates, filepath.Join(home, ".nmodcleanerrc"))
	}
	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, ".nmodcleanerrc"))
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", os.ErrNotExist
}
