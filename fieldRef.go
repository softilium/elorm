package elorm

import (
	"fmt"
	"reflect"
)

const refSplitter = "$$"   //separator between unique ID and entity type in reference strings
const refFieldLength = 107 // length of Ref field in characters, used for string representation of references

// FieldValueRef is the reference (navigation) field value implementation. It supports lazy loading of referenced entity.
type FieldValueRef struct {
	fieldValueBase
	factory *Factory
	v       string
	old     string
}

func isNilInterfaceValue(i interface{}) bool {
	if i == nil {
		return true
	}
	v := reflect.ValueOf(i)
	if v.IsValid() && (v.Kind() == reflect.Ptr || v.Kind() == reflect.Slice ||
		v.Kind() == reflect.Map || v.Kind() == reflect.Func || v.Kind() == reflect.Interface) {
		return v.IsNil()
	}
	return false
}

func (T *FieldValueRef) SetFactory(newValue *Factory) {
	T.factory = newValue
}

func (T *FieldValueRef) Set(newValue any) error {
	stringValue := ""
	switch v := newValue.(type) {
	case string:
		stringValue = v
	case IEntity:
		if !isNilInterfaceValue(v) {
			stringValue = v.RefString()
		}
	default:
		if newValue != nil {
			return fmt.Errorf("FieldValueRef.Set: type assertion failed: expected string or entityRef pointer for field, got %T", newValue)
		}
	}

	if stringValue == "" {
		T.v = ""
		T.isDirty = T.isDirty || stringValue != T.v
		return nil
	}

	if T.factory == nil {
		return fmt.Errorf("FieldValueRef.Set: missing factory")
	}
	ok, deft := T.factory.IsRef(stringValue)
	if !ok {
		return fmt.Errorf("FieldValueRef.Set: invalid ref %s", stringValue)
	}

	if T.def == nil {
		T.def = deft.RefField
	}

	if T.def != nil && deft.ObjectName != T.def.EntityDef.ObjectName {
		return fmt.Errorf("FieldValueRef.Set: ref %s does not match field type %s", deft.ObjectName, T.def.Name)
	}

	T.isDirty = T.isDirty || stringValue != T.v
	T.v = stringValue
	return nil
}

func (T *FieldValueRef) Get() (any, error) {
	r, err := T.factory.LoadEntityWrapped(T.v)
	if err != nil {
		return nil, fmt.Errorf("FieldValueRef.Get: failed to load entity: %w", err)
	}
	return r, nil
}

func (T *FieldValueRef) GetOld() (any, error) {
	r, err := T.factory.LoadEntityWrapped(T.old)
	if err != nil {
		return nil, fmt.Errorf("FieldValueRef.GetOld: failed to load entity: %w", err)
	}
	return r, nil
}

func (T *FieldValueRef) resetOld() {
	T.old = T.v
}

func (T *FieldValueRef) SqlStringValue(v ...any) (string, error) {
	v2 := T.v
	if len(v) == 1 {
		ok := false
		v2, ok = v[0].(string)
		if !ok {
			return "", fmt.Errorf("FieldValueRef.SqlStringValue: expected string value for field %s, got %T", T.def.Name, v)
		}
	}

	if T.factory == nil {
		return "", fmt.Errorf("FieldValueRef.SqlStringValue: missing factory")
	}
	return fmt.Sprintf("'%s'", v2), nil
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
		return fmt.Errorf("FieldValueRef.Scan: type assertion failed: expected string or []uint8 for field %s, got %T", T.def.Name, v)
	}

	if asStr == "" {
		T.v = ""
		T.isDirty = false
		T.old = T.v
		return nil
	}

	if T.def != nil {
		ok, defTypeFromID := T.def.EntityDef.Factory.IsRef(asStr)
		if !ok {
			return fmt.Errorf("FieldValueRef.Scan: invalid ref %s for field %s", asStr, T.def.Name)
		}
		if defTypeFromID.ObjectName != T.def.EntityDef.ObjectName {
			return fmt.Errorf("FieldValueRef.Scan: ref %s does not match field type %s", defTypeFromID.ObjectName, T.def.Name)
		}
	} else {
		ok, defTypeFromID := T.factory.IsRef(asStr)
		if !ok {
			return fmt.Errorf("FieldValueRef.Scan: invalid ref %s", asStr)
		}
		T.def = defTypeFromID.RefField
	}
	T.v = asStr
	T.isDirty = false
	T.old = T.v
	return nil
}
