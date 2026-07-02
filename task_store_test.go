package main

import (
	"reflect"
	"testing"
	"time"
)

func TestCloneTaskCopiesAllFields(t *testing.T) {
	var original Task
	originalValue := reflect.ValueOf(&original).Elem()
	originalType := originalValue.Type()

	for i := 0; i < originalValue.NumField(); i++ {
		field := originalValue.Field(i)
		if !field.CanSet() {
			t.Fatalf("Task.%s cannot be set by the test", originalType.Field(i).Name)
		}

		value, ok := testValueForTaskField(field.Type())
		if !ok {
			t.Fatalf("missing test value for Task.%s with type %s", originalType.Field(i).Name, field.Type())
		}
		field.Set(value)
	}

	cloned := cloneTask(&original)
	clonedValue := reflect.ValueOf(cloned)

	for i := 0; i < originalValue.NumField(); i++ {
		fieldName := originalType.Field(i).Name
		originalField := originalValue.Field(i).Interface()
		clonedField := clonedValue.Field(i).Interface()

		if !reflect.DeepEqual(clonedField, originalField) {
			t.Fatalf("cloneTask did not copy Task.%s: got %#v, want %#v", fieldName, clonedField, originalField)
		}
	}
}

func testValueForTaskField(fieldType reflect.Type) (reflect.Value, bool) {
	switch {
	case fieldType == reflect.TypeOf(time.Time{}):
		return reflect.ValueOf(time.Date(2026, 7, 2, 3, 4, 5, 6, time.UTC)), true
	case fieldType == reflect.TypeOf((*time.Time)(nil)):
		value := time.Date(2026, 7, 2, 4, 5, 6, 7, time.UTC)
		return reflect.ValueOf(&value), true
	case fieldType == reflect.TypeOf((*string)(nil)):
		value := "task error"
		return reflect.ValueOf(&value), true
	case fieldType.Kind() == reflect.Interface:
		return reflect.ValueOf(map[string]any{"ok": true}), true
	case fieldType.Kind() == reflect.String:
		value := reflect.New(fieldType).Elem()
		value.SetString("task-field-value")
		return value, true
	default:
		return reflect.Value{}, false
	}
}
