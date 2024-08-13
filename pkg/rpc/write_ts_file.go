package rpc

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sjc5/kit/pkg/fsutil"
)

func writeTSFile(outDest, content string) error {
	err := fsutil.EnsureDir(outDest)
	if err != nil {
		return errors.New("failed to ensure out dest dir: " + err.Error())
	}

	err = os.WriteFile(filepath.Join(outDest, "api-types.ts"), []byte(content), os.ModePerm)
	if err != nil {
		return errors.New("failed to write ts file: " + err.Error())
	}

	return nil
}
