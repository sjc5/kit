package tsgen

import (
	"strings"
	"testing"
	"time"
)

/////////////////////////////////////////////////////////////////////
/////// OLD TESTS
/////////////////////////////////////////////////////////////////////

func TestMakeTSType(t *testing.T) {
	seenTypes := make(map[trimmedType][]cleanName)

	name := "TestStruct"
	inputStruct := struct{ Field string }{"Value"}

	target, err := makeTSType(makeTSTypeInput{
		typeInstance:    inputStruct,
		seenTypes:       &seenTypes,
		name:            name,
		nameIsOverride:  false,
		typeDefinitions: &map[string]string{},
		usedTypeNames:   &map[string]string{},
		typeAliases:     &map[string]string{},
	})
	if err != nil {
		t.Fatalf("makeTSType failed: %s", err)
	}

	if target == "" {
		t.Fatal("Expected non-empty TypeScript string and target")
	}

	if target != name {
		t.Errorf("Expected target to be 'TestStruct', got %q", target)
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

/////////////////////////////////////////////////////////////////////
/////// NEW TESTS
/////////////////////////////////////////////////////////////////////

// TestGenerateTSContent_SimpleTypes tests generation of simple types
func TestGenerateTSContent_SimpleTypes(t *testing.T) {
	type SimpleType struct {
		Field string
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/simple"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &SimpleType{}, TSTypeName: "SimpleOutput"},
				},
			},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify output
	assertContains(t, content, "export type SimpleOutput = T")
	assertContains(t, content, "type T1_ = {\n\tField: string;\n}")
	assertContains(t, content, "phantomOutputType: null as unknown as SimpleOutput")
}

// TestGenerateTSContent_DuplicateTypes tests handling of duplicate types
func TestGenerateTSContent_DuplicateTypes(t *testing.T) {
	type SharedType struct {
		ID int
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/first"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &SharedType{}, TSTypeName: "FirstOutput"},
				},
			},
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/second"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &SharedType{}, TSTypeName: "SecondOutput"},
				},
			},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify output - both types should reference the same core type
	assertContains(t, content, "export type FirstOutput = T1_")
	assertContains(t, content, "export type SecondOutput = T1_")
	assertContains(t, content, "type T1_ = {\n\tID: number;\n}")

	// There should only be one core type definition
	occurrences := strings.Count(content, "type T")
	if occurrences != 1 {
		t.Errorf("Expected 1 core type definition, found %d", occurrences)
	}
}

// TestGenerateTSContent_DifferentTypesWithSameName tests handling of different types with same name
func TestGenerateTSContent_DifferentTypesWithSameName(t *testing.T) {
	type Type1 struct {
		Field1 string
	}

	type Type2 struct {
		Field2 int
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/path"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &Type1{}, TSTypeName: "SameNameOutput"},
				},
			},
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/path/$"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &Type2{}, TSTypeName: "SameNameOutput"},
				},
			},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify that the second type got a numeric suffix
	assertContains(t, content, "export type SameNameOutput = T")
	assertContains(t, content, "export type SameNameOutput1 = T")

	// Verify both type definitions exist
	assertContains(t, content, "Field1: string;")
	assertContains(t, content, "Field2: number;")
}

// TestGenerateTSContent_DollarSignInName tests handling of $ in route patterns
func TestGenerateTSContent_DollarSignInName(t *testing.T) {
	type SimpleType struct {
		Name string
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/user/$id"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &SimpleType{}, TSTypeName: "/user/$idOutput"},
				},
			},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify the $ was replaced with "Dollar"
	assertContains(t, content, "export type UserIdOutput = T")
	assertNotContains(t, content, "export type User$idOutput")
}

// TestGenerateTSContent_ComplexNestedTypes tests handling of complex nested types
func TestGenerateTSContent_ComplexNestedTypes(t *testing.T) {
	type NestedType struct {
		Nested string
	}

	type ParentType struct {
		Name  string
		Child NestedType
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/complex"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &ParentType{}, TSTypeName: "ComplexOutput"},
				},
			},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify nested type was handled correctly
	assertContains(t, content, "export type ComplexOutput = T")
	assertContains(t, content, "Child: NestedType;")
}

