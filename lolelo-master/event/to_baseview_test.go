package event

import (
	"testing"

	"github.com/VantageSports/lolstats/baseview"

	"gopkg.in/tylerb/is.v1"
)

func TestFirstMatching(t *testing.T) {
	is := is.New(t)

	creations := []*OnCreate{
		{
			SenderName: "sightward",
			Position:   baseview.Position{3900.0, 5200.0, 0.0},
			NetworkID:  1000,
			TeamID:     200,
		},
		{
			SenderName: "visionward",
			Position:   baseview.Position{4400.0, 5400.0, 0.0},
			NetworkID:  1001,
			TeamID:     100,
		},
		{
			SenderName: "jammerdevice",
			Position:   baseview.Position{5900.0, 2200.0, 0.0},
			NetworkID:  1002,
			TeamID:     100,
		},
	}
	creations[0].SetMatchSeconds(123.9)
	creations[1].SetMatchSeconds(124.1)
	creations[2].SetMatchSeconds(139)

	spells := []*SpellCast{
		{
			NetworkID:   101,
			SpellName:   "itemghostward",
			PositionEnd: baseview.Position{4000.0, 5000.0, 0.0},
		},
		{
			NetworkID:   106,
			SpellName:   "TrinketTotemLVL1",
			PositionEnd: baseview.Position{4100.0, 5200.0, 0.0},
		},
		{
			NetworkID:   102,
			SpellName:   "JammerDevice",
			PositionEnd: baseview.Position{6000.0, 2000.0, 0.0},
		},
	}
	spells[0].SetMatchSeconds(123.4)
	spells[1].SetMatchSeconds(124.5)
	spells[2].SetMatchSeconds(137)

	netIDMap := NewNetworkIDMap()
	netIDMap.AddParticipant(1, "hero1")
	netIDMap.AddParticipant(2, "hero2")
	netIDMap.AddParticipant(6, "hero6")
	netIDMap.AddEvent(&NetworkIDMapping{SenderName: "hero1", NetworkID: 101, EloType: "ID_HERO"})
	netIDMap.AddEvent(&NetworkIDMapping{SenderName: "hero2", NetworkID: 102, EloType: "ID_HERO"})
	netIDMap.AddEvent(&NetworkIDMapping{SenderName: "hero6", NetworkID: 106, EloType: "ID_HERO"})
	for _, c := range creations {
		netIDMap.AddEvent(c)
	}

	bv, err := placeWards(creations[0:2], spells, netIDMap)
	is.Err(err)

	bv, err = placeWards(creations, spells[0:2], netIDMap)
	is.Err(err)

	bv, err = placeWards(creations, spells, netIDMap)
	is.Equal(3, len(bv))

	isWardEqual(is, bv[0], 123.9, baseview.YellowTrinket, 6)
	isWardEqual(is, bv[1], 124.1, baseview.SightWard, 1)
	isWardEqual(is, bv[2], 139, baseview.VisionWard, 2)
}

func isWardEqual(is *is.Is, wpv baseview.Event, seconds float64, itemType baseview.WardItemType, pID int64) {
	wp, ok := wpv.(*baseview.WardPlaced)
	is.Msg("expected *WardPlaced, got %T", wpv).True(ok)

	is.True(seconds-wp.Seconds() < 0.5)
	is.Equal(itemType, wp.WardItemType)
	is.Equal(pID, wp.ParticipantID)
}
