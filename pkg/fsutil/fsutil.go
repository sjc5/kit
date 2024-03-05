package fsutil

import "os"

func EnsureDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}
