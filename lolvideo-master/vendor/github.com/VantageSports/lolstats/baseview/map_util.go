package baseview

const (
	AreaBlueBase      int64 = 1
	AreaBlueTopLane   int64 = 2
	AreaBlueTopJungle int64 = 3
	AreaBlueMidLane   int64 = 4
	AreaBlueBotJungle int64 = 5
	AreaBlueBotLane   int64 = 6

	AreaTopRiver    int64 = 7
	AreaBottomRiver int64 = 8

	AreaRedBase      int64 = 9
	AreaRedTopLane   int64 = 10
	AreaRedTopJungle int64 = 11
	AreaRedMidLane   int64 = 12
	AreaRedBotJungle int64 = 13
	AreaRedBotLane   int64 = 14
)

// MapRegions are more general game concepts that aren't tied to a particular side of the map
type MapRegion string

const (
	RegionTop    MapRegion = "TOP"
	RegionMid    MapRegion = "MID"
	RegionBot    MapRegion = "BOT"
	RegionJungle MapRegion = "JUNGLE"
	RegionBase   MapRegion = "BASE"
	RegionOther  MapRegion = "OTHER"
)

// There should be a mapping from every area to a region
func AreaToRegion(id int64) MapRegion {
	if id == AreaBlueBase || id == AreaRedBase {
		return RegionBase
	} else if id == AreaBlueTopLane || id == AreaRedTopLane {
		return RegionTop
	} else if id == AreaBlueMidLane || id == AreaRedMidLane {
		return RegionMid
	} else if id == AreaBlueBotLane || id == AreaRedBotLane {
		return RegionBot
	} else if id == AreaBlueTopJungle || id == AreaBlueBotJungle ||
		id == AreaRedTopJungle || id == AreaRedBotJungle ||
		id == AreaTopRiver || id == AreaBottomRiver {
		return RegionJungle
	}
	return RegionOther
}

const (
	// Summoners rift constants
	// Derived from https://developer.riotgames.com/docs/game-constants
	// If you want to modify, make sure you play around with the jsfiddle they
	// link to on that site for an easy way to vizualize various points.
	SRXMin float64 = -120
	SRXMax float64 = 14870
	SRYMin float64 = -120
	SRYMax float64 = 14980
)

// GetArea returns the id associated with the provided position, or -1 if it
// can't be discerned.
func AreaID(p Position) int64 {
	transposed := !isBlue(p)
	if transposed {
		p = Position{X: SRYMax - p.Y, Y: SRXMax - p.X, Z: p.Z}
	}

	max, min := p.X, p.Y
	if max < min {
		max, min = min, max
	}

	switch {
	case p.X < 4400 && p.Y < 4400:
		return ternary(transposed, AreaRedBase, AreaBlueBase)
	case p.X < 1900 || p.Y > 11700:
		return ternary(transposed, AreaRedTopLane, AreaBlueTopLane)
	case p.Y < 1900 || p.X > 11700:
		return ternary(transposed, AreaRedBotLane, AreaBlueBotLane)
	case min/max > 0.86:
		return ternary(transposed, AreaRedMidLane, AreaBlueMidLane)
	case p.X <= p.Y:
		return ternary(transposed, AreaRedTopJungle, AreaBlueTopJungle)
	case p.X > p.Y:
		return ternary(transposed, AreaRedBotJungle, AreaBlueBotJungle)

	default:
		return 0
	}
}

func ternary(cond bool, a int64, b int64) int64 {
	if cond {
		return a
	}
	return b
}

func isBlue(p Position) bool {
	distXMin := p.X - SRYMin
	distXMax := SRXMax - p.X
	distYMin := p.Y - SRYMin
	distYMax := SRYMax - p.Y

	if distXMin < distXMax && distXMin < distYMin && distXMin < distYMax {
		// return true (team blue) if the position is closest to XMin...
		return true
	} else if distYMin < distXMin && distYMin < distXMax && distYMin < distYMax {
		// .. or YMin
		return true
	}
	return false
}
