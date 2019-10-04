package gcd

import (
	"reflect"
	"strings"
	"testing"
)

func TestJsonEqualsDatastore(t *testing.T) {
	row := HistoryRow{}
	s := reflect.TypeOf(row)
	for i := 0; i < s.NumField(); i++ {
		field := s.Field(i)

		jsonTag := field.Tag.Get("json")
		datastoreTag := field.Tag.Get("datastore")

		jsonTagName := strings.Split(jsonTag, ",")[0]
		datastoreName := strings.Split(datastoreTag, ",")[0]

		// Json marshal and datastore translation follow the same rules:
		// If empty, then the field name is used
		// If "-", then omit the field

		if jsonTagName != datastoreName {
			t.Error("Field " + field.Name + " has mismatched json and datastore tags")
		}
	}
}
