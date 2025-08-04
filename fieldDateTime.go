package elorm

import (
	"fmt"
	"sync"
	"time"
)

// FieldValueDateTime is the datetime field value implementation that stores date and time values without timezone.
type FieldValueDateTime struct {
	fieldValueBase
	v    time.Time
	old  time.Time
	lock sync.Mutex
}

func (T *FieldValueDateTime) Set(newValue time.Time) {
	T.lock.Lock()
	defer T.lock.Unlock()

	T.isDirty = T.isDirty || newValue.Compare(T.v) != 0
	T.v = newValue
}

func (T *FieldValueDateTime) Get() time.Time {
	T.lock.Lock()
	defer T.lock.Unlock()

	return T.v
}

func (T *FieldValueDateTime) GetOld() time.Time {
	T.lock.Lock()
	defer T.lock.Unlock()

	return T.old
}

func (T *FieldValueDateTime) resetOld() {
	T.lock.Lock()
	defer T.lock.Unlock()

	T.old = T.v
}

func (T *FieldValueDateTime) SqlStringValue(v ...any) (string, error) {
	T.lock.Lock()
	defer T.lock.Unlock()

	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(time.Time)
		if !ok {
			return "", fmt.Errorf("FieldValueDateTime.SqlStringValue: expected time.Time value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("FieldValueDateTime.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}

	if v2.IsZero() {
		return "NULL", nil
	}

	return fmt.Sprintf("'%s'", v2.Format(T.def.DateTimeJSONFormat)), nil
}

func (T *FieldValueDateTime) AsString() string {
	T.lock.Lock()
	defer T.lock.Unlock()

	return T.v.Format(T.def.DateTimeJSONFormat)
}

func (T *FieldValueDateTime) Scan(v any) error {
	T.lock.Lock()
	defer T.lock.Unlock()

	if v == nil {
		T.v = time.Time{}
		T.old = T.v
		return nil
	}
	switch vtyped := v.(type) {
	case time.Time:
		T.v = vtyped
	case []uint8:
		vt2 := string(vtyped)
		if vt2 == "infinity" || vt2 == "-infinity" { //sqlite
			T.v = time.Time{}
		} else {
			parsedTime, err := time.Parse(T.def.DateTimeJSONFormat, vt2)
			if err != nil {
				return fmt.Errorf("fieldValueDateTime.Scan: failed to parse time from []uint8 for field %s: %v", T.def.Name, err)
			}
			T.v = parsedTime
		}
	default:
		return fmt.Errorf("fieldValueDateTime.Scan: expected time.Time or []uint8 for field %s, got %T", T.def.Name, v)
	}
	T.isDirty = false
	T.old = T.v
	return nil
}
