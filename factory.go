package elorm

import (
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
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

type Factory struct {
	dbDialect              int
	DB                     *sql.DB
	EntityDefs             []*EntityDef
	loadedEntities         *expirable.LRU[string, *Entity]
	dataVersionCheckMode   int
	AggressiveReadingCache bool // It assumes each database has only one factory instance, so it can cache entities aggressively.

	//for sqlite
	activeTx      *sql.Tx // Active transaction, if any. Used to ensure that all entities are created in the same transaction.
	nestedTxLevel int     // Used to track nested transactions, so we can commit or rollback correctly.
}

func addHandler[ht any](
	f *Factory, dest any, handler ht, errPrefix string,
	getter func(def *EntityDef) []ht,
	setter func(def *EntityDef, newValue []ht)) error {
	if dest == nil {
		return fmt.Errorf("%s: dest is nil", errPrefix)
	}
	switch v := dest.(type) {
	case *EntityDef: // particular entity def
		if getter(v) == nil {
			setter(v, make([]ht, 0))
		}
		setter(v, append(getter(v), handler))
	case string: // fragment name
		found := false
		for _, def := range f.EntityDefs {
			if slices.Contains(def.Fragments, v) {
				if getter(def) == nil {
					setter(def, make([]ht, 0))
				}
				setter(def, append(getter(def), handler))
			}
		}
		if !found {
			return fmt.Errorf("%s: no entity types for fragment %s", errPrefix, v)
		}
	default:
		return fmt.Errorf("%s: unsupported destination type %T", errPrefix, dest)
	}
	return nil
}

func (f *Factory) AddFillNewHandler(dest any, handler EntityHandlerFunc) error {
	return addHandler(f, dest, handler, "AddFillNewHandler",
		func(def *EntityDef) []EntityHandlerFunc { return def.fillNewHandlers },
		func(def *EntityDef, newValue []EntityHandlerFunc) { def.fillNewHandlers = newValue })
}

func (f *Factory) AddBeforeSaveHandlerByRef(dest any, handler EntityHandlerFuncByRef) error {
	return addHandler(f, dest, handler, "AddBeforeSaveHandlerByRef",
		func(def *EntityDef) []EntityHandlerFuncByRef { return def.beforeSaveHandlerByRefs },
		func(def *EntityDef, newValue []EntityHandlerFuncByRef) { def.beforeSaveHandlerByRefs = newValue })
}

func (f *Factory) AddBeforeSaveHandler(dest any, handler EntityHandlerFunc) error {
	return addHandler(f, dest, handler, "AddBeforeSaveHandler",
		func(def *EntityDef) []EntityHandlerFunc { return def.beforeSaveHandlers },
		func(def *EntityDef, newValue []EntityHandlerFunc) { def.beforeSaveHandlers = newValue })
}

func (f *Factory) AddAfterSaveHandlerByRef(dest any, handler EntityHandlerFuncByRef) error {
	return addHandler(f, dest, handler, "AddAfterSaveHandlerByRef",
		func(def *EntityDef) []EntityHandlerFuncByRef { return def.afterSaveHandlerByRefs },
		func(def *EntityDef, newValue []EntityHandlerFuncByRef) { def.afterSaveHandlerByRefs = newValue })
}

func (f *Factory) AddAfterSaveHandler(dest any, handler EntityHandlerFunc) error {
	return addHandler(f, dest, handler, "AddAfterSaveHandler",
		func(def *EntityDef) []EntityHandlerFunc { return def.afterSaveHandlers },
		func(def *EntityDef, newValue []EntityHandlerFunc) { def.afterSaveHandlers = newValue })
}

func (f *Factory) AddBeforeDeleteHandlerByRef(dest any, handler EntityHandlerFuncByRef) error {
	return addHandler(f, dest, handler, "AddBeforeDeleteHandlerByRef",
		func(def *EntityDef) []EntityHandlerFuncByRef { return def.beforeDeleteHandlerByRefs },
		func(def *EntityDef, newValue []EntityHandlerFuncByRef) { def.beforeDeleteHandlerByRefs = newValue })
}

func (f *Factory) AddBeforeDeleteHandler(dest any, handler EntityHandlerFunc) error {
	return addHandler(f, dest, handler, "AddBeforeDeleteHandler",
		func(def *EntityDef) []EntityHandlerFunc { return def.beforeDeleteHandlers },
		func(def *EntityDef, newValue []EntityHandlerFunc) { def.beforeDeleteHandlers = newValue })
}

func (f *Factory) BeginTran() (*sql.Tx, error) {

	if f.dbDialect != DbDialectSQLite {
		return f.DB.Begin()
	}

	if f.nestedTxLevel == 0 {
		newTx, err := f.DB.Begin()
		if err != nil {
			return nil, fmt.Errorf("Factory.BeginTran: failed to begin transaction: %w", err)
		}
		f.activeTx = newTx
		f.nestedTxLevel = 1
	} else {
		f.nestedTxLevel++
	}

	return f.activeTx, nil
}

func (f *Factory) CommitTran(tx *sql.Tx) error {
	if f.dbDialect != DbDialectSQLite {
		return tx.Commit()
	} else {

		if f.nestedTxLevel == 0 {
			return fmt.Errorf("Factory.CommitTran: no active transaction to commit")
		}
		f.nestedTxLevel--
		if f.nestedTxLevel == 0 {
			err := f.activeTx.Commit()
			if err != nil {
				_ = f.RollbackTran(tx)
			}
			f.activeTx = nil
			return err
		}
		return nil
	}
}

func (f *Factory) Query(query string, args ...any) (*sql.Rows, error) {
	if f.dbDialect == DbDialectSQLite {
		if f.nestedTxLevel > 0 {
			return f.activeTx.Query(query, args...)
		}
		return f.DB.Query(query, args...)
	}
	return f.DB.Query(query, args...)
}

func (f *Factory) RollbackTran(tx *sql.Tx) error {
	if f.dbDialect != DbDialectSQLite {
		return tx.Rollback()
	} else {
		if f.nestedTxLevel == 0 {
			return fmt.Errorf("Factory.RollbackTran: no active transaction to rollback")
		}
		err := f.activeTx.Rollback()
		f.activeTx = nil
		return err
	}
}

func CreateFactory(dbDialect string, connectionString string) (*Factory, error) {
	if dbDialect == "" {
		return nil, fmt.Errorf("Factory.CreateFactory: dbDialect is empty")
	}
	if connectionString == "" {
		return nil, fmt.Errorf("Factory.CreateFactory: connectionString is empty")
	}

	var dbd int
	switch dbDialect {
	case "postgres":
		dbd = DbDialectPostgres
	case "mssql":
		dbd = DbDialectMSSQL
	case "mysql":
		dbd = DbDialectMySQL
	case "sqlite", "sqlite3":
		dbd = DbDialectSQLite
	default:
		return nil, fmt.Errorf("Factory.CreateFactory: unsupported db dialect: %s", dbDialect)
	}

	r := &Factory{
		dbDialect:              dbd,
		EntityDefs:             make([]*EntityDef, 0),
		loadedEntities:         expirable.NewLRU[string, *Entity](0, nil, time.Minute*10),
		dataVersionCheckMode:   DataVersionCheckAlways,
		AggressiveReadingCache: false,
	}
	var err error
	r.DB, err = sql.Open(dbDialect, connectionString)
	if err != nil {
		return nil, fmt.Errorf("Factory.CreateFactory: failed to open DB: %w", err)
	}

	err = r.DB.Ping()
	if err != nil {
		return nil, fmt.Errorf("Factory.CreateFactory: failed to ping DB: %w", err)
	}

	return r, nil
}

func (T *Factory) DbDialect() int {
	return T.dbDialect
}

func (T *Factory) SetDataVersionCheckMode(mode int) error {
	if mode != DataVersionCheckNever && mode != DataVersionCheckAlways {
		return fmt.Errorf("factory.SetDataVersionCheckMode: invalid mode %d, must be one of -1, 1", mode)
	}
	T.dataVersionCheckMode = mode
	return nil
}

func (T *Factory) CreateEntityDef(ObjectName string, TableName string) (*EntityDef, error) {
	if ObjectName == "" {
		return nil, fmt.Errorf("Factory.CreateEntityDef: ObjectName is empty")
	}
	if TableName == "" {
		return nil, fmt.Errorf("Factory.CreateEntityDef: TableName is empty")
	}
	for _, def := range T.EntityDefs {

		if strings.EqualFold(def.ObjectName, ObjectName) {
			return nil, fmt.Errorf("Factory.CreateEntityDef: entity definition with name %s already exists", ObjectName)
		}

		if strings.EqualFold(def.TableName, TableName) {
			return nil, fmt.Errorf("Factory.CreateEntityDef: entity definition with name %s already exists", ObjectName)
		}
	}

	r := &EntityDef{
		ObjectName:           ObjectName,
		TableName:            TableName,
		Factory:              T,
		FieldDefs:            make([]*FieldDef, 0),
		IndexDefs:            make([]*IndexDef, 0),
		DataVersionCheckMode: DataVersionCheckDefault,
	}
	T.EntityDefs = append(T.EntityDefs, r)

	var err error

	r.RefField, err = r.AddRefFieldDef(RefFieldName, r)
	if err != nil {
		return nil, fmt.Errorf("Factory.CreateEntityDef: error creating Ref field for %s: %w", ObjectName, err)
	}

	r.IsDeletedField, err = r.AddBoolFieldDef(IsDeletedFieldName, false)
	if err != nil {
		return nil, fmt.Errorf("Factory.CreateEntityDef: error creating IsDeleted field for %s: %w", ObjectName, err)
	}

	r.DataVersionField, err = r.AddStringFieldDef(DataVersionFieldName, 20, "")
	if err != nil {
		return nil, fmt.Errorf("Factory.CreateEntityDef: error creating DataVersion field for %s: %w", ObjectName, err)
	}

	return r, nil
}

func (T *Factory) NewRef(def *EntityDef) string {
	if def == nil {
		return NewRef()
	}
	return fmt.Sprintf("%s%s%s", NewRef(), refSplitter, strings.ToLower(def.ObjectName))
}

func (T *Factory) IsRef(s string) (bool, *EntityDef) {
	if s == "" {
		return false, nil
	}
	parts := strings.Split(s, refSplitter)
	if len(parts) != 2 {
		return false, nil
	}
	for _, def := range T.EntityDefs {
		if strings.EqualFold(def.ObjectName, parts[1]) {
			return true, def
		}
	}
	return false, nil
}

func (T *Factory) CreateEntity(def *EntityDef) (*Entity, error) {
	if def == nil {
		return nil, fmt.Errorf("Factory.CreateEntity: def is nil")
	}
	r := &Entity{
		Factory:   T,
		entityDef: def,
		isNew:     true,
		Values:    make(map[string]IFieldValue),
	}
	for _, fd := range def.FieldDefs {
		fv, err := fd.CreateFieldValue(r)
		if err != nil {
			return nil, fmt.Errorf("Factory.CreateEntity: error creating field value for %s: %w", fd.Name, err)
		}
		r.Values[fd.Name] = fv
	}
	r.ref = r.Values[RefFieldName].(*FieldValueRef)
	if err := r.ref.Set(T.NewRef(def)); err != nil {
		return nil, fmt.Errorf("Factory.CreateEntity: error setting Ref: %w", err)
	}

	r.isDeleted = r.Values[IsDeletedFieldName].(*FieldValueBool)
	r.isDeleted.Set(false)

	r.dataVersion = r.Values[DataVersionFieldName].(*FieldValueString)

	T.loadedEntities.Add(r.RefString(), r)

	for _, handler := range def.fillNewHandlers {
		err := handler(r.entityDef.Wrap(r))
		if err != nil {
			return nil, fmt.Errorf("Factory.CreateEntity: fillNewHandler failed: %w", err)
		}
	}

	return r, nil
}

func (T *Factory) CreateEntityWrapped(def *EntityDef) (any, error) {
	if def == nil {
		return nil, fmt.Errorf("Factory.CreateEntityWrapped: def is nil")
	}

	r, err := T.CreateEntity(def)
	if err != nil {
		return nil, fmt.Errorf("Factory.CreateEntityWrapped: failed to create entity: %w", err)
	}

	if def.Wrap != nil {
		return def.Wrap(r), nil
	}

	return r, nil

}

func (T *Factory) LoadEntity(Ref string) (*Entity, error) {

	ok, def := T.IsRef(Ref)
	if !ok {
		return nil, fmt.Errorf("Factory.LoadEntity: invalid ref %s", Ref)
	}

	tableName, err := def.SqlTableName()
	if err != nil {
		return nil, fmt.Errorf("Factory.LoadEntity: failed to get SQL table name for entity %s: %w", def.ObjectName, err)
	}

	dvcm := def.DataVersionCheckMode
	if dvcm == DataVersionCheckDefault {
		dvcm = T.dataVersionCheckMode
	}

	fromCache, ok := T.loadedEntities.Get(Ref)
	if ok {
		if !T.AggressiveReadingCache && dvcm != DataVersionCheckNever {

			row := T.DB.QueryRow(fmt.Sprintf("select 1 from %s where Ref=%s and DataVersion=%s", tableName, Ref, fromCache.DataVersion()))
			var scanBuffer *int
			if row.Scan(scanBuffer) == sql.ErrNoRows {
				// The entity is not in the database or it changed, so we need to reload it.
				T.loadedEntities.Remove(Ref)
			} else {
				return fromCache, nil
			}
		} else {
			return fromCache, nil
		}
	}

	// Pre-initialize slices with capacity
	fn := make([]string, 0, len(def.FieldDefs))
	fp := make([]any, 0, len(def.FieldDefs))

	res, err := T.CreateEntity(def)
	if err != nil {
		return nil, err
	}

	if err := res.ref.Set(Ref); err != nil {
		return nil, fmt.Errorf("Factory.LoadEntity: error setting Ref: %w", err)
	}

	for _, v := range def.FieldDefs {
		if v.Name != RefFieldName {
			fn = append(fn, v.Name)
			fp = append(fp, res.Values[v.Name].(any))
		}
	}

	sql := ""
	switch T.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL:
		sql = fmt.Sprintf("select %s from %s where ref=$1", strings.Join(fn, ", "), tableName)
	case DbDialectMySQL, DbDialectSQLite:
		sql = fmt.Sprintf("select %s from %s where ref=?", strings.Join(fn, ", "), tableName)
	}
	rows, err := T.Query(sql, Ref)
	if err != nil {
		return nil, fmt.Errorf("Factory.LoadEntity: failed to query select statement: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("Factory.LoadEntity: rows error: %w", err)
		}
		return nil, fmt.Errorf("Factory.LoadEntity: entity not found in database")
	}

	err = rows.Scan(fp...)
	if err != nil {
		return nil, fmt.Errorf("Factory.LoadEntity: failed to scan row: %w", err)
	}
	res.isNew = false
	T.loadedEntities.Add(Ref, res)

	return res, nil
}

