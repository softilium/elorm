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

func AddFilterEQ(leftField *FieldDef, rightValue any) *Filter {
	return &Filter{
		Op:      FilterEQ,
		LeftOp:  leftField,
		RightOp: rightValue,
	}
}

func AddFilterLIKE(leftField *FieldDef, rightValue string) *Filter {

	if leftField.Type != fieldDefTypeString {
		return nil
	}
	return &Filter{
		Op:      FilterLIKE,
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

func (T *Filter) renderWhereClause() string {
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
	case FilterLIKE:
		if T.LeftOp != nil && T.RightOp != nil {
			colname, err := T.LeftOp.SqlColumnName()
			if err != nil {
				return fmt.Sprintf("Error: %s", err.Error())
			}
			return fmt.Sprintf("%s LIKE '%%%v%%'", colname, T.RightOp)
		}
	case FilterAndGroup, FilterOrGroup:
		renderOps := make(map[int]string)
		renderOps[FilterAndGroup] = " and "
		renderOps[FilterOrGroup] = " or "

		results := make([]string, len(T.Childs))
		for i, v := range T.Childs {
			results[i] = v.renderWhereClause()
		}
		return fmt.Sprintf("(%s)", strings.Join(results, renderOps[T.Op]))

	default:
		return "NOT IMPLEMENTED YET"
	}
	return ""
}

type SortItem struct {
	Field *FieldDef
	Asc   bool
}

func (T *EntityDef) SelectEntities(filters []*Filter, sorts []*SortItem) ([]*Entity, error) {

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
			query += f.renderWhereClause()
		}
	}

	if len(sorts) > 0 {
		query += " order by "
		sortClauses := make([]string, 0, len(sorts))
		for _, s := range sorts {
			if s.Field == nil {
				continue
			}
			coln, err := s.Field.SqlColumnName()
			if err != nil {
				return nil, fmt.Errorf("EntityDef.SelectEntities: failed to get SQL column name for field %s: %w", s.Field.Name, err)
			}
			if s.Asc {
				sortClauses = append(sortClauses, fmt.Sprintf("%s asc", coln))
			} else {
				sortClauses = append(sortClauses, fmt.Sprintf("%s desc", coln))
			}
		}
		query += strings.Join(sortClauses, ", ")
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
