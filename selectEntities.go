package elorm

import (
	"fmt"
	"strings"
)

const (
	FilterEQ       = 50
	FilterLIKE     = 55
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

var renderOpsMap = map[int]string{
	FilterEQ:   "=",
	FilterNOEQ: "<>",
	FilterGT:   ">",
	FilterGE:   ">=",
	FilterLT:   "<",
	FilterLE:   "<=",
}

var renderGroupOpsMap = map[int]string{
	FilterAndGroup: " and ",
	FilterOrGroup:  " or ",
}

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

func AddFilterLIKE(leftField *FieldDef, rightValue string) *Filter {
	if leftField == nil || leftField.Type != fieldDefTypeString {
		return nil
	}
	return &Filter{
		Op:      FilterLIKE,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

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

func (T *Filter) renderWhereClause() string {
	switch T.Op {
	case FilterEQ, FilterNOEQ, FilterGE, FilterGT, FilterLT, FilterLE:
		if T.LeftOp != nil && T.RightOp != nil {
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return fmt.Sprintf("Filter.renderWhereClause: failed to get SQL column name: %s", err.Error())
			}
			fv, err := T.LeftOp.CreateFieldValue(nil)
			if err != nil {
				return fmt.Sprintf("Filter.renderWhereClause: failed to create field value: %s", err.Error())
			}
			rv, err := fv.SqlStringValue(T.RightOp)
			if err != nil {
				return fmt.Sprintf("Filter.renderWhereClause: failed to get SQL string value: %s", err.Error())
			}
			return fmt.Sprintf("%s %s %v", colname, renderOpsMap[T.Op], rv)
		}
	case FilterLIKE:
		if T.LeftOp != nil && T.RightOp != nil {
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return fmt.Sprintf("Filter.renderWhereClause: failed to get SQL column name: %s", err.Error())
			}
			return fmt.Sprintf("%s LIKE '%%%v%%'", colname, T.RightOp)
		}
	case FilterAndGroup, FilterOrGroup:
		results := make([]string, len(T.Childs))
		for i, v := range T.Childs {
			results[i] = v.renderWhereClause()
		}
		return fmt.Sprintf("(%s)", strings.Join(results, renderGroupOpsMap[T.Op]))
	}
	return ""
}

type SortItem struct {
	Field *FieldDef
	Asc   bool
}

func (T *EntityDef) SelectEntities(filters []*Filter, sorts []*SortItem, pageNo int, pageSize int) (result []*Entity, pages int, err error) {
	if filters == nil {
		filters = []*Filter{}
	}
	if sorts == nil {
		sorts = []*SortItem{}
	}
	if pageNo < 0 || pageSize < 0 {
		return nil, 0, fmt.Errorf("EntityDef.SelectEntities: invalid pageNo or pageSize")
	}
	if pageNo > 0 && pageSize > 0 && len(sorts) == 0 {
		return nil, 0, fmt.Errorf("EntityDef.SelectEntities: pagination is only supported with sorting")
	}
	result = make([]*Entity, 0)
	pages = 0
	getSql := func(totals bool) (string, error) {
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

		tn, err := T.SqlTableName()
		if err != nil {
			return "", fmt.Errorf("EntityDef.SelectEntities: failed to get SQL table name: %w", err)
		}

		query := fmt.Sprintf("select %s from %s", fields, tn)

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
				query += f.renderWhereClause()
			}
		}

		if len(sorts) > 0 && !totals {
			query += " order by "
			sortClauses := make([]string, 0, len(sorts))
			for _, s := range sorts {
				if s.Field == nil {
					continue
				}
				coln, err := s.Field.SqlColumnName()
				if err != nil {
					return "", fmt.Errorf("EntityDef.SelectEntities: failed to get SQL column name for field %s: %w", s.Field.Name, err)
				}
				if s.Asc {
					sortClauses = append(sortClauses, fmt.Sprintf("%s asc", coln))
				} else {
					sortClauses = append(sortClauses, fmt.Sprintf("%s desc", coln))
				}
			}
			query += strings.Join(sortClauses, ", ")
			query += " ##afterall##"
		}

		if !totals && pageNo > 0 && pageSize > 0 {
			switch T.Factory.DbDialect() {
			case DbDialectPostgres:
				query = strings.ReplaceAll(query, "##afterall##", fmt.Sprintf("limit %d offset %d", pageSize, (pageNo-1)*pageSize))
			case DbDialectMSSQL:
				query = strings.ReplaceAll(query, "##afterall##", fmt.Sprintf("OFFSET %d ROWS FETCH NEXT %d ROWS ONLY", (pageNo-1)*pageSize, pageSize))
			case DbDialectMySQL:
				query = strings.ReplaceAll(query, "##afterall##", fmt.Sprintf("limit %d offset %d", pageSize, (pageNo-1)*pageSize))
			case DbDialectSQLite:
				query = strings.ReplaceAll(query, "##afterall##", fmt.Sprintf("limit %d offset %d", pageSize, (pageNo-1)*pageSize))
			default:
				return "", fmt.Errorf("EntityDef.SelectEntities: pagination not supported for this DB dialect: %d", T.Factory.DbDialect())
			}
		} else {
			query = strings.ReplaceAll(query, "###afterall#", "")
		}
		return query, nil
	}

	query, err := getSql(false)
	if err != nil {
		return result, pages, fmt.Errorf("EntityDef.SelectEntities: failed to get SQL query: %w", err)
	}

	rows, err := T.Factory.DB.Query(query)
	if err != nil {
		return result, pages, fmt.Errorf("EntityDef.SelectEntities: failed to execute query '%s': %w", query, err)
	}
	defer rows.Close()

	for rows.Next() {

		res, err := T.Factory.CreateEntity(T)
		if err != nil {
			return result, pages, fmt.Errorf("EntityDef.SelectEntities: failed to create entity: %w", err)
		}

		fp := make([]any, 0, len(T.FieldDefs))
		for _, v := range T.FieldDefs {
			fp = append(fp, res.Values[v.Name].(any))
		}
		err = rows.Scan(fp...)
		if err != nil {
			return result, pages, fmt.Errorf("EntityDef.SelectEntities: failed to scan row: %w", err)
		}
		res.isNew = false
		T.Factory.loadedEntities.Add(res.RefString(), res)

		result = append(result, res)
	}

	if pageNo > 0 && pageSize > 0 {
		countQuery, err := getSql(true)
		if err != nil {
			return result, pages, fmt.Errorf("EntityDef.SelectEntities: failed to get count SQL query: %w", err)
		}
		countRows, err := T.Factory.DB.Query(countQuery)
		if err != nil {
			return result, pages, fmt.Errorf("EntityDef.SelectEntities: failed to execute count query '%s': %w", countQuery, err)
		}
		defer countRows.Close()
		if countRows.Next() {
			err = countRows.Scan(&pages)
			if err != nil {
				return result, pages, fmt.Errorf("EntityDef.SelectEntities: failed to scan count row: %w", err)
			}
			pages = (pages + pageSize - 1) / pageSize // calculate total pages
		} else {
			return result, pages, fmt.Errorf("EntityDef.SelectEntities: no rows returned for count query '%s'", countQuery)
		}
	}

	return result, pages, nil

}
