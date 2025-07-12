package elorm

import (
	"fmt"
	"strings"
	"time"
)

const (
	// 6 field types are supported
	FieldDefTypeString   = 100
	FieldDefTypeInt      = 200
	FieldDefTypeBool     = 300
	FieldDefTypeRef      = 400
	FieldDefTypeNumeric  = 500
	FieldDefTypeDateTime = 600
)

const (
	DbDialectPostgres = 100
	DbDialectMSSQL    = 200
	DbDialectMySQL    = 300
	DbDialectSQLite   = 400
)

const (
	DataVersionCheckNever   = -1
	DataVersionCheckDefault = 0
	DataVersionCheckAlways  = 1
)

const refSplitter = "$$"

const refFieldLength = 107 // length of Ref field in characters, used for string representation of references

const (
	RefFieldName         = "Ref"
	IsDeletedFieldName   = "IsDeleted"
	DataVersionFieldName = "DataVersion"
)

type FieldDef struct {
	EntityDef *EntityDef
	Name      string
	Type      int
	Len       int //for string
	Precision int //for numeric
	Scale     int //for numeric
	DefValue  any
}

func (T *FieldDef) CreateFieldValue(entity *Entity) (IFieldValue, error) {
	switch T.Type {
	case FieldDefTypeString:
		x := &FieldValueString{v: T.DefValue.(string)}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeInt:
		x := &FieldValueInt{v: T.DefValue.(int64)}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeBool:
		x := &FieldValueBool{v: T.DefValue.(bool)}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeRef:
		x := &FieldValueRef{factory: T.EntityDef.Factory}
		vt, ok := T.DefValue.(string)
		if ok {
			x.v = vt
		}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeNumeric:
		x := &FieldValueNumeric{v: T.DefValue.(float64)}
		x.entity = entity
		x.def = T
		return x, nil
	case FieldDefTypeDateTime:
		x := &FieldValueDateTime{}
		tv, ok := T.DefValue.(time.Time)
		if ok {
			x.v = tv
		}
		x.entity = entity
		x.def = T
		return x, nil
	default:
		return nil, fmt.Errorf("fieldDef.CreateFieldValue: unknown field type %d for field %s", T.Type, T.Name)
	}
}

func (T *FieldDef) SqlColumnName() (string, error) {

	switch T.EntityDef.Factory.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL, DbDialectMySQL, DbDialectSQLite:
		return strings.ToLower(T.Name), nil
	default:
		return "", fmt.Errorf("fieldDef.SqlColumnName: Unknown database type")
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
		return "", fmt.Errorf("fieldDef.SqlColumnType: unknown database type %d", dialect)
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
		return "varchar(36)", nil
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

func (T *EntityDef) AddStringFieldDef(name string, size int, defValue string) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		EntityDef: T,
		Name:      name,
		Type:      FieldDefTypeString,
		Len:       size,
		DefValue:  defValue,
	}
	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

func (T *EntityDef) AddBoolFieldDef(name string, defValue bool) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		EntityDef: T,
		Name:      name,
		Type:      FieldDefTypeBool,
		DefValue:  defValue,
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

func (T *EntityDef) AddDateTimeFieldDef(name string) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		EntityDef: T,
		Name:      name,
		Type:      FieldDefTypeDateTime,
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

func (T *EntityDef) AddIntFieldDef(name string, defValue int64) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		EntityDef: T,
		Name:      name,
		Type:      FieldDefTypeInt,
		DefValue:  defValue,
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

func (T *EntityDef) AddRefFieldDef(name string, refType *EntityDef) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		Name:      name,
		Type:      FieldDefTypeRef,
		EntityDef: refType,
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

func (T *EntityDef) AddNumericFieldDef(name string, Precision int, Scale int, DefValue float64) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		Name:      name,
		Type:      FieldDefTypeNumeric,
		DefValue:  DefValue,
		Precision: Precision,
		Scale:     Scale,
		EntityDef: T,
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

type IndexDef struct {
	PK        bool
	Unique    bool
	FieldDefs []*FieldDef
}

type EntityHandlerFunc func(entity any) error      // we should instantiate entity before calling this handler
type EntityByRefHandlerFunc func(ref string) error // we can call this handler without instantiating entity, just by reference for better performance

type EntityDef struct {
	Factory              *Factory
	DataVersionCheckMode int // DataVersionCheckNever, DataVersionCheckDefault, DataVersionCheckAlways
	ObjectName           string
	TableName            string
	FieldDefs            []*FieldDef
	IndexDefs            []*IndexDef
	RefField             *FieldDef
	IsDeletedField       *FieldDef                // field for soft delete
	DataVersionField     *FieldDef                // field for data versioning
	Wrap                 func(source *Entity) any // optional function to wrap the entity type into custom one

	FillNewHandler           EntityHandlerFunc
	BeforeSaveHandler        EntityHandlerFunc
	AfterSaveHandler         EntityHandlerFunc
	BeforeDeleteHandler      EntityHandlerFunc
	BeforeSaveHandlerByRef   EntityByRefHandlerFunc
	AfterSaveHandlerByRef    EntityByRefHandlerFunc
	BeforeDeleteHandlerByRef EntityByRefHandlerFunc
}

func (T *EntityDef) FieldDefByName(name string) *FieldDef {
	for _, v := range T.FieldDefs {
		if strings.EqualFold(v.Name, name) {
			return v
		}
	}
	return nil
}

func (T *EntityDef) SqlTableName() (string, error) {
	switch T.Factory.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL, DbDialectMySQL, DbDialectSQLite:
		return strings.ToLower(T.TableName), nil
	default:
		return "", fmt.Errorf("EntityDef.SqlTableName: Unknown database type %d", T.Factory.dbDialect)
	}
}

