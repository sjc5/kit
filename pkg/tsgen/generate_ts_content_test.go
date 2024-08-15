package tsgen

import "testing"

func TestMakeTSType(t *testing.T) {
	prereqsMap := make(map[string]int)
	seenTypes := make(map[trimmedType][]cleanName)

	name := "TestStruct"
	inputStruct := struct{ Field string }{"Value"}

	target, prereqs, err := makeTSType(makeTSTypeInput{
		typeInstance:   inputStruct,
		prereqsMap:     &prereqsMap,
		seenTypes:      &seenTypes,
		name:           name,
		nameIsOverride: false,
	})
	if err != nil {
		t.Fatalf("makeTSType failed: %s", err)
	}

	if len(prereqs) == 0 || target == "" {
		t.Fatal("Expected non-empty TypeScript string and target")
	}

	if target != name {
		t.Errorf("Expected target to be 'TestStruct', got %q", target)
	}

	// Ensure that duplicate types with different names don't cause issues
	name2 := "TestStruct2"

	target2, prereqs2, err := makeTSType(makeTSTypeInput{
		typeInstance:   inputStruct,
		prereqsMap:     &prereqsMap,
		seenTypes:      &seenTypes,
		name:           name2,
		nameIsOverride: true,
	})
	if err != nil {
		t.Fatalf("makeTSType failed: %s", err)
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
