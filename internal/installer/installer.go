package installer

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// binaryName returns the platform-correct name for the given package manager binary.
// On Windows, npm, yarn, and pnpm ship as .cmd wrappers.
func binaryName(manager string) string {
	if runtime.GOOS == "windows" {
		return manager + ".cmd"
	}
	return manager
}

// RunInstall executes the appropriate package manager install command within
// the given project directory. The manager argument should be "pnpm", "yarn",
// or "npm".
func RunInstall(cwd string, manager string) error {
	bin := binaryName(manager)
	cmd := exec.Command(bin, "install")
	cmd.Dir = cwd
	cmd.Stderr = os.Stderr // Only pipe errors directly, standard output hidden to keep console clean

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s install failed in %s: %w", manager, cwd, err)
	}
	return nil
}

// RunPnpmInstall is kept for backwards compatibility. Prefer RunInstall.
func RunPnpmInstall(cwd string) error {
	return RunInstall(cwd, "pnpm")
}
