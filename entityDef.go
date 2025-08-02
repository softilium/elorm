package elorm

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Predefined field names for common fields in entities.
const (
	RefFieldName         = "Ref"
	IsDeletedFieldName   = "IsDeleted"
	DataVersionFieldName = "DataVersion"
)

// EntityHandlerFuncNoContext is a function type for handling entities without context.
// Used for fillNewHandlers.
type EntityHandlerFuncNoContext func(entity any) error

// EntityHandlerFunc is a function type for handling entities with context.
// Used for beforeSave, afterSave, beforeDelete handlers.
type EntityHandlerFunc func(ctx context.Context, entity any) error

// EntityHandlerFuncByRef is a function type for handling entities by reference with context.
// Entity is not instantiated; only reference is provided.
type EntityHandlerFuncByRef func(ctx context.Context, ref string) error

// EntityDef describes the definition of an entity.
type EntityDef struct {
	Factory                 *Factory                 // Owning factory, used to access database and other resources
	DataVersionCheckMode    int                      // DataVersionCheckNever, DataVersionCheckDefault, DataVersionCheckAlways
	ObjectName              string                   // name of the object, elorm-gen created strongly typed structs based on this name
	TableName               string                   // SQL table name, if empty, it will be generated from ObjectName
	Fragments               []string                 // fragments are used to define reusable parts of entity definitions
	FieldDefs               []*FieldDef              // defined fields. All predefined fields (Ref, IsDeleted, DataVersion) are automatically added to this list.
	IndexDefs               []*IndexDef              // defined indexes. PK doesn't need to be defined here, it is always created automatically
	RefField                *FieldDef                // Primary Key, ID
	IsDeletedField          *FieldDef                // field for soft delete
	DataVersionField        *FieldDef                // field for data versioning
	Wrap                    func(source *Entity) any // optional function to wrap the entity type into custom struct (used by elorm-gen)
	AutoExpandFieldsForJSON map[*FieldDef]bool       // if specified, these fields will be automatically expanded when serializing to JSON

	fillNewHandlers           []EntityHandlerFuncNoContext
	beforeSaveHandlerByRefs   []EntityHandlerFuncByRef
	beforeSaveHandlers        []EntityHandlerFunc
	afterSaveHandlers         []EntityHandlerFunc
	beforeDeleteHandlerByRefs []EntityHandlerFuncByRef
	beforeDeleteHandlers      []EntityHandlerFunc
}

// AddStringFieldDef adds a string field definition to this entity def.
func (T *EntityDef) AddStringFieldDef(name string, size int, defValue string) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		EntityDef: T,
		Name:      name,
		Type:      FieldDefTypeString,
		Len:       size,
	}
	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

// AddBoolFieldDef adds a boolean field definition to this entity def.
func (T *EntityDef) AddBoolFieldDef(name string, defValue bool) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		EntityDef: T,
		Name:      name,
		Type:      FieldDefTypeBool,
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

// AddDateTimeFieldDef adds a datetime field definition to this entity def.
func (T *EntityDef) AddDateTimeFieldDef(name string) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		EntityDef:          T,
		Name:               name,
		Type:               FieldDefTypeDateTime,
		DateTimeJSONFormat: time.DateTime, // default format, can be changed later
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

// AddIntFieldDef adds an integer field definition to this entity def.
func (T *EntityDef) AddIntFieldDef(name string, defValue int64) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		EntityDef: T,
		Name:      name,
		Type:      FieldDefTypeInt,
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

// AddRefFieldDef adds a reference field definition to this entity def.
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

// AddNumericFieldDef adds a numeric field definition to this entity def.
func (T *EntityDef) AddNumericFieldDef(name string, Precision int, Scale int, DefValue float64) (*FieldDef, error) {
	if err := T.checkName(name); err != nil {
		return nil, err
	}
	nr := &FieldDef{
		Name:      name,
		Type:      FieldDefTypeNumeric,
		Precision: Precision,
		Scale:     Scale,
		EntityDef: T,
	}

	T.FieldDefs = append(T.FieldDefs, nr)
	return nr, nil
}

