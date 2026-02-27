package cleaner

import (
	"os"
)

// DeleteNodeModules safely deletes the entire node_modules directory
func DeleteNodeModules(dirPath string) error {
	return os.RemoveAll(dirPath)
}
