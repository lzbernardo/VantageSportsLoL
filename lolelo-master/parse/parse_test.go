package parse

import (
	"testing"

	"gopkg.in/tylerb/is.v1"
)

func TestGetVal(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		expected interface{}
		input    string
		err      bool
	}{
		{1.2, "1.2", false},
		{12, "12", false},
		{"12", `"12"`, false},
		{-1.004, "-1.004", false},
		{"1 2 3", "1 2 3", false},
		{-12, "-12", false},
		{"one two", "one two", false},
		{true, "true", false},
		{false, "False", false},
		{"False Prophet", "False Prophet", false},
		{"False Prophet", "\"False Prophet\"", false},
		{map[string]interface{}{"x": 1.0, "y": 2.1, "z": -3.4}, "X:1 Y:2.1 Z:-3.4", false},
	}

	for i, c := range cases {
		caseIs := is.Msg("case %d", i+1)
		actual, err := getVal(c.input)
		if c.err {
			caseIs.Err(err)
			continue
		}

		caseIs.NotErr(err)
		caseIs.Equal(c.expected, actual)
	}
}
