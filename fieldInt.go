package elorm

import (
	"fmt"
)

type fieldValueInt struct {
	fieldValueBase
	v   int64
	old int64
}

func (T *fieldValueInt) Get() (any, error) {
	return T.v, nil
}

func (T *fieldValueInt) GetOld() (any, error) {
	return T.old, nil
}

func (T *fieldValueInt) resetOld() {
	T.old = T.v
}

func (T *fieldValueInt) Set(newValue any) error {
	intValue, ok := newValue.(int64)
	if !ok {
		return fmt.Errorf("fieldValueInt.Set: expected int64 value for field %s, got %T", T.def.Name, newValue)
	}
	T.isDirty = T.isDirty || intValue != T.v
	T.v = intValue
	return nil
}

func (T *fieldValueInt) SqlStringValue() (string, error) {
	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	switch T.def.EntityDef.Factory.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL, DbDialectMySQL, DbDialectSQLite:
		return fmt.Sprintf("%d", T.v), nil
	default:
		return "", fmt.Errorf("fieldValueInt.SqlStringValue: unknown database type %d for field %s", T.def.EntityDef.Factory.dbDialect, T.def.Name)
	}
}

func (T *fieldValueInt) AsString() string {
	return fmt.Sprintf("%d", T.v)
}

func (T *fieldValueInt) Scan(v any) error {
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
