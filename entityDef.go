package elorm

import (
	"fmt"
	"strings"
)

const (
	RefFieldName         = "Ref"
	IsDeletedFieldName   = "IsDeleted"
	DataVersionFieldName = "DataVersion"
)

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
type EntityHandlerFuncByRef func(ref string) error // we can call this handler without instantiating entity, just by reference for better performance

type EntityDef struct {
	Factory              *Factory
	DataVersionCheckMode int // DataVersionCheckNever, DataVersionCheckDefault, DataVersionCheckAlways
	ObjectName           string
	TableName            string
	Fragments            []string // fragments are used to define reusable parts of entity definitions
	FieldDefs            []*FieldDef
	IndexDefs            []*IndexDef
	RefField             *FieldDef
	IsDeletedField       *FieldDef                // field for soft delete
	DataVersionField     *FieldDef                // field for data versioning
	Wrap                 func(source *Entity) any // optional function to wrap the entity type into custom struct

	fillNewHandlers           []EntityHandlerFunc
	beforeSaveHandlerByRefs   []EntityHandlerFuncByRef
	beforeSaveHandlers        []EntityHandlerFunc
	afterSaveHandlerByRefs    []EntityHandlerFuncByRef
	afterSaveHandlers         []EntityHandlerFunc
	beforeDeleteHandlerByRefs []EntityHandlerFuncByRef
	beforeDeleteHandlers      []EntityHandlerFunc
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
			_ = rows.Close()
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