func (T *Factory) LoadEntityWrapped(Ref string) (any, error) {
	res, err := T.LoadEntity(Ref)
	if err != nil {
		return nil, err
	}
	if res.entityDef.Wrap != nil {
		return res.entityDef.Wrap(res), nil
	}
	return res, nil
}

func (T *Factory) FetchRowMap(rows *sql.Rows) (map[string]any, error) {
	if rows == nil {
		return nil, fmt.Errorf("Factory.FetchRowMap: rows cannot be nil")
	}
	rc, _ := rows.Columns()
	cts, _ := rows.ColumnTypes()

	// Pre-allocate maps and slices with known capacity
	colCount := len(cts)
	hm := make(map[string]any, colCount)
	tmp := make([]any, colCount)

	for idx := range tmp {

		colTypeName := strings.ToLower(cts[idx].DatabaseTypeName())
		colLen, colLenReceived := cts[idx].Length()
		tmp[idx] = new(any)

		switch T.dbDialect {
		case DbDialectPostgres:
			if colLenReceived && colTypeName == "varchar" && colLen == refFieldLength {
				r := &FieldValueRef{}
				r.SetFactory(T)
				tmp[idx] = r
			}
		case DbDialectMSSQL:
			if colLenReceived && colTypeName == "nvarchar" && colLen == refFieldLength {
				r := &FieldValueRef{}
				r.SetFactory(T)
				tmp[idx] = r
			}
		case DbDialectSQLite:
			if colTypeName == fmt.Sprintf("varchar(%d)", refFieldLength) {
				r := &FieldValueRef{}
				r.SetFactory(T)
				tmp[idx] = r
			}
		}
	}
	err := rows.Scan(tmp...)
	if err != nil {
		return nil, fmt.Errorf("Factory.FetchRowMap: scan error: %w", err)
	}
	for cidx, c := range rc {
		switch tmp[cidx].(type) {
		case *FieldValueRef:
			hm[c] = tmp[cidx].(*FieldValueRef)
		default:
			hm[c] = tmp[cidx]
		}
	}

	// MySql cannot provide us column length, so we need to check if the column is a reference
	if T.dbDialect == DbDialectMySQL {
		for _, c := range rc {
			if q, ok := hm[c].(*any); ok && q != nil {
				if q2, ok := (*q).([]uint8); ok {
					refStr := string(q2)
					itsRef, _ := T.IsRef(refStr)
					if itsRef {
						r := &FieldValueRef{}
						r.SetFactory(T)
						err = r.Set(refStr)
						if err != nil {
							return nil, fmt.Errorf("Factory.FetchRowMap: error setting ref value: %w", err)
						}
						hm[c] = r
					}
				}
			}
		}
	}

	return hm, nil
}

