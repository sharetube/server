package omitnilpointers

import (
	"reflect"
)

func OmitNilPointers(fields map[string]any) map[string]any {
	omitted := make(map[string]any)
	for key, value := range fields {
		if value == nil {
			continue
		}

		// Use reflection to check if it's a pointer
		v := reflect.ValueOf(value)
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				continue
			}
			// Dereference the pointer and store the actual value
			omitted[key] = v.Elem().Interface()
		} else {
			// Non-pointer value, store as is
			omitted[key] = value
		}
	}

	return omitted
}
