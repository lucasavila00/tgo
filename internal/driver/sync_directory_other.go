//go:build !unix

package driver

// syncDirectory has no portable operation on this platform.
func syncDirectory(string) error {
	return nil
}
