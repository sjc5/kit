package rpc

import (
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

func TestGenerateTypeScript(t *testing.T) {
	tempDir := t.TempDir()

	opts := Opts{
		OutDest: tempDir,
		RouteDefs: []RouteDef{
			{
				Key:    "testQuery",
				Type:   TypeQuery,
				Input:  struct{ Name string }{"TestName"},
				Output: struct{ Result string }{"TestResult"},
			},
			{
				Key:    "testMutation",
				Type:   TypeMutation,
				Input:  struct{ ID int }{1},
				Output: struct{ Success bool }{true},
			},
		},
		AdHocTypes: []AdHocType{
			{
				Struct: struct{ Data string }{"TestData"},
				Name:   "TestAdHocType",
			},
		},
	}

	err := GenerateTypeScript(opts)
	if err != nil {
		t.Fatalf("GenerateTypeScript failed: %s", err)
	}

	expectedFile := filepath.Join(tempDir, "api-types.ts")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("Expected TypeScript file not found: %s", expectedFile)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read generated TypeScript file: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript file is empty")
	}

	// Check that the output contains specific strings
	contentStr := string(content)

	// Check for the presence of TypeScript interfaces
	if !strings.Contains(contentStr, "export interface testQuery_input {") {
		t.Error("Expected TypeScript interface for testQuery_input not found")
	}

	if !strings.Contains(contentStr, "export interface testQuery_output {") {
		t.Error("Expected TypeScript interface for testQuery_output not found")
	}

	if !strings.Contains(contentStr, "export interface testMutation_input {") {
		t.Error("Expected TypeScript interface for testMutation_input not found")
	}

	if !strings.Contains(contentStr, "export interface testMutation_output {") {
		t.Error("Expected TypeScript interface for testMutation_output not found")
	}

	if !strings.Contains(contentStr, "export interface TestAdHocType {") {
		t.Error("Expected TypeScript interface for TestAdHocType not found")
	}

	cleanUpTestFiles(t, tempDir)
}

func TestGenerateTypeScriptNoRoutes(t *testing.T) {
	tempDir := t.TempDir()

	opts := Opts{
		OutDest:   tempDir,
		RouteDefs: []RouteDef{},
	}

	err := GenerateTypeScript(opts)
	if err != nil {
		t.Fatalf("GenerateTypeScript failed: %s", err)
	}

	expectedFile := filepath.Join(tempDir, "api-types.ts")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Fatalf("Expected TypeScript file not found: %s", expectedFile)
	}

	content, err := os.ReadFile(expectedFile)
	if err != nil {
		t.Fatalf("Failed to read generated TypeScript file: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript file is empty")
	}

	cleanUpTestFiles(t, tempDir)
}

func TestConvertToTSVariableName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"simpleName", "simpleName"},
		{"with spaces", "with_spaces"},
		{"special@chars", "special_chars"},
		{"123startsWithNumber", "startsWithNumber"},              // leading numbers are removed
		{"___multiple__underscores___", "multiple_underscores_"}, // leading underscores are removed, internal are limited to one
	}

	for _, test := range tests {
		output := convertToTSVariableName(test.input)
		if output != test.expected {
			t.Errorf("convertToTSVariableName(%q) = %q; want %q", test.input, output, test.expected)
		}
	}
}

func TestMakeTSStr(t *testing.T) {
	prereqsMap := make(map[string]int)
	seenTypes := make(map[trimmedType][]cleanName)

	target := ""
	name := "TestStruct"
	inputStruct := struct{ Field string }{"Value"}

	prereqs, err := makeTSStr(makeTSStrInput{
		target:         &target,
		t:              inputStruct,
		prereqsMap:     &prereqsMap,
		name:           name,
		seenTypes:      &seenTypes,
		nameIsOverride: false,
	})
	if err != nil {
		t.Fatalf("makeTSStr failed: %s", err)
	}

	if len(prereqs) == 0 || target == "" {
		t.Fatal("Expected non-empty TypeScript string and target")
	}

	if target != name {
		t.Errorf("Expected target to be 'TestStruct', got %q", target)
	}

	// Ensure that duplicate types with different names don't cause issues
	target2 := ""
	name2 := "TestStruct2"

	prereqs2, err := makeTSStr(makeTSStrInput{
		target:         &target2,
		t:              inputStruct,
		prereqsMap:     &prereqsMap,
		name:           name2,
		seenTypes:      &seenTypes,
		nameIsOverride: true,
	})
	if err != nil {
		t.Fatalf("makeTSStr failed: %s", err)
	}

	if len(prereqs2) == 0 || target2 == "" {
		t.Fatal("Expected non-empty TypeScript string and target")
	}

	if target2 == target {
		t.Errorf("Expected different targets for different names, got %q and %q", target, target2)
	}
}

func TestGetIsAnonName(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"", true},
		{" ", true},
		{"_", true},
		{"Name", false},
	}

	for _, test := range tests {
		result := getIsAnonName(test.input)
		if result != test.expected {
			t.Errorf("getIsAnonName(%q) = %v; want %v", test.input, result, test.expected)
		}
	}
}