// FieldDefByName returns the field definition with the specified name.
func (T *EntityDef) FieldDefByName(name string) *FieldDef {
	for _, v := range T.FieldDefs {
		if strings.EqualFold(v.Name, name) {
			return v
		}
	}
	return nil
}

// SqlTableName returns the SQL table name for this entity definition.
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
		err := T.ensureDBStructurePostgres()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to ensure DB structure for Postgres: %w", err)
		}
	case DbDialectMSSQL:
		err := T.ensureDBStructureMSSQL()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to ensure DB structure for MSSQL: %w", err)
		}
	case DbDialectMySQL:
		err := T.ensureDBStructureMySQL()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to ensure DB structure for MySQL: %w", err)
		}
	case DbDialectSQLite:
		err := T.ensureDBStructureSQLite()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to ensure DB structure for SQLite: %w", err)
		}
	default:
		return fmt.Errorf("unknown database type %d", T.Factory.dbDialect)
	}

	err := T.ensureDatabaseIndexes()
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructure: failed to ensure DB structure: %w", err)
	}

	return nil
}

func (T *EntityDef) ensureDBStructurePostgres() error {
	tran, err := T.Factory.BeginTran()
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to begin transaction: %w", err)
	}

	tn, err := T.SqlTableName()
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to get SQL table name: %w", err)
	}

	_, err = tran.Exec(fmt.Sprintf("create table if not exists %s()", tn))
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to create table: %w", err)
	}

	for _, v := range T.FieldDefs {
		colType, err := v.SqlColumnType()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to get SQL column type for field %s: %w", v.Name, err)
		}

		coln, err := v.SqlColumnName()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to get SQL column name for field %s: %w", v.Name, err)
		}

		_, err = tran.Exec(fmt.Sprintf("alter table %s add column if not exists %s %s", tn, coln, colType))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to add column %s: %w", coln, err)
		}

		_, err = tran.Exec(fmt.Sprintf("alter table %s alter column %s type %s", tn, coln, colType))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to alter column %s: %w", coln, err)
		}
	}

	var cnt int
	row := tran.QueryRow(fmt.Sprintf("select count(*) as chk from information_schema.constraint_column_usage where table_name='%s' and constraint_name='%s_pk'", tn, tn))

	err = row.Scan(&cnt)
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to scan constraint count: %w", err)
	}

	if cnt == 0 {
		_, err = tran.Exec(fmt.Sprintf("alter table %s add constraint %s_pk primary key (Ref)", tn, tn))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to add primary key constraint: %w", err)
		}
	}

	err = T.Factory.CommitTran(tran)
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructurePostgres: failed to commit transaction: %w", err)
	}
	return nil
}

func (T *EntityDef) ensureDBStructureMSSQL() error {
	tran, err := T.Factory.BeginTran()
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructureMSSQL: failed to begin transaction: %w", err)
	}

	tn, err := T.SqlTableName()
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructureMSSQL: failed to get SQL table name: %w", err)
	}

	_, err = tran.Exec(fmt.Sprintf("if not exists (select * from sysobjects where name='%s' and xtype='U') create table %s (ref nvarchar(%d) primary key)", tn, tn, refFieldLength))
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructureMSSQL: failed to create table: %w", err)
	}

	for _, v := range T.FieldDefs {
		colType, err := v.SqlColumnType()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureMSSQL: failed to get SQL column type for field %s: %w", v.Name, err)
		}
		coln, err := v.SqlColumnName()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureMSSQL: failed to get SQL column name for field %s: %w", v.Name, err)
		}
		_, err = tran.Exec(fmt.Sprintf("if not exists (select * from syscolumns where id=object_id('%s') and name='%s') alter table %s add %s %s", tn, coln, tn, coln, colType))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureMSSQL: failed to add column %s: %w", coln, err)
		}
	}
	err = T.Factory.CommitTran(tran)
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructureMSSQL: failed to commit transaction: %w", err)
	}
	return nil
}

