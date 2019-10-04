package baseview

import (
	"testing"

	"gopkg.in/tylerb/is.v1"
)

func TestTeamID(t *testing.T) {
	is := is.New(t)

	for i := -2; i < 12; i++ {
		var expected int64
		if i <= 5 && i >= 1 {
			expected = 100
		} else if i <= 10 && i >= 6 {
			expected = 200
		}

		is.Msg("failure when i = %d", i).Equal(expected, TeamID(int64(i)))
	}
}
