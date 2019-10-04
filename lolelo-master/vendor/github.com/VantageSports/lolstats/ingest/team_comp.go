// TeamComp sums the relative strengths of one team's champions vs another.
// The strength values themselves are kept is a super-secret spreadsheet in
// google docs, and are periodically updated, translated into json, and kept in
// the champ_comp.json file in this directory. This file is read at startup.
//
// The TeamComp map maps from attributes ('carry', 'snowball', etc), to sides,
// ('us', 'them', 'me') to float values, which are the sum of the champions'
// abilities in that category on that side.

package ingest

import (
	"fmt"
	"strconv"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats/baseview"
)

type TeamComp struct {
	// Attributes maps from attribute (e.g. 'carry', 'cc', 'snowball') to
	// side-key, (e.g. 'us', 'them', or 'me') to value.
	Attributes map[string]map[string]float64 `json:"attributes,omitempty"`
	// KeyAttributes maps from side-key to attributes, to value for the top
	// 3 attributes for each side.
	KeyAttributes map[string]map[string]float64 `json:"key_attributes,omitempty"`
	// KeyContribution percent is the percentage of contribution to subject
	// champion's key attributes.
	KeyContributionPct float64 `json:"key_contribution_pct,omitempty"`
}

func (tc *TeamComp) AddParticipant(me, player baseview.Participant) {
	champIDStr := strconv.FormatInt(player.ChampionID, 10)
	champAttributes := compAttributesByChampIDStr[champIDStr]

	if champAttributes == nil {
		log.Warning(fmt.Sprintf("champ with no attributes found: " + champIDStr))
		return
	}

	if tc.Attributes == nil {
		tc.Attributes = map[string]map[string]float64{}
	}

	for name, val := range champAttributes {
		if tc.Attributes[name] == nil {
			tc.Attributes[name] = map[string]float64{}
		}

		myTeam := baseview.TeamID(me.ParticipantID)
		playerTeam := baseview.TeamID(player.ParticipantID)

		sideKey := "them"
		if myTeam == playerTeam {
			sideKey = "us"
		}

		tc.Attributes[name][sideKey] += val
		if me.ParticipantID == player.ParticipantID {
			tc.Attributes[name]["me"] += val
		}
	}

}

func (tc *TeamComp) ComputeKeyAttributes() {
	tc.KeyAttributes = map[string]map[string]float64{
		"us":   map[string]float64{},
		"them": map[string]float64{},
	}

	// Compute both team's top 3 characteristics.
	for attr, sideMap := range tc.Attributes {
		for sideKey, val := range sideMap {
			if sideKey == "me" {
				continue
			}
			teamKeys := tc.KeyAttributes[sideKey]
			if len(teamKeys) < 3 {
				teamKeys[attr] = val
			} else if minKey, minVal := minVal(teamKeys); minVal <= val {
				// map iteration order is undefined, so break ties
				// alphabetically by key name
				if minVal < val || attr < minKey {
					delete(teamKeys, minKey)
					teamKeys[attr] = val
				}
			}
		}
	}

	// Compute my contribution to my team's key characteristics
	tc.KeyAttributes["me"] = map[string]float64{}
	myContribution, teamTotal := 0.0, 0.0
	for attr, val := range tc.KeyAttributes["us"] {
		attrContribution := tc.Attributes[attr]["me"]
		tc.KeyAttributes["me"][attr] = attrContribution
		myContribution += attrContribution
		teamTotal += val
	}
	tc.KeyContributionPct = 100.0 * myContribution / teamTotal
}

func minVal(m map[string]float64) (minKey string, minVal float64) {
	for key, val := range m {
		// if (a) this is the first key we've encountered, or
		//    (b) val is the minimum, or
		//    (c) val is equal but the key is alphabetically AFTER minkey
		// then we've found our new min
		if minKey == "" || val < minVal || (val == minVal && key > minKey) {
			minKey, minVal = key, val
		}
	}
	return minKey, minVal
}
