package elorm

import (
	"testing"
)

func mockFieldValueString() *FieldValueString {
	return &FieldValueString{
		fieldValueBase: fieldValueBase{
			def: &FieldDef{
				Name: "testField",
				EntityDef: &EntityDef{
					Factory: &Factory{},
				},
			},
		},
	}
}

func TestFieldValueString_SqlStringValue(t *testing.T) {
	field := mockFieldValueString()
	type line struct {
		values []any
		expStr string
		expErr bool
	}
	field.Set("testValue")
	testCases := []line{
		{values: []any{}, expStr: "'testValue'", expErr: false},
		{values: []any{"newTestValue"}, expStr: "'newTestValue'", expErr: false},
		{values: []any{"'123"}, expStr: "'''123'", expErr: false},
		{values: []any{123}, expStr: "", expErr: true},
	}
	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			expStr, err := field.SqlStringValue(tc.values...)
			if (err != nil) != tc.expErr {
				t.Errorf("expected error: %v, got: %v", tc.expErr, err != nil)
			}
			if expStr != tc.expStr {
				t.Errorf("expected: %s, got: %s", tc.expStr, expStr)
			}
		})
	}
}

func TestFieldValueString_SetAndGet(t *testing.T) {
	field := mockFieldValueString()

	field.Set("initialValue")
	if val := field.Get(); val != "initialValue" {
		t.Errorf("expected 'initialValue', got '%s'", val)
	}

	field.Set("newValue")
	if val := field.Get(); val != "newValue" {
		t.Errorf("expected 'newValue', got '%s'", val)
	}
}
func TestFieldValueString_Old(t *testing.T) {
	field := mockFieldValueString()

	field.Set("initialValue")
	if val := field.Old(); val != "" {
		t.Errorf("expected '', got '%s'", val)
	}

	field.resetOld()

	field.Set("newValue")
	if val := field.Old(); val != "initialValue" {
		t.Errorf("expected '', got '%s'", val)
	}
}

func TestFieldValueString_AsString(t *testing.T) {
	field := mockFieldValueString()

	field.Set("testValue")
	if val := field.AsString(); val != "testValue" {
		t.Errorf("expected 'testValue', got '%s'", val)
	}

	field.Set("anotherValue")
	if val := field.AsString(); val != "anotherValue" {
		t.Errorf("expected 'anotherValue', got '%s'", val)
	}
}

func TestFieldValueString_Scan(t *testing.T) {
	field := mockFieldValueString()

	type scanTest struct {
		input    any
		expected string
		err      bool
	}

	tests := []scanTest{
		{input: "test", expected: "test", err: false},
		{input: []uint8("test"), expected: "test", err: false},
		{input: nil, expected: "", err: false},
		{input: 123, expected: "", err: true},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			err := field.Scan(tt.input)
			if (err != nil) != tt.err {
				t.Errorf("expected error: %v, got: %v", tt.err, err)
			}
			if field.Get() != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, field.Get())
			}
			if field.Old() != tt.expected {
				t.Errorf("expected old value to match new value '%s', got '%s'", tt.expected, field.Old())
			}
		})
	}
}
