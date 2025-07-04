package elorm

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

type IEntity interface {
	RefString() string
	IsNew() bool
	Save() error
	LoadFrom(src IEntity, predefinedFields bool) error
}

type Entity struct {
	Factory   *Factory
	entityDef *EntityDef
	Values    map[string]IFieldValue

	ref         *FieldValueRef
	isDeleted   *FieldValueBool
	dataVersion *FieldValueString

	isNew bool
}

func (T *Entity) RefString() string {
	return T.ref.AsString()
}

func (T *Entity) IsNew() bool {
	return T.isNew
}

func (T *Entity) IsDeleted() bool {
	return T.isDeleted.Get()
}

func (T *Entity) SetIsDeleted(newValue bool) {
	T.isDeleted.Set(newValue)
}

func (T *Entity) DataVersion() string {
	return T.dataVersion.Get()
}

func (T *Entity) Save() error {
	var err error

	dvCheck := T.entityDef.DataVersionCheckMode
	if dvCheck == DataVersionCheckDefault {
		dvCheck = T.Factory.dataVersionCheckMode
	}

	if T.entityDef.BeforeSaveHandler != nil {
		if err := T.entityDef.BeforeSaveHandler(T.entityDef.Wrap(T)); err != nil {
			return fmt.Errorf("Entity.Save: beforeSaveHandler failed: %w", err)
		}
	}

	tableName, err := T.entityDef.SqlTableName()
	if err != nil {
		return fmt.Errorf("Entity.Save: failed to get SQL table name for entity %s: %w", T.entityDef.ObjectName, err)
	}

	if T.RefString() == "" {
		return fmt.Errorf("Entity.Save: cannot save entity with empty Ref field")
	}

	fieldCount := len(T.entityDef.FieldDefs)
	fn := make([]string, 0, fieldCount)

	// Pre-compute all column names outside the loop for better performance
	columnNames := make(map[string]string, fieldCount)
	for _, v := range T.entityDef.FieldDefs {
		coln, err := v.SqlColumnName()
		if err != nil {
			return fmt.Errorf("Entity.Save: failed to get SQL column name for field %s: %w", v.Name, err)
		}
		columnNames[v.Name] = coln
		fn = append(fn, coln)
	}

	tx, err := T.Factory.DB.Begin()
	if err != nil {
		return fmt.Errorf("Entity.Save: failed to begin transaction: %w", err)
	}
	defer tx.Rollback() // rollback transaction if not committed

	if T.isNew {

		if dvCheck == DataVersionCheckAlways {
			T.dataVersion.Set(NewRef())
		}

		fv := make([]string, 0, fieldCount)
		for _, v := range T.entityDef.FieldDefs {
			sqlv, err := T.Values[v.Name].SqlStringValue()
			if err != nil {
				return fmt.Errorf("Entity.Save: failed to get SQL value for field %s: %w", v.Name, err)
			}
			fv = append(fv, sqlv)
		}

		sql := fmt.Sprintf(`insert into %s (%s) values (%s)`,
			tableName, strings.Join(fn, ", "), strings.Join(fv, ", "))
		_, err = tx.Exec(sql)
		if err != nil {
			return fmt.Errorf("Entity.Save: failed to insert: %w", err)
		}

	} else {
		refsv, err := T.ref.SqlStringValue()
		if err != nil {
			return fmt.Errorf("Entity.Save: failed to get SQL value for Ref field: %w", err)
		}
		setlist := make([]string, 0, fieldCount)
		for _, v := range T.entityDef.FieldDefs {
			coln := columnNames[v.Name] // Use pre-computed column name

			sv, err := T.Values[v.Name].SqlStringValue()
			if err != nil {
				return fmt.Errorf("Entity.Save: failed to get SQL value for field %s: %w", v.Name, err)
			}
			setlist = append(setlist, fmt.Sprintf("%s = %s", coln, sv))
		}

		if dvCheck == DataVersionCheckAlways {
			oldDV := T.DataVersion()
			T.dataVersion.Set(NewRef())

			sql := fmt.Sprintf(`update %s set %s where ref=%s and dataversion='%s'`,
				tableName, strings.Join(setlist, ", "), refsv, oldDV)
			res, err := tx.Exec(sql)
			if err != nil {
				T.dataVersion.Set(oldDV)
				return fmt.Errorf("Entity.Save: failed to update: %w", err)
			}
			rowsAffected, err := res.RowsAffected()
			if err != nil {
				T.dataVersion.Set(oldDV)
				return fmt.Errorf("Entity.Save: failed to get rows affected: %w", err)
			}
			if rowsAffected != 1 {
				T.dataVersion.Set(oldDV)
				return fmt.Errorf("Entity.Save: update failed, entity %s was changed by another user", T.RefString())
			}

		} else {

			sql := fmt.Sprintf(`update %s set %s where ref=%s`,
				tableName, strings.Join(setlist, ", "), refsv)
			_, err = tx.Exec(sql)
			if err != nil {
				return fmt.Errorf("Entity.Save: failed to update: %w", err)
			}
		}
	}

	err = tx.Commit()

	T.isNew = false
	if err == nil {
		for _, v := range T.Values {
			v.resetOld()
		}
	} else {
		return fmt.Errorf("Entity.Save: failed to commit transaction: %w", err)
	}

	if T.entityDef.AfterSaveHandler != nil {
		if err := T.entityDef.AfterSaveHandler(T.entityDef.Wrap(T)); err != nil {
			return fmt.Errorf("Entity.Save: afterSaveHandler failed: %w", err)
		}
	}

	return nil
}

