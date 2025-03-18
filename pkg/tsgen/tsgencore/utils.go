package tsgencore

import (
	"reflect"
	"strings"
	"time"
)

func isBasicType(t reflect.Type) bool {
	if t == nil {
		return false
	}

	if t == reflect.TypeOf(time.Time{}) || t == reflect.TypeOf(time.Duration(0)) {
		return true
	}

	switch t.Kind() {
	case reflect.Interface,
		reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64,
		reflect.String:
		return true
	default:
		return false
	}
}

func isUnexported(field reflect.StructField) bool {
	return field.PkgPath != ""
}

func isOptionalField(field reflect.StructField) bool {
	if field.Type.Kind() == reflect.Ptr {
		return true
	}
	tag := field.Tag.Get("json")
	if tag != "" {
		parts := strings.Split(tag, ",")
		for _, part := range parts[1:] {
			if part == "omitempty" || part == "omitzero" {
				return true
			}
		}
	}
	return false
}

func getJSONFieldName(field reflect.StructField) string {
	tag := field.Tag.Get("json")
	if tag == "" {
		return field.Name
	}
	parts := strings.Split(tag, ",")
	if parts[0] == "-" {
		return ""
	}
	if parts[0] != "" {
		return parts[0]
	}
	return field.Name
}

func shouldOmitField(field reflect.StructField) bool {
	tag := field.Tag.Get("json")
	return tag == "-" || strings.HasPrefix(tag, "-,")
}

func getCustomTypeScriptType(field reflect.StructField) string {
	return field.Tag.Get("ts_type")
}

func buildObj(fields []string) string {
	if len(fields) == 0 {
		return "Record<never, never>"
	}
	var sb strings.Builder
	sb.WriteString("{\n")
	for _, field := range fields {
		sb.WriteString("\t")
		sb.WriteString(field)
		sb.WriteString(";\n")
	}
	sb.WriteString("}")
	return sb.String()
}
