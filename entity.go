package elorm

import (
	"fmt"
	"strings"
)

type IReferableEntity interface {
	RefString() string
	IsNew() bool
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

			refsv, err := T.ref.SqlStringValue()
			if err != nil {
				return fmt.Errorf("Entity.Save: failed to get SQL value for Ref field: %w", err)
			}

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
				tableName, strings.Join(setlist, ", "), T.RefString())
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
