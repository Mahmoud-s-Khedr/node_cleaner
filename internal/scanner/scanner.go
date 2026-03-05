package scanner

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type ScanResult struct {
	ProjectPath     string
	NodeModulesPath string
	IsPnpm          bool
	PackageManager  string // "pnpm", "yarn", "npm"
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

// restrictedPaths returns OS-specific paths that should never contain user-managed node_modules.
func restrictedPaths() []string {
	if runtime.GOOS == "windows" {
		return []string{
			`C:\Windows`,
			`C:\Program Files`,
			`C:\Program Files (x86)`,
		}
	}
	// Linux / macOS
	return []string{
		"/usr/lib",
		"/usr/share",
		"/var/lib/flatpak",
		"/snap/",
	}
}

// shouldSkipPath checks if the absolute path falls under known system directories
// that should never contain user-managed node_modules.
func shouldSkipPath(path string) bool {
	for _, restricted := range restrictedPaths() {
		if strings.Contains(path, restricted) {
			return true
		}
	}
	return false
}

// DetectPackageManager inspects the project directory for lock files to decide
// which package manager owns the project. Priority: pnpm > yarn > npm.
func DetectPackageManager(modPath string, projPath string) string {
	if _, err := os.Stat(filepath.Join(modPath, ".modules.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(projPath, "pnpm-lock.yaml")); err == nil {
		return "pnpm"
	}
	if _, err := os.Stat(filepath.Join(projPath, "yarn.lock")); err == nil {
		return "yarn"
	}
	if _, err := os.Stat(filepath.Join(projPath, "package-lock.json")); err == nil {
		return "npm"
	}
	// Fallback: if there is a package.json we default to npm
	return "npm"
}

// isPnpmManaged checks if a directory contains pnpm optimization signatures
func isPnpmManaged(modPath string, projPath string) bool {
	return DetectPackageManager(modPath, projPath) == "pnpm"
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
			pm := DetectPackageManager(path, projectPath)

			results = append(results, ScanResult{
				ProjectPath:     projectPath,
				NodeModulesPath: path,
				IsPnpm:          pm == "pnpm",
				PackageManager:  pm,
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
