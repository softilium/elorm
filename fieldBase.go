package elorm

// IFieldValue is the interface for field values in elorm.
type IFieldValue interface {
	Def() *FieldDef
	Entity() *Entity
	SqlStringValue(v ...any) (string, error)
	Scan(v any) error
	AsString() string
	resetOld()
}

type fieldValueBase struct {
	def     *FieldDef
	entity  *Entity
	isDirty bool
}

func (T *fieldValueBase) Def() *FieldDef {
	return T.def
}

func (T *fieldValueBase) Entity() *Entity {
	return T.entity
}