// TestGenerateTSContent_AdHocTypes tests handling of ad-hoc types
func TestGenerateTSContent_AdHocTypes(t *testing.T) {
	type SomeType struct {
		Field string
	}

	opts := Opts{
		Items: []Item{},
		AdHocTypes: []AdHocType{
			{TSTypeName: "CustomType", Struct: &SomeType{}},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify ad-hoc type was included
	assertContains(t, content, "export type CustomType = T")
	assertContains(t, content, "type T1_ = {\n\tField: string;\n}")
}

// TestGenerateTSContent_TypesWithTimeField tests handling of time.Time fields
func TestGenerateTSContent_TypesWithTimeField(t *testing.T) {
	type TypeWithTime struct {
		Created time.Time
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/with-time"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &TypeWithTime{}, TSTypeName: "TimeOutput"},
				},
			},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify time.Time was handled correctly (implementation-dependent)
	assertContains(t, content, "export type TimeOutput = T")
	assertContains(t, content, "Created: ")
}

// TestGenerateTSContent_OrderOfTypesAndRoutes tests the order of declarations in the output
func TestGenerateTSContent_OrderOfTypesAndRoutes(t *testing.T) {
	type SimpleType struct {
		Field string
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/order-test"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &SimpleType{}, TSTypeName: "OrderTestOutput"},
				},
			},
		},
		ItemsArrayVarName: "routes",
		ExtraTSCode:       "// Extra TS code\nexport type Helper = string;",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Check order: first export types, then routes array, then extra code, then core types
	firstExportTypePos := strings.Index(content, "export type OrderTestOutput")
	routesArrayPos := strings.Index(content, "const routes =")
	extraCodePos := strings.Index(content, "// Extra TS code")
	coreTypePos := strings.Index(content, "type T1_ =")

	if firstExportTypePos > routesArrayPos ||
		routesArrayPos > extraCodePos ||
		extraCodePos > coreTypePos {
		t.Errorf("Elements are in wrong order in generated content")
	}
}

// TestGenerateTSContent_EmptyNameHandling tests handling of empty or anonymous names
func TestGenerateTSContent_EmptyNameHandling(t *testing.T) {
	type AnonymousType struct {
		Field string
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/anon"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &AnonymousType{}, TSTypeName: ""},
				},
			},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify empty name was handled with default
	assertContains(t, content, "export type AnonType")
}

// Helper functions for assertions
func assertContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("Expected content to contain '%s' but it didn't.\nContent: %s", needle, haystack)
	}
}

func assertNotContains(t *testing.T, haystack, needle string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("Expected content to NOT contain '%s' but it did.\nContent: %s", needle, haystack)
	}
}

// TestGenerateTSContent_WithCustomSorting tests the custom sorting feature
func TestGenerateTSContent_WithCustomSorting(t *testing.T) {
	type SimpleType struct {
		Field string
	}

	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/c"},
					{Name: "order", Value: "3"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &SimpleType{}, TSTypeName: "COutput"},
				},
			},
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/a"},
					{Name: "order", Value: "1"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &SimpleType{}, TSTypeName: "AOutput"},
				},
			},
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/b"},
					{Name: "order", Value: "2"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: &SimpleType{}, TSTypeName: "BOutput"},
				},
			},
		},
		ItemsArrayVarName:             "routes",
		ArbitraryPropertyNameToSortBy: "order",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Find positions of patterns in the output
	aPos := strings.Index(content, `pattern: "/a"`)
	bPos := strings.Index(content, `pattern: "/b"`)
	cPos := strings.Index(content, `pattern: "/c"`)

	// Verify they're in the correct order
	if !(aPos < bPos && bPos < cPos) {
		t.Errorf("Items not sorted correctly by 'order' property")
	}
}

// TestGenerateTSContent_NullTypes tests handling of nil type instances
func TestGenerateTSContent_NullTypes(t *testing.T) {
	opts := Opts{
		Items: []Item{
			{
				ArbitraryProperties: []ArbitraryProperty{
					{Name: "pattern", Value: "/null-type"},
					{Name: "routeType", Value: "loader"},
				},
				PhantomTypes: []PhantomType{
					{PropertyName: "phantomOutputType", TypeInstance: nil, TSTypeName: "ShouldBeUndefined"},
				},
			},
		},
		ItemsArrayVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify nil type instance handled correctly
	assertContains(t, content, "phantomOutputType: undefined")
	assertNotContains(t, content, "export type ShouldBeUndefined")
}
