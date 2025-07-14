package elorm

import (
	"fmt"
)

type FieldValueInt struct {
	fieldValueBase
	v   int64
	old int64
}

func (T *FieldValueInt) Set(newValue int64) {
	T.isDirty = T.isDirty || newValue != T.v
	T.v = newValue
}

func (T *FieldValueInt) Get() int64 {
	return T.v
}

func (T *FieldValueInt) GetOld() int64 {
	return T.old
}

func (T *FieldValueInt) resetOld() {
	T.old = T.v
}

func (T *FieldValueInt) SqlStringValue(v ...any) (string, error) {
	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(int64)
		if !ok {
			return "", fmt.Errorf("FieldValueInt.SqlStringValue: expected int64 value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("FieldValueInt.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	return fmt.Sprintf("%d", v2), nil
}

func (T *FieldValueInt) AsString() string {
	return fmt.Sprintf("%d", T.v)
}

func (T *FieldValueInt) Scan(v any) error {
	if v == nil {
		return fmt.Errorf("fieldValueInt.Scan: nil value for field %s", T.def.Name)
	}
	asInt, ok := v.(int64)
	if !ok {
		return fmt.Errorf("fieldValueInt.Scan: expected int64 for field %s, got %T", T.def.Name, v)
	}
	T.v = asInt
	T.isDirty = false
	T.old = T.v
	return nil
}
