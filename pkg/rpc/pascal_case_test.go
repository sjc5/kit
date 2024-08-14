package rpc

import "testing"

func TestConvertToPascalCase(t *testing.T) {
	tests := [][]string{
		{"simpleName", "SimpleName"},
		{"with spaces", "WithSpaces"},
		{"special@chars", "SpecialChars"},
		{"123startsWithNumber", "StartsWithNumber"},
		{"___multiple__underscores___", "MultipleUnderscores"},
		{"number444Inside", "Number444Inside"},
		{"number_at_end_444", "NumberAtEnd444"},
		{"number_at_end_wo_us444", "NumberAtEndWoUs444"},
		{"ALLCAPS", "ALLCAPS"}, // All caps should remain the same
		{"snake_case", "SnakeCase"},
		{"camelCase", "CamelCase"},
		{"MixedCAse_with_underscores", "MixedCAseWithUnderscores"}, // No way to fix the CAse
		{"with.dots.and-dashes", "WithDotsAndDashes"},
		{"   leading spaces", "LeadingSpaces"},
		{"trailing spaces   ", "TrailingSpaces"},
		{"__leading_underscores", "LeadingUnderscores"},
		{"trailing_underscores__", "TrailingUnderscores"},
		{"1234", ""}, // All numbers should result in an empty string
		{"a1B2c3", "A1B2c3"},
		{"", ""}, // Empty string should remain empty
	}

	for _, test := range tests {
		t.Run(test[0], func(t *testing.T) {
			output := convertToPascalCase(test[0])
			if output != test[1] {
				t.Errorf("convertToPascalCase(%q) = %q; want %q", test[0], output, test[1])
			}
		})
	}
}
