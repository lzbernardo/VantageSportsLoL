package ingest

import (
	"fmt"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats/baseview"
)

type TimeInterval struct {
	Begin float64
	End   float64
}

type wardLife struct {
	Begin          float64
	End            float64
	Position       *baseview.Position
	Creator        int64
	ItemType       baseview.WardItemType
	Type           string
	EndReason      string
	AttacksAgainst []baseview.Attack
	ClearedBy      int64
	Reveals        map[int64][]*TimeInterval
}

// JsonWardLife is a slimmed down version of WardLife, suitable for use outside this file.
// Mostly, AttacksAgainst is removed, and Reveals is turned into a count instead of intervals
type JsonWardLife struct {
	Begin     float64            `json:"begin"`
	End       float64            `json:"end"`
	Position  *baseview.Position `json:"position"`
	Type      string             `json:"type"`
	EndReason string             `json:"end_reason"`
	ClearedBy int64              `json:"cleared_by,omitempty"`
	// map of participantID -> number of times that participant was revealed by this ward
	Reveals map[int64]int `json:"reveals"`
}

// wardAnalysis keeps track of all wards
// When a player places a ward, add it to pending placed
// When a ward is created, try to find the closest pending placed ward, and move it to ActiveWards
// When a ward dies, look for it in ActiveWards. Fill in end info, and move it to PlayerWards
type wardAnalysis struct {
	ActiveWards map[int64]*wardLife
	PlayerWards map[int64][]*wardLife
}

func NewWardAnalysis() *wardAnalysis {
	return &wardAnalysis{
		ActiveWards: map[int64]*wardLife{},
		PlayerWards: map[int64][]*wardLife{},
	}
}

func (wa *wardAnalysis) AddWardPlaced(t *baseview.WardPlaced, gameLengthSeconds float64) {
	life := &wardLife{
		Begin: t.Seconds(),
		// End gets overridden if we get a ward_death event
		End:            gameLengthSeconds,
		Position:       &t.Position,
		Creator:        t.ParticipantID,
		ItemType:       t.WardItemType,
		Type:           t.WardItemType.Ward().Type(),
		AttacksAgainst: []baseview.Attack{},
		Reveals:        map[int64][]*TimeInterval{},
	}
	wa.ActiveWards[t.WardID] = life
	wa.PlayerWards[t.ParticipantID] = append(wa.PlayerWards[t.ParticipantID], life)
}

func (wa *wardAnalysis) AddAttack(t *baseview.Attack) {
	if t.TargetID == 0 {
		return
	}

	activeWard, exists := wa.ActiveWards[t.TargetID]
	if !exists {
		// There are basic attacks on other things. Just ignore those in this file
		return
	}

	activeWard.AttacksAgainst = append(activeWard.AttacksAgainst, *t)
}

func (wa *wardAnalysis) AddWardDeath(t *baseview.WardDeath) error {
	ward, exists := wa.ActiveWards[t.WardID]

	if !exists {
		return fmt.Errorf("unable to find matching active ward for death at %.1f: %v", t.Seconds(), t.WardID)
	}

	ward.End = t.Seconds()
	duration := ward.End - ward.Begin

	// If it was attacked recently then it was cleared
	// If the player has too many wards of that type, then it's a replacement
	// If the ward expires, and has a reasonable lifetime, then it's an expiration
	var lastAttack *baseview.Attack
	if len(ward.AttacksAgainst) > 0 {
		lastAttack = &ward.AttacksAgainst[len(ward.AttacksAgainst)-1]
	}

	if lastAttack != nil && ward.End-lastAttack.Seconds() < 5 {
		ward.EndReason = "cleared"
		ward.ClearedBy = lastAttack.AttackerID
	} else if wa.exceedsWardLimit(ward) {
		ward.EndReason = "replaced"
	} else if ward.ItemType.Ward().Type() == "yellow" && duration >= 58 {
		// YellowWards expire after 60 seconds, but we account for 2 seconds
		// of wiggle room.
		ward.EndReason = "expired"
		if duration >= 155 {
			// Sightstone wards expire after 150 seconds, but if there is a
			// processing hiccup/pause, we may not see these wards actually
			// expire for ~165 seconds, so we just warn.
			log.Warning(fmt.Sprintf("yellow ward died at %.1f with duration of %.1f", t.Seconds(), duration))
		}
	} else {
		// Its unlikely, but possible, that we missed an attack on this ward.
		// As of 9/30/2016, we know that we don't see Rengar bush jumps, for
		// instance. For now, we're marking all wards as having been cleared
		// and crediting _SOME_ champ on the other team.
		ward.EndReason = "cleared"
		ward.ClearedBy = likelyWardClearer(ward)
		log.Warning(fmt.Sprintf("unknown ward death reason at %.1f (marked cleared): %v", t.Seconds(), ward))
	}

	// Remove it from ActiveWards
	delete(wa.ActiveWards, t.WardID)
	return nil
}

