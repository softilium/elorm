package elorm

import (
	"fmt"
	"slices"
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

type targetItem struct {
	Name    string
	Unique  bool
	Fields  []string
	Matched bool
}

func (T *EntityDef) compileIndexTargets() ([]*targetItem, error) {
	targets := make([]*targetItem, 0)
	for _, id := range T.IndexDefs {
		if id == nil || len(id.FieldDefs) == 0 {
			continue
		}
		newTarget := targetItem{Unique: id.Unique}
		buf := make([]string, 0)
		for _, fd := range id.FieldDefs {
			coln, err := fd.SqlColumnName()
			if err != nil {
				return nil, fmt.Errorf("EntityDef.compileIndexTargets: failed to get SQL column name for field %s: %w", fd.Name, err)
			}
			buf = append(buf, coln)
		}
		tableName, err := T.SqlTableName()
		if err != nil {
			return nil, fmt.Errorf("EntityDef.compileIndexTargets: failed to get SQL table name: %w", err)
		}
		newTarget.Fields = buf
		newTarget.Name = tableName + "_idx_by_" + strings.Join(newTarget.Fields, "_")
		if newTarget.Unique {
			newTarget.Name += "__uniq"
		}
		targets = append(targets, &newTarget)
	}
	return targets, nil
}

func (T *EntityDef) ensureDatabaseIndexes() error {

	var real []*targetItem
	var err error

	switch T.Factory.dbDialect {
	case DbDialectPostgres:
		real, err = T.loadDatabaseIndexesPostgres()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to ensure indexes for Postgres: %w", err)
		}
	case DbDialectMSSQL:
		real, err = T.loadDatabaseIndexesMSSQL()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to ensure indexes for MSSQL: %w", err)
		}
	case DbDialectMySQL:
		real, err = T.loadDatabaseIndexesMySQL()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to ensure indexes for MySQL: %w", err)
		}
	case DbDialectSQLite:
		real, err = T.loadDatabaseIndexesSQLite()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to ensure indexes for SQLite: %w", err)
		}
	default:
		return nil
	}

	tableName, err := T.SqlTableName()
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to get SQL table name: %w", err)
	}

	targets, err := T.compileIndexTargets()
	if err != nil {
		return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to compile index targets: %w", err)
	}

	for _, v1 := range targets {
		for _, v2 := range real {
			if v1.Name == v2.Name && slices.Compare(v1.Fields, v2.Fields) == 0 && v1.Unique == v2.Unique {
				v1.Matched = true
				v2.Matched = true
			}
		}
	}
	for _, v1 := range real {
		for _, v2 := range targets {
			if v1.Matched || v2.Matched {
				continue
			}
			if v1.Name == v2.Name && slices.Compare(v1.Fields, v2.Fields) == 0 && v1.Unique == v2.Unique {
				v1.Matched = true
				v2.Matched = true
			}
		}
	}

	for _, v1 := range targets {
		if v1.Matched {
			continue
		}
		buf := strings.Join(v1.Fields, ", ")
		if v1.Unique {
			_, err = T.Factory.Exec(fmt.Sprintf("create unique index %s on %s (%s)", v1.Name, tableName, buf))
		} else {
			_, err = T.Factory.Exec(fmt.Sprintf("create index %s on %s (%s)", v1.Name, tableName, buf))
		}
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to create index %s: %w", v1.Name, err)
		}
	}
	for _, v2 := range real {
		if v2.Matched {
			continue
		}
		switch T.Factory.dbDialect {
		case DbDialectMSSQL, DbDialectMySQL:
			_, err = T.Factory.Exec(fmt.Sprintf("drop index %s on %s", v2.Name, tableName))
		case DbDialectPostgres, DbDialectSQLite:
			_, err = T.Factory.Exec(fmt.Sprintf("drop index %s", v2.Name))
		default:
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: unknown database type %d", T.Factory.dbDialect)
		}
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to drop index %s: %w", v2.Name, err)
		}
	}
	return nil
}

func (T *EntityDef) loadDatabaseIndexesPostgres() ([]*targetItem, error) {

	tableName, err := T.SqlTableName()
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadRealIndexes: failed to get SQL table name: %w", err)
	}

	rows, err := T.Factory.Query(fmt.Sprintf(`

select
    i.relname as iname,
	ix.indisunique as uni,
    a.attname as cname
from
    pg_class t, pg_class i, pg_index ix, pg_attribute a 
where
    t.oid = ix.indrelid
    and i.oid = ix.indexrelid
    and a.attrelid = t.oid
    and t.relkind = 'r'
    and a.attnum = ANY(ix.indkey)
    and t.relname like '%s'
	and ix.indisprimary=false
order by
    i.relname,
	array_position(ix.indkey, a.attnum)	

	`, tableName))
	if err != nil {
		return nil, fmt.Errorf("EntityDef.ensureDBIndexesPostgres: failed to query existing indexes: %w", err)
	}
	defer rows.Close()
	real := make([]*targetItem, 0)
	for rows.Next() {
		var iname, cname string
		var uni bool
		if err := rows.Scan(&iname, &uni, &cname); err != nil {
			return nil, fmt.Errorf("EntityDef.ensureDBIndexesPostgres: failed to scan index row: %w", err)
		}
		if len(iname) == 0 || len(cname) == 0 {
			continue
		}
		found := false
		for _, v := range real {
			if v.Name == iname {
				v.Fields = append(v.Fields, cname)
				found = true
				break
			}
		}
		if !found {
			newItem := targetItem{Name: iname, Unique: uni}
			newItem.Fields = []string{cname}
			real = append(real, &newItem)
		}
	}
	return real, nil
}

