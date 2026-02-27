package scanner

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type ScanResult struct {
	ProjectPath     string
	NodeModulesPath string
	IsPnpm          bool
}

// shouldSkip checks against common heavy or system directories we don't need to traverse
func shouldSkip(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	skipDirs := map[string]bool{
		"dist":  true,
		"build": true,
	}
	return skipDirs[name]
}

// shouldSkipPath checks if the absolute path falls under known system directories
// that should never contain user-managed node_modules.
func shouldSkipPath(path string) bool {
	restrictedSubstrings := []string{
		"/usr/lib",
		"/usr/share",
		"/var/lib/flatpak",
		"/snap/",
	}

	for _, restricted := range restrictedSubstrings {
		if strings.Contains(path, restricted) {
			return true
		}
	}
	return false
}

// isPnpmManaged checks if a directory contains pnpm optimization signatures
func isPnpmManaged(modPath string, projPath string) bool {
	if _, err := os.Stat(filepath.Join(modPath, ".modules.yaml")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(projPath, "pnpm-lock.yaml")); err == nil {
		return true
	}
	return false
}

// FindNodeModules securely walks down the filesystem starting at root
// limiting memory usage by yielding entries directly and avoiding deep stats.
func FindNodeModules(root string) ([]ScanResult, error) {
	var results []ScanResult

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Skip permissions or missing directory errors quietly
			if errors.Is(err, fs.ErrPermission) || errors.Is(err, fs.ErrNotExist) {
				return filepath.SkipDir
			}
			return nil
		}

		if !d.IsDir() {
			return nil
		}

		dirName := d.Name()

		// Do not traverse into known skip directories
		if shouldSkip(dirName) {
			return filepath.SkipDir
		}

		// Do not traverse restricted system paths
		if shouldSkipPath(path) {
			return filepath.SkipDir
		}

		// Don't traverse inside a node_modules folder itself
		if strings.Contains(path, "node_modules"+string(os.PathSeparator)) {
			return filepath.SkipDir
		}

		if dirName == "node_modules" {
			projectPath := filepath.Dir(path)
			isPnpm := isPnpmManaged(path, projectPath)

			results = append(results, ScanResult{
				ProjectPath:     projectPath,
				NodeModulesPath: path,
				IsPnpm:          isPnpm,
			})

			// Skip going deeper inside this node_modules
			return filepath.SkipDir
		}

		return nil
	})

	if err != nil {
		return results, err
	}

	return results, nil
}