func (T *Factory) DeleteEntity(ref string) error {

	if ref == "" {
		return fmt.Errorf("Factory.DeleteEntity: ref is empty")
	}

	ok, def := T.IsRef(ref)
	if !ok {
		return fmt.Errorf("Factory.DeleteEntity: invalid ref %s", ref)
	}

	var err error
	tx, err := T.BeginTran()
	if err != nil {
		return fmt.Errorf("Factory.DeleteEntity: failed to begin transaction: %w", err)
	}

	// before delete handlers
	for _, handler := range def.beforeDeleteHandlerByRefs {
		err := handler(ref)
		if err != nil {
			_ = T.RollbackTran(tx)
			return fmt.Errorf("Factory.DeleteEntity: BeforeDeleteHandlerByRef failed: %w", err)
		}
	}
	if len(def.beforeDeleteHandlers) > 0 {
		loaded, err := T.LoadEntity(ref)
		if err != nil {
			_ = T.RollbackTran(tx)
			return fmt.Errorf("Factory.DeleteEntity: failed to load entity for deletion (for running BeforeDeleteHandler): %w", err)
		}
		for _, handler := range def.beforeDeleteHandlers {
			err = handler(loaded)
			if err != nil {
				_ = T.RollbackTran(tx)
				return fmt.Errorf("Factory.DeleteEntity: BeforeDeleteHandler failed: %w", err)
			}
		}
	}

	tableName, err := def.SqlTableName()
	if err != nil {
		_ = T.RollbackTran(tx)
		return fmt.Errorf("Factory.DeleteEntity: failed to get SQL table name for entity %s: %w", def.ObjectName, err)
	}

	switch T.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL:
		sql := fmt.Sprintf("delete from %s where Ref=$1", tableName)
		_, err = tx.Exec(sql, ref)
	case DbDialectMySQL, DbDialectSQLite:
		sql := fmt.Sprintf("delete from %s where Ref=?", tableName)
		_, err = tx.Exec(sql, ref)
	default:
		_ = T.RollbackTran(tx)
		return fmt.Errorf("Factory.DeleteEntity: unsupported db dialect: %d", T.dbDialect)
	}

	if err != nil {
		_ = T.RollbackTran(tx)
		return fmt.Errorf("Factory.DeleteEntity: failed to delete entity: %w", err)
	}

	T.loadedEntities.Remove(ref)

	return T.CommitTran(tx)
}

