package ingest

import (
	"testing"

	"gopkg.in/tylerb/is.v1"

	"github.com/VantageSports/lolstats/baseview"
)

var (
	wardPos  = baseview.Position{1000.0, 2000.0, 55.0}
	inRange  = baseview.Position{1100.0, 1900.0, 55.0}
	outRange = baseview.Position{9000.0, 10000.0, 55.0}
)

func TestWardPlaced(t *testing.T) {
	is := is.New(t)
	// Basic case
	wa := NewWardAnalysis()

	// Player 1 places a ward, and then ward id 90 is created soon after
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.YellowTrinket, 1.0), 1000)

	// We should have an active ward with id 90 linked to the creator
	is.Equal(1, wa.ActiveWards[90].Creator)
}

func TestWardDeathCleared(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places a ward, and then ward id 90 is created soon after
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.YellowTrinket, 1.0), 1000)

	// Player 6 attacks the ward three times
	wa.AddAttack(wardAttack(90, 6, 2.0))
	wa.AddAttack(wardAttack(90, 6, 2.5))
	wa.AddAttack(wardAttack(90, 6, 3.0))

	// Ward dies
	wa.AddWardDeath(wardDeath(90, 100, 3.5))

	// We should have a player ward with the right properties
	is.Equal(0, len(wa.ActiveWards))
	is.Equal(1, wa.PlayerWards[1][0].Begin)
	is.Equal(3.5, wa.PlayerWards[1][0].End)
	is.Equal(1, wa.PlayerWards[1][0].Creator)
	is.Equal("cleared", wa.PlayerWards[1][0].EndReason)
	is.Equal(6, wa.PlayerWards[1][0].ClearedBy)
}

func TestWardDeathReplaced(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places 3 yellow wards and a pink ward
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.YellowTrinket, 1.0), 1000)
	wa.AddWardPlaced(wardPlaced(1, 91, baseview.SightWard, 2.0), 1000)
	wa.AddWardPlaced(wardPlaced(1, 92, baseview.SightWard, 3.0), 1000)
	wa.AddWardPlaced(wardPlaced(1, 93, baseview.VisionWard, 4.0), 1000)

	// A 4th yellow ward is placed, and the first ward dies
	wa.AddWardPlaced(wardPlaced(1, 94, baseview.SightWard, 5.0), 1000)
	wa.AddWardDeath(wardDeath(90, 100, 5.0))

	// Check the reason. It should be replaced
	is.Equal(1.0, wa.PlayerWards[1][0].Begin)
	is.Equal(5.0, wa.PlayerWards[1][0].End)
	is.Equal("replaced", wa.PlayerWards[1][0].EndReason)

	// A 2nd pink ward is placed
	wa.AddWardPlaced(wardPlaced(1, 95, baseview.VisionWard, 6.0), 1000)
	wa.AddWardDeath(wardDeath(93, 100, 6.0))

	// Check the reason. It should be replaced
	is.Equal(4.0, wa.PlayerWards[1][3].Begin)
	is.Equal(6.0, wa.PlayerWards[1][3].End)
	is.Equal("replaced", wa.PlayerWards[1][3].EndReason)
}

func TestWardDeathExpired(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places a yellow ard that expires
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.YellowTrinket, 1.0), 1000)
	wa.AddWardDeath(wardDeath(90, 100, 63))

	t.Logf(wa.PlayerWards[1][0].Type)

	// Check the reason. It should be expired
	is.Equal(1.0, wa.PlayerWards[1][0].Begin)
	is.Equal(63, wa.PlayerWards[1][0].End)
	is.Equal("expired", wa.PlayerWards[1][0].EndReason)
}