// likelyWardClearer returns a participant id on the opposite of the ward's
// creator that is 'likely' to be the one that cleared the ward. This is used
// when we don't have a last attack on the ward (which may happen - rarely -
// in cases when we don't have certain champs attacks, like Rengar's bush jumps
// as of 9/30/2016). We first attempt to find a champ that is currently
// 'revealed', and just guess after that.
func likelyWardClearer(ward *wardLife) int64 {
	var lastRevealTime float64
	var lastRevealPID int64

	for pID, intervals := range ward.Reveals {
		if baseview.TeamID(ward.Creator) == baseview.TeamID(pID) || len(intervals) == 0 {
			// this shouldn't ever happen, but just being safe.
			continue
		}
		reveal := intervals[len(intervals)-1]
		if lastRevealTime < reveal.End {
			lastRevealTime = reveal.End
			lastRevealPID = pID
		}
	}

	if lastRevealPID > 0 {
		return lastRevealPID
	} else if ward.Creator >= 6 {
		return 1
	}
	return 6
}

func (wa *wardAnalysis) AddStateUpdate(t *baseview.StateUpdate) {
	// Don't reveal dead players
	if t.Health == 0 {
		return
	}
	// Look for players in range of active wards
	for _, activeWard := range wa.ActiveWards {
		// Ignore champions on the same team as the ward
		if baseview.TeamID(activeWard.Creator) == baseview.TeamID(t.ParticipantID) {
			continue
		}

		sightRange := activeWard.ItemType.Ward().SightRange()

		// Add a reveal if they're within range.
		// This doesn't account for things like terrain or bushes, so it's a bit inaccurate
		if activeWard.Position.DistanceXY(t.Position) <= sightRange {
			reveals := activeWard.Reveals[t.ParticipantID]
			if len(reveals) > 0 {
				lastReveal := reveals[len(reveals)-1]
				if t.Seconds()-lastReveal.End < 5 {
					// If it's within 5 seconds, continue the last reveal
					lastReveal.End = t.Seconds()
					continue
				}
			}

			// Otherwise, create a new reveal
			activeWard.Reveals[t.ParticipantID] = append(reveals,
				&TimeInterval{Begin: t.Seconds(), End: t.Seconds()})
		}
	}
}

func (wa *wardAnalysis) exceedsWardLimit(ward *wardLife) bool {
	activeByType := map[string]float64{}

	for _, w := range wa.ActiveWards {
		if w.Creator != ward.Creator {
			continue
		}
		max := w.ItemType.Ward().MaxConcurrent()
		activeByType[w.Type]++

		if activeByType[w.Type] > max {
			return true
		}
	}

	return false
}

func (wa *wardAnalysis) ExportPlayerWards() map[int64][]*JsonWardLife {
	res := map[int64][]*JsonWardLife{}
	for pID, val := range wa.PlayerWards {
		for _, wl := range val {
			revealMap := map[int64]int{}
			for rID, arry := range wl.Reveals {
				revealMap[rID] = len(arry)
			}
			res[pID] = append(res[pID], &JsonWardLife{
				Begin:     wl.Begin,
				End:       wl.End,
				Position:  wl.Position,
				Type:      wl.Type,
				EndReason: wl.EndReason,
				ClearedBy: wl.ClearedBy,
				Reveals:   revealMap,
			})
		}
	}
	return res
}

func (wa *wardAnalysis) RevealsPerWardAverage(participantID int64) float64 {
	reveals := 0.0
	total := 0.0
	for _, val := range wa.PlayerWards[participantID] {
		for _, arry := range val.Reveals {
			reveals += float64(len(arry))
		}
		total++
	}
	if total == 0 {
		total = 1.0
	}
	return reveals / total
}

func (wa *wardAnalysis) LiveWardsAverage(participantID int64, matchDuration float64) map[string]float64 {
	res := map[string]float64{
		"yellow":          0,
		"pink":            0,
		"blue":            0,
		"yellow_and_blue": 0,
	}

	for _, ward := range wa.PlayerWards[participantID] {
		res[ward.Type] += ward.End - ward.Begin
		if ward.Type == "blue" || ward.Type == "yellow" {
			res["yellow_and_blue"] += ward.End - ward.Begin
		}
	}

	for k := range res {
		res[k] = res[k] / matchDuration
	}
	return res
}

// For debugging
func (ward *wardLife) String() string {

	s := fmt.Sprintf("%v [%.1f, %.1f] %s", ward.Type, ward.Begin, ward.End, ward.EndReason)
	if ward.EndReason == "cleared" {
		s += fmt.Sprintf(" by %v", ward.ClearedBy)
	}
	s += fmt.Sprintf(". Reveals: ")
	for player, revealTimes := range ward.Reveals {
		s += fmt.Sprintf("%v: ", player)
		for _, revealTime := range revealTimes {
			s += fmt.Sprintf("[%.1f, %.1f] ", revealTime.Begin, revealTime.End)
		}
	}
	return s + "\n"
}
