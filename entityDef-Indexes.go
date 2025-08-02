package elorm

import (
	"fmt"
	"slices"
	"strings"
)

// IndexDef represents an index definition for an Entity struct.
type IndexDef struct {
	Unique    bool
	FieldDefs []*FieldDef
}

type indexItem struct {
	name    string
	unique  bool
	fields  []string
	matched bool
}

func (T *EntityDef) compileIndexTargets() ([]*indexItem, error) {
	targets := make([]*indexItem, 0)
	for _, id := range T.IndexDefs {
		if id == nil || len(id.FieldDefs) == 0 {
			continue
		}
		newTarget := indexItem{unique: id.Unique}
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
		newTarget.fields = buf
		newTarget.name = fmt.Sprintf("%s_idx_by_%s", tableName, strings.Join(newTarget.fields, "_"))
		if newTarget.unique {
			newTarget.name += "__uniq"
		}
		targets = append(targets, &newTarget)
	}
	return targets, nil
}

func (T *EntityDef) ensureDatabaseIndexes() error {

	var real []*indexItem
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
			if v1.name == v2.name && slices.Compare(v1.fields, v2.fields) == 0 && v1.unique == v2.unique {
				v1.matched = true
				v2.matched = true
			}
		}
	}
	for _, v1 := range real {
		for _, v2 := range targets {
			if v1.matched || v2.matched {
				continue
			}
			if v1.name == v2.name && slices.Compare(v1.fields, v2.fields) == 0 && v1.unique == v2.unique {
				v1.matched = true
				v2.matched = true
			}
		}
	}

	for _, v1 := range targets {
		if v1.matched {
			continue
		}
		buf := strings.Join(v1.fields, ", ")
		if v1.unique {
			_, err = T.Factory.Exec(fmt.Sprintf("create unique index %s on %s (%s)", v1.name, tableName, buf))
		} else {
			_, err = T.Factory.Exec(fmt.Sprintf("create index %s on %s (%s)", v1.name, tableName, buf))
		}
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to create index %s: %w", v1.name, err)
		}
	}
	for _, v2 := range real {
		if v2.matched {
			continue
		}
		switch T.Factory.dbDialect {
		case DbDialectMSSQL, DbDialectMySQL:
			_, err = T.Factory.Exec(fmt.Sprintf("drop index %s on %s", v2.name, tableName))
		case DbDialectPostgres, DbDialectSQLite:
			_, err = T.Factory.Exec(fmt.Sprintf("drop index %s", v2.name))
		default:
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: unknown database type %d", T.Factory.dbDialect)
		}
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDatabaseIndexes: failed to drop index %s: %w", v2.name, err)
		}
	}
	return nil
}

func (T *EntityDef) loadDatabaseIndexesPostgres() ([]*indexItem, error) {

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
	defer func() {
		_ = rows.Close()
	}()
	real := make([]*indexItem, 0)
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
			if v.name == iname {
				v.fields = append(v.fields, cname)
				found = true
				break
			}
		}
		if !found {
			newItem := indexItem{name: iname, unique: uni}
			newItem.fields = []string{cname}
			real = append(real, &newItem)
		}
	}
	return real, nil
}

func (T *EntityDef) loadDatabaseIndexesMSSQL() ([]*indexItem, error) {

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
	defer func() {
		_ = rows.Close()
	}()

	real := make([]*indexItem, 0)
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
			if v.name == iname {
				v.fields = append(v.fields, cname)
				found = true
				break
			}
		}
		if !found {
			newItem := indexItem{name: iname, unique: uni}
			newItem.fields = []string{cname}
			real = append(real, &newItem)
		}
	}
	return real, nil
}

func (T *EntityDef) loadDatabaseIndexesMySQL() ([]*indexItem, error) {
	tableName, err := T.SqlTableName()
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMySQL: failed to get SQL table name: %w", err)
	}

	query := fmt.Sprintf("show index from %s where Key_name!='PRIMARY'", tableName)
	rows, err := T.Factory.Query(query)
	if err != nil {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMySQL: failed to query existing indexes: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	rescols, _ := rows.Columns()
	key_name_colidx := slices.Index(rescols, "Key_name")
	column_name_colidx := slices.Index(rescols, "Column_name")
	nonUnique_colidx := slices.Index(rescols, "Non_unique")
	if key_name_colidx < 0 || column_name_colidx < 0 || nonUnique_colidx < 0 {
		return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesMySQL: missing required columns in index query result")
	}

	real := make([]*indexItem, 0)
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
			if v.name == key_name {
				v.fields = append(v.fields, column_name)
				found = true
				break
			}
		}
		if !found {
			newItem := indexItem{name: key_name, unique: nonUnique == 0}
			newItem.fields = []string{column_name}
			real = append(real, &newItem)
		}
	}
	return real, nil
}

func (T *EntityDef) loadDatabaseIndexesSQLite() ([]*indexItem, error) {
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
	defer func() {
		_ = rows.Close()
	}()

	fake := new(any)

	real := make([]*indexItem, 0)
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
		defer func() {
			_ = indexRows.Close()
		}()

		fields := make([]string, 0)
		for indexRows.Next() {
			var seqno int
			var cname string
			if err := indexRows.Scan(&seqno, fake, &cname); err != nil {
				return nil, fmt.Errorf("EntityDef.loadDatabaseIndexesSQLite: failed to scan index info row: %w", err)
			}
			fields = append(fields, cname)
		}

		real = append(real, &indexItem{
			name:   iname,
			unique: unique == 1,
			fields: fields,
		})
	}
	return real, nil
}

// AddIndex adds a new index to the entity definition.
// The index can be unique or non-unique, and is defined over one or more fields.
// Index will be created or updated automatically in scope of ensureDatabaseIndexes() method.
// Parameters:
//   - Unique: specifies whether the index should enforce uniqueness.
//   - fld: variadic list of FieldDef representing the fields to include in the index.
//
// Returns:
//   - error: describing the reason for failure, or nil if the index was added successfully.
func (T *EntityDef) AddIndex(Unique bool, fld ...FieldDef) error {
	if len(fld) == 0 {
		return fmt.Errorf("EntityDef.AddIndex: no fields provided for index")
	}
	if T.IndexDefs == nil {
		T.IndexDefs = make([]*IndexDef, 0)
	}
	if len(fld) == 1 && fld[0].Name == RefFieldName {
		return fmt.Errorf("EntityDef.AddIndex: cannot create index on reference field %s (it is already indexed as primary key)", RefFieldName)
	}

	idxAsStr := func(idx IndexDef) string {
		buf := make([]string, 0)
		for _, v := range idx.FieldDefs {
			buf = append(buf, v.Name)
		}
		slices.Sort(buf)
		return strings.Join(buf, ",")
	}

	newIndex := &IndexDef{Unique: Unique, FieldDefs: make([]*FieldDef, 0)}
	for _, v := range fld {
		if T.FieldDefByName(v.Name) == nil {
			return fmt.Errorf("EntityDef.AddIndex: field %s does not belong to entity %s", v.Name, T.ObjectName)
		}
		newIndex.FieldDefs = append(newIndex.FieldDefs, &v)
	}

	for _, v := range T.IndexDefs {
		if idxAsStr(*v) == idxAsStr(*newIndex) {
			return fmt.Errorf("EntityDef.AddIndex: index with fields %s already exists for entity %s", idxAsStr(*newIndex), T.ObjectName)
		}
	}

	T.IndexDefs = append(T.IndexDefs, newIndex)
	return nil
}
