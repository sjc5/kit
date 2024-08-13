package rpc

import "github.com/tkrajina/typescriptify-golang-structs/typescriptify"

// newConverter creates a new TypeScriptify converter
func newConverter() *typescriptify.TypeScriptify {
	converter := typescriptify.New()
	converter.CreateInterface = true
	return converter
}
