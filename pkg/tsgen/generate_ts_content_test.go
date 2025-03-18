package tsgen

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestGenerateTypeScript(t *testing.T) {
	// Test case without AdHocTypes
	opts := Opts{
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{
					"type": "query",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomInputType": {
						TypeInstance: struct{ Name string }{"TestName"},
						TSTypeName:   "TestQueryInput",
					},
					"phantomOutputType": {
						TypeInstance: struct{ Result string }{"TestResult"},
						TSTypeName:   "TestQueryOutput",
					},
				},
			},
			{
				ArbitraryProperties: map[string]any{
					"type": "mutation",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomInputType": {
						TypeInstance: struct{ ID int }{1},
						TSTypeName:   "TestMutationInput",
					},
					"phantomOutputType": {
						TypeInstance: struct{ Success bool }{true},
						TSTypeName:   "TestMutationOutput",
					},
				},
			},
		},
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript content is empty")
	}

	// Check that the output contains specific strings
	contentStrMinimized := whiteSpaceToSingleSpace(content)

	var expectedStrs = []string{mainTypes, items}

	for _, expectedStr := range expectedStrs {
		if !strings.Contains(contentStrMinimized, whiteSpaceToSingleSpace(expectedStr)) {
			t.Errorf("Expected string not found in generated TypeScript content: %q", expectedStr)
		}
	}

	// Check for the presence of TypeScript types
	if !strings.Contains(content, "export type TestQueryInput = {") {
		t.Error("Expected TypeScript type for TestQueryInput not found")
	}

	if !strings.Contains(content, "export type TestQueryOutput = {") {
		t.Error("Expected TypeScript type for TestQueryOutput not found")
	}

	if !strings.Contains(content, "export type TestMutationInput = {") {
		t.Error("Expected TypeScript type for TestMutationInput not found")
	}

	if !strings.Contains(content, "export type TestMutationOutput = {") {
		t.Error("Expected TypeScript type for TestMutationOutput not found")
	}

	// Check if AdHocTypes are correctly handled when not provided
	if strings.Contains(content, "export type TestAdHocType = ") {
		t.Error("TypeScript type for TestAdHocType found, but AdHocTypes were not provided")
	}

	// Test case with AdHocTypes
	opts.AdHocTypes = []*AdHocType{
		{
			TypeInstance: struct{ Data string }{"TestData"},
			TSTypeName:   "TestAdHocType",
		},
	}

	content, err = GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript content is empty")
	}

	contentStrMinimized = whiteSpaceToSingleSpace(content)

	expectedStrs = append(expectedStrs, adHocTypes)

	for _, expectedStr := range expectedStrs {
		if !strings.Contains(contentStrMinimized, whiteSpaceToSingleSpace(expectedStr)) {
			t.Errorf("Expected string not found in generated TypeScript content: %q", expectedStr)
		}
	}

	// Check for the presence of TypeScript types again
	if !strings.Contains(content, "export type TestQueryInput = {") {
		t.Error("Expected TypeScript types for TestQueryInput not found")
	}

	if !strings.Contains(content, "export type TestQueryOutput = {") {
		t.Error("Expected TypeScript types for TestQueryOutput not found")
	}

	if !strings.Contains(content, "export type TestMutationInput = {") {
		t.Error("Expected TypeScript types for TestMutationInput not found")
	}

	if !strings.Contains(content, "export type TestMutationOutput = {") {
		t.Error("Expected TypeScript types for TestMutationOutput not found")
	}

	// Now check for the presence of AdHocTypes
	if !strings.Contains(content, "export type TestAdHocType = {") {
		t.Error("Expected TypeScript types for TestAdHocType not found")
	}
}

func TestGenerateTypeScriptNoItems(t *testing.T) {
	opts := Opts{
		Collection: []CollectionItem{},
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript content is empty")
	}
}

