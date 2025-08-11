package elorm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	_ "modernc.org/sqlite"
)

func TestEntity_Save(t *testing.T) {

	ent1 := mockEntity_OrderLine()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		T       *Entity
		args    args
		wantErr bool
	}{
		{name: "insert entity", T: ent1, args: args{ctx: context.Background()}, wantErr: false},
		{name: "update entity", T: ent1, args: args{ctx: context.Background()}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.T.Save(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Entity.Save() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEntity_JSON(t *testing.T) {
	ent1 := mockEntity_OrderLine()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		T       *Entity
		args    args
		wantErr bool
	}{
		{name: "entity to json", T: ent1, args: args{ctx: context.Background()}, wantErr: false},
	}
	for _, tt := range tests {
		buf := bytes.NewBuffer(nil)
		t.Run(tt.name, func(t *testing.T) {
			err := json.NewEncoder(buf).Encode(tt.T)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entity.JSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
		t.Run(tt.name+" (deserialization)", func(t *testing.T) {

			ent2 := mockEntity_OrderLine()

			err := json.NewDecoder(buf).Decode(ent2)
			if (err != nil) != tt.wantErr {
				t.Errorf("Entity.JSON() error = %v, wantErr %v", err, tt.wantErr)
			}

			ent3 := mockEntity_OrderLine()

			err = ent2.LoadFrom(ent3, true)
			if err != nil {
				t.Errorf("Entity.LoadFrom() error = %v", err)
			}

			err = ent2.LoadFrom(ent3, false)
			if err != nil {
				t.Errorf("Entity.LoadFrom() error = %v", err)
			}

			nbr := mockEntityDef_Orders(mockFactory()).FieldDefByName("OrderNbr")
			oldNbr := ent2.Values[nbr.Name].(*FieldValueString).v
			ent3.Values[nbr.Name].(*FieldValueString).Set(oldNbr + "_updated")

			err = ent3.Save(context.Background())
			if err != nil {
				t.Errorf("Entity.Save() error = %v, wantErr %v", err, tt.wantErr)
			}

			ent2, err = ent3.Factory.LoadEntity(ent2.RefString())
			if err != nil {
				t.Errorf("Entity.LoadEntity() error = %v", err)
			}

			_ = ent2

		})
	}
}

func TestSelectEntities(t *testing.T) {

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

	const tstcnt = 50
	for i := 0; i < tstcnt; i++ {
		ent1 := mockEntity_OrderLine()
		ent1.Values["OrderNbr"].(*FieldValueString).Set(fmt.Sprintf("OrderNbr_%d", i))
		err = ent1.Save(context.Background())
		if err != nil {
			t.Errorf("Entity.Save() error = %v", err)
		}
	}

	Nbr := ordersDef.FieldDefByName("OrderNbr")

	allLines, _, err = ordersDef.SelectEntities([]*Filter{
		AddFilterEQ(Nbr, "ww"),
		AddFilterNOEQ(Nbr, "qq"),
	}, nil, 10, 1)
	if err != nil {
		t.Errorf("SelectEntities() error = %v", err)
	}

	_ = allLines

}
