//go:build !unix

package compiler

// syncDirectory has no portable operation on this platform.
func syncDirectory(string) error {
	return nil
}
