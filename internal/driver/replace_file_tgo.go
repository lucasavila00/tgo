package driver

import "os"

// replaceFile uses the platform rename contract for one staged file.
func replaceFile(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}
