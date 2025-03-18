package tsgencore

import (
	"fmt"
	"reflect"
	"time"
)

type typeCollector struct {
	types             map[reflect.Type]*typeEntry
	rootType          reflect.Type
	rootRequestedName string
}

type typeEntry struct {
	originalGoType reflect.Type
	resolvedName   string
	usedAsEmbedded bool
	isReferenced   bool
	visited        bool
	coreType       string
	requestedName  string
}

func newTypeCollector() *typeCollector {
	return &typeCollector{types: make(map[reflect.Type]*typeEntry)}
}

func (c *typeCollector) getOrCreateEntry(t reflect.Type, userDefinedAlias ...string) *typeEntry {
	if entry, exists := c.types[t]; exists {
		if t == c.rootType && c.rootRequestedName != "" && entry.requestedName == "" {
			entry.requestedName = c.rootRequestedName
		}
		return entry
	}

	entry := &typeEntry{originalGoType: t}
	if t == c.rootType && c.rootRequestedName != "" {
		entry.requestedName = c.rootRequestedName
	} else if len(userDefinedAlias) > 0 && userDefinedAlias[0] != "" {
		entry.requestedName = userDefinedAlias[0]
	} else {
		var requestedName string
		if !isBasicType(t) {
			requestedName = t.Name()
		}
		if requestedName != "" {
			entry.requestedName = requestedName
		}
	}

	c.types[t] = entry
	return entry
}

func (c *typeCollector) collectType(t reflect.Type, userDefinedAlias ...string) {
	isRoot := (t == c.rootType)

	if t.Name() != "" || isRoot {
		entry := c.getOrCreateEntry(t, userDefinedAlias...)
		if entry.visited {
			return
		}
		entry.visited = true

		if isBasicType(t) && !isRoot {
			entry.coreType = c.getTypeScriptType(t)
			return
		}
	} else {
		if !isRoot && isBasicType(t) {
			return
		}
	}

	switch t.Kind() {
	case reflect.Struct:
		c.collectStructFields(t)

	case reflect.Ptr:
		if t.Name() != "" {
			c.getOrCreateEntry(t, userDefinedAlias...)
		}
		c.collectType(t.Elem())

	case reflect.Slice, reflect.Array:
		if t.Name() != "" {
			c.getOrCreateEntry(t, userDefinedAlias...)
		}
		c.collectType(t.Elem())

	case reflect.Map:
		if t.Name() != "" {
			c.getOrCreateEntry(t, userDefinedAlias...)
		}
		c.collectType(t.Key())
		c.collectType(t.Elem())
	}
}

func (c *typeCollector) collectStructFields(t reflect.Type) {
	for i := range t.NumField() {
		field := t.Field(i)
		if isUnexported(field) {
			continue
		}
		fieldType := field.Type

		if field.Anonymous {
			if fieldType.Kind() == reflect.Struct {
				embeddedEntry := c.getOrCreateEntry(fieldType)
				embeddedEntry.usedAsEmbedded = true
				c.collectType(fieldType)
				continue
			}

			if fieldType.Kind() == reflect.Ptr && fieldType.Elem().Kind() == reflect.Struct {
				embeddedType := fieldType.Elem()
				embeddedEntry := c.getOrCreateEntry(embeddedType)
				embeddedEntry.usedAsEmbedded = true
				embeddedEntry.isReferenced = true
				c.collectType(embeddedType)
				continue
			}
		}

		c.collectFieldType(fieldType)
	}
}

func (c *typeCollector) collectFieldType(t reflect.Type) {
	switch t.Kind() {
	case reflect.Struct:
		entry := c.getOrCreateEntry(t)
		entry.isReferenced = true
		c.collectType(t)

	case reflect.Ptr:
		if t.Name() != "" {
			entry := c.getOrCreateEntry(t)
			entry.isReferenced = true
		}
		if t.Elem().Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(t.Elem())
			entry.isReferenced = true
		}
		c.collectType(t.Elem())

	case reflect.Slice, reflect.Array:
		if t.Name() != "" {
			entry := c.getOrCreateEntry(t)
			entry.isReferenced = true
		}
		elemType := t.Elem()
		if elemType.Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(elemType)
			entry.isReferenced = true
		} else if elemType.Kind() == reflect.Ptr && elemType.Elem().Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(elemType.Elem())
			entry.isReferenced = true
		}
		c.collectType(elemType)

	case reflect.Map:
		if t.Name() != "" {
			entry := c.getOrCreateEntry(t)
			entry.isReferenced = true
		}

		// Map key
		keyType := t.Key()
		if keyType.Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(keyType)
			entry.isReferenced = true
		}
		c.collectType(keyType)

		// Map value
		valueType := t.Elem()
		if valueType.Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(valueType)
			entry.isReferenced = true
		} else if valueType.Kind() == reflect.Ptr && valueType.Elem().Kind() == reflect.Struct {
			entry := c.getOrCreateEntry(valueType.Elem())
			entry.isReferenced = true
		}
		c.collectType(valueType)
	}
}