// Database structure related methods

func (T *Factory) refColumnType() (string, error) {
	switch T.dbDialect {
	case DbDialectPostgres, DbDialectMSSQL:
		return "elorm_ref_type", nil
	case DbDialectMySQL:
		return fmt.Sprintf("VARCHAR(%d)", refFieldLength), nil
	default:
		return "", fmt.Errorf("factory.RefColumnType: unsupported db dialect: %d", T.dbDialect)
	}
}

func (T *Factory) createRefColumnType() error {

	switch T.dbDialect {
	case DbDialectPostgres:

		refObjIdDomain, err := T.refColumnType()
		if err != nil {
			return fmt.Errorf("Factory.createRefColumnType: failed to get ref column type: %w", err)
		}

		row, err := T.DB.Query("select count(*) as cnt from pg_type where typname=$1", refObjIdDomain)
		if err != nil {
			return fmt.Errorf("Factory.createRefColumnType: failed to query pg_type: %w", err)
		}
		defer row.Close()

		row.Next()
		if err = row.Err(); err != nil {
			return fmt.Errorf("Factory.createRefColumnType: row error: %w", err)
		}

		cnt := 0
		if err = row.Scan(&cnt); err != nil {
			return fmt.Errorf("Factory.createRefColumnType: failed to scan count: %w", err)
		}

		if cnt == 0 {

			_, err = T.DB.Exec(fmt.Sprintf("create domain %s as varchar(%d)", refObjIdDomain, refFieldLength))
			if err != nil {
				return fmt.Errorf("Factory.createRefColumnType: failed to create domain: %w", err)
			}
		}

		return nil
	case DbDialectMSSQL:
		// For MSSQL, check if the type exists, if not, create it as a user-defined type (UDT)
		refObjIdDomain, err := T.refColumnType()
		if err != nil {
			return fmt.Errorf("Factory.createRefColumnType: failed to get ref column type: %w", err)
		}
		var cnt int
		row := T.DB.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM sys.types WHERE name = '%s'", refObjIdDomain))
		err = row.Scan(&cnt)
		if err != nil {
			return fmt.Errorf("Factory.createRefColumnType: failed to query sys.types: %w", err)
		}
		if cnt == 0 {
			_, err = T.DB.Exec(fmt.Sprintf("CREATE TYPE %s FROM nvarchar(%d)", refObjIdDomain, refFieldLength))
			if err != nil {
				return fmt.Errorf("Factory.createRefColumnType: failed to create type: %w", err)
			}
		}
		return nil
	case DbDialectMySQL, DbDialectSQLite:
		// MySQL does not support custom domains/types like Postgres/MSSQL, so just ensure columns use VARCHAR(107)
		return nil
	default:
		return fmt.Errorf("factory.CreateRefColumnType: unsupported db dialect: %d", T.dbDialect)
	}

}

func (T *Factory) EnsureDBStructure() error {
	err := T.createRefColumnType()
	if err != nil {
		return fmt.Errorf("Factory.EnsureDBStructure: failed to create ref column type: %w", err)
	}

	for _, def := range T.EntityDefs {
		err := def.ensureDBStructure()
		if err != nil {
			return fmt.Errorf("Factory.EnsureDBStructure: failed to ensure DB structure: %w", err)
		}
	}
	return nil
}
