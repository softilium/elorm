package elorm

import (
	"bytes"
	"context"
	"encoding/json"
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

			ldesc := mockEntityDef_OrderLines(mockFactory()).FieldDefByName("LineDescription")
			oldNbr := ent2.Values[ldesc.Name].(*FieldValueString).v
			ent2.Values[ldesc.Name].(*FieldValueString).Set(oldNbr + "_updated")

			err = ent2.Save(context.Background())
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
	ordersDef := mockEntityDef_Orders(factory)

	mock_ClearEntities(t)

	mock_SeedEntities(t)

	var err error

	Nbr := ordersDef.FieldDefByName("OrderNbr")

	allLines, _, err := ordersDef.SelectEntities([]*Filter{
		AddFilterEQ(Nbr, "ww"),
		AddFilterNOEQ(Nbr, "qq"),
	}, nil, 10, 1)
	if err != nil {
		t.Errorf("SelectEntities() error = %v", err)
	}

	_ = allLines

}
