package elorm

import (
	"testing"
)

func TestFieldValueNumeric_Scan(t *testing.T) {
	type args struct {
		v any
	}
	tests := []struct {
		name    string
		T       *FieldValueNumeric
		args    args
		wantErr bool
		wantV   float64
	}{
		{name: "valid float64", T: mockFieldValueNumeric(), args: args{v: 123.456}, wantErr: false, wantV: 123.46},
		{name: "invalid string", T: mockFieldValueNumeric(), args: args{v: "not a float"}, wantErr: true, wantV: 7.62},
		{name: "valid string", T: mockFieldValueNumeric(), args: args{v: "123.123"}, wantErr: false, wantV: 123.12},
		{name: "invalid type (bool)", T: mockFieldValueNumeric(), args: args{v: true}, wantErr: true, wantV: 7.62},
		{name: "nil value", T: mockFieldValueNumeric(), args: args{v: nil}, wantErr: true, wantV: 7.62},
		{name: "zero value", T: mockFieldValueNumeric(), args: args{v: 0.0}, wantErr: false, wantV: 0.0},
		{name: "[]uint8 valid value", T: mockFieldValueNumeric(), args: args{v: []uint8("123.45")}, wantErr: false, wantV: 123.45},
		{name: "[]uint8 invalid value", T: mockFieldValueNumeric(), args: args{v: []uint8("not a float")}, wantErr: true, wantV: 7.62},
		{name: "int64", T: mockFieldValueNumeric(), args: args{v: int64(12345)}, wantErr: false, wantV: 12345.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.T.Scan(tt.args.v)
			if (err != nil) != tt.wantErr {
				t.Errorf("FieldValueNumeric.Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.T.v != tt.wantV {
				t.Errorf("FieldValueNumeric.Scan() v = %v, want %v", tt.T.v, tt.wantV)
			}
		})
	}
}

func TestFieldValueNumeric_SqlStringValue(t *testing.T) {
	tests := []struct {
		name    string
		T       *FieldValueNumeric
		args    []any
		want    string
		wantErr bool
	}{
		{name: "default value", T: mockFieldValueNumeric(), args: nil, want: "7.62", wantErr: false},
		{name: "custom value", T: mockFieldValueNumeric(), args: []any{123.456}, want: "123.46", wantErr: false},
		{name: "invalid type", T: mockFieldValueNumeric(), args: []any{"not a float"}, want: "", wantErr: true},
		{name: "empty args", T: mockFieldValueNumeric(), args: []any{}, want: "7.62", wantErr: false},
		{name: "nil args", T: mockFieldValueNumeric(), args: nil, want: "7.62", wantErr: false},
		{name: "zero value", T: mockFieldValueNumeric(), args: []any{0.0}, want: "0.00", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.T.SqlStringValue(tt.args...)
			if (err != nil) != tt.wantErr {
				t.Errorf("FieldValueNumeric.SqlStringValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("FieldValueNumeric.SqlStringValue() = %v, want %v", got, tt.want)
			}
		})
	}
}
