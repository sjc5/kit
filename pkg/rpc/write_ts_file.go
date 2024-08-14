package rpc

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sjc5/kit/pkg/fsutil"
)

func writeTSFile(outDest, content string) error {
	err := fsutil.EnsureDir(outDest)
	if err != nil {
		return errors.New("failed to ensure out dest dir: " + err.Error())
	}

	path := filepath.Join(outDest, "api-types.ts")
	cleaned := cleanContent(content)
	err = os.WriteFile(path, []byte(cleaned), os.ModePerm)
	if err != nil {
		return errors.New("failed to write ts file: " + err.Error())
	}

	return nil
}

// cleanContent replace all instances of four spaces with a tab
// and replaces the empty object with Record<string, never>
func cleanContent(content string) string {
	cleaned := strings.ReplaceAll(content, "    ", "\t")
	cleaned = strings.ReplaceAll(cleaned, "{\n\n}", "Record<string, never>")
	return cleaned
}
