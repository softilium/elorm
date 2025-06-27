package elorm

import (
	"fmt"
	"strconv"
)

type FieldValueNumeric struct {
	fieldValueBase
	v   float64
	old float64
}

func (T *FieldValueNumeric) Get() (any, error) {
	return T.v, nil
}

func (T *FieldValueNumeric) GetOld() (any, error) {
	return T.old, nil
}

func (T *FieldValueNumeric) resetOld() {
	T.old = T.v
}

func (T *FieldValueNumeric) Set(newValue any) error {
	floatValue, ok := newValue.(float64)
	if !ok {
		return fmt.Errorf("fieldValueNumeric.Set: expected float64 value for field %s, got %T", T.def.Name, newValue)
	}
	T.isDirty = T.isDirty || floatValue != T.v
	T.v = floatValue
	return nil
}

func (T *FieldValueNumeric) mask() string {
	return "%" + fmt.Sprintf("%d", T.def.Precision) + "." + fmt.Sprintf("%d", T.def.Scale) + "f"
}

func (T *FieldValueNumeric) SqlStringValue(v ...any) (string, error) {
	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(float64)
		if !ok {
			return "", fmt.Errorf("fieldValueNumeric.SqlStringValue: expected float64 value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("fieldValueNumeric.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	return fmt.Sprintf(T.mask(), v2), nil
}

func (T *FieldValueNumeric) AsString() string {
	return fmt.Sprintf(T.mask(), T.v)
}

func (T *FieldValueNumeric) Scan(v any) error {
	if v == nil {
		return fmt.Errorf("fieldValueNumeric.Scan: nil value for field %s", T.def.Name)
	}
	switch vtyped := v.(type) {
	case float64:
		T.v = vtyped
	case string:
		vt, err := strconv.ParseFloat(vtyped, 64)
		if err != nil {
			return fmt.Errorf("fieldValueNumeric.Scan: cannot parse string '%s' as float64 for field %s", vtyped, T.def.Name)
		}
		T.v = vt
	case []uint8:
		vts2 := string(vtyped)
		vt, err := strconv.ParseFloat(vts2, 64)
		if err != nil {
			return fmt.Errorf("fieldValueNumeric.Scan: cannot parse []uint8 '%s' as float64 for field %s", vts2, T.def.Name)
		}
		T.v = vt
	case int64:
		T.v = float64(vtyped)
	default:
		return fmt.Errorf("fieldValueNumeric.Scan: unsupported type %T for field %s", v, T.def.Name)
	}
	T.isDirty = false
	T.old = T.v
	return nil
}
