package elorm

import (
	"fmt"
	"strings"
)

// Filter operation constants for entity filtering in SelectEntities.
const (
	FilterEQ        = 50
	FilterLIKE      = 55
	FilterNOEQ      = 60
	FilterGT        = 70
	FilterGE        = 80
	FilterLT        = 90
	FilterLE        = 100
	FilterIN        = 110
	FilterNOTIN     = 120
	FilterIsNULL    = 130
	FilterIsNOTNULL = 140
	FilterAndGroup  = 200
	FilterOrGroup   = 210
)

// Filter represents a filter condition for entity selection (SelectEntities).
type Filter struct {
	Op      int
	LeftOp  *FieldDef
	RightOp any
	Childs  []*Filter
}

// AddFilterEQ creates a filter for equality comparison.
func AddFilterEQ(leftField *FieldDef, rightValue any) *Filter {
	if leftField == nil {
		return nil
	}
	return &Filter{
		Op:      FilterEQ,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

// AddFilterLIKE creates a filter for LIKE string comparison.
func AddFilterLIKE(leftField *FieldDef, rightValue string) *Filter {
	if leftField == nil || leftField.Type != FieldDefTypeString {
		return nil
	}
	return &Filter{
		Op:      FilterLIKE,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

// AddFilterNOEQ creates a filter for inequality comparison.
func AddFilterNOEQ(leftField *FieldDef, rightValue any) *Filter {
	if leftField == nil {
		return nil
	}
	return &Filter{
		Op:      FilterNOEQ,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

// AddFilterGT creates a filter for greater than comparison.
func AddFilterGT(leftField *FieldDef, rightValue any) *Filter {
	if leftField == nil {
		return nil
	}
	return &Filter{
		Op:      FilterGT,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

// AddFilterGE creates a filter for greater than or equal comparison.
func AddFilterGE(leftField *FieldDef, rightValue any) *Filter {
	if leftField == nil {
		return nil
	}
	return &Filter{
		Op:      FilterGE,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

// AddFilterLT creates a filter for less than comparison.
func AddFilterLT(leftField *FieldDef, rightValue any) *Filter {
	if leftField == nil {
		return nil
	}
	return &Filter{
		Op:      FilterLT,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

// AddFilterLE creates a filter for less than or equal comparison.
func AddFilterLE(leftField *FieldDef, rightValue any) *Filter {
	if leftField == nil {
		return nil
	}
	return &Filter{
		Op:      FilterLE,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

// AddFilterIN creates a filter for IN comparison with multiple values.
func AddFilterIN(leftField *FieldDef, rightValues ...any) *Filter {
	if leftField == nil || len(rightValues) == 0 {
		return nil
	}
	return &Filter{
		Op:      FilterIN,
		LeftOp:  leftField,
		RightOp: rightValues,
	}
}

// AddFilterNOTIN creates a filter for NOT IN comparison with multiple values.
func AddFilterNOTIN(leftField *FieldDef, rightValues ...any) *Filter {
	if leftField == nil || len(rightValues) == 0 {
		return nil
	}
	return &Filter{
		Op:      FilterNOTIN,
		LeftOp:  leftField,
		RightOp: rightValues,
	}
}

// AddFilterIsNULL creates a filter for NULL value check.
func AddFilterIsNULL(leftField *FieldDef) *Filter {
	if leftField == nil {
		return nil
	}
	return &Filter{
		Op:      FilterIsNULL,
		LeftOp:  leftField,
		RightOp: nil,
	}
}

// AddAndGroup creates a filter group with AND logic for combining multiple filters.
func AddAndGroup(childs ...*Filter) *Filter {
	return &Filter{
		Op:      FilterAndGroup,
		LeftOp:  nil,
		RightOp: nil,
		Childs:  childs,
	}
}

// AddOrGroup creates a filter group with OR logic for combining multiple filters.
func AddOrGroup(childs ...*Filter) *Filter {
	return &Filter{
		Op:      FilterOrGroup,
		LeftOp:  nil,
		RightOp: nil,
		Childs:  childs,
	}
}

var renderOpsMap = map[int]string{
	FilterEQ:    "=",
	FilterNOEQ:  "<>",
	FilterGT:    ">",
	FilterGE:    ">=",
	FilterLT:    "<",
	FilterLE:    "<=",
	FilterIN:    "IN",
	FilterNOTIN: "NOT IN",
}

var renderGroupOpsMap = map[int]string{
	FilterAndGroup: " and ",
	FilterOrGroup:  " or ",
}

func (T *Filter) renderWhereClause() (string, error) {
	switch T.Op {
	case FilterEQ, FilterNOEQ, FilterGE, FilterGT, FilterLT, FilterLE:
		if T.LeftOp != nil && T.RightOp != nil {
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to get SQL column name: %w", err)
			}
			fv, err := T.LeftOp.CreateFieldValue(nil)
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to create field value: %w", err)
			}
			rv, err := fv.SqlStringValue(T.RightOp)
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to get SQL string value: %w", err)
			}
			return fmt.Sprintf("%s %s %v", colname, renderOpsMap[T.Op], rv), nil
		}
	case FilterLIKE:
		if T.LeftOp != nil && T.RightOp != nil {
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to get SQL column name: %w", err)
			}
			return fmt.Sprintf("%s LIKE '%%%v%%'", colname, T.RightOp), nil
		}
	case FilterIN, FilterNOTIN:
		if T.LeftOp != nil && T.RightOp != nil {
			fv, err := T.LeftOp.CreateFieldValue(nil)
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to create field value: %w", err)
			}
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to get SQL column name: %w", err)
			}
			var values []string
			for _, v := range T.RightOp.([]any) {
				rv, err := fv.SqlStringValue(v)
				if err != nil {
					return "", fmt.Errorf("Filter.renderWhereClause: failed to get SQL string value for IN/NOT IN: %w", err)
				}
				values = append(values, rv)
			}
			return fmt.Sprintf("%s %s (%s)", colname, renderOpsMap[T.Op], strings.Join(values, ", ")), nil
		}
	case FilterIsNULL:
		if T.LeftOp != nil {
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to get SQL column name: %w", err)
			}
			return fmt.Sprintf("%s IS NULL", colname), nil
		}
	case FilterIsNOTNULL:
		if T.LeftOp != nil {
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to get SQL column name: %w", err)
			}
			return fmt.Sprintf("NOT %s IS NULL", colname), nil
		}
	case FilterAndGroup, FilterOrGroup:
		results := make([]string, len(T.Childs))
		for i, v := range T.Childs {
			clause, err := v.renderWhereClause()
			if err != nil {
				return "", fmt.Errorf("Filter.renderWhereClause: failed to render child clause: %w", err)
			}
			results[i] = clause
		}
		return fmt.Sprintf("(%s)", strings.Join(results, renderGroupOpsMap[T.Op])), nil
	}
	return "", nil
}

// SortItem represents a sort condition element for SelectEntities
type SortItem struct {
	Field *FieldDef
	Asc   bool
}

// SelectEntities retrieves entities from the database with filtering, sorting, and pagination.
func (T *EntityDef) SelectEntities(filters []*Filter, sorts []*SortItem, pageNo int, pageSize int) (result []*Entity, pagesCount int, err error) {
	if filters == nil {
		filters = []*Filter{}
	}
	if sorts == nil {
		// sort by ref by default
		sorts = []*SortItem{{Field: T.RefField, Asc: true}}
	}
	if pageNo < 0 || pageSize < 0 {
		return nil, 0, fmt.Errorf("EntityDef.SelectEntities: invalid pageNo or pageSize")
	}
	if pageNo > 0 && pageSize > 0 && len(sorts) == 0 {
		return nil, 0, fmt.Errorf("EntityDef.SelectEntities: pagination is only supported with sorting")
	}
	result = make([]*Entity, 0)
	pagesCount = 0
	getSql := func(totals bool) (string, error) {
		var builder strings.Builder
		fields := ""
		if totals {
			fields = "count(*) as total"
		} else {
			fnames := make([]string, 0, len(T.FieldDefs))
			for _, v := range T.FieldDefs {
				coln, err := v.SqlColumnName()
				if err != nil {
					return "", fmt.Errorf("EntityDef.SelectEntities: failed to get SQL column name for field %s: %w", v.Name, err)
				}
				fnames = append(fnames, coln)
			}
			fields = strings.Join(fnames, ", ")
		}
		builder.WriteString("select ")
		builder.WriteString(fields)
		builder.WriteString(" from ")
		tablename, err := T.SqlTableName()
		if err != nil {
			return "", fmt.Errorf("EntityDef.SelectEntities: failed to get SQL table name: %w", err)
		}
		builder.WriteString(tablename)
		for i := len(filters) - 1; i >= 0; i-- {
			if filters[i] == nil {
				filters = append(filters[:i], filters[i+1:]...)
			}
		}
		if len(filters) > 0 {
			builder.WriteString(" where ")
			for i, f := range filters {
				if i > 0 {
					builder.WriteString(" and ")
				}
				clause, err := f.renderWhereClause()
				if err != nil {
					return "", fmt.Errorf("EntityDef.SelectEntities: failed to render where clause: %w", err)
				}
				builder.WriteString(clause)
			}
		}
		if len(sorts) > 0 && !totals {
			builder.WriteString(" order by ")
			sortClauses := make([]string, 0, len(sorts))
			for _, s := range sorts {
				if s.Field == nil {
					continue
				}
				coln, err := s.Field.SqlColumnName()
				if err != nil {
					return "", fmt.Errorf("EntityDef.SelectEntities: failed to get SQL column name for sort field: %w", err)
				}
				order := "ASC"
				if !s.Asc {
					order = "DESC"
				}
				sortClauses = append(sortClauses, fmt.Sprintf("%s %s", coln, order))
			}
			builder.WriteString(strings.Join(sortClauses, ", "))
			if pageNo > 0 && pageSize > 0 {
				switch T.Factory.DbDialect() {
				case DbDialectSQLite, DbDialectPostgres:
					builder.WriteString(fmt.Sprintf(" limit %d offset %d", pageSize, (pageNo-1)*pageSize))
				case DbDialectMySQL:
					builder.WriteString(fmt.Sprintf(" limit %d, %d", (pageNo-1)*pageSize, pageSize))
				case DbDialectMSSQL:
					builder.WriteString(fmt.Sprintf(" offset %d rows fetch next %d rows only", (pageNo-1)*pageSize, pageSize))
				}
			}
		}
		return builder.String(), nil
	}

	query, err := getSql(false)
	if err != nil {
		return result, pagesCount, fmt.Errorf("EntityDef.SelectEntities: failed to get SQL query: %w", err)
	}

	rows, err := T.Factory.Query(query)
	if err != nil {
		return result, pagesCount, fmt.Errorf("EntityDef.SelectEntities: failed to execute query '%s': %w", query, err)
	}
	defer func() {
		_ = rows.Close()
	}()

	fp := make([]any, len(T.FieldDefs))
	for rows.Next() {

		res, err := T.Factory.CreateEntity(T)
		if err != nil {
			return result, pagesCount, fmt.Errorf("EntityDef.SelectEntities: failed to create entity: %w", err)
		}
		for i, v := range T.FieldDefs {
			fp[i] = res.Values[v.Name].(any)
		}

		err = rows.Scan(fp...)
		if err != nil {
			return result, pagesCount, fmt.Errorf("EntityDef.SelectEntities: failed to scan row: %w", err)
		}
		res.isNew = false

		cached, ok := T.Factory.loadedEntities.Get(res.RefString())
		if ok {
			if cached.dataVersion.v == res.dataVersion.v {
				res = cached
			} else {
				T.Factory.loadedEntities.Remove(res.RefString())
			}
		}

		T.Factory.loadedEntities.Add(res.RefString(), res)
		result = append(result, res)
	}

	if pageNo > 0 && pageSize > 0 {
		countQuery, err := getSql(true)
		if err != nil {
			return result, pagesCount, fmt.Errorf("EntityDef.SelectEntities: failed to get count SQL query: %w", err)
		}
		countRows, err := T.Factory.Query(countQuery)
		if err != nil {
			return result, pagesCount, fmt.Errorf("EntityDef.SelectEntities: failed to execute count query '%s': %w", countQuery, err)
		}
		defer func() {
			_ = countRows.Close()
		}()
		if countRows.Next() {
			err = countRows.Scan(&pagesCount)
			if err != nil {
				return result, pagesCount, fmt.Errorf("EntityDef.SelectEntities: failed to scan count row: %w", err)
			}
			pagesCount = (pagesCount + pageSize - 1) / pageSize // calculate total pages
		} else {
			return result, pagesCount, fmt.Errorf("EntityDef.SelectEntities: no rows returned for count query '%s'", countQuery)
		}
	}

	return result, pagesCount, nil
}
