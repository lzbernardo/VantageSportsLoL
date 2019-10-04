package baseview

import "math"

type Ward struct {
	wardType   string
	sightRange float64
	concurrent float64
}

func (w Ward) Type() string           { return w.wardType }
func (w Ward) SightRange() float64    { return w.sightRange }
func (w Ward) MaxConcurrent() float64 { return w.concurrent }

var (
	BlueWard   = Ward{wardType: "blue", sightRange: 500.0, concurrent: math.Inf(1)}
	PinkWard   = Ward{wardType: "pink", sightRange: 1100.0, concurrent: 1.0}
	YellowWard = Ward{wardType: "yellow", sightRange: 1100.0, concurrent: 3.0}
)

type WardItemType string

const (
	BlueTrinket   WardItemType = "BLUE_TRINKET"
	SightWard     WardItemType = "SIGHT_WARD"
	YellowTrinket WardItemType = "YELLOW_TRINKET"
	VisionWard    WardItemType = "VISION_WARD"
)

func (w WardItemType) Ward() Ward {
	switch w {
	case SightWard, YellowTrinket:
		return YellowWard
	case BlueTrinket:
		return BlueWard
	case VisionWard:
		return PinkWard
	}
	return Ward{"unknown", 0, 1}
}
