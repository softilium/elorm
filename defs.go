package elorm

import (
	"database/sql"
	"fmt"
	"strings"
	"time"
)

const (
	// 6 field types are supported
	fieldDefTypeString   = 100
	fieldDefTypeInt      = 200
	fieldDefTypeBool     = 300
	fieldDefTypeRef      = 400
	fieldDefTypeNumeric  = 500
	fieldDefTypeDateTime = 600
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
	case fieldDefTypeString:
		x := &FieldValueString{v: T.DefValue.(string)}
		x.entity = entity
		x.def = T
		return x, nil
	case fieldDefTypeInt:
		x := &FieldValueInt{v: T.DefValue.(int64)}
		x.entity = entity
		x.def = T
		return x, nil
	case fieldDefTypeBool:
		x := &FieldValueBool{v: T.DefValue.(bool)}
		x.entity = entity
		x.def = T
		return x, nil
	case fieldDefTypeRef:
		x := &FieldValueRef{factory: T.EntityDef.Factory, v: T.DefValue.(string)}
		x.entity = entity
		x.def = T
		return x, nil
	case fieldDefTypeNumeric:
		x := &FieldValueNumeric{v: T.DefValue.(float64)}
		x.entity = entity
		x.def = T
		return x, nil
	case fieldDefTypeDateTime:
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
	case fieldDefTypeString:
		return fmt.Sprintf("varchar(%d)", T.Len), nil
	case fieldDefTypeInt:
		return "int", nil
	case fieldDefTypeBool:
		return "bool", nil
	case fieldDefTypeRef:
		t, err := T.EntityDef.Factory.refColumnType()
		if err != nil {
			return "", err
		}
		return t, nil
	case fieldDefTypeDateTime:
		return "timestamp without time zone", nil
	case fieldDefTypeNumeric:
		return fmt.Sprintf("decimal(%d,%d)", T.Precision, T.Scale), nil
	default:
		return "", fmt.Errorf("fieldDef.sqlColumnTypePostgres: unknown field type: %d", T.Type)
	}
}

func (T *FieldDef) sqlColumnTypeMSSQL() (string, error) {
	switch T.Type {
	case fieldDefTypeString:
		return fmt.Sprintf("nvarchar(%d)", T.Len), nil
	case fieldDefTypeInt:
		return "bigint", nil
	case fieldDefTypeBool:
		return "bit", nil
	case fieldDefTypeRef:
		return fmt.Sprintf("nvarchar(%d)", refFieldLength), nil
	case fieldDefTypeDateTime:
		return "datetime", nil
	case fieldDefTypeNumeric:
		return fmt.Sprintf("decimal(%d,%d)", T.Precision, T.Scale), nil
	default:
		return "", fmt.Errorf("fieldDef.sqlColumnTypeMSSQL: unknown field type: %d", T.Type)
	}
}

func (T *FieldDef) sqlColumnTypeMySQL() (string, error) {
	switch T.Type {
	case fieldDefTypeString:
		return fmt.Sprintf("varchar(%d)", T.Len), nil
	case fieldDefTypeInt:
		return "int", nil
	case fieldDefTypeBool:
		return "tinyint(1)", nil
	case fieldDefTypeRef:
		return "varchar(36)", nil
	case fieldDefTypeDateTime:
		return "datetime", nil
	case fieldDefTypeNumeric:
		return fmt.Sprintf("decimal(%d,%d)", T.Precision, T.Scale), nil
	default:
		return "", fmt.Errorf("fieldDef.sqlColumnTypeMySQL: unknown field type: %d", T.Type)
	}
}

