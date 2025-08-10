package elorm

import (
	"fmt"
	"sync"
)

// FieldValueBool is the bool field value implementation.
type FieldValueBool struct {
	fieldValueBase
	v    bool
	old  bool
	lock sync.Mutex
}

func (T *FieldValueBool) Set(newValue bool) {
	T.lock.Lock()
	defer T.lock.Unlock()

	T.v = newValue
}

func (T *FieldValueBool) Get() bool {
	T.lock.Lock()
	defer T.lock.Unlock()

	return T.v
}

func (T *FieldValueBool) Old() bool {
	T.lock.Lock()
	defer T.lock.Unlock()

	return T.old
}

func (T *FieldValueBool) resetOld() {
	T.lock.Lock()
	defer T.lock.Unlock()

	T.old = T.v
}

func (T *FieldValueBool) SqlStringValue(v ...any) (string, error) {
	T.lock.Lock()
	defer T.lock.Unlock()

	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(bool)
		if !ok {
			return "", fmt.Errorf("FieldValueBool.SqlStringValue: expected bool value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.def == nil || T.def.EntityDef == nil || T.def.EntityDef.Factory == nil {
		return "", fmt.Errorf("FieldValueBool.SqlStringValue: missing definition or factory for field %s", T.def.Name)
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
		return "", fmt.Errorf("FieldValueBool.SqlStringValue: unknown database type %d for field %s", T.def.EntityDef.Factory.dbDialect, T.def.Name)
	}
}

func (T *FieldValueBool) AsString() string {
	T.lock.Lock()
	defer T.lock.Unlock()

	if T.v {
		return "TRUE"
	}
	return "FALSE"
}

func (T *FieldValueBool) Scan(v any) error {
	T.lock.Lock()
	defer T.lock.Unlock()

	if v == nil {
		T.v = false
	} else {
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
	}
	T.old = T.v
	return nil
}
