package ingest

import (
	"testing"

	"gopkg.in/tylerb/is.v1"

	"github.com/VantageSports/lolstats/baseview"
)

// Sanity check that our champ comp attributes json parsed correctly.
func TestParseChampCompAttrs(t *testing.T) {
	numChamps := len(compAttributesByChampIDStr)
	is.New(t).Msg("only found %d champ comps", numChamps).True(numChamps > 120)
}

func TestAddParticipant(t *testing.T) {
	is := is.New(t)

	me := tcPart(1, 117)

	tc := TeamComp{}
	tc.AddParticipant(*me, *me)
	for _, sideMap := range tc.Attributes {
		is.Msg("all maps should contain exactly 1 'us' and 1 'me' key").Equal(2, len(sideMap))
	}

	teammate := tcPart(2, 112)
	tc.AddParticipant(*me, *teammate)
	numUsGT := 0
	for _, sideMap := range tc.Attributes {
		// some keys should have 'us' > 'me' now
		if sideMap["us"] > sideMap["me"] {
			numUsGT++
		}
	}
	is.Msg("side keys with us > me = %d", numUsGT).True(numUsGT >= 15)

	enemy := tcPart(6, 119)
	tc.AddParticipant(*me, *enemy)
	numThemGT := 0
	for _, sideMap := range tc.Attributes {
		// some keys should have 'them' > 'us' (even though there's two of us? probably)
		if sideMap["them"] > sideMap["us"] {
			numThemGT++
		}
	}
	is.Msg("side keys with them > us = %d", numThemGT).True(numThemGT > 1)

}

func tcPart(pID, champID int64) *baseview.Participant {
	return &baseview.Participant{
		ChampionID:    champID,
		Division:      "wood",
		ParticipantID: pID,
		SummonerID:    pID + 10000,
	}
}
