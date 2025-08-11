package elorm

import "os"

func mockFieldValueNumeric() *FieldValueNumeric {
	return &FieldValueNumeric{
		fieldValueBase: fieldValueBase{
			def: &FieldDef{
				Name:      "NumericField",
				Precision: 10,
				Scale:     2,
				EntityDef: &EntityDef{
					Factory: &Factory{},
				},
			},
		},
		v:   7.62,
		old: 3.14,
	}
}

func mockFieldValueString() *FieldValueString {
	return &FieldValueString{
		fieldValueBase: fieldValueBase{
			def: &FieldDef{
				Name: "StringField",
				EntityDef: &EntityDef{
					Factory: &Factory{},
				},
			},
		},
	}
}

var mckFactory *Factory

func mockFactory() *Factory {

	if mckFactory != nil {
		return mckFactory
	}

	err := os.Remove("tests.db")
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}

	result, err := CreateFactory("sqlite", "file:tests.db")
	if err != nil {
		panic(err)
	}
	result.EntityDefs = append(result.EntityDefs, mockEntityDef_Orders(result))
	result.EntityDefs = append(result.EntityDefs, mockLinesEntityDef_Order(result))

	err = result.EnsureDBStructure()
	if err != nil {
		panic(err)
	}

	mckFactory = result
	return result
}

var mckEntityDef_Orders *EntityDef

func mockEntityDef_Orders(factory *Factory) *EntityDef {

	if mckEntityDef_Orders != nil {
		return mckEntityDef_Orders
	}

	orderdef, err := factory.CreateEntityDef("TestOrder", "TestOrders")
	if err != nil {
		panic(err)
	}

	nbr, _ := orderdef.AddStringFieldDef("OrderNbr", 255, "")
	_, _ = orderdef.AddNumericFieldDef("OrderQty", 10, 2, 0.0)
	_, _ = orderdef.AddDateTimeFieldDef("OrderDate")
	_, _ = orderdef.AddBoolFieldDef("OrderApproved", false)
	_, _ = orderdef.AddIntFieldDef("OrderStatus", 0)

	orderdef.IndexDefs = append(orderdef.IndexDefs, &IndexDef{
		Unique:    true,
		FieldDefs: []*FieldDef{nbr},
	})

	mckEntityDef_Orders = orderdef
	return orderdef
}

var mckLinesEntityDef_Order *EntityDef

func mockLinesEntityDef_Order(factory *Factory) *EntityDef {

	if mckLinesEntityDef_Order != nil {
		return mckLinesEntityDef_Order
	}

	orderdef := mockEntityDef_Orders(factory)

	orderlinedef, err := factory.CreateEntityDef("TestOrderLine", "TestOrderLines")
	if err != nil {
		panic(err)
	}

	ownerf, _ := orderlinedef.AddRefFieldDef("OrderRef", orderdef)
	_, _ = orderlinedef.AddNumericFieldDef("LineQty", 10, 2, 0.0)
	_, _ = orderlinedef.AddStringFieldDef("LineDescription", 255, "")
	_, _ = orderlinedef.AddIntFieldDef("LineDescription", 0)
	orderlinedef.AutoExpandFieldsForJSON = map[*FieldDef]bool{ownerf: true}

	mckLinesEntityDef_Order = orderlinedef
	return orderlinedef

}

func mockEntity_OrderLine() *Entity {

	factory := mockFactory()
	def := factory.EntityDefs[1]

	result, err := factory.CreateEntity(def)
	if err != nil {
		panic(err)
	}

	return result
}
