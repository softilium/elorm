package elorm

import "testing"

func TestEntityDef_AddIndex(t *testing.T) {

	f := mockFactory()
	od := mockEntityDef_Orders(f)
	ld := mockEntityDef_OrderLines(f)
	nbr := od.FieldDefByName("OrderNbr")

	err := od.AddIndex(true)
	if err == nil {
		t.Error("Expected error when adding index without fields")
	}

	err = od.AddIndex(true, od.RefField)
	if err == nil {
		t.Error("Expected error when adding index without fields")
	}

	err = ld.AddIndex(true, ld.RefField, nbr)
	if err == nil {
		t.Error("Expected error when adding index without fields")
	}

	od.IndexDefs = nil // Reset index definitions for the test

	err = od.AddIndex(true, nbr)
	if err != nil {
		t.Errorf("AddIndex() error = %v", err)
	}

	err = od.AddIndex(true, nbr)
	if err == nil {
		t.Error("Expected error when adding the same index again")
	}

}
