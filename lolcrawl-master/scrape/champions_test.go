package scrape

import (
	"testing"

	"gopkg.in/tylerb/is.v1"
)

func TestGetRecentLolVersion(t *testing.T) {
	is := is.New(t)

	var versionTests = []struct {
		in  []string
		out string
		err bool
	}{
		{[]string{"6.3.1", "5.24.2"}, "6.3.1", false},
		{[]string{"4.3.1", "5.24.2"}, "5.24.2", false},
		{[]string{"3.6.15", "3.6.14", "0.154.3"}, "3.6.15", false},
		{[]string{"6.0.1", "1.2.3", "lolpatch_5.18"}, "6.0.1", false},
		{[]string{"lolpatch_5.18", "6.0.1", "1.2.3"}, "6.0.1", false},
		{[]string{"1.2.3"}, "1.2.3", false},
		{[]string{}, "", true},
	}

	for _, vt := range versionTests {
		v, err := GetRecentLolVersion(vt.in)
		if vt.err {
			is.Err(err)
		} else {
			is.NotErr(err)
		}
		is.Equal(v, vt.out)
	}
}