func (T *EntityDef) loadDatabaseIndexesMSSQL() ([]*targetItem, error) {

	tableName, err := T.SqlTableName()
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMSSQL: failed to get SQL table name: %w", err)
	}

	query := fmt.Sprintf(`

SELECT
    i.name AS iname,
    i.is_unique AS uni,
    c.name AS cname
FROM
    sys.indexes i
    INNER JOIN sys.index_columns ic ON i.object_id = ic.object_id AND i.index_id = ic.index_id
    INNER JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
    INNER JOIN sys.tables t ON i.object_id = t.object_id
WHERE
    t.name = '%s'
    AND i.is_primary_key = 0
ORDER BY
    i.name, ic.index_column_id
	
	`, tableName)

	rows, err := T.Factory.Query(query)
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMSSQL: failed to query existing indexes: %w", err)
	}
	defer rows.Close()

	real := make([]*targetItem, 0)
	for rows.Next() {
		var iname, cname string
		var uni bool
		if err := rows.Scan(&iname, &uni, &cname); err != nil {
			return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMSSQL: failed to scan index row: %w", err)
		}
		if len(iname) == 0 || len(cname) == 0 {
			continue
		}
		found := false
		for _, v := range real {
			if v.Name == iname {
				v.Fields = append(v.Fields, cname)
				found = true
				break
			}
		}
		if !found {
			newItem := targetItem{Name: iname, Unique: uni}
			newItem.Fields = []string{cname}
			real = append(real, &newItem)
		}
	}
	return real, nil
}

func (T *EntityDef) loadDatabaseIndexesMySQL() ([]*targetItem, error) {
	tableName, err := T.SqlTableName()
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMySQL: failed to get SQL table name: %w", err)
	}

	query := fmt.Sprintf("show index from %s where Key_name!='PRIMARY'", tableName)
	rows, err := T.Factory.Query(query)
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMySQL: failed to query existing indexes: %w", err)
	}
	defer rows.Close()

	rescols, _ := rows.Columns()
	key_name_colidx := slices.Index(rescols, "Key_name")
	column_name_colidx := slices.Index(rescols, "Column_name")
	nonUnique_colidx := slices.Index(rescols, "Non_unique")
	if key_name_colidx < 0 || column_name_colidx < 0 || nonUnique_colidx < 0 {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMySQL: missing required columns in index query result")
	}

	real := make([]*targetItem, 0)
	for rows.Next() {
		buf := make([]any, 15)
		for i := range buf {
			buf[i] = new(any)
		}
		var key_name string
		var column_name string
		var nonUnique int
		buf[key_name_colidx] = &key_name
		buf[column_name_colidx] = &column_name
		buf[nonUnique_colidx] = &nonUnique
		if err := rows.Scan(buf...); err != nil {
			return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMySQL: failed to scan index row: %w", err)
		}
		if len(key_name) == 0 || len(column_name) == 0 {
			continue
		}
		found := false
		for _, v := range real {
			if v.Name == key_name {
				v.Fields = append(v.Fields, column_name)
				found = true
				break
			}
		}
		if !found {
			newItem := targetItem{Name: key_name, Unique: nonUnique == 0}
			newItem.Fields = []string{column_name}
			real = append(real, &newItem)
		}
	}
	return real, nil
}

func (T *EntityDef) loadDatabaseIndexesSQLite() ([]*targetItem, error) {
	tableName, err := T.SqlTableName()
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesSQLite: failed to get SQL table name: %w", err)
	}

	query := fmt.Sprintf(`
	PRAGMA index_list('%s')
	`, tableName)

	rows, err := T.Factory.Query(query)
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesSQLite: failed to query existing indexes: %w", err)
	}
	defer rows.Close()

	fake := new(any)

	real := make([]*targetItem, 0)
	for rows.Next() {
		var iname string
		var unique int
		var creationmode string
		if err := rows.Scan(fake, &iname, &unique, &creationmode, fake); err != nil {
			return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesSQLite: failed to scan index row: %w", err)
		}
		if len(iname) == 0 || creationmode != "c" {
			continue
		}

		indexQuery := fmt.Sprintf(`
		PRAGMA index_info('%s')
		`, iname)

		indexRows, err := T.Factory.Query(indexQuery)
		if err != nil {
			return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesSQLite: failed to query index info: %w", err)
		}
		defer indexRows.Close()

		fields := make([]string, 0)
		for indexRows.Next() {
			var seqno int
			var cname string
			if err := indexRows.Scan(&seqno, fake, &cname); err != nil {
				return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesSQLite: failed to scan index info row: %w", err)
			}
			fields = append(fields, cname)
		}

		real = append(real, &targetItem{
			Name:   iname,
			Unique: unique == 1,
			Fields: fields,
		})
	}
	return real, nil
}

func (T *EntityDef) AddIndex(Unique bool, fld ...FieldDef) error {
	if len(fld) == 0 {
		return fmt.Errorf("EntityDef.AddIndex: no fields provided for index")
	}
	if T.IndexDefs == nil {
		T.IndexDefs = make([]*IndexDef, 0)
	}
	newIndex := &IndexDef{Unique: Unique, FieldDefs: make([]*FieldDef, 0)}
	for _, v := range fld {
		if T.FieldDefByName(v.Name) == nil {
			return fmt.Errorf("EntityDef.AddIndex: field %s does not belong to entity %s", v.Name, T.ObjectName)
		}
		newIndex.FieldDefs = append(newIndex.FieldDefs, &v)
	}
	T.IndexDefs = append(T.IndexDefs, newIndex)
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
		defer rows.Close()
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