func TestExtraTS(t *testing.T) {
	opts := Opts{
		ExtraTSCode: "export const extraCode = 'extra';",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %s", err)
	}

	if len(content) == 0 {
		t.Fatal("Generated TypeScript content is empty")
	}

	if !strings.Contains(content, "export const extraCode = 'extra';") {
		t.Error("Expected extra TypeScript code not found")
	}
}

const mainTypes = "export type TestMutationInput = {"

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

const adHocTypes = "export type TestAdHocType = {"

func whiteSpaceToSingleSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// TestGenerateTSContent_SimpleTypes tests generation of simple types
func TestGenerateTSContent_SimpleTypes(t *testing.T) {
	type SimpleType struct {
		Field string
	}

	opts := Opts{
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/simple",
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &SimpleType{}, TSTypeName: "SimpleOutput"},
				},
			},
		},
		CollectionVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify output
	assertContains(t, content, "export type SimpleOutput = {\n\tField: string;\n}")
	assertContains(t, content, "phantomOutputType: null as unknown as SimpleOutput")
}

// TestGenerateTSContent_DuplicateTypes tests handling of duplicate types
func TestGenerateTSContent_DuplicateTypes(t *testing.T) {
	type Inner struct {
		Name string
	}

	type Outer struct {
		X Inner
	}

	opts := Opts{
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/first",
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &Outer{}, TSTypeName: "FirstOutput"},
				},
			},
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/second",
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &Outer{}, TSTypeName: "SecondOutput"},
				},
			},
		},
		CollectionVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify output - both types should reference the same core type
	assertContains(t, content, "export type FirstOutput = {\n\tX: Inner;\n\n}")
	assertContains(t, content, "export type SecondOutput = { X: Inner; }")
	assertContains(t, content, "type Inner = { Name: string; }")

	// There should only be one core type definition
	occurrences := strings.Count(content, "type Inner =")
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

	type Type3ForAdHoc struct {
		Field3 int
	}

	opts := Opts{
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/path",
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &Type1{}, TSTypeName: "SameNameOutput"},
				},
			},
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/path/$",
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &Type2{}, TSTypeName: "SameNameOutput"},
				},
			},
		},
		AdHocTypes: []*AdHocType{
			{TSTypeName: "SameNameOutput", TypeInstance: &Type3ForAdHoc{}},
		},
		CollectionVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify that the second type got a numeric suffix
	assertContains(t, content, "export type SameNameOutput = {")
	assertContains(t, content, "export type SameNameOutput_2 = {")
	assertContains(t, content, "export type SameNameOutput_3 = {")

	// Verify both type definitions exist
	assertContains(t, content, "Field1: string;")
	assertContains(t, content, "Field2: number;")
	assertContains(t, content, "Field3: number;")
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
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/complex",
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &ParentType{}, TSTypeName: "ComplexOutput"},
				},
			},
		},
		CollectionVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify nested type was handled correctly
	assertContains(t, content, "export type ComplexOutput = {")
	assertContains(t, content, "Child: NestedType;")
}

// TestGenerateTSContent_AdHocTypes tests handling of ad-hoc types
func TestGenerateTSContent_AdHocTypes(t *testing.T) {
	type SomeType struct {
		Field string
	}

	opts := Opts{
		Collection: []CollectionItem{},
		AdHocTypes: []*AdHocType{
			{TSTypeName: "CustomType", TypeInstance: &SomeType{}},
		},
		CollectionVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify ad-hoc type was included
	assertContains(t, content, "export type CustomType = { Field: string; };")
}

// TestGenerateTSContent_TypesWithTimeField tests handling of time.Time fields
func TestGenerateTSContent_TypesWithTimeField(t *testing.T) {
	type TypeWithTime struct {
		Created time.Time
	}

	opts := Opts{
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/with-time",
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &TypeWithTime{}, TSTypeName: "TimeOutput"},
				},
			},
		},
		CollectionVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify time.Time was handled correctly (implementation-dependent)
	assertContains(t, content, "export type TimeOutput = {")
	assertContains(t, content, "Created: ")
}

