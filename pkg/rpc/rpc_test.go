package rpc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Helper function to clean up files created during tests
func cleanUpTestFiles(t *testing.T, path string) {
	err := os.RemoveAll(path)
	if err != nil {
		t.Fatalf("failed to clean up test files: %s", err)
	}
}

const testFileName = "api-types.ts"

func TestGenerateTypeScript(t *testing.T) {
	tempDir := t.TempDir()

	opts := Opts{
		OutPath: filepath.Join(tempDir, testFileName),
		RouteDefs: []RouteDef{
			{
				Key:        "testQuery",
				ActionType: ActionTypeQuery,
				Input:      struct{ Name string }{"TestName"},
				Output:     struct{ Result string }{"TestResult"},
			},
			{
				Key:        "testMutation",
				ActionType: ActionTypeMutation,
				Input:      struct{ ID int }{1},
				Output:     struct{ Success bool }{true},
			},
		},
		AdHocTypes: []AdHocType{
			{
				Struct:     struct{ Data string }{"TestData"},
				TSTypeName: "TestAdHocType",
			},
		},
	}

	err := GenerateTypeScript(opts)
	if err != nil {
		t.Fatalf("GenerateTypeScript failed: %s", err)
	}

	if _, err := os.Stat(opts.OutPath); os.IsNotExist(err) {
		t.Fatalf("Expected TypeScript file not found: %s", opts.OutPath)
	}

	content, err := os.ReadFile(opts.OutPath)
	if err != nil {
		t.Fatalf("Failed to read generated TypeScript file: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript file is empty")
	}

	// Check that the output contains specific strings
	contentStr := string(content)

	contestStrMinimized := whiteSpaceToSingleSpace(contentStr)

	for _, expectedStr := range expectedStrs {
		if !strings.Contains(contestStrMinimized, whiteSpaceToSingleSpace(expectedStr)) {
			t.Errorf(
				"Expected string not found in generated TypeScript content: %s",
				whiteSpaceToSingleSpace(expectedStr),
			)
		}
	}

	// Check for the presence of TypeScript interfaces
	if !strings.Contains(contentStr, "export type TestQueryInput = {") {
		t.Error("Expected TypeScript interface for TestQueryInput not found")
	}

	if !strings.Contains(contentStr, "export type TestQueryOutput = {") {
		t.Error("Expected TypeScript interface for TestQueryOutput not found")
	}

	if !strings.Contains(contentStr, "export type TestMutationInput = {") {
		t.Error("Expected TypeScript interface for TestMutationInput not found")
	}

	if !strings.Contains(contentStr, "export type TestMutationOutput = {") {
		t.Error("Expected TypeScript interface for TestMutationOutput not found")
	}

	if !strings.Contains(contentStr, "export type TestAdHocType = {") {
		t.Error("Expected TypeScript interface for TestAdHocType not found")
	}

	cleanUpTestFiles(t, tempDir)
}

func TestGenerateTypeScriptNoRoutes(t *testing.T) {
	tempDir := t.TempDir()

	opts := Opts{
		OutPath:   filepath.Join(tempDir, testFileName),
		RouteDefs: []RouteDef{},
	}

	err := GenerateTypeScript(opts)
	if err != nil {
		t.Fatalf("GenerateTypeScript failed: %s", err)
	}

	if _, err := os.Stat(opts.OutPath); os.IsNotExist(err) {
		t.Fatalf("Expected TypeScript file not found: %s", opts.OutPath)
	}

	content, err := os.ReadFile(opts.OutPath)
	if err != nil {
		t.Fatalf("Failed to read generated TypeScript file: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript file is empty")
	}

	cleanUpTestFiles(t, tempDir)
}

func TestExtraTS(t *testing.T) {
	tempDir := t.TempDir()

	opts := Opts{
		OutPath:     filepath.Join(tempDir, testFileName),
		RouteDefs:   []RouteDef{},
		ExtraTSCode: "export const extraCode = 'extra';",
	}

	err := GenerateTypeScript(opts)
	if err != nil {
		t.Fatalf("GenerateTypeScript failed: %s", err)
	}

	if _, err := os.Stat(opts.OutPath); os.IsNotExist(err) {
		t.Fatalf("Expected TypeScript file not found: %s", opts.OutPath)
	}

	content, err := os.ReadFile(opts.OutPath)
	if err != nil {
		t.Fatalf("Failed to read generated TypeScript file: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript file is empty")
	}

	contentStr := string(content)

	fmt.Println(contentStr)

	if !strings.Contains(contentStr, "export const extraCode = 'extra';") {
		t.Error("Expected extra TypeScript code not found")
	}

	cleanUpTestFiles(t, tempDir)
}

const mainTypes = `export type TestMutationInput = {
	ID: number;
}
export type TestMutationOutput = {
	Success: boolean;
}
export type TestQueryInput = {
	Name: string;
}
export type TestQueryOutput = {
	Result: string;
}`

const routes = `const routes = [
	{
		actionType: "mutation",
		key: "testMutation",
		phantomInputType: null as unknown as TestMutationInput,
		phantomOutputType: null as unknown as TestMutationOutput,
	},
	{
		actionType: "query",
		key: "testQuery",
		phantomInputType: null as unknown as TestQueryInput,
		phantomOutputType: null as unknown as TestQueryOutput,
	},
] as const;`

const adHocTypes = `export type TestAdHocType = {
	Data: string;
}`

var expectedStrs = []string{mainTypes, routes, adHocTypes, extraTSCode}

func whiteSpaceToSingleSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
