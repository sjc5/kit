package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func GetCallerDir() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Dir(file)
}
