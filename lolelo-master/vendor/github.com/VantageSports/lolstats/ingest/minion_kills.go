package ingest

import (
	"fmt"

	"github.com/VantageSports/lolstats/baseview"
)

type MinionStatTracker struct {
	// MinionsKilled tracks participant_id + level to minions killed, grouped by region
	MinionsKilled map[int64]map[int64]map[baseview.MapRegion]int64
	// JungleMinionInteractions is the sum of attacks and damage from jungle minions
	JungleMinionInteractions map[int64]map[int64]int64

	LastChampCS    map[int64]int64
	LastChampLevel map[int64]int64
}

func NewMinionStatTracker() *MinionStatTracker {
	mkt := &MinionStatTracker{
		MinionsKilled:            map[int64]map[int64]map[baseview.MapRegion]int64{},
		JungleMinionInteractions: map[int64]map[int64]int64{},
		LastChampCS:              map[int64]int64{},
		LastChampLevel:           map[int64]int64{},
	}
	// Everyone starts at level 1, not 0
	for i := int64(1); i <= 10; i++ {
		mkt.LastChampLevel[i] = 1
	}
	return mkt
}

func (mkt *MinionStatTracker) AddStateUpdate(t *baseview.StateUpdate) {
	// If there's a diff in the cs, then add it to the appropriate level+region bucket
	// NOTE: As of 6.21 EloBuddy patch, NeutralMinionsKilled is always 0, so
	// this is probably useless for tracking junglers!
	if t.MinionsKilled+t.NeutralMinionsKilled != mkt.LastChampCS[t.ParticipantID] {
		if _, ok := mkt.MinionsKilled[t.ParticipantID]; !ok {
			mkt.MinionsKilled[t.ParticipantID] = map[int64]map[baseview.MapRegion]int64{}
		}
		champMap := mkt.MinionsKilled[t.ParticipantID]
		if _, ok := champMap[mkt.LastChampLevel[t.ParticipantID]]; !ok {
			champMap[mkt.LastChampLevel[t.ParticipantID]] = map[baseview.MapRegion]int64{}
		}
		champLevelMap := champMap[mkt.LastChampLevel[t.ParticipantID]]

		areaID := baseview.AreaID(t.Position)
		region := baseview.AreaToRegion(areaID)
		champLevelMap[region]++
	}

	mkt.LastChampCS[t.ParticipantID] = t.MinionsKilled + t.NeutralMinionsKilled
}

func (mkt *MinionStatTracker) AddAttack(t *baseview.Attack) {
	mkt.addJungleMinionInteraction(t.AttackerID, t.TargetID)
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

func (mkt *MinionStatTracker) AddLevelUp(t *baseview.LevelUp) {
	mkt.LastChampLevel[t.ParticipantID]++
}

// For debugging
func (mkt *MinionStatTracker) String() string {
	s := ""
	for k, v := range mkt.MinionsKilled {
		s += fmt.Sprintf("Participant: %v, Last CS: %v, Last Level: %v\n", k, mkt.LastChampCS[k], mkt.LastChampLevel[k])
		for level, minionMap := range v {
			s += fmt.Sprintf("  %v:\n", level)
			for region, count := range minionMap {
				s += fmt.Sprintf("    %v = %v\n", region, count)
			}
		}
	}
	return s
}
