package elorm

type IFieldValue interface {
	Def() *fieldDef
	Entity() *Entity
	SqlStringValue() (string, error)
	Scan(v any) error
	AsString() string
	Set(newValue any) error
	Get() (any, error)
	GetOld() (any, error)
	resetOld()
}

type fieldValueBase struct {
	def     *fieldDef
	entity  *Entity
	isDirty bool
}

func (T *fieldValueBase) Def() *fieldDef {
	return T.def
}

func (T *fieldValueBase) Entity() *Entity {
	return T.entity
}
