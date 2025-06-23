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

func (T *fieldValueDateTime) SqlStringValue() (string, error) {
	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("fieldValueDateTime.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	switch T.def.EntityDef.Factory.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL, DbDialectMySQL, DbDialectSQLite:
		return fmt.Sprintf("'%s'", T.v.Format(time.DateTime)), nil
	default:
		return "", fmt.Errorf("fieldValueDateTime.SqlStringValue: unknown database type %d for field %s", T.def.EntityDef.Factory.dbDialect, T.def.Name)
	}
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