// TestGenerateTSContent_EmptyNameHandling tests handling of empty or anonymous names
func TestGenerateTSContent_AnonAndUnnamedShouldBeSkipped(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{{TypeInstance: struct{ Field string }{}, TSTypeName: ""}},
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify empty name was handled with default
	assertNotContains(t, content, "export type")
}

// Helper functions for assertions
func assertContains(t *testing.T, outer, inner string) {
	cleanOuter := normalizeWhiteSpace(outer)
	cleanInner := normalizeWhiteSpace(inner)

	t.Helper()
	if !strings.Contains(cleanOuter, cleanInner) {
		t.Errorf("Expected content to contain '%s' but it didn't.\nContent: %s", cleanInner, cleanOuter)
	}
}

func assertNotContains(t *testing.T, outer, inner string) {
	cleanOuter := normalizeWhiteSpace(outer)
	cleanInner := normalizeWhiteSpace(inner)

	t.Helper()
	if strings.Contains(cleanOuter, cleanInner) {
		t.Errorf("Expected content to NOT contain '%s' but it did.\nContent: %s", cleanInner, cleanOuter)
	}
}

// TestGenerateTSContent_WithCustomSorting tests the custom sorting feature
func TestGenerateTSContent_WithCustomSorting(t *testing.T) {
	type SimpleType struct {
		Field string
	}

	opts := Opts{
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/c",
					"order":     3,
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &SimpleType{}, TSTypeName: "COutput"},
				},
			},
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/a",
					"order":     1,
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &SimpleType{}, TSTypeName: "AOutput"},
				},
			},
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/b",
					"order":     2,
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: &SimpleType{}, TSTypeName: "BOutput"},
				},
			},
		},
		CollectionVarName: "routes",
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
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{
					"pattern":   "/null-type",
					"routeType": "loader",
				},
				PhantomTypes: map[string]AdHocType{
					"phantomOutputType": {TypeInstance: nil, TSTypeName: "ShouldBeUndefined"},
				},
			},
		},
		CollectionVarName: "routes",
	}

	content, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("GenerateTSContent failed: %v", err)
	}

	// Verify nil type instance handled correctly
	assertContains(t, content, "phantomOutputType: undefined")
	assertNotContains(t, content, "export type ShouldBeUndefined")
}

// Test structs with identical field structures but different types
type Person struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Animal struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

// Test different primitive types
type AllPrimitives struct {
	BoolField    bool    `json:"boolField"`
	IntField     int     `json:"intField"`
	Int8Field    int8    `json:"int8Field"`
	Int16Field   int16   `json:"int16Field"`
	Int32Field   int32   `json:"int32Field"`
	Int64Field   int64   `json:"int64Field"`
	UintField    uint    `json:"uintField"`
	Uint8Field   uint8   `json:"uint8Field"`
	Uint16Field  uint16  `json:"uint16Field"`
	Uint32Field  uint32  `json:"uint32Field"`
	Uint64Field  uint64  `json:"uint64Field"`
	Float32Field float32 `json:"float32Field"`
	Float64Field float64 `json:"float64Field"`
	StringField  string  `json:"stringField"`
	ByteField    byte    `json:"byteField"`
	RuneField    rune    `json:"runeField"`
}

// Test pointer types
type WithPointers struct {
	StringPtr *string         `json:"stringPtr"`
	IntPtr    *int            `json:"intPtr"`
	BoolPtr   *bool           `json:"boolPtr"`
	StructPtr *Person         `json:"structPtr"`
	SlicePtr  *[]int          `json:"slicePtr"`
	MapPtr    *map[string]int `json:"mapPtr"`
}

// Test nested structs
type Address struct {
	Street  string `json:"street"`
	City    string `json:"city"`
	Country string `json:"country"`
}

