package tsgen

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

	// Test case without AdHocTypes
	opts := Opts{
		ArbitraryPropertyNameToSortBy: "type",
		OutPath:                       filepath.Join(tempDir, testFileName),
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{
						Name:  "type",
						Value: "query",
					},
				},
				PhantomTypes: []PhantomType{
					{
						PropertyName: "phantomInputType",
						TypeInstance: struct{ Name string }{"TestName"},
						TSTypeName:   "testQueryInput",
					},
					{
						PropertyName: "phantomOutputType",
						TypeInstance: struct{ Result string }{"TestResult"},
						TSTypeName:   "testQueryOutput",
					},
				},
			},
			{
				ArbitraryProperties: []ArbitraryProperty{
					{
						Name:  "type",
						Value: "mutation",
					},
				},
				PhantomTypes: []PhantomType{
					{
						PropertyName: "phantomInputType",
						TypeInstance: struct{ ID int }{1},
						TSTypeName:   "testMutationInput",
					},
					{
						PropertyName: "phantomOutputType",
						TypeInstance: struct{ Success bool }{true},
						TSTypeName:   "testMutationOutput",
					},
				},
			},
		},
	}

	err := GenerateTSToFile(opts)
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

	contentStrMinimized := whiteSpaceToSingleSpace(contentStr)

	var expectedStrs = []string{mainIntroComment, mainTypes, items}

	for _, expectedStr := range expectedStrs {
		if !strings.Contains(contentStrMinimized, whiteSpaceToSingleSpace(expectedStr)) {
			t.Errorf("Expected string not found in generated TypeScript content: %q", expectedStr)
		}
	}

	// Check for the presence of TypeScript types
	if !strings.Contains(contentStr, "export type TestQueryInput = _T") {
		t.Error("Expected TypeScript type for TestQueryInput not found")
	}

	if !strings.Contains(contentStr, "export type TestQueryOutput = _T") {
		t.Error("Expected TypeScript type for TestQueryOutput not found")
	}

	if !strings.Contains(contentStr, "export type TestMutationInput = _T") {
		t.Error("Expected TypeScript type for TestMutationInput not found")
	}

	if !strings.Contains(contentStr, "export type TestMutationOutput = _T") {
		t.Error("Expected TypeScript type for TestMutationOutput not found")
	}

	// Check if AdHocTypes are correctly handled when not provided
	if strings.Contains(contentStr, "export type TestAdHocType = _T") {
		t.Error("TypeScript type for TestAdHocType found, but AdHocTypes were not provided")
	}

	// Clean up before testing with AdHocTypes
	cleanUpTestFiles(t, tempDir)

	// Test case with AdHocTypes
	opts.AdHocTypes = []AdHocType{
		{
			Struct:     struct{ Data string }{"TestData"},
			TSTypeName: "TestAdHocType",
		},
	}

	err = GenerateTSToFile(opts)
	if err != nil {
		t.Fatalf("GenerateTypeScript failed: %s", err)
	}

	if _, err := os.Stat(opts.OutPath); os.IsNotExist(err) {
		t.Fatalf("Expected TypeScript file not found: %s", opts.OutPath)
	}

	content, err = os.ReadFile(opts.OutPath)
	if err != nil {
		t.Fatalf("Failed to read generated TypeScript file: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript file is empty")
	}

	contentStr = string(content)

	contentStrMinimized = whiteSpaceToSingleSpace(contentStr)

	expectedStrs = append(expectedStrs, adHocTypes)

	for _, expectedStr := range expectedStrs {
		if !strings.Contains(contentStrMinimized, whiteSpaceToSingleSpace(expectedStr)) {
			t.Errorf("Expected string not found in generated TypeScript content: %q", expectedStr)
		}
	}

	// Check for the presence of TypeScript types again
	if !strings.Contains(contentStr, "export type TestQueryInput = _T") {
		t.Error("Expected TypeScript types for TestQueryInput not found")
	}

	if !strings.Contains(contentStr, "export type TestQueryOutput = _T") {
		t.Error("Expected TypeScript types for TestQueryOutput not found")
	}

	if !strings.Contains(contentStr, "export type TestMutationInput = _T") {
		t.Error("Expected TypeScript types for TestMutationInput not found")
	}

	if !strings.Contains(contentStr, "export type TestMutationOutput = _T") {
		t.Error("Expected TypeScript types for TestMutationOutput not found")
	}

	// Now check for the presence of AdHocTypes
	if !strings.Contains(contentStr, "export type TestAdHocType = _T") {
		t.Error("Expected TypeScript types for TestAdHocType not found")
	}

	cleanUpTestFiles(t, tempDir)
}

func TestGenerateTypeScriptNoItems(t *testing.T) {
	tempDir := t.TempDir()

	opts := Opts{
		OutPath: filepath.Join(tempDir, testFileName),
		Items:   []Item{},
	}

	err := GenerateTSToFile(opts)
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
		ExtraTSCode: "export const extraCode = 'extra';",
	}

	err := GenerateTSToFile(opts)
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

const mainTypes = "export type TestMutationInput = _T"

const items = ` = [
	{
		phantomInputType: null as unknown as TestMutationInput,
		phantomOutputType: null as unknown as TestMutationOutput,
		type: "mutation",
	},
	{
		phantomInputType: null as unknown as TestQueryInput,
		phantomOutputType: null as unknown as TestQueryOutput,
		type: "query",
	},
] as const;`

const adHocTypes = "export type TestAdHocType = _T"

func whiteSpaceToSingleSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
