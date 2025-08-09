package elorm

import (
	"fmt"
	"strings"
)

// Supported field types
const (
	FieldDefTypeString   = 100
	FieldDefTypeInt      = 200
	FieldDefTypeBool     = 300
	FieldDefTypeRef      = 400
	FieldDefTypeNumeric  = 500
	FieldDefTypeDateTime = 600
)

// FieldDef describes a field in an entity.
type FieldDef struct {
	EntityDef          *EntityDef
	Name               string
	Type               int
	Len                int    //for string
	Precision          int    //for numeric
	Scale              int    //for numeric
	DateTimeJSONFormat string //for date time, e.g. "2006-01-02T15:04:05Z07:00"
}

func (T *FieldDef) CreateFieldValue(entity *Entity) (IFieldValue, error) {
	switch T.Type {
	case FieldDefTypeString:
		x := &FieldValueString{}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeInt:
		x := &FieldValueInt{}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeBool:
		x := &FieldValueBool{}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeRef:
		x := &FieldValueRef{factory: T.EntityDef.Factory}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeNumeric:
		x := &FieldValueNumeric{}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeDateTime:
		x := &FieldValueDateTime{}
		x.entity = entity
		x.def = T
		return x, nil
	default:
		return nil, fmt.Errorf("FieldDef.CreateFieldValue: unknown field type %d for field %s", T.Type, T.Name)
	}
}

func (T *FieldDef) SqlColumnName() (string, error) {
	switch T.EntityDef.Factory.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL, DbDialectMySQL, DbDialectSQLite:
		return strings.ToLower(T.Name), nil
	default:
		return "", fmt.Errorf("FieldDef.SqlColumnName: Unknown database type")
	}
}

func (T *FieldDef) SqlColumnType() (string, error) {
	dialect := T.EntityDef.Factory.dbDialect
	switch dialect {
	case DbDialectPostgres:
		return T.sqlColumnTypePostgres()
	case DbDialectMSSQL:
		return T.sqlColumnTypeMSSQL()
	case DbDialectMySQL:
		return T.sqlColumnTypeMySQL()
	case DbDialectSQLite:
		return T.sqlColumnTypeSQLite()
	default:
		return "", fmt.Errorf("FieldDef.SqlColumnType: unknown database type %d", dialect)
	}
}

func (T *FieldDef) sqlColumnTypePostgres() (string, error) {
	switch T.Type {
	case FieldDefTypeString:
		return fmt.Sprintf("varchar(%d)", T.Len), nil
	case FieldDefTypeInt:
		return "int", nil
	case FieldDefTypeBool:
		return "bool", nil
	case FieldDefTypeRef:
		t, err := T.EntityDef.Factory.refColumnType()
		if err != nil {
			return "", err
		}
		return t, nil
	case FieldDefTypeDateTime:
		return "timestamp without time zone", nil
	case FieldDefTypeNumeric:
		return fmt.Sprintf("decimal(%d,%d)", T.Precision, T.Scale), nil
	default:
		return "", fmt.Errorf("fieldDef.sqlColumnTypePostgres: unknown field type: %d", T.Type)
	}
}

func (T *FieldDef) sqlColumnTypeMSSQL() (string, error) {
	switch T.Type {
	case FieldDefTypeString:
		return fmt.Sprintf("nvarchar(%d)", T.Len), nil
	case FieldDefTypeInt:
		return "bigint", nil
	case FieldDefTypeBool:
		return "bit", nil
	case FieldDefTypeRef:
		return fmt.Sprintf("nvarchar(%d)", refFieldLength), nil
	case FieldDefTypeDateTime:
		return "datetime", nil
	case FieldDefTypeNumeric:
		return fmt.Sprintf("decimal(%d,%d)", T.Precision, T.Scale), nil
	default:
		return "", fmt.Errorf("fieldDef.sqlColumnTypeMSSQL: unknown field type: %d", T.Type)
	}
}

func (T *FieldDef) sqlColumnTypeMySQL() (string, error) {
	switch T.Type {
	case FieldDefTypeString:
		return fmt.Sprintf("varchar(%d)", T.Len), nil
	case FieldDefTypeInt:
		return "int", nil
	case FieldDefTypeBool:
		return "tinyint(1)", nil
	case FieldDefTypeRef:
		return fmt.Sprintf("varchar(%d)", refFieldLength), nil
	case FieldDefTypeDateTime:
		return "datetime", nil
	case FieldDefTypeNumeric:
		return fmt.Sprintf("decimal(%d,%d)", T.Precision, T.Scale), nil
	default:
		return "", fmt.Errorf("fieldDef.sqlColumnTypeMySQL: unknown field type: %d", T.Type)
	}
}

func (T *FieldDef) sqlColumnTypeSQLite() (string, error) {
	switch T.Type {
	case FieldDefTypeString:
		return fmt.Sprintf("varchar(%d)", T.Len), nil
	case FieldDefTypeInt:
		return "integer", nil
	case FieldDefTypeBool:
		return "boolean", nil
	case FieldDefTypeRef:
		return fmt.Sprintf("varchar(%d)", refFieldLength), nil
	case FieldDefTypeDateTime:
		return "datetime", nil
	case FieldDefTypeNumeric:
		return fmt.Sprintf("decimal(%d,%d)", T.Precision, T.Scale), nil
	default:
		return "", fmt.Errorf("fieldDef.sqlColumnTypeSQLite: unknown field type: %d", T.Type)
	}
}

func (T *EntityDef) checkName(name string) error {
	for _, v := range T.FieldDefs {
		if v.Name == name {
			return fmt.Errorf("field %s already exists in entity %s", name, T.ObjectName)
		}
	}
	return nil
}
