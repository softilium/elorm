package elorm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// IEntity is the interface for entities in elorm.
type IEntity interface {
	RefString() string
	IsNew() bool
	IsDeleted() bool
	Save(ctx context.Context) error
	LoadFrom(src IEntity, predefinedFields bool) error
	GetValues() map[string]IFieldValue
	Def() *EntityDef
}

// Entity represents a database entity in elorm.
type Entity struct {
	Factory   *Factory
	entityDef *EntityDef
	Values    map[string]IFieldValue

	ref         *FieldValueRef
	isDeleted   *FieldValueBool
	dataVersion *FieldValueString

	isNew bool
}

// Def returns the entity definition for this entity.
func (T *Entity) Def() *EntityDef {
	return T.entityDef
}

// GetValues returns a map of all field values for this entity.
func (T *Entity) GetValues() map[string]IFieldValue {
	return T.Values
}

// RefString returns the string representation of this entity's reference.
func (T *Entity) RefString() string {
	return T.ref.AsString()
}

// IsNew returns true if this entity has not been saved to the database yet.
func (T *Entity) IsNew() bool {
	return T.isNew
}

// IsDeleted returns true if this entity is marked for deletion.
func (T *Entity) IsDeleted() bool {
	return T.isDeleted.Get()
}

// SetIsDeleted sets the deletion status of this entity.
func (T *Entity) SetIsDeleted(newValue bool) {
	T.isDeleted.Set(newValue)
}

// DataVersion returns the current data version of this entity.
func (T *Entity) DataVersion() string {
	return T.dataVersion.Get()
}

