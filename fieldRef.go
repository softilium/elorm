package elorm

import (
	"fmt"
)

type FieldValueRef struct {
	fieldValueBase
	factory *Factory
	v       string
	old     string
}

func (T *FieldValueRef) SetFactory(newValue *Factory) {
	T.factory = newValue
}

func (T *FieldValueRef) Set(newValue any) error {

	stringValue := ""
	switch v := newValue.(type) {
	case string:
		stringValue = v
	case IReferableEntity:
		stringValue = v.RefString()
	default:
		if newValue != nil {
			return fmt.Errorf("fieldValueRef.Set: expected string or entityRef pointer for field, got %T", newValue)
		}
	}

	if T.factory == nil {
		return fmt.Errorf("fieldValueRef.Set: missing factory")
	}
	ok, deft := T.factory.IsRef(stringValue)
	if !ok {
		return fmt.Errorf("fieldValueRef.Set: invalid ref %s", stringValue)
	}

	if T.def == nil {
		T.def = deft.RefField
	}

	if T.def != nil && deft.ObjectName != T.def.EntityDef.ObjectName {
		return fmt.Errorf("fieldValueRef.Set: ref %s does not match field type %s", deft.ObjectName, T.def.Name)
	}

	T.isDirty = T.isDirty || stringValue != T.v
	T.v = stringValue
	return nil
}

func (T *FieldValueRef) Get() (any, error) {

	r, err := T.factory.LoadEntityWrapped(T.v)
	if err != nil {
		return nil, err
	}
	return r, nil

}

func (T *FieldValueRef) GetOld() (any, error) {

	r, err := T.factory.LoadEntityWrapped(T.old)
	if err != nil {
		return nil, err
	}
	return r, nil

}

func (T *FieldValueRef) resetOld() {
	T.old = T.v
}

func (T *FieldValueRef) SqlStringValue() (string, error) {
	if T.factory == nil {
		return "", fmt.Errorf("fieldValueRef.SqlStringValue: missing factory")
	}
	switch T.factory.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL, DbDialectMySQL, DbDialectSQLite:
		if T.v == "" {
			return "NULL", nil
		}
		return fmt.Sprintf("'%s'", T.v), nil
	default:
		return "", fmt.Errorf("fieldValueRef.SqlStringValue: unknown database type %d for field %s", T.factory.dbDialect, T.def.Name)
	}
}

func (T *FieldValueRef) AsString() string {
	return T.v
}

func (T *FieldValueRef) Scan(v any) error {
	if v == nil {
		T.v = ""
		T.isDirty = false
		return nil
	}

	asStr := ""

	switch v := v.(type) {
	case string:
		asStr = v
	case []uint8: //MySql
		asStr = string(v)
	default:
		return fmt.Errorf("fieldValueRef.Scan: expected string or []uint8 for field %s, got %T", T.def.Name, v)
	}

	if T.def != nil {
		ok, defTypeFromID := T.def.EntityDef.Factory.IsRef(asStr)
		if !ok {
			return fmt.Errorf("fieldValueRef.Scan: invalid ref %s for field %s", asStr, T.def.Name)
		}
		if defTypeFromID.ObjectName != T.def.EntityDef.ObjectName {
			return fmt.Errorf("fieldValueRef.Scan: ref %s does not match field type %s", defTypeFromID.ObjectName, T.def.Name)
		}
	} else {
		ok, defTypeFromID := T.factory.IsRef(asStr)
		if !ok {
			return fmt.Errorf("fieldValueRef.Scan: invalid ref %s", asStr)
		}
		T.def = defTypeFromID.RefField
	}
	T.v = asStr
	T.isDirty = false
	T.old = T.v
	return nil
}