func (T *EntityDef) ensureDBStructure() error {

	switch T.Factory.dbDialect {
	case DbDialectPostgres:

		tran, err := T.Factory.BeginTran()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to begin transaction: %w", err)
		}

		tn, err := T.SqlTableName()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL table name: %w", err)
		}

		_, err = tran.Exec(fmt.Sprintf("create table if not exists %s()", tn))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to create table: %w", err)
		}

		for _, v := range T.FieldDefs {

			colType, err := v.SqlColumnType()
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column type for field %s: %w", v.Name, err)
			}

			coln, err := v.SqlColumnName()
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column name for field %s: %w", v.Name, err)
			}

			_, err = tran.Exec(fmt.Sprintf("alter table %s add column if not exists %s %s", tn, coln, colType))
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to add column %s: %w", coln, err)
			}

			_, err = tran.Exec(fmt.Sprintf("alter table %s alter column %s type %s", tn, coln, colType))
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to alter column %s: %w", coln, err)
			}

		}

		var cnt int
		row := tran.QueryRow(fmt.Sprintf("select count(*) as chk from information_schema.constraint_column_usage where table_name='%s' and constraint_name='%s_pk'", tn, tn))

		err = row.Scan(&cnt)
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to scan constraint count: %w", err)
		}

		if cnt == 0 {
			_, err = tran.Exec(fmt.Sprintf("alter table %s add constraint %s_pk primary key (Ref)", tn, tn))
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to add primary key constraint: %w", err)
			}
		}

		err = T.Factory.CommitTran(tran)
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to commit transaction: %w", err)
		}
		return nil
	case DbDialectMSSQL:
		tran, err := T.Factory.BeginTran()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to begin transaction: %w", err)
		}

		// Table name
		tn, err := T.SqlTableName()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL table name: %w", err)
		}

		_, err = tran.Exec(fmt.Sprintf("if not exists (select * from sysobjects where name='%s' and xtype='U') create table %s (ref nvarchar(%d) primary key)", tn, tn, refFieldLength))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to create table: %w", err)
		}

		for _, v := range T.FieldDefs {
			colType, err := v.SqlColumnType()
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column type for field %s: %w", v.Name, err)
			}
			coln, err := v.SqlColumnName()
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column name for field %s: %w", v.Name, err)
			}
			_, err = tran.Exec(fmt.Sprintf("if not exists (select * from syscolumns where id=object_id('%s') and name='%s') alter table %s add %s %s", tn, coln, tn, coln, colType))
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to add column %s: %w", coln, err)
			}
		}
		err = T.Factory.CommitTran(tran)
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to commit transaction: %w", err)
		}
		return nil
	case DbDialectMySQL:
		tran, err := T.Factory.BeginTran()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to begin transaction: %w", err)
		}

		// Table name
		tn, err := T.SqlTableName()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL table name: %w", err)
		}

		_, err = tran.Exec(fmt.Sprintf("create table if not exists %s (ref varchar(36) primary key)", tn))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to create table: %w", err)
		}

		for _, v := range T.FieldDefs {
			colType, err := v.SqlColumnType()
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column type for field %s: %w", v.Name, err)
			}
			coln, err := v.SqlColumnName()
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column name for field %s: %w", v.Name, err)
			}

			rows, err := tran.Query(fmt.Sprintf("SELECT 1 FROM information_schema.columns WHERE table_name = '%s' AND column_name = '%s'", tn, coln))
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get column information for field %s: %w", v.Name, err)
			}
			defer rows.Close()
			if !rows.Next() {
				_, err = tran.Exec(fmt.Sprintf("alter table %s add column %s %s", tn, coln, colType))
				if err != nil {
					_ = T.Factory.RollbackTran(tran)
					return fmt.Errorf("EntityDef.ensureDBStructure: failed to add column %s: %w", coln, err)
				}
			}
			rows.Close()
		}
		err = T.Factory.CommitTran(tran)
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to commit transaction: %w", err)
		}
		return nil
	case DbDialectSQLite:
		tran, err := T.Factory.BeginTran()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to begin transaction: %w", err)
		}

		// Table name
		tn, err := T.SqlTableName()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL table name: %w", err)
		}

		// Create table if not exists with ref as primary key
		_, err = tran.Exec(fmt.Sprintf("create table if not exists %s (ref varchar(%d) primary key)", tn, refFieldLength))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to create table: %w", err)
		}

		for _, v := range T.FieldDefs {
			colType, err := v.SqlColumnType()
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column type for field %s: %w", v.Name, err)
			}
			coln, err := v.SqlColumnName()
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column name for field %s: %w", v.Name, err)
			}

			// Check if column exists
			rows, err := tran.Query(fmt.Sprintf("PRAGMA table_info(%s)", tn))
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get column information for field %s: %w", v.Name, err)
			}
			defer rows.Close()
			colExists := false
			for rows.Next() {
				var cid int
				var name, ctype string
				var notnull, pk int
				var dfltValue any
				if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err == nil {
					if name == coln {
						colExists = true
						break
					}
				}
			}
			rows.Close()
			if !colExists {
				_, err = tran.Exec(fmt.Sprintf("alter table %s add column %s %s", tn, coln, colType))
				if err != nil {
					_ = T.Factory.RollbackTran(tran)
					return fmt.Errorf("EntityDef.ensureDBStructure: failed to add column %s: %w", coln, err)
				}
			}
		}
		err = T.Factory.CommitTran(tran)
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to commit transaction: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown database type %d", T.Factory.dbDialect)
	}
}