// Save persists the Entity to the database. It handles both insert and update operations
// depending on whether the entity is new or existing. The method performs the following steps:
//   - Executes the BeforeSaveHandler if defined.
//   - Begins a database transaction.
//   - If the entity is new, inserts a new record into the database.
//   - If the entity exists, updates the corresponding record, optionally performing
//     data version checks to prevent concurrent modifications.
//   - Commits the transaction if all operations succeed, or rolls back on error.
//   - Executes the AfterSaveHandler if defined.
//
// Returns an error if any step fails, including handler execution, SQL operations,
// or transaction management.
func (T *Entity) Save(ctx context.Context) error {
	var err error

	if !T.Def().UseSoftDelete && T.IsDeleted() {
		return fmt.Errorf("Entity.Save: cannot save entity with IsDeleted=true, UseSoftDelete is false")
	}

	dvCheck := T.entityDef.DataVersionCheckMode
	if dvCheck == DataVersionCheckDefault {
		dvCheck = T.Factory.dataVersionCheckMode
	}

	// before save handlers
	for _, hndl := range T.entityDef.beforeSaveHandlerByRefs {
		if err := hndl(ctx, T.RefString()); err != nil {
			return fmt.Errorf("Entity.Save: beforeSaveHandlerByRef failed for ref %s: %w", T.ref.AsString(), err)
		}
	}
	for _, hndl := range T.entityDef.beforeSaveHandlers {
		var err error
		if T.entityDef.Wrap == nil {
			err = hndl(ctx, T.entityDef)
		} else {
			err = hndl(ctx, T.entityDef.Wrap(T))
		}
		if err != nil {
			return fmt.Errorf("Entity.Save: beforeSaveHandler failed for ref %s: %w", T.ref.AsString(), err)
		}
	}

	T.Factory.loadsaveLock.Lock()
	defer T.Factory.loadsaveLock.Unlock()

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

	tx, err := T.Factory.BeginTran()
	if err != nil {
		return fmt.Errorf("Entity.Save: failed to begin transaction: %w", err)
	}

	if T.isNew {

		T.dataVersion.Set(NewRef())

		fv := make([]string, 0, fieldCount)
		for _, v := range T.entityDef.FieldDefs {
			sqlv, err := T.Values[v.Name].SqlStringValue()
			if err != nil {
				_ = T.Factory.RollbackTran(tx)
				return fmt.Errorf("Entity.Save: failed to get SQL value for field %s: %w", v.Name, err)
			}
			fv = append(fv, sqlv)
		}

		sql := fmt.Sprintf(`insert into %s (%s) values (%s)`,
			tableName, strings.Join(fn, ", "), strings.Join(fv, ", "))
		_, err = tx.Exec(sql)
		if err != nil {
			_ = T.Factory.RollbackTran(tx)
			return fmt.Errorf("Entity.Save: failed to insert: %w", err)
		}

	} else {

		oldDV := T.DataVersion()
		T.dataVersion.Set(NewRef())

		refsv, err := T.ref.SqlStringValue()
		if err != nil {
			_ = T.Factory.RollbackTran(tx)
			return fmt.Errorf("Entity.Save: failed to get SQL value for Ref field: %w", err)
		}
		setlist := make([]string, 0, fieldCount)
		for _, v := range T.entityDef.FieldDefs {
			coln := columnNames[v.Name] // Use pre-computed column name

			sv, err := T.Values[v.Name].SqlStringValue()
			if err != nil {
				_ = T.Factory.RollbackTran(tx)
				return fmt.Errorf("Entity.Save: failed to get SQL value for field %s: %w", v.Name, err)
			}
			setlist = append(setlist, fmt.Sprintf("%s = %s", coln, sv))
		}

		if dvCheck == DataVersionCheckAlways {

			sql := fmt.Sprintf(`update %s set %s where ref=%s and dataversion='%s'`,
				tableName, strings.Join(setlist, ", "), refsv, oldDV)
			res, err := tx.Exec(sql)
			if err != nil {
				T.dataVersion.Set(oldDV)
				_ = T.Factory.RollbackTran(tx)
				return fmt.Errorf("Entity.Save: failed to update: %w", err)
			}
			rowsAffected, err := res.RowsAffected()
			if err != nil {
				T.dataVersion.Set(oldDV)
				_ = T.Factory.RollbackTran(tx)
				return fmt.Errorf("Entity.Save: failed to get rows affected: %w", err)
			}
			if rowsAffected != 1 {
				T.dataVersion.Set(oldDV)
				_ = T.Factory.RollbackTran(tx)
				return fmt.Errorf("Entity.Save: update failed, entity %s was changed by another user", T.RefString())
			}

		} else {

			sql := fmt.Sprintf(`update %s set %s where ref=%s`,
				tableName, strings.Join(setlist, ", "), refsv)
			_, err = tx.Exec(sql)
			if err != nil {
				_ = T.Factory.RollbackTran(tx)
				return fmt.Errorf("Entity.Save: failed to update: %w", err)
			}
		}
	}

	err = T.Factory.CommitTran(tx)
	if err != nil {
		return fmt.Errorf("Entity.Save: failed to commit transaction: %w", err)
	}

	// after save handlers
	for _, handler := range T.entityDef.afterSaveHandlers {
		if err := handler(ctx, T.entityDef.Wrap(T)); err != nil {
			return fmt.Errorf("Entity.Save: afterSaveHandler failed for ref %s: %w", T.RefString(), err)
		}
	}

	for _, v := range T.Values {
		v.resetOld()
	}
	T.isNew = false
	T.Factory.loadedEntities.Add(T.RefString(), T)
	return nil
}

