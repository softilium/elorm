package elorm

import (
	"context"
	"fmt"
	"os"
	"testing"
)

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

	nbr, _ := orderdef.AddStringFieldDef("OrderNbr", 255)
	_, _ = orderdef.AddNumericFieldDef("OrderQty", 10, 2)
	_, _ = orderdef.AddDateTimeFieldDef("OrderDate")
	_, _ = orderdef.AddBoolFieldDef("OrderApproved")
	_, _ = orderdef.AddIntFieldDef("OrderStatus")

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
	_, _ = orderlinedef.AddNumericFieldDef("LineQty", 10, 2)
	_, _ = orderlinedef.AddStringFieldDef("LineDescription", 255)
	_, _ = orderlinedef.AddIntFieldDef("LineDescription")
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

func mock_ClearEntities(t *testing.T) {
	factory := mockFactory()
	linesDef := mockEntityDef_Orders(factory)
	ordersDef := mockEntityDef_Orders(factory)

	allLines, _, err := linesDef.SelectEntities(nil, nil, 0, 0)
	if err != nil {
		t.Errorf("SelectEntities() error = %v", err)
	}
	for _, line := range allLines {
		err = factory.DeleteEntity(context.Background(), line.RefString())
		if err != nil {
			t.Errorf("DeleteEntity() error = %v", err)
		}
	}

	allOrders, _, err := ordersDef.SelectEntities(nil, nil, 0, 0)
	if err != nil {
		t.Errorf("SelectEntities() error = %v", err)
	}
	for _, order := range allOrders {
		err = factory.DeleteEntity(context.Background(), order.RefString())
		if err != nil {
			t.Errorf("DeleteEntity() error = %v", err)
		}
	}

}

const seedCount = 50

func mock_SeedEntities(t *testing.T) []string {
	result := make([]string, seedCount)
	for i := range seedCount {
		ent1 := mockEntity_OrderLine()
		ent1.Values["OrderNbr"].(*FieldValueString).Set(fmt.Sprintf("OrderNbr_%d", i))
		err := ent1.Save(context.Background())
		if err != nil {
			t.Errorf("Entity.Save() error = %v", err)
		}
		result[i] = ent1.RefString()
	}
	return result
}
