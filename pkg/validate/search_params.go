package validate

import (
	"fmt"
	"reflect"
	"strconv"
)

func parseURLValues(values map[string][]string, dst any) error {
	dstValue := reflect.ValueOf(dst)
	if dstValue.Kind() != reflect.Ptr || dstValue.IsNil() {
		return fmt.Errorf("destination must be a non-nil pointer")
	}

	dstElem := dstValue.Elem()
	if dstElem.Kind() != reflect.Struct {
		return fmt.Errorf("destination must be a pointer to a struct")
	}

	dstType := dstElem.Type()
	for i := 0; i < dstElem.NumField(); i++ {
		field := dstType.Field(i)
		fieldValue := dstElem.Field(i)

		fmt.Println("Field", field.Name, field.Type, fieldValue.Kind(), fieldValue.CanSet())
		if !fieldValue.CanSet() {
			continue
		}

		tag := field.Tag.Get("json")
		if tag == "" {
			tag = field.Name
		}

		if value, ok := values[tag]; ok {
			err := setField(fieldValue, value)
			if err != nil {
				return fmt.Errorf("error setting field %s: %w", field.Name, err)
			}
		}
	}

	return nil
}

func setField(field reflect.Value, values []string) error {
	if len(values) == 0 {
		return nil
	}

	// fmt.Println("Setting field", field.Type(), values, field.CanSet())

	switch field.Kind() {
	case reflect.Slice:
		return setSliceField(field, values)
	case reflect.String, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64, reflect.Bool:
		return setSingleValueField(field, values[0])
	default:
		return fmt.Errorf("unsupported field type %s", field.Type())
	}
}

func setSliceField(field reflect.Value, values []string) error {
	slice := reflect.MakeSlice(field.Type(), len(values), len(values))

	for i, value := range values {
		err := setSingleValueField(slice.Index(i), value)
		if err != nil {
			return err
		}
	}

	field.Set(slice)
	return nil
}

func setSingleValueField(field reflect.Value, value string) error {
	if !field.CanSet() {
		return fmt.Errorf("field is not settable")
	}
	switch field.Kind() {
	case reflect.String:
		fmt.Println("Setting string field", value, reflect.TypeOf(value), field.CanSet())
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetInt(intValue)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintValue, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return err
		}
		field.SetUint(uintValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		field.SetFloat(floatValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(boolValue)
	default:
		return fmt.Errorf("unsupported field type %s", field.Type())
	}
	return nil
}