func (T *Entity) MarshalJSON() ([]byte, error) {
	vm := make(map[string]any, len(T.Values))
	for _, v := range T.Values {
		vm[v.Def().Name] = v.AsString()
	}
	return json.Marshal(vm)
}

func (T *Entity) UnmarshalJSON(b []byte) error {

	oldRef := T.RefString()

	vm := make(map[string]any, len(T.Values))

	err := json.Unmarshal(b, &vm)
	if err != nil {
		return fmt.Errorf("Entity.UnmarshalJSON: failed to unmarshal JSON: %w", err)
	}
	for _, v := range T.Values {
		if val, ok := vm[v.Def().Name]; ok {
			switch v.Def().Type {
			case FieldDefTypeString:
				v.(*FieldValueString).Set(val.(string))
			case FieldDefTypeInt:
				switch val.(type) {
				case int:
					v.(*FieldValueInt).Set(int64(val.(int)))
				case int64:
					v.(*FieldValueInt).Set(val.(int64))
				case string:
					valInt, err := strconv.ParseInt(strings.TrimSpace(val.(string)), 10, 64)
					if err != nil {
						return fmt.Errorf("Entity.LoadFromJSON: failed to parse integer value for field %s: %w", v.Def().Name, err)
					}
					v.(*FieldValueInt).Set(valInt)
				default:
					return fmt.Errorf("Entity.LoadFromJSON: unexpected type for integer field %s: %T", v.Def().Name, val)
				}
			case FieldDefTypeBool:
				switch val.(type) {
				case bool:
					v.(*FieldValueBool).Set(val.(bool))
				case string:
					asStr := strings.ToLower(val.(string))
					v.(*FieldValueBool).Set(asStr == "true" || asStr == "1" || asStr == "yes" || asStr == "on")
				default:
					return fmt.Errorf("Entity.LoadFromJSON: unexpected type for boolean field %s: %T", v.Def().Name, val)
				}
			case FieldDefTypeRef:
				v.(*FieldValueRef).Set(val.(string))
			case FieldDefTypeDateTime:
				strVal := strings.TrimSpace(val.(string))
				tv, err := time.Parse(time.RFC3339, strVal)
				if err != nil {
					return fmt.Errorf("Entity.LoadFromJSON: failed to parse date time: %w", err)
				}
				v.(*FieldValueDateTime).Set(tv)
			case FieldDefTypeNumeric:
				switch val.(type) {
				case float32:
					v.(*FieldValueNumeric).Set(float64(val.(float32)))
				case float64:
					v.(*FieldValueNumeric).Set(val.(float64))
				case string:
					valFloat, err := strconv.ParseFloat(strings.TrimSpace(val.(string)), 64)
					if err != nil {
						return fmt.Errorf("Entity.LoadFromJSON: failed to parse numeric value for field %s: %w", v.Def().Name, err)
					}
					v.(*FieldValueNumeric).Set(valFloat)
				default:
					return fmt.Errorf("Entity.LoadFromJSON: unexpected type for numeric field %s: %T", v.Def().Name, val)
				}
			}
		}
	}

	// force to check existence of entity in database
	T.Factory.loadedEntities.Remove(oldRef)
	T.Factory.loadedEntities.Remove(T.RefString())

	existsCopy, _ := T.Factory.LoadEntity(T.RefString())
	T.isNew = existsCopy == nil

	return nil
}

func (T *Entity) LoadFrom(src IEntity, predefinedFields bool) error {
	if src == nil {
		return fmt.Errorf("Entity.LoadFrom: source entity is nil")
	}

	srcT, ok := src.(*Entity)
	if !ok {
		return fmt.Errorf("Entity.LoadFrom: source entity is not of type *EntityDef")
	}

	if T.entityDef != srcT.entityDef {
		return fmt.Errorf("Entity.LoadFrom: source entity has different definition")
	}

	for idx, v := range T.Values {

		if !predefinedFields && (v.Def().Name == RefFieldName || v.Def().Name == DataVersionFieldName) {
			continue
		}
		switch ft := T.Values[idx].(type) {

		case *FieldValueString:
			ft.v = srcT.Values[idx].(*FieldValueString).v
		case *FieldValueInt:
			ft.v = srcT.Values[idx].(*FieldValueInt).v
		case *FieldValueBool:
			ft.v = srcT.Values[idx].(*FieldValueBool).v
		case *FieldValueRef:
			ft.v = srcT.Values[idx].(*FieldValueRef).v
		case *FieldValueDateTime:
			ft.v = srcT.Values[idx].(*FieldValueDateTime).v
		case *FieldValueNumeric:
			ft.v = srcT.Values[idx].(*FieldValueNumeric).v
		}
	}

	return nil
}
