package tsgen

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sjc5/kit/pkg/fsutil"
)

type Opts struct {
	// Path, including filename, where the resulting TypeScript file will be written
	OutPath           string
	Items             []Item
	AdHocTypes        []AdHocType
	ExtraTSCode       string
	ItemsArrayVarName string // Defaults to "tsgenItems"
	ExportItemsArray  bool
}

// Item represents a TypeScript object type with arbitrary properties and phantom types.
// It will be added to a constant array in the generated TypeScript file with the name
// assigned to ItemsArrayVarName in Opts.
type Item struct {
	ArbitraryProperties []ArbitraryProperty
	PhantomTypes        []PhantomType
}

// Anything you'd like to add to a TypeScript type object,
// other than the phantom types. Value must be JSON-serializable.
type ArbitraryProperty struct {
	Name  string
	Value any
}

type PhantomType struct {
	PropertyName string
	TSTypeName   string
	TypeInstance any
}

type AdHocType struct {
	// Instance of the struct to generate TypeScript for
	Struct any

	// Name is required only if struct is anonymous, otherwise optional override
	TSTypeName string
}

// GenerateTSToFile generates a TypeScript file from the provided Opts.
func GenerateTSToFile(opts Opts) error {
	if opts.OutPath == "" {
		return errors.New("outpath is required")
	}

	tsContent, err := GenerateTSContent(opts)
	if err != nil {
		return err
	}

	err = fsutil.EnsureDir(filepath.Dir(opts.OutPath))
	if err != nil {
		return errors.New("failed to ensure out dest dir: " + err.Error())
	}

	err = os.WriteFile(opts.OutPath, []byte(tsContent), os.ModePerm)
	if err != nil {
		return errors.New("failed to write ts file: " + err.Error())
	}

	return nil
}
