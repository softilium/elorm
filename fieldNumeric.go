package elorm

import (
	"fmt"
	"strconv"
	"strings"
)

type FieldValueNumeric struct {
	fieldValueBase
	v   float64
	old float64
}

func (T *FieldValueNumeric) Set(newValue float64) {
	T.isDirty = T.isDirty || newValue != T.v
	T.v = newValue
}

func (T *FieldValueNumeric) Get() float64 {
	return T.v
}

func (T *FieldValueNumeric) GetOld() float64 {
	return T.old
}

func (T *FieldValueNumeric) resetOld() {
	T.old = T.v
}

func (T *FieldValueNumeric) mask() string {
	return fmt.Sprintf("%%%d.%df", T.def.Precision, T.def.Scale)
}

func (T *FieldValueNumeric) SqlStringValue(v ...any) (string, error) {
	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(float64)
		if !ok {
			return "", fmt.Errorf("FieldValueNumeric.SqlStringValue: type assertion failed: expected float64 value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("FieldValueNumeric.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	return strings.TrimSpace(fmt.Sprintf(T.mask(), v2)), nil
}

func (T *FieldValueNumeric) AsString() string {
	return strings.TrimSpace(fmt.Sprintf(T.mask(), T.v))
}

func (T *FieldValueNumeric) Scan(v any) error {
	if v == nil {
		return fmt.Errorf("FieldValueNumeric.Scan: nil value for field %s", T.def.Name)
	}
	switch vtyped := v.(type) {
	case float64:
		T.v = vtyped
	case string:
		vt, err := strconv.ParseFloat(vtyped, 64)
		if err != nil {
			return fmt.Errorf("FieldValueNumeric.Scan: cannot parse string '%s' as float64 for field %s", vtyped, T.def.Name)
		}
		T.v = vt
	case []uint8:
		vts2 := string(vtyped)
		vt, err := strconv.ParseFloat(vts2, 64)
		if err != nil {
			return fmt.Errorf("FieldValueNumeric.Scan: cannot parse []uint8 '%s' as float64 for field %s", vts2, T.def.Name)
		}
		T.v = vt
	case int64:
		T.v = float64(vtyped)
	default:
		return fmt.Errorf("FieldValueNumeric.Scan: unsupported type %T for field %s", v, T.def.Name)
	}
	T.isDirty = false
	T.old = T.v
	return nil
}