type Customer struct {
	ID      int     `json:"id"`
	Name    string  `json:"name"`
	Address Address `json:"address"`
}

// Test embedded structs
type Named struct {
	Name string `json:"name"`
}

type Aged struct {
	Age int `json:"age"`
}

type Employee struct {
	Named
	Aged
	Department string `json:"department"`
}

// Test slices and arrays
type WithCollections struct {
	IntSlice    []int     `json:"intSlice"`
	StringSlice []string  `json:"stringSlice"`
	StructSlice []Person  `json:"structSlice"`
	IntArray    [3]int    `json:"intArray"`
	StringArray [2]string `json:"stringArray"`
	StructArray [2]Person `json:"structArray"`
}

// Test maps
type WithMaps struct {
	StringToInt    map[string]int    `json:"stringToInt"`
	StringToString map[string]string `json:"stringToString"`
	IntToString    map[int]string    `json:"intToString"`
	StringToPerson map[string]Person `json:"stringToPerson"`
}

// Test empty structs
type Empty struct{}

// Test time.Time fields
type WithTime struct {
	Created time.Time  `json:"created"`
	Updated *time.Time `json:"updated"`
}

// Test interfaces
type WithInterfaces struct {
	EmptyInterface    interface{}  `json:"emptyInterface"`
	StringerInterface fmt.Stringer `json:"stringerInterface"`
}

// Test optional fields with json:"omitempty"
type WithOptionalFields struct {
	Required string `json:"required"`
	Optional string `json:"optional,omitempty"`
}

// Test custom TS type annotations
type WithCustomTSType struct {
	ID     int    `json:"id" ts_type:"string"` // Override int with string
	Custom string `json:"custom" ts_type:"CustomType"`
}

// TestAllPrimitiveTypes ensures all primitive types are handled correctly
func TestAllPrimitiveTypes(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: AllPrimitives{}, TSTypeName: "Primitives"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check for boolean type
	if !strings.Contains(output, "boolField: boolean") {
		t.Error("Boolean field not properly typed")
	}

	// Check for number types
	numericFields := []string{"intField", "int8Field", "int16Field", "int32Field",
		"int64Field", "uintField", "uint8Field", "uint16Field", "uint32Field",
		"uint64Field", "float32Field", "float64Field", "byteField", "runeField"}

	for _, field := range numericFields {
		if !strings.Contains(output, field+": number") {
			t.Errorf("Numeric field %s not properly typed", field)
		}
	}

	// Check string type
	if !strings.Contains(output, "stringField: string") {
		t.Error("String field not properly typed")
	}
}

// TestPointerTypes ensures pointer types are handled correctly
func TestPointerTypes(t *testing.T) {
	stringVal := "test"
	intVal := 42
	boolVal := true
	sliceVal := []int{1, 2, 3}
	mapVal := map[string]int{"one": 1}

	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: WithPointers{
				StringPtr: &stringVal,
				IntPtr:    &intVal,
				BoolPtr:   &boolVal,
				StructPtr: &Person{Name: "Test", Age: 30},
				SlicePtr:  &sliceVal,
				MapPtr:    &mapVal,
			}, TSTypeName: "PointerTest"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check optional marking for pointer types
	if !strings.Contains(output, "stringPtr?: string") {
		t.Error("String pointer not marked as optional")
	}
	if !strings.Contains(output, "intPtr?: number") {
		t.Error("Int pointer not marked as optional")
	}
	if !strings.Contains(output, "boolPtr?: boolean") {
		t.Error("Bool pointer not marked as optional")
	}

	// Check that struct pointers are handled correctly
	if !strings.Contains(output, "structPtr?:") {
		t.Error("Struct pointer not marked as optional")
	}

	// Check slice and map pointers
	if !strings.Contains(output, "slicePtr?: Array<number>") {
		t.Error("Slice pointer not handled correctly")
	}
	if !strings.Contains(output, "mapPtr?: Record<string, number>") {
		t.Error("Map pointer not handled correctly")
	}
}

