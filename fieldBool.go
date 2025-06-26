package elorm

import (
	"fmt"
)

type fieldValueBool struct {
	fieldValueBase
	v   bool
	old bool
}

func (T *fieldValueBool) Set(newValue any) error {
	boolValue, ok := newValue.(bool)
	if !ok {
		return fmt.Errorf("fieldValueBool.Set: expected bool value for field %s, got %T", T.def.Name, newValue)
	}
	T.isDirty = T.isDirty || boolValue != T.v
	T.v = boolValue
	return nil
}

func (T *fieldValueBool) Get() (any, error) {
	return T.v, nil
}

func (T *fieldValueBool) GetOld() (any, error) {
	return T.old, nil
}

func (T *fieldValueBool) resetOld() {
	T.old = T.v
}

func (T *fieldValueBool) SqlStringValue(v ...any) (string, error) {
	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(bool)
		if !ok {
			return "", fmt.Errorf("fieldValueBool.SqlStringValue: expected int64 value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("fieldValueBool.SqlStringValue: missing definition or factory for field %s", T.def.Name)
	}
	switch T.def.EntityDef.Factory.dbDialect {
	case DbDialectPostgres, DbDialectSQLite:
		if v2 {
			return "TRUE", nil
		}
		return "FALSE", nil
	case DbDialectMSSQL, DbDialectMySQL:
		if v2 {
			return "1", nil
		}
		return "0", nil
	default:
		return "", fmt.Errorf("fieldValueBool.SqlStringValue: unknown database type %d for field %s", T.def.EntityDef.Factory.dbDialect, T.def.Name)
	}
}

func (T *fieldValueBool) AsString() string {
	if T.v {
		return "TRUE"
	}
	return "FALSE"
}

func (T *fieldValueBool) Scan(v any) error {
	if v == nil {
		return fmt.Errorf("fieldValueBool.Scan: nil value for field %s", T.def.Name)
	}
	switch vtyped := v.(type) {
	case bool:
		T.v = vtyped
	case int64:
		if vtyped == 0 {
			T.v = false
		} else {
			T.v = true
		}
	default:
		return fmt.Errorf("fieldValueBool.Scan: expected bool or int64 for field %s, got %T", T.def.Name, v)
	}
	T.isDirty = false
	T.old = T.v
	return nil
}