func (T *Entity) valuesToMap(defs map[*FieldDef]bool) (map[string]any, error) {
	vm := make(map[string]any, len(T.Values))
	for _, v := range T.Values {
		if len(defs) > 0 {
			if _, ok := defs[v.Def()]; !ok {
				continue
			}
		}

		switch vt := v.(type) {
		case *FieldValueString:
			vm[v.Def().Name] = vt.v
		case *FieldValueInt:
			vm[v.Def().Name] = vt.v
		case *FieldValueBool:
			vm[v.Def().Name] = vt.v
		case *FieldValueRef:
			ok, def := T.Factory.IsRef(vt.v)
			if !ok {
				vm[v.Def().Name] = vt.v
			} else {
				if len(def.AutoExpandFieldsForJSON) > 0 && vt.def.Name != RefFieldName {
					entity, err := T.Factory.LoadEntity(vt.v)
					if err != nil {
						return nil, fmt.Errorf("Entity.MarshalJSON: failed to load entity for (entity type=%s, ref=%s): %w", vt.def.Name, vt.v, err)
					}
					vm2, err := entity.valuesToMap(def.AutoExpandFieldsForJSON)
					if err != nil {
						return nil, fmt.Errorf("Entity.MarshalJSON: failed to convert entity to map for ref %s: %w", vt.v, err)
					}
					vm[v.Def().Name] = vm2
				} else {
					vm[v.Def().Name] = vt.v
				}
			}
		case *FieldValueDateTime:
			vm[v.Def().Name] = vt.v.Format(v.Def().DateTimeJSONFormat)
		case *FieldValueNumeric:
			vm[v.Def().Name] = vt.v
		default:
			return nil, fmt.Errorf("Entity.MarshalJSON: unsupported field type %d for field %s", v.Def().Type, v.Def().Name)
		}
	}
	return vm, nil
}

// MarshalJSON implements json.Marshaler interface for JSON serialization.
func (T *Entity) MarshalJSON() ([]byte, error) {
	vm, err := T.valuesToMap(nil)
	if err != nil {
		return nil, fmt.Errorf("Entity.MarshalJSON: failed to convert values to map: %w", err)
	}
	return json.Marshal(vm)
}

// UnmarshalJSON implements json.Unmarshaler interface for JSON deserialization.
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
				case float64:
					v.(*FieldValueInt).Set(int64(val.(float64)))
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
				stringVal := ""
				switch vt := val.(type) {
				case string:
					stringVal = vt
				case map[string]any:
					if ref, ok := vt[RefFieldName]; ok {
						stringVal, ok = ref.(string)
						if !ok {
							return fmt.Errorf("Entity.LoadFromJSON: expected string for reference field %s, got %T", v.Def().Name, ref)
						}
					} else {
						return fmt.Errorf("Entity.LoadFromJSON: missing reference field %s in map", RefFieldName)
					}
				}
				err = v.(*FieldValueRef).Set(stringVal)
				if err != nil {
					return fmt.Errorf("Entity.LoadFromJSON: failed to set reference field %s: %w", v.Def().Name, err)
				}
			case FieldDefTypeDateTime:
				strVal := strings.TrimSpace(val.(string))
				tv, err := time.Parse(v.Def().DateTimeJSONFormat, strVal)
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

// LoadFrom copies field values from another entity into this entity.
func (T *Entity) LoadFrom(src IEntity, predefinedFields bool) error {
	if src == nil {
		return fmt.Errorf("Entity.LoadFrom: source entity is nil")
	}

	if T.entityDef != src.Def() {
		return fmt.Errorf("Entity.LoadFrom: source entity has different definition")
	}

	vals := src.GetValues()

	for idx, v := range vals {

		if !predefinedFields && (v.Def().Name == RefFieldName || v.Def().Name == DataVersionFieldName) {
			continue
		}
		switch ft := T.Values[idx].(type) {

		case *FieldValueString:
			ft.v = vals[idx].(*FieldValueString).v
		case *FieldValueInt:
			ft.v = vals[idx].(*FieldValueInt).v
		case *FieldValueBool:
			ft.v = vals[idx].(*FieldValueBool).v
		case *FieldValueRef:
			ft.v = vals[idx].(*FieldValueRef).v
		case *FieldValueDateTime:
			ft.v = vals[idx].(*FieldValueDateTime).v
		case *FieldValueNumeric:
			ft.v = vals[idx].(*FieldValueNumeric).v
		}
	}

	return nil
}
