package ingest

import (
	"fmt"
	"math"

	"github.com/VantageSports/lolstats/baseview"
)

const (
	// These are the max width and max height of the summoner's rift map.
	// Technically, you can have slightly negative x and y positions, but we're
	// ignoring that for simplicity, as it's unlikely to make a significant
	// difference in the computation of map coverage percent.
	// https://developer.riotgames.com/docs/game-constants
	width  = 14870.0
	height = 14980.0
)

// A CoverageMap is a way to track what percentage of the map has been visted
// at least once. For each position added to the CoverageMap, we compute the
// grid "square" referenced by the position and mark it as visited. The total
// number of keys in our map relative to the total possible number of keys
// represents the amount of the map visited.
//
// The gridSize represents the number of grid squares in a grid. Therefore, a
// gridSize of 12 would represent a 12x12 grid, whereas a gridSize of 1 would
// mean 1x1.
type CoverageMap struct {
	// keyed by game-stage and then grid-key
	stageToSeen map[string]map[string]int64
	gridSize    float64
}

func NewCoverageMap(gridSize int64) *CoverageMap {
	return &CoverageMap{
		stageToSeen: map[string]map[string]int64{},
		gridSize:    float64(gridSize),
	}
}

// Add computes the grid key for the specified position and marks it as seen.
func (c *CoverageMap) Add(seconds float64, p baseview.Position) {
	stage := gameStage(seconds)

	x := math.Min(width, math.Max(p.X, 0))
	xKey := int64(x / (width / c.gridSize))

	y := math.Min(height, math.Max(p.Y, 0))
	yKey := int64(y / (height / c.gridSize))

	gridKey := fmt.Sprintf("%d-%d", xKey, yKey)

	if _, found := c.stageToSeen[stage]; !found {
		c.stageToSeen[stage] = map[string]int64{}
	}
	if _, found := c.stageToSeen["full"]; !found {
		c.stageToSeen["full"] = map[string]int64{}
	}

	c.stageToSeen[stage][gridKey]++
	c.stageToSeen["full"][gridKey]++
}

// Percent returns the percentage of "visited" grid squares relative to the
// total grid squares.
func (c *CoverageMap) Percents() map[string]float64 {
	res := map[string]float64{}
	max := c.gridSize * c.gridSize

	for stage, keys := range c.stageToSeen {
		res[stage] = 100.0 * float64(len(keys)) / max
	}

	return res
}
