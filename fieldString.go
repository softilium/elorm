package elorm

import (
	"fmt"
)

type fieldValueString struct {
	fieldValueBase
	v   string
	old string
}

func (T *fieldValueString) SqlStringValue() (string, error) {
	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("fieldValueString.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	switch T.def.EntityDef.Factory.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL, DbDialectMySQL, DbDialectSQLite:
		return fmt.Sprintf("'%s'", T.v), nil
	default:
		return "", fmt.Errorf("fieldValueString.SqlStringValue: unknown database type %d for field %s", T.def.EntityDef.Factory.dbDialect, T.def.Name)
	}
}

func (T *fieldValueString) Set(newValue any) error {
	stringValue, ok := newValue.(string)
	if !ok {
		return fmt.Errorf("fieldValueString.Set: expected string value for field %s, got %T", T.def.Name, newValue)
	}
	T.isDirty = T.isDirty || stringValue != T.v
	T.v = stringValue
	return nil
}

func (T *fieldValueString) Get() (any, error) {
	return T.v, nil
}

func (T *fieldValueString) GetOld() (any, error) {
	return T.old, nil
}

func (T *fieldValueString) resetOld() {
	T.old = T.v
}

func (T *fieldValueString) AsString() string {
	return T.v
}

func (T *fieldValueString) Scan(v any) error {
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
