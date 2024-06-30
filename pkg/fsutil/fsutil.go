package fsutil

import (
	"encoding/gob"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, os.ModePerm)
}

func GetCallerDir() string {
	_, file, _, _ := runtime.Caller(1)
	return filepath.Dir(file)
}

// copyDir recursively copies a directory from src to dst.
func CopyDir(src, dst string) error {
	// Get properties of source dir
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	// Create the destination directory
	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	// Read the directory contents
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		fileInfo, err := entry.Info()
		if err != nil {
			return err
		}

		// If the entry is a directory, recurse
		if fileInfo.IsDir() {
			if err := CopyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			// If it's a file, copy it
			if err := CopyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// copyFile copies a single file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}
	return destFile.Sync()
}

func FromGobInto(file fs.File, dest any) error {
	dec := gob.NewDecoder(file)
	err := dec.Decode(dest)
	if err != nil {
		return fmt.Errorf("failed to decode bytes into dest: %w", err)
	}
	return nil
}
