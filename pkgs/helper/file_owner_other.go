//go:build !unix

package helper

import "os"

func preserveFileOwner(_ *os.File, _ os.FileInfo) error {
	return nil
}
