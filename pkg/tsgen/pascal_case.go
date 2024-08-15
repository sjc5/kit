package tsgen

import (
	"regexp"
	"strings"
	"unicode"
)

// convertToPascalCase converts a string to PascalCase
func convertToPascalCase(input string) string {
	var builder strings.Builder
	capitalize := true
	var lastChar rune

	for _, r := range input {
		if isIllegalCharacter(r) {
			if lastChar != '_' {
				builder.WriteRune('_')
			}
			capitalize = true
		} else if unicode.IsDigit(r) && builder.Len() == 0 {
			continue // Skip leading digits
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
	re := regexp.MustCompile(`_+`)
	result = re.ReplaceAllString(result, "_")

	// Capitalize after each underscore and remove the underscore
	parts := strings.Split(result, "_")
	for i, part := range parts {
		if part != "" {
			parts[i] = string(unicode.ToUpper(rune(part[0]))) + part[1:]
		}
	}
	result = strings.Join(parts, "")

	return result
}

// isIllegalCharacter checks if a character is illegal for TypeScript variable names
func isIllegalCharacter(r rune) bool {
	return !(unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_')
}
