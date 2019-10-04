package baseview

import (
	"testing"

	"gopkg.in/tylerb/is.v1"
)

func TestPositionDistance(t *testing.T) {
	is := is.New(t)

	pos := Position{X: 5000, Y: 5000, Z: 100}
	is.Equal(0.0, pos.Distance(Position{X: 5000, Y: 5000, Z: 100}))
	is.Equal(0.0, pos.DistanceXY(Position{X: 5000, Y: 5000, Z: 100}))
	is.Equal(100.0, pos.Distance(Position{X: 5000, Y: 5000, Z: 0}))
	is.Equal(0.0, pos.DistanceXY(Position{X: 5000, Y: 5000, Z: 0}))
	is.Equal(5000.0, pos.Distance(Position{X: 2000, Y: 1000, Z: 100}))
	is.Equal(5000.0, pos.DistanceXY(Position{X: 2000, Y: 1000, Z: 100}))
	is.Equal(5000.0, pos.Distance(Position{X: 1000, Y: 8000, Z: 100}))
	is.Equal(5000.0, pos.DistanceXY(Position{X: 1000, Y: 8000, Z: 100}))
	is.Equal(5000.0, pos.Distance(Position{X: 5000, Y: 8000, Z: -3900}))
	is.Equal(3000.0, pos.DistanceXY(Position{X: 5000, Y: 8000, Z: -3900}))
}
