package ingest

import (
	"fmt"

	"github.com/VantageSports/lolstats/baseview"
)

type MinionStatTracker struct {
	// MinionInteractions is the sum of attacks and damage to lane minions
	MinionInteractions map[int64]map[int64]map[baseview.MapRegion]int64
	// JungleMinionInteractions is the sum of attacks and damage from jungle minions
	JungleMinionInteractions map[int64]map[int64]int64

	LastChampLevel map[int64]int64
}

func NewMinionStatTracker() *MinionStatTracker {
	mkt := &MinionStatTracker{
		MinionInteractions:       map[int64]map[int64]map[baseview.MapRegion]int64{},
		JungleMinionInteractions: map[int64]map[int64]int64{},
		LastChampLevel:           map[int64]int64{},
	}
	// Everyone starts at level 1, not 0
	for i := int64(1); i <= 10; i++ {
		mkt.LastChampLevel[i] = 1
	}
	return mkt
}

// Elobuddy has issues with minion kills and neutral minion kills in the ping events. Don't rely on it!
// func (mkt *MinionStatTracker) AddStateUpdate(t *baseview.StateUpdate) {
// }

func (mkt *MinionStatTracker) AddAttack(t *baseview.Attack) {
	mkt.addJungleMinionInteraction(t.AttackerID, t.TargetID)
	mkt.addLaneMinionInteraction(t.AttackerID, t.TargetType, t.TargetPosition)
}

func (mkt *MinionStatTracker) AddDamage(t *baseview.Damage) {
	mkt.addJungleMinionInteraction(t.VictimID, t.AttackerID)
}

func (mkt *MinionStatTracker) addJungleMinionInteraction(pID, candidateID int64) {
	team := baseview.TeamID(pID)
	if team != baseview.BlueTeam && team != baseview.RedTeam {
		return
	}
	if !baseview.IsJungleMinion(candidateID) {
		return
	}
	if _, found := mkt.JungleMinionInteractions[pID]; !found {
		mkt.JungleMinionInteractions[pID] = map[int64]int64{}
	}

	mkt.JungleMinionInteractions[pID][mkt.LastChampLevel[pID]]++
}

func (mkt *MinionStatTracker) addLaneMinionInteraction(pID int64, candidateType baseview.ActorType, position *baseview.Position) {
	team := baseview.TeamID(pID)
	if team != baseview.BlueTeam && team != baseview.RedTeam {
		return
	}

	if candidateType != baseview.ActorMinion || position == nil {
		return
	}

	if _, found := mkt.MinionInteractions[pID]; !found {
		mkt.MinionInteractions[pID] = map[int64]map[baseview.MapRegion]int64{}
	}
	minionLevelMap := mkt.MinionInteractions[pID]

	if _, found := minionLevelMap[mkt.LastChampLevel[pID]]; !found {
		minionLevelMap[mkt.LastChampLevel[pID]] = map[baseview.MapRegion]int64{}
	}
	minionLevelRegionMap := minionLevelMap[mkt.LastChampLevel[pID]]

	areaID := baseview.AreaID(*position)
	region := baseview.AreaToRegion(areaID)
	minionLevelRegionMap[region]++
}

func (mkt *MinionStatTracker) AddLevelUp(t *baseview.LevelUp) {
	mkt.LastChampLevel[t.ParticipantID]++
}

// For debugging
func (mkt *MinionStatTracker) String() string {
	s := ""
	for k, v := range mkt.MinionInteractions {
		s += fmt.Sprintf("Participant: %v, Last Level: %v\n", k, mkt.LastChampLevel[k])
		for level, minionMap := range v {
			s += fmt.Sprintf("  %v:\n", level)
			for region, count := range minionMap {
				s += fmt.Sprintf("    %v = %v\n", region, count)
			}
		}
	}
	return s
}
