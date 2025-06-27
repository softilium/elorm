package elorm

import (
	"fmt"
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
			return "", fmt.Errorf("fieldValueString.SqlStringValue: expected string value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("fieldValueString.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	return fmt.Sprintf("'%s'", v2), nil
}

func (T *FieldValueString) Set(newValue any) error {
	stringValue, ok := newValue.(string)
	if !ok {
		return fmt.Errorf("fieldValueString.Set: expected string value for field %s, got %T", T.def.Name, newValue)
	}
	T.isDirty = T.isDirty || stringValue != T.v
	T.v = stringValue
	return nil
}

func (T *FieldValueString) Get() (any, error) {
	return T.v, nil
}

func (T *FieldValueString) GetOld() (any, error) {
	return T.old, nil
}

func (T *FieldValueString) resetOld() {
	T.old = T.v
}

func (T *FieldValueString) AsString() string {
	return T.v
}

func (T *FieldValueString) Scan(v any) error {
	if v == nil {
		return fmt.Errorf("fieldValueString.Scan: nil value for field %s", T.def.Name)
	}
	switch vtyped := v.(type) {
	case string:
		T.v = vtyped

	case []uint8:
		T.v = string(vtyped)
	default:
		return fmt.Errorf("fieldValueString.Scan: expected string or []uint8 for field %s, got %T", T.def.Name, v)
	}

	T.isDirty = false
	T.old = T.v
	return nil
}
