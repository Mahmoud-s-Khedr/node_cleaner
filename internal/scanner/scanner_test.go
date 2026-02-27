package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestShouldSkip(t *testing.T) {
	skip := []string{".git", ".node", "dist", "build"}
	for _, name := range skip {
		if !shouldSkip(name) {
			t.Errorf("shouldSkip(%q) = false, want true", name)
		}
	}

	keep := []string{"src", "my-project", "packages", "apps"}
	for _, name := range keep {
		if shouldSkip(name) {
			t.Errorf("shouldSkip(%q) = true, want false", name)
		}
	}
}

func TestShouldSkipPath(t *testing.T) {
	skip := []string{
		"/usr/lib/something",
		"/usr/share/cursor",
		"/var/lib/flatpak/app",
		"/snap/core/current",
	}
	for _, path := range skip {
		if !shouldSkipPath(path) {
			t.Errorf("shouldSkipPath(%q) = false, want true", path)
		}
	}

	keep := []string{
		"/home/user/projects",
		"/opt/myapp",
		"/tmp/workspace",
	}
	for _, path := range keep {
		if shouldSkipPath(path) {
			t.Errorf("shouldSkipPath(%q) = true, want false", path)
		}
	}
}

func TestIsPnpmManaged_ModulesYaml(t *testing.T) {
	modPath := t.TempDir()
	projPath := t.TempDir()

	// No markers yet — should not be pnpm
	if isPnpmManaged(modPath, projPath) {
		t.Error("isPnpmManaged = true before any markers, want false")
	}

	// Add .modules.yaml inside node_modules
	if err := os.WriteFile(filepath.Join(modPath, ".modules.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if !isPnpmManaged(modPath, projPath) {
		t.Error("isPnpmManaged = false after .modules.yaml, want true")
	}
}

func TestIsPnpmManaged_LockFile(t *testing.T) {
	modPath := t.TempDir()
	projPath := t.TempDir()

	// Add pnpm-lock.yaml in the project root
	if err := os.WriteFile(filepath.Join(projPath, "pnpm-lock.yaml"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}
	if !isPnpmManaged(modPath, projPath) {
		t.Error("isPnpmManaged = false after pnpm-lock.yaml, want true")
	}
}

func TestFindNodeModules(t *testing.T) {
	// Build a temp tree:
	//   root/
	//     projectA/
	//       node_modules/
	//     projectB/
	//       node_modules/
	//         .modules.yaml    <- pnpm-managed, should be flagged IsPnpm=true
	//     dist/                <- should be skipped entirely
	//       node_modules/

	root := t.TempDir()

	mkdirAll := func(parts ...string) {
		if err := os.MkdirAll(filepath.Join(parts...), 0755); err != nil {
			t.Fatal(err)
		}
	}
	writeFile := func(content string, parts ...string) {
		if err := os.WriteFile(filepath.Join(parts...), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mkdirAll(root, "projectA", "node_modules")
	mkdirAll(root, "projectB", "node_modules")
	writeFile("", root, "projectB", "node_modules", ".modules.yaml")
	mkdirAll(root, "dist", "node_modules") // inside "dist", should be skipped

	results, err := FindNodeModules(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find exactly 2 (not the one inside dist/)
	if len(results) != 2 {
		t.Errorf("FindNodeModules found %d results, want 2", len(results))
	}

	pnpmCount := 0
	for _, r := range results {
		if r.IsPnpm {
			pnpmCount++
		}
	}
	if pnpmCount != 1 {
		t.Errorf("IsPnpm=true count = %d, want 1", pnpmCount)
	}
}
