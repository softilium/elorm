package elorm

import (
	"fmt"
	"strings"
)

type FieldValueString struct {
	fieldValueBase
	v   string
	old string
}

func (T *FieldValueString) SqlStringValue(v ...any) (string, error) {
	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(string)
		if !ok {
			return "", fmt.Errorf("FieldValueString.SqlStringValue: expected string value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("FieldValueString.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	v2 = strings.ReplaceAll(v2, "'", "''") // Escape single quotes for SQL
	return fmt.Sprintf("'%s'", v2), nil
}

func (T *FieldValueString) Set(newValue string) {
	T.isDirty = T.isDirty || newValue != T.v
	T.v = newValue
}

func (T *FieldValueString) Get() string {
	return T.v
}

func (T *FieldValueString) GetOld() string {
	return T.old
}

func (T *FieldValueString) resetOld() {
	T.old = T.v
}

func (T *FieldValueString) AsString() string {
	return T.v
}

func (T *FieldValueString) Scan(v any) error {
	if v == nil {
		T.v = ""
		T.isDirty = false
		T.old = T.v
		return nil
	}
	switch vtyped := v.(type) {
	case string:
		T.v = vtyped
	case []uint8:
		T.v = string(vtyped)
	default:
		return fmt.Errorf("FieldValueString.Scan: type assertion failed: expected string or []uint8 for field %s, got %T", T.def.Name, v)
	}

	T.isDirty = false
	T.old = T.v
	return nil
}