func TestWardDeathComplexCleared(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places a yellow ward
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.SightWard, 1.0), 1000)

	// Has a couple attacks early
	wa.AddAttack(wardAttack(90, 6, 2.5))
	wa.AddAttack(wardAttack(90, 6, 3.0))

	// Player 1 places 2 more wards
	wa.AddWardPlaced(wardPlaced(1, 91, baseview.SightWard, 8.0), 1000)
	wa.AddWardPlaced(wardPlaced(1, 92, baseview.SightWard, 9.0), 1000)

	// Later on, the first ward is cleared
	wa.AddAttack(wardAttack(90, 8, 130))
	// The ward death event can sometimes be pretty delayed
	wa.AddWardDeath(wardDeath(90, 100, 134))

	// Check the reason. It should be cleared
	is.Equal(1.0, wa.PlayerWards[1][0].Begin)
	is.Equal(134, wa.PlayerWards[1][0].End)
	is.Equal("cleared", wa.PlayerWards[1][0].EndReason)
	is.Equal(8, wa.PlayerWards[1][0].ClearedBy)
}

func TestWardDeathComplexReplaced(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places a yellow ward
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.SightWard, 1.0), 1000)

	// Has a couple attacks early
	wa.AddAttack(wardAttack(90, 6, 2.5))
	wa.AddAttack(wardAttack(90, 6, 3.0))

	// Player 1 places 2 more wards
	wa.AddWardPlaced(wardPlaced(1, 91, baseview.SightWard, 8.0), 1000)
	wa.AddWardPlaced(wardPlaced(1, 92, baseview.SightWard, 9.0), 1000)

	// Later on, the first ward is replaced
	wa.AddWardPlaced(wardPlaced(1, 93, baseview.SightWard, 130.0), 1000)
	wa.AddWardDeath(wardDeath(90, 100, 134))

	// Check the reason. It should be replaced
	is.Equal(1.0, wa.PlayerWards[1][0].Begin)
	is.Equal(134, wa.PlayerWards[1][0].End)
	is.Equal("replaced", wa.PlayerWards[1][0].EndReason)
}

func TestWardDeathComplexExpired(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places a yellow ward
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.SightWard, 1.0), 1000)

	// Has a couple attacks early
	wa.AddAttack(wardAttack(90, 6, 2.5))
	wa.AddAttack(wardAttack(90, 6, 3.0))

	// Player 1 places 2 more wards
	wa.AddWardPlaced(wardPlaced(1, 91, baseview.SightWard, 8.0), 1000)
	wa.AddWardPlaced(wardPlaced(1, 92, baseview.SightWard, 9.0), 1000)

	// Later on, the first ward is replaced
	wa.AddWardDeath(wardDeath(90, 100, 134))

	// Check the reason. It should be expired
	is.Equal(1.0, wa.PlayerWards[1][0].Begin)
	is.Equal(134, wa.PlayerWards[1][0].End)
	is.Equal("expired", wa.PlayerWards[1][0].EndReason)
}

func TestWardReveal(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places a yellow ward
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.SightWard, 1.0), 1000)

	// Put some people near it
	// Some friendly people, who shouldn't count
	wa.AddStateUpdate(stateUpdate(1, 500, inRange, 1.2))
	wa.AddStateUpdate(stateUpdate(5, 500, inRange, 1.2))
	// Some enemies, who should count
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 1.2))
	wa.AddStateUpdate(stateUpdate(10, 500, inRange, 1.2))
	// An enemy was dead in the area before the ward was placed. Don't count him
	wa.AddStateUpdate(stateUpdate(9, 0, inRange, 1.2))

	// There should be a reveal for each enemy
	is.Equal(1, len(wa.ActiveWards[90].Reveals[6]))
	is.Equal(1, len(wa.ActiveWards[90].Reveals[10]))
	is.Equal(0, len(wa.ActiveWards[90].Reveals[9]))
}