// TestNestedStructs ensures nested struct types are handled correctly
func TestNestedStructs(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: Customer{}, TSTypeName: "Customer"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check that Customer type is defined
	if !strings.Contains(output, "export type Customer = ") {
		t.Error("Customer type not found in output")
	}

	// Check that Address type is defined or inlined
	addr := `export type Address = { street: string; city: string; country: string; };`

	if !strings.Contains(normalizeWhiteSpace(output), normalizeWhiteSpace(addr)) {
		t.Log(output)
		t.Log(addr)
		t.Error("Nested Address struct not properly formatted")
	}
}

// TestEmbeddedStructs ensures embedded struct fields are handled correctly
func TestEmbeddedStructs(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: Employee{}, TSTypeName: "Employee"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check that Employee type is defined
	if !strings.Contains(output, "export type Employee = ") {
		t.Error("Employee type not found in output")
	}

	// Check for embedded struct fields
	if !strings.Contains(output, "name: string") {
		t.Error("Embedded Name field not found in Employee")
	}
	if !strings.Contains(output, "age: number") {
		t.Error("Embedded Age field not found in Employee")
	}
	if !strings.Contains(output, "department: string") {
		t.Error("Department field not found in Employee")
	}
}

// TestCollectionTypes ensures slices and arrays are handled correctly
func TestCollectionTypes(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: WithCollections{}, TSTypeName: "CollectionTest"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check slice types
	if !strings.Contains(output, "intSlice: Array<number>") {
		t.Error("Int slice not properly typed")
	}
	if !strings.Contains(output, "stringSlice: Array<string>") {
		t.Error("String slice not properly typed")
	}

	// Check array types (should be the same as slices in TypeScript)
	if !strings.Contains(output, "intArray: Array<number>") {
		t.Error("Int array not properly typed")
	}
	if !strings.Contains(output, "stringArray: Array<string>") {
		t.Error("String array not properly typed")
	}

	// Check struct collections
	if !strings.Contains(output, "structSlice: Array<") {
		t.Error("Struct slice not properly typed")
	}
	if !strings.Contains(output, "structArray: Array<") {
		t.Error("Struct array not properly typed")
	}
}

// TestMapTypes ensures maps are handled correctly
func TestMapTypes(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: WithMaps{}, TSTypeName: "MapTest"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check map types
	if !strings.Contains(output, "stringToInt: Record<string, number>") {
		t.Error("String to int map not properly typed")
	}
	if !strings.Contains(output, "stringToString: Record<string, string>") {
		t.Error("String to string map not properly typed")
	}
	if !strings.Contains(output, "intToString: Record<number, string>") {
		t.Error("Int to string map not properly typed")
	}
	if !strings.Contains(output, "stringToPerson: Record<string,") {
		t.Error("String to struct map not properly typed")
	}
}

// TestTimeType ensures time.Time is handled correctly
func TestTimeType(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: WithTime{}, TSTypeName: "TimeTest"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check that time.Time is mapped to string
	if !strings.Contains(output, "created: string") {
		t.Log("Output:", output)
		t.Error("time.Time not mapped to string")
	}

	// Check that *time.Time is optional and mapped to string
	if !strings.Contains(output, "updated?: string") {
		t.Error("*time.Time not mapped to optional string")
	}
}

// TestInterfaceTypes ensures interface types are handled correctly
func TestInterfaceTypes(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: WithInterfaces{}, TSTypeName: "InterfaceTest"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check that empty interface is mapped to any
	if !strings.Contains(output, "emptyInterface: unknown") {
		t.Log(output)
		t.Error("Empty interface not mapped to 'unknown'")
	}

	// Check that non-empty interface is mapped to unknown
	if !strings.Contains(output, "stringerInterface: unknown") {
		t.Error("Non-empty interface not mapped to 'unknown'")
	}
}

