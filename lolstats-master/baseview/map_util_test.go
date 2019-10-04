package baseview

import (
	"fmt"
	"testing"

	"gopkg.in/tylerb/is.v1"
)

func TestAreas(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		expected int64
		pos      Position
	}{
		{AreaRedTopLane, Position{X: 2000, Y: 13500}},
		{AreaRedTopLane, Position{X: 7000, Y: 13500}},
		{AreaRedBase, Position{X: 12000, Y: 13500}},
		{AreaBlueTopJungle, Position{X: 2000, Y: 8000}},
		{AreaRedMidLane, Position{X: 7000, Y: 8000}},
		{AreaRedBotJungle, Position{X: 12000, Y: 8000}},
		{AreaBlueBase, Position{X: 2000, Y: 3000}},
		{AreaBlueBotJungle, Position{X: 7000, Y: 3000}},
		{AreaRedBotLane, Position{X: 12000, Y: 3000}},
		{AreaBlueBase, Position{X: 2000, Y: 1500}},
		{AreaBlueBotLane, Position{X: 7000, Y: 1500}},
		{AreaBlueBotLane, Position{X: 12000, Y: 1500}},
	}

	for i, c := range cases {
		is.Msg(fmt.Sprintf("case %d %v", i+1, c.pos)).Equal(c.expected, AreaID(c.pos))
	}
}

func TestBlue(t *testing.T) {
	is := is.New(t)

	cases := []struct {
		expected bool
		pos      Position
		msg      string
	}{
		{true, Position{X: 10696.51, Y: 1594.512, Z: 49.3109}, "blue bot lane"},
		{false, Position{X: 13124, Y: 2506, Z: 51.36691}, "bot lane (edge)"},
		{true, Position{X: 9786, Y: 1476, Z: 51.32457}, "blue bot lane"},
		{false, Position{X: 13150, Y: 6040, Z: 55.46784}, "red bot lane"},
		{true, Position{X: 7713.833, Y: 5215.289, Z: 48.56312}, "blue bot jungle"},
		{true, Position{X: 4171.234, Y: 4620.626, Z: 56.88552}, "blue mid lane"},
		{false, Position{X: 9984, Y: 9188, Z: 51.84162}, "red mid lane"},
	}

	for i, c := range cases {
		is.Msg(fmt.Sprintf("case %d: %s", i+1, c.msg)).Equal(c.expected, isBlue(c.pos))
	}
}
