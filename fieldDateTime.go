package elorm

import (
	"fmt"
	"time"
)

// This field type stores date + time value, without timezone.
type fieldValueDateTime struct {
	fieldValueBase
	v   time.Time
	old time.Time
}

func (T *fieldValueDateTime) Get() (any, error) {
	return T.v, nil
}

func (T *fieldValueDateTime) GetOld() (any, error) {
	return T.old, nil
}

func (T *fieldValueDateTime) resetOld() {
	T.old = T.v
}

func (T *fieldValueDateTime) Set(newValue any) error {
	timeValue, ok := newValue.(time.Time)
	if !ok {
		return fmt.Errorf("fieldValueDateTime.Set: expected time.Time value for field %s, got %T", T.def.Name, newValue)
	}
	T.isDirty = T.isDirty || timeValue.Compare(T.v) != 0
	T.v = timeValue
	return nil
}

func (T *fieldValueDateTime) SqlStringValue(v ...any) (string, error) {
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
	return fmt.Sprintf("'%s'", v2.Format(time.DateTime)), nil
}

func (T *fieldValueDateTime) AsString() string {
	return T.v.Format(time.RFC3339)
}

func (T *fieldValueDateTime) Scan(v any) error {
	if v == nil {
		return fmt.Errorf("fieldValueDateTime.Scan: nil value for field %s", T.def.Name)
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
