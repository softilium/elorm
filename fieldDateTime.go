package elorm

import (
	"fmt"
	"time"
)

// This field type stores date + time value, without timezone.
type FieldValueDateTime struct {
	fieldValueBase
	v   time.Time
	old time.Time
}

func (T *FieldValueDateTime) Set(newValue time.Time) {
	T.isDirty = T.isDirty || newValue.Compare(T.v) != 0
	T.v = newValue
}

func (T *FieldValueDateTime) Get() time.Time {
	return T.v
}

func (T *FieldValueDateTime) GetOld() time.Time {
	return T.old
}

func (T *FieldValueDateTime) resetOld() {
	T.old = T.v
}

func (T *FieldValueDateTime) SqlStringValue(v ...any) (string, error) {
	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(time.Time)
		if !ok {
			return "", fmt.Errorf("fieldValueDateTime.SqlStringValue: expected time.Time value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("fieldValueDateTime.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}

	if v2.IsZero() {
		return "NULL", nil
	}

	return fmt.Sprintf("'%s'", v2.Format(time.DateTime)), nil
}

func (T *FieldValueDateTime) AsString() string {
	return T.v.Format(time.RFC3339)
}

func (T *FieldValueDateTime) Scan(v any) error {
	if v == nil {
		T.v = time.Time{}
		T.old = T.v
		return nil
	}
	switch vtyped := v.(type) {
	case time.Time:
		T.v = vtyped
	case []uint8:
		parsedTime, err := time.Parse(time.DateTime, string(vtyped))
		if err != nil {
			return fmt.Errorf("fieldValueDateTime.Scan: failed to parse time from []uint8 for field %s: %v", T.def.Name, err)
		}
		T.v = parsedTime
	default:
		return fmt.Errorf("fieldValueDateTime.Scan: expected time.Time or []uint8 for field %s, got %T", T.def.Name, v)
	}
	T.isDirty = false
	T.old = T.v
	return nil
}
