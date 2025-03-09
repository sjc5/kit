package tsgen

import (
	"regexp"
	"strings"
	"unicode"
)

var multipleUnderscoresRegex = regexp.MustCompile(`_+`)

// convertToPascalCase converts a string to PascalCase (but with
// an underscore prepended if the input starts with a digit).
func convertToPascalCase(input string) string {
	// Check if the input starts with a digit
	startsWithDigit := len(input) > 0 && unicode.IsDigit(rune(input[0]))

	var builder strings.Builder
	capitalize := true
	var lastChar rune

	for _, r := range input {
		if isIllegalCharacter(r) {
			if lastChar != '_' {
				builder.WriteRune('_')
			}
			capitalize = true
		} else {
			if capitalize {
				builder.WriteRune(unicode.ToUpper(r))
				capitalize = false
			} else {
				builder.WriteRune(r)
			}
		}
		lastChar = r
	}

	result := builder.String()

	// Remove leading underscores
	result = strings.TrimLeft(result, "_")

	// Replace multiple underscores with a single underscore
	result = multipleUnderscoresRegex.ReplaceAllString(result, "_")

	// Capitalize after each underscore and remove the underscore
	parts := strings.Split(result, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = string(unicode.ToUpper(rune(part[0]))) + part[1:]
		}
	}
	result = strings.Join(parts, "")

	// If the original input started with a digit, prepend an underscore to the final result
	if startsWithDigit {
		result = "_" + result
	}

	return result
}

// isIllegalCharacter checks if a character is illegal for TypeScript variable names
func isIllegalCharacter(r rune) bool {
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
}
