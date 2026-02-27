package installer

import (
	"fmt"
	"os"
	"os/exec"
)

// RunPnpmInstall executes pnpm install within a given directory
func RunPnpmInstall(cwd string) error {
	cmd := exec.Command("pnpm", "install")
	cmd.Dir = cwd
	cmd.Stderr = os.Stderr // Only pipe errors directly, standard output hidden to keep console clean

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pnpm install failed in %s: %w", cwd, err)
	}
	return nil
}