func TestWardMultiReveal(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places a yellow ward
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.SightWard, 1.0), 1000)

	// Enemy walks by
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 1.2))
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 2))
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 3))
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 4))
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 5))
	// Left the ward range
	wa.AddStateUpdate(stateUpdate(6, 500, outRange, 6))
	// Steps back in briefly. This should continue the same reveal
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 7))
	wa.AddStateUpdate(stateUpdate(6, 500, outRange, 8))
	// Steps back in after a longer period. This should be a new reveal
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 20))
	wa.AddStateUpdate(stateUpdate(6, 500, outRange, 21))

	// There should be 2 reveals
	is.Equal(2, len(wa.ActiveWards[90].Reveals[6]))
	// First one should be from 1.2 -> 7
	is.Equal(1.2, wa.ActiveWards[90].Reveals[6][0].Begin)
	is.Equal(7, wa.ActiveWards[90].Reveals[6][0].End)
	// Second one should be from 20 -> 20
	is.Equal(20, wa.ActiveWards[90].Reveals[6][1].Begin)
	is.Equal(20, wa.ActiveWards[90].Reveals[6][1].End)
}

func TestWardRevealTruncation(t *testing.T) {
	is := is.New(t)
	wa := NewWardAnalysis()

	// Player 1 places a yellow ward
	wa.AddWardPlaced(wardPlaced(1, 90, baseview.SightWard, 1.0), 1000)

	// Enemy walks by
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 80))
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 81))
	// The ward expires, during the enemy's visit
	wa.AddWardDeath(wardDeath(90, 100, 81))
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 82))
	wa.AddStateUpdate(stateUpdate(6, 500, inRange, 83))
	// Left the ward range
	wa.AddStateUpdate(stateUpdate(6, 500, outRange, 84))

	// Someone else comes by after the ward dies at 81
	wa.AddStateUpdate(stateUpdate(7, 500, inRange, 84))

	// There should be a reveal that gets truncated at 81 seconds
	is.Equal(1, len(wa.PlayerWards[1][0].Reveals[6]))
	is.Equal(1, wa.PlayerWards[1][0].Begin)
	is.Equal(81, wa.PlayerWards[1][0].End)
	is.Equal(80, wa.PlayerWards[1][0].Reveals[6][0].Begin)
	is.Equal(81, wa.PlayerWards[1][0].Reveals[6][0].End)

	// Player 7 should get deleted from the reveals
	is.Equal(0, len(wa.PlayerWards[1][0].Reveals[7]))
}

func TestLikelyWardClearer(t *testing.T) {
	is := is.New(t)
	life := &wardLife{Creator: 2}

	is.Msg("no attacks or reveals should guess").Equal(6, likelyWardClearer(life))

	life.Creator = 7
	is.Msg("no attacks or reveals should guess").Equal(1, likelyWardClearer(life))

	life.Reveals = map[int64][]*TimeInterval{
		// contrived test-case, adding reveals for same-team
		6: []*TimeInterval{
			{Begin: 200, End: 220},
		},
		1: []*TimeInterval{
			{Begin: 100, End: 110}, {Begin: 150, End: 160},
		},
		2: []*TimeInterval{
			{Begin: 95, End: 100}, {Begin: 145, End: 215},
		},
	}
	is.Msg("should find 2's last reveal").Equal(2, likelyWardClearer(life))
}

func wardPlaced(participantID, wardID int64, item baseview.WardItemType, t float64) *baseview.WardPlaced {
	b := &baseview.WardPlaced{
		ParticipantID: participantID,
		Position:      wardPos,
		WardItemType:  item,
		WardType:      item.Ward().Type(),
		WardID:        wardID,
	}
	b.SetType("ward_placed")
	b.SetSeconds(t)
	return b
}

func wardDeath(wardID int64, teamID int64, t float64) *baseview.WardDeath {
	b := &baseview.WardDeath{
		WardID: wardID,
	}
	b.SetType("ward_death")
	b.SetSeconds(t)
	return b
}

func wardAttack(wardID, attackerID int64, t float64) *baseview.Attack {
	b := &baseview.Attack{
		AttackerID:     attackerID,
		TargetID:       wardID,
		TargetPosition: &wardPos,
	}
	b.SetType("attack")
	b.SetSeconds(t)
	return b
}

func stateUpdate(participantID int64, health float64, pos baseview.Position, secs float64) *baseview.StateUpdate {
	b := &baseview.StateUpdate{
		Health:        health,
		Position:      pos,
		ParticipantID: participantID,
	}
	b.SetType("state_update")
	b.SetSeconds(secs)
	return b
}
