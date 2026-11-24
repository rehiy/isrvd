//go:build !unix

package helper

import "os"

// PreserveFileOwner is a no-op on platforms without Unix ownership.
func PreserveFileOwner(_ *os.File, _ os.FileInfo) error {
	return nil
}
