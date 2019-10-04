package ingest

import (
	"fmt"

	"github.com/VantageSports/lolstats/baseview"
)

// BuddyFinder figures out which teammate you spent the most time next to.
// We use it to calculate role positions, by looking for the sup/adc combo
type BuddyFinder struct {
	// For each participant, there's a buddy score for each team member
	// Right now, we only care about early game (<10 minutes)
	EarlyGameBuddyScore map[int64]map[int64]int64

	LastStateUpdates map[int64]baseview.StateUpdate
}

func NewBuddyFinder() *BuddyFinder {
	bf := &BuddyFinder{
		EarlyGameBuddyScore: map[int64]map[int64]int64{},
		LastStateUpdates:    map[int64]baseview.StateUpdate{},
	}
	return bf
}

func (bf *BuddyFinder) AddStateUpdate(t *baseview.StateUpdate) {
	// Ignore the period before minions start.
	// Also, ignore everything past the early game
	if t.Seconds() < 100 || t.Seconds() > 600 {
		return
	}

	bf.LastStateUpdates[t.ParticipantID] = *t

	// Make sure we have an initial position for each champion
	if len(bf.LastStateUpdates) != 10 {
		return
	}

	teamID := baseview.TeamID(t.ParticipantID)
	for k, v := range bf.LastStateUpdates {
		// Skip yourself, and players on the other team
		if k == t.ParticipantID || baseview.TeamID(k) != teamID {
			continue
		}
		// Add 1 point for each tick in which a team member is closer than 2000 units away
		if t.Position.DistanceXY(v.Position) < 2000 {
			if _, ok := bf.EarlyGameBuddyScore[t.ParticipantID]; !ok {
				bf.EarlyGameBuddyScore[t.ParticipantID] = map[int64]int64{}
			}
			bf.EarlyGameBuddyScore[t.ParticipantID][k]++
		}
	}
}

func (bf *BuddyFinder) BuddyFor(participantID int64, assignedRoles map[int64]baseview.RolePosition) int64 {
	var highestScore int64
	var bestMatch int64 = -1
	for k, v := range bf.EarlyGameBuddyScore[participantID] {
		if _, found := assignedRoles[k]; found {
			continue
		}
		if v > highestScore {
			highestScore = v
			bestMatch = k
		}
	}
	return bestMatch
}

func (bf *BuddyFinder) String() string {
	s := ""
	for k, v := range bf.EarlyGameBuddyScore {
		s += fmt.Sprintf("Participant: %v\n", k)
		for buddy, score := range v {
			s += fmt.Sprintf("  %v: %v\n", buddy, score)
		}
	}
	return s
}