func (T *FieldDef) sqlColumnTypeSQLite() (string, error) {
	switch T.Type {
	case fieldDefTypeString:
		return fmt.Sprintf("varchar(%d)", T.Len), nil
	case fieldDefTypeInt:
		return "integer", nil
	case fieldDefTypeBool:
		return "boolean", nil
	case fieldDefTypeRef:
		return fmt.Sprintf("varchar(%d)", refFieldLength), nil
	case fieldDefTypeDateTime:
		return "datetime", nil
	case fieldDefTypeNumeric:
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
		Type:      fieldDefTypeString,
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
		Type:      fieldDefTypeBool,
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
		Type:      fieldDefTypeDateTime,
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
		Type:      fieldDefTypeInt,
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
		Type:      fieldDefTypeRef,
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
		Type:      fieldDefTypeNumeric,
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

type EntityHandlerFunc func(entity any) error

type EntityDef struct { //base for any real entity types
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
	selectStmt           *sql.Stmt

	FillNewHandler    EntityHandlerFunc
	BeforeSaveHandler EntityHandlerFunc
	AfterSaveHandler  EntityHandlerFunc
}

func (T *EntityDef) FieldDefByName(name string) *FieldDef {
	for _, v := range T.FieldDefs {
		if v.Name == name {
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

const (
	FilterEQ       = 50
	FilterNOEQ     = 60
	FilterGT       = 70
	FilterGE       = 80
	FilterLT       = 90
	FilterLE       = 100
	FilterAndGroup = 200
	FilterOrGroup  = 210
)

type Filter struct {
	Op      int
	LeftOp  *FieldDef
	RightOp any
	Childs  []*Filter
}

func AddFilterEQ(leftField *FieldDef, rightValue any) *Filter {
	return &Filter{
		Op:      FilterEQ,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

func AddFilterNOEQ(leftField *FieldDef, rightValue any) *Filter {
	return &Filter{
		Op:      FilterNOEQ,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

func AddFilterGT(leftField *FieldDef, rightValue any) *Filter {
	return &Filter{
		Op:      FilterGT,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}
func AddFilterGE(leftField *FieldDef, rightValue any) *Filter {
	return &Filter{
		Op:      FilterGE,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}
func AddFilterLT(leftField *FieldDef, rightValue any) *Filter {
	return &Filter{
		Op:      FilterLT,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}
func AddFilterLE(leftField *FieldDef, rightValue any) *Filter {
	return &Filter{
		Op:      FilterLE,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

func AddAndGroup(childs ...*Filter) *Filter {
	return &Filter{
		Op:      FilterAndGroup,
		LeftOp:  nil,
		RightOp: nil,
		Childs:  childs,
	}
}
func AddOrGroup(childs ...*Filter) *Filter {
	return &Filter{
		Op:      FilterOrGroup,
		LeftOp:  nil,
		RightOp: nil,
		Childs:  childs,
	}
}

func (T *Filter) RenderSql() string {
	switch T.Op {
	case FilterEQ, FilterNOEQ, FilterGE, FilterGT, FilterLT, FilterLE:
		if T.LeftOp != nil && T.RightOp != nil {
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return fmt.Sprintf("Error: %s", err.Error())
			}
			renderOps := make(map[int]string)
			renderOps[FilterEQ] = "="
			renderOps[FilterNOEQ] = "<>"
			renderOps[FilterGT] = ">"
			renderOps[FilterGE] = ">="
			renderOps[FilterLT] = "<"
			renderOps[FilterLE] = "<="
			fv, _ := T.LeftOp.CreateFieldValue(nil)
			rv, _ := fv.SqlStringValue(T.RightOp)
			return fmt.Sprintf("%s %s %v", colname, renderOps[T.Op], rv)
		}
	case FilterAndGroup, FilterOrGroup:
		renderOps := make(map[int]string)
		renderOps[FilterAndGroup] = " and "
		renderOps[FilterOrGroup] = " or "

		results := make([]string, len(T.Childs))
		for i, v := range T.Childs {
			results[i] = v.RenderSql()
		}
		return fmt.Sprintf("(%s)", strings.Join(results, renderOps[T.Op]))

	default:
		return "NOT IMPLEMENTED YET"
	}
	return ""
}

func (T *EntityDef) SelectEntities(filters []*Filter) ([]*Entity, error) {

	fnames := make([]string, 0, len(T.FieldDefs))

	for _, v := range T.FieldDefs {
		coln, err := v.SqlColumnName()
		if err != nil {
			return nil, fmt.Errorf("EntityDef.SelectEntities: failed to get SQL column name for field %s: %w", v.Name, err)
		}
		fnames = append(fnames, coln)
	}

	tn, err := T.SqlTableName()
	if err != nil {
		return nil, fmt.Errorf("EntityDef.SelectEntities: failed to get SQL table name: %w", err)
	}

	query := fmt.Sprintf("select %s from %s", strings.Join(fnames, ", "), tn)

	// remove nil filters
	for i := len(filters) - 1; i >= 0; i-- {
		if filters[i] == nil {
			filters = append(filters[:i], filters[i+1:]...)
			break
		}
	}

	if len(filters) > 0 {
		query += " where "
		for i, f := range filters {
			if f == nil {
				continue
			}
			if i > 0 {
				query += " and "
			}
			query += f.RenderSql()
		}
	}

	result := make([]*Entity, 0)

	rows, err := T.Factory.DB.Query(query)
	if err != nil {
		return nil, fmt.Errorf("EntityDef.SelectEntities: failed to execute query '%s': %w", query, err)
	}
	defer rows.Close()

	for rows.Next() {

		res, err := T.Factory.CreateEntity(T)
		if err != nil {
			return nil, fmt.Errorf("EntityDef.SelectEntities: failed to create entity: %w", err)
		}

		fp := make([]any, 0, len(T.FieldDefs))
		for _, v := range T.FieldDefs {
			fp = append(fp, res.Values[v.Name].(any))
		}
		err = rows.Scan(fp...)
		if err != nil {
			return nil, fmt.Errorf("EntityDef.SelectEntities: failed to scan row: %w", err)
		}
		res.isNew = false
		T.Factory.loadedEntities.Add(res.RefString(), res)

		result = append(result, res)
	}

	return result, nil

}

func (T *EntityDef) ensureDBStructure() error {

	switch T.Factory.dbDialect {
	case DbDialectPostgres:

		tran, err := T.Factory.DB.Begin()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to begin transaction: %w", err)
		}
		defer tran.Rollback()

		tn, err := T.SqlTableName()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL table name: %w", err)
		}

		_, err = tran.Exec(fmt.Sprintf("create table if not exists %s()", tn))
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to create table: %w", err)
		}

		for _, v := range T.FieldDefs {

			colType, err := v.SqlColumnType()
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column type for field %s: %w", v.Name, err)
			}

			coln, err := v.SqlColumnName()
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column name for field %s: %w", v.Name, err)
			}

			_, err = tran.Exec(fmt.Sprintf("alter table %s add column if not exists %s %s", tn, coln, colType))
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to add column %s: %w", coln, err)
			}

			_, err = tran.Exec(fmt.Sprintf("alter table %s alter column %s type %s", tn, coln, colType))
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to alter column %s: %w", coln, err)
			}

		}

		var cnt int
		row := tran.QueryRow(fmt.Sprintf("select count(*) as chk from information_schema.constraint_column_usage where table_name='%s' and constraint_name='%s_pk'", tn, tn))

		err = row.Scan(&cnt)
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to scan constraint count: %w", err)
		}

		if cnt == 0 {
			_, err = tran.Exec(fmt.Sprintf("alter table %s add constraint %s_pk primary key (Ref)", tn, tn))
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to add primary key constraint: %w", err)
			}
		}

		err = tran.Commit()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to commit transaction: %w", err)
		}
		return nil
	case DbDialectMSSQL:
		tran, err := T.Factory.DB.Begin()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to begin transaction: %w", err)
		}
		defer tran.Rollback()

		// Table name
		tn, err := T.SqlTableName()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL table name: %w", err)
		}

		_, err = tran.Exec(fmt.Sprintf("if not exists (select * from sysobjects where name='%s' and xtype='U') create table %s (ref nvarchar(%d) primary key)", tn, tn, refFieldLength))
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to create table: %w", err)
		}

		for _, v := range T.FieldDefs {
			colType, err := v.SqlColumnType()
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column type for field %s: %w", v.Name, err)
			}
			coln, err := v.SqlColumnName()
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column name for field %s: %w", v.Name, err)
			}
			_, err = tran.Exec(fmt.Sprintf("if not exists (select * from syscolumns where id=object_id('%s') and name='%s') alter table %s add %s %s", tn, coln, tn, coln, colType))
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to add column %s: %w", coln, err)
			}
		}
		err = tran.Commit()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to commit transaction: %w", err)
		}
		return nil
	case DbDialectMySQL:
		tran, err := T.Factory.DB.Begin()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to begin transaction: %w", err)
		}
		defer tran.Rollback()

		// Table name
		tn, err := T.SqlTableName()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL table name: %w", err)
		}

		_, err = tran.Exec(fmt.Sprintf("create table if not exists %s (ref varchar(36) primary key)", tn))
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to create table: %w", err)
		}

		for _, v := range T.FieldDefs {
			colType, err := v.SqlColumnType()
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column type for field %s: %w", v.Name, err)
			}
			coln, err := v.SqlColumnName()
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column name for field %s: %w", v.Name, err)
			}

			rows, err := tran.Query(fmt.Sprintf("SELECT 1 FROM information_schema.columns WHERE table_name = '%s' AND column_name = '%s'", tn, coln))
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get column information for field %s: %w", v.Name, err)
			}
			if !rows.Next() {
				_, err = tran.Exec(fmt.Sprintf("alter table %s add column %s %s", tn, coln, colType))
				if err != nil {
					return fmt.Errorf("EntityDef.ensureDBStructure: failed to add column %s: %w", coln, err)
				}
			}
			rows.Close()
		}
		err = tran.Commit()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to commit transaction: %w", err)
		}
		return nil
	case DbDialectSQLite:
		tran, err := T.Factory.DB.Begin()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to begin transaction: %w", err)
		}
		defer tran.Rollback()

		// Table name
		tn, err := T.SqlTableName()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL table name: %w", err)
		}

		// Create table if not exists with ref as primary key
		_, err = tran.Exec(fmt.Sprintf("create table if not exists %s (ref varchar(%d) primary key)", tn, refFieldLength))
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to create table: %w", err)
		}

		for _, v := range T.FieldDefs {
			colType, err := v.SqlColumnType()
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column type for field %s: %w", v.Name, err)
			}
			coln, err := v.SqlColumnName()
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get SQL column name for field %s: %w", v.Name, err)
			}

			// Check if column exists
			rows, err := tran.Query(fmt.Sprintf("PRAGMA table_info(%s)", tn))
			if err != nil {
				return fmt.Errorf("EntityDef.ensureDBStructure: failed to get column information for field %s: %w", v.Name, err)
			}
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
					return fmt.Errorf("EntityDef.ensureDBStructure: failed to add column %s: %w", coln, err)
				}
			}
		}
		err = tran.Commit()
		if err != nil {
			return fmt.Errorf("EntityDef.ensureDBStructure: failed to commit transaction: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown database type %d", T.Factory.dbDialect)
	}
}