func (T *EntityDef) ensureDBStructureMySQL() error {
	tran, err := T.Factory.BeginTran()
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructureMySQL: failed to begin transaction: %w", err)
	}

	tn, err := T.SqlTableName()
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructureMySQL: failed to get SQL table name: %w", err)
	}

	_, err = tran.Exec(fmt.Sprintf("create table if not exists %s (ref varchar(36) primary key)", tn))
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructureMySQL: failed to create table: %w", err)
	}

	for _, v := range T.FieldDefs {
		colType, err := v.SqlColumnType()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureMySQL: failed to get SQL column type for field %s: %w", v.Name, err)
		}
		coln, err := v.SqlColumnName()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureMySQL: failed to get SQL column name for field %s: %w", v.Name, err)
		}

		rows, err := tran.Query(fmt.Sprintf("SELECT 1 FROM information_schema.columns WHERE table_name = '%s' AND column_name = '%s'", tn, coln))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureMySQL: failed to get column information for field %s: %w", v.Name, err)
		}
		defer func() {
			_ = rows.Close()
		}()
		if !rows.Next() {
			_, err = tran.Exec(fmt.Sprintf("alter table %s add column %s %s", tn, coln, colType))
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructureMySQL: failed to add column %s: %w", coln, err)
			}
		}
		_ = rows.Close()
	}
	err = T.Factory.CommitTran(tran)
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructureMySQL: failed to commit transaction: %w", err)
	}
	return nil
}

func (T *EntityDef) ensureDBStructureSQLite() error {
	tran, err := T.Factory.BeginTran()
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructureSQLite: failed to begin transaction: %w", err)
	}

	tn, err := T.SqlTableName()
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructureSQLite: failed to get SQL table name: %w", err)
	}

	_, err = tran.Exec(fmt.Sprintf("create table if not exists %s (ref varchar(%d) primary key)", tn, refFieldLength))
	if err != nil {
		_ = T.Factory.RollbackTran(tran)
		return fmt.Errorf("EntityDef.ensureDBStructureSQLite: failed to create table: %w", err)
	}

	for _, v := range T.FieldDefs {
		colType, err := v.SqlColumnType()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureSQLite: failed to get SQL column type for field %s: %w", v.Name, err)
		}
		coln, err := v.SqlColumnName()
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureSQLite: failed to get SQL column name for field %s: %w", v.Name, err)
		}

		rows, err := tran.Query(fmt.Sprintf("PRAGMA table_info(%s)", tn))
		if err != nil {
			_ = T.Factory.RollbackTran(tran)
			return fmt.Errorf("EntityDef.ensureDBStructureSQLite: failed to get column information for field %s: %w", v.Name, err)
		}
		defer func() {
			_ = rows.Close()
		}()
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
		if !colExists {
			_, err = tran.Exec(fmt.Sprintf("alter table %s add column %s %s", tn, coln, colType))
			if err != nil {
				_ = T.Factory.RollbackTran(tran)
				return fmt.Errorf("EntityDef.ensureDBStructureSQLite: failed to add column %s: %w", coln, err)
			}
		}
	}
	err = T.Factory.CommitTran(tran)
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDBStructureSQLite: failed to commit transaction: %w", err)
	}
	return nil
}

// ActualDataVersionCheckMode returns the effective data version check mode for this entity.
func (T *EntityDef) ActualDataVersionCheckMode() int {
	if T.Factory.AggressiveReadingCache {
		return DataVersionCheckNever
	}
	dvcm := T.DataVersionCheckMode
	if dvcm == DataVersionCheckDefault {
		dvcm = T.Factory.dataVersionCheckMode
	}
	return dvcm
}