func (c *typeCollector) buildDefinitions() (_results, IDStr) {
	if len(c.types) > 0 && c.rootType != nil {
		hasStructs := false
		for t := range c.types {
			if t == nil {
				continue
			}
			if t.Kind() == reflect.Struct {
				hasStructs = true
				break
			}
		}

		if !hasStructs {
			id := getIDFromReflectType(c.rootType, c.rootRequestedName)

			results := map[IDStr]*TypeInfo{id: {
				_id:          id,
				OriginalName: c.rootRequestedName,
				ResolvedName: c.types[c.rootType].resolvedName,
				ReflectType:  c.rootType,
				TSStr:        c.getTypeScriptType(c.rootType),
			}}

			return results, id
		}
	}

	for t, entry := range c.types {
		if entry.coreType == "" {
			if t.Kind() == reflect.Struct {
				fields := c.generateTypeFields(t)
				entry.coreType = buildObj(fields)
			} else {
				entry.coreType = c.getTypeScriptType(t)
			}
		}
	}

	reflectTypeToID := make(map[reflect.Type]IDStr)

	for t, entry := range c.types {
		if t.Kind() == reflect.Struct {
			if t != c.rootType && entry.usedAsEmbedded && !entry.isReferenced {
				continue
			}
		}

		requestedName := entry.requestedName
		if t == c.rootType && c.rootRequestedName != "" {
			requestedName = c.rootRequestedName
		}

		id := getIDFromReflectType(t, requestedName)
		reflectTypeToID[t] = id
	}

	finalTypes := make(map[IDStr]*TypeInfo)

	for t, id := range reflectTypeToID {
		requestedName := ""

		entry := c.types[t]

		if t == c.rootType {
			requestedName = c.rootRequestedName
		} else if entry.requestedName != "" {
			requestedName = entry.requestedName
		}

		finalTypes[id] = &TypeInfo{
			_id:          id,
			OriginalName: requestedName,
			ResolvedName: entry.resolvedName,
			ReflectType:  t,
			TSStr:        c.types[t].coreType,
		}
	}

	if c.rootType != nil {
		return finalTypes, reflectTypeToID[c.rootType]
	}

	panic("tsgencore error: something went wrong")
}

func (c *typeCollector) generateTypeFields(t reflect.Type) []string {
	if t.Kind() != reflect.Struct {
		return nil
	}

	var fields []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.PkgPath != "" {
			continue
		}
		if shouldOmitField(field) {
			continue
		}

		if field.Anonymous {
			if field.Type.Kind() == reflect.Struct {
				for j := 0; j < field.Type.NumField(); j++ {
					embField := field.Type.Field(j)
					if embField.PkgPath != "" || shouldOmitField(embField) {
						continue
					}

					fieldName := getJSONFieldName(embField)
					if fieldName == "" {
						continue
					}

					customType := getCustomTypeScriptType(embField)
					var fieldType string
					if customType != "" {
						fieldType = customType
					} else {
						fieldType = c.getTypeScriptType(embField.Type)
					}

					if isOptionalField(embField) {
						fields = append(fields, fmt.Sprintf("%s?: %s", fieldName, fieldType))
					} else {
						fields = append(fields, fmt.Sprintf("%s: %s", fieldName, fieldType))
					}
				}
				continue
			} else if field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct {
				ptrType := field.Type.Elem()
				structName := ptrType.Name()

				fieldName := getJSONFieldName(field)
				if fieldName == "" {
					fieldName = structName
				}

				elemType := c.getTypeScriptType(field.Type.Elem())
				fields = append(fields, fmt.Sprintf("%s?: %s", fieldName, elemType))
				continue
			}
		}

		fieldName := getJSONFieldName(field)
		if fieldName == "" {
			continue
		}

		customType := getCustomTypeScriptType(field)
		var fieldType string
		if customType != "" {
			fieldType = customType
		} else {
			if field.Type.Kind() == reflect.Ptr {
				fieldType = c.getTypeScriptType(field.Type.Elem())
			} else {
				fieldType = c.getTypeScriptType(field.Type)
			}
		}

		if isOptionalField(field) {
			fields = append(fields, fmt.Sprintf("%s?: %s", fieldName, fieldType))
		} else {
			fields = append(fields, fmt.Sprintf("%s: %s", fieldName, fieldType))
		}
	}

	return fields
}

func getBasicTSType(t reflect.Type) string {
	if t == nil {
		return "undefined"
	}

	switch t.Kind() {
	case reflect.Bool:
		return "boolean"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number"
	case reflect.String:
		return "string"
	default:
		return "unknown"
	}
}

func (c *typeCollector) getTypeScriptType(t reflect.Type) string {
	if t == nil {
		return "undefined"
	}

	var typeStr string

	switch t.Kind() {
	case reflect.Interface:
		typeStr = "unknown"

	case reflect.Bool:
		typeStr = "boolean"

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		typeStr = "number"

	case reflect.String:
		typeStr = "string"

	case reflect.Ptr:
		typeStr = c.getTypeScriptType(t.Elem())

	case reflect.Slice, reflect.Array:
		elemType := c.getTypeScriptType(t.Elem())
		typeStr = fmt.Sprintf("Array<%s>", elemType)

	case reflect.Map:
		keyType := c.getTypeScriptType(t.Key())
		valueType := c.getTypeScriptType(t.Elem())
		typeStr = fmt.Sprintf("Record<%s, %s>", keyType, valueType)

	case reflect.Struct:
		switch {
		case t == reflect.TypeOf(time.Time{}):
			typeStr = "string"
		case t == reflect.TypeOf(time.Duration(0)):
			typeStr = "number"
		case t.Name() != "" && c.types[t] != nil:
			entry := c.getOrCreateEntry(t)
			requestedName := entry.requestedName

			if t == c.rootType && c.rootRequestedName != "" {
				requestedName = c.rootRequestedName
			}

			// ID will be replaced later with the correct resolved name
			typeStr = getIDFromReflectType(t, requestedName)
		default:
			fields := c.generateTypeFields(t)
			typeStr = buildObj(fields)
		}

	default:
		typeStr = "unknown"
	}

	if IsMarkedNullable(t) {
		typeStr = fmt.Sprintf("%s | null", typeStr)
	}
	if IsMarkedOptional(t) {
		typeStr = fmt.Sprintf("%s | undefined", typeStr)
	}

	return typeStr
}