// TestOptionalFields ensures fields with omitempty are marked as optional
func TestOptionalFields(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: WithOptionalFields{}, TSTypeName: "OptionalTest"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check required and optional fields
	if !strings.Contains(output, "required: string") {
		t.Error("Required field not properly typed")
	}
	if !strings.Contains(output, "optional?: string") {
		t.Error("Optional field not marked with ?")
	}
}

// TestCustomTSTypes ensures custom ts_type tags are honored
func TestCustomTSTypes(t *testing.T) {
	opts := Opts{
		AdHocTypes: []*AdHocType{
			{TypeInstance: WithCustomTSType{}, TSTypeName: "CustomTypeTest"},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check custom type overrides
	if !strings.Contains(output, "id: string") {
		t.Error("ts_type override for ID not honored")
	}
	if !strings.Contains(output, "custom: CustomType") {
		t.Error("ts_type for custom field not honored")
	}
}

// TestTypeGeneration tests that the generator produces valid TypeScript
func TestTypeGeneration(t *testing.T) {
	// Create a complex set of types to test many features at once
	opts := Opts{
		CollectionVarName: "complexTest",
		AdHocTypes: []*AdHocType{
			{TypeInstance: Person{}, TSTypeName: "Person"},
			{TypeInstance: Animal{}, TSTypeName: "Animal"},
			{TypeInstance: Customer{}, TSTypeName: "Customer"},
			{TypeInstance: Employee{}, TSTypeName: "Employee"},
			{TypeInstance: WithCollections{}, TSTypeName: "Collections"},
			{TypeInstance: WithMaps{}, TSTypeName: "Maps"},
			{TypeInstance: WithPointers{}, TSTypeName: "Pointers"},
			{TypeInstance: WithTime{}, TSTypeName: "TimeFields"},
			{TypeInstance: WithInterfaces{}, TSTypeName: "Interfaces"},
			{TypeInstance: WithOptionalFields{}, TSTypeName: "Optionals"},
			{TypeInstance: Empty{}, TSTypeName: "Empty"},
		},
		Collection: []CollectionItem{
			{
				ArbitraryProperties: map[string]any{"strProp": "string value", "numProp": 42, "boolProp": true},
				PhantomTypes: map[string]AdHocType{
					"personRef": {TSTypeName: "PersonRef", TypeInstance: Person{Name: "Jane", Age: 25}},
					"animalRef": {TSTypeName: "AnimalRef", TypeInstance: Animal{Name: "Rex", Age: 3}},
				},
			},
		},
	}

	output, err := GenerateTSContent(opts)
	if err != nil {
		t.Fatalf("Failed to generate TypeScript: %v", err)
	}

	// Check the output has all expected exported types
	expectedTypes := []string{
		"Person", "Animal", "Customer", "Employee", "Collections",
		"Maps", "Pointers", "TimeFields", "Interfaces", "Optionals",
		"Empty", "PersonRef", "AnimalRef",
	}

	for _, typeName := range expectedTypes {
		if !strings.Contains(output, "export type "+typeName+" = ") {
			t.Errorf("Expected type %s not found in output", typeName)
		}
	}

	// Check that items array is generated
	if !strings.Contains(output, "const complexTest = [") {
		t.Error("Items array not generated")
	}

	// Check arbitrary properties in items array
	if !strings.Contains(output, "strProp: \"string value\"") {
		t.Error("String property not correctly added to items array")
	}
	if !strings.Contains(output, "numProp: 42") {
		t.Error("Number property not correctly added to items array")
	}
	if !strings.Contains(output, "boolProp: true") {
		t.Error("Boolean property not correctly added to items array")
	}

	// Check phantom types in items array
	if !strings.Contains(output, "personRef: null as unknown as PersonRef") {
		t.Error("Person phantom type not correctly added to items array")
	}
	if !strings.Contains(output, "animalRef: null as unknown as AnimalRef") {
		t.Error("Animal phantom type not correctly added to items array")
	}
}

func normalizeWhiteSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
