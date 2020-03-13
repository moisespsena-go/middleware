package middleware

import (
	"reflect"
	"testing"
)


func TestExtensions_Copy(t *testing.T) {
	tests := []struct {
		name  string
		this  Extensions
		wantM Extensions
	}{
		{"copy0", Extensions{"a":true}, Extensions{"a":true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotM := tt.this.Copy(); !reflect.DeepEqual(gotM, tt.wantM) {
				t.Errorf("Extensions.Copy() = %v, want %v", gotM, tt.wantM)
			} else {
				tt.this["b"] = true
				if reflect.DeepEqual(gotM, tt.this) {
					t.Errorf("Extensions.Copy() same value")
				}
			}
		})
	}
}

func TestExtensions_Update(t *testing.T) {
	type args struct {
		news []Extensions
	}
	tests := []struct {
		name string
		this Extensions
		args args
		want Extensions
	}{
		{"", Extensions{}, args{[]Extensions{{"a":true}}}, Extensions{"a":true}},
		{"", Extensions{}, args{[]Extensions{{"a":true}, {"b":false}}}, Extensions{"a":true,"b":false}},
		{"", Extensions{}, args{[]Extensions{{"a":true}, {"b":false},{"a":false}}}, Extensions{"a":false,"b":false}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.this.Update(tt.args.news...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Extensions.Update() = %v, want %v", got, tt.want)
			} else if reflect.DeepEqual(got, tt.this) {
				t.Errorf("Extensions.Update() = %v, want %v: same", got, tt.this)
			}
		})
	}
}

func TestExtensions_PtrUpdate(t *testing.T) {
	type args struct {
		news []Extensions
	}
	tests := []struct {
		name string
		this Extensions
		args args
		want Extensions
	}{
		{"", Extensions{}, args{[]Extensions{{"a":true}}}, Extensions{"a":true}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.this.PtrUpdate(tt.args.news...); !reflect.DeepEqual(got, &tt.want) {
				t.Errorf("Extensions.Update() = %v, want %v", got, tt.want)
			} else if !reflect.DeepEqual(got, &tt.this) {
				t.Errorf("Extensions.Update() = %v, want %v: not same", got, &tt.this)
			}
		})
	}
}