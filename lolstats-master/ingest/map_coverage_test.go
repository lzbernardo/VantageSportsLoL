package ingest

import (
	"testing"

	"gopkg.in/tylerb/is.v1"

	"github.com/VantageSports/lolstats/baseview"
)

func TestMapCoverage(t *testing.T) {
	is := is.New(t)

	mcSmall := NewCoverageMap(1)
	mcBig := NewCoverageMap(12)

	positionTimes := []PositionTime{
		{Seconds: 500, Position: baseview.Position{X: 500, Y: 500}},      // early
		{Seconds: 500, Position: baseview.Position{X: 500, Y: 500}},      // early (same)
		{Seconds: 500, Position: baseview.Position{X: 500, Y: 500}},      // early (same)
		{Seconds: 1000, Position: baseview.Position{X: 500, Y: 10000}},   // mid
		{Seconds: 1500, Position: baseview.Position{X: 10000, Y: 500}},   // late
		{Seconds: 2000, Position: baseview.Position{X: 10000, Y: 10000}}, // late
	}

	for _, p := range positionTimes {
		mcSmall.Add(p.Seconds, p.Position)
		mcBig.Add(p.Seconds, p.Position)
	}

	smallPercents := mcSmall.Percents()
	is.Equal(100, smallPercents["full"])
	is.Equal(100, smallPercents["early"])
	is.Equal(100, smallPercents["late"])

	bigPercents := mcBig.Percents()
	is.Equal(100*4.0/144.0, bigPercents["full"])
	is.Equal(100*1.0/144, bigPercents["early"])
	is.Equal(100*2.0/144, bigPercents["late"])
}
