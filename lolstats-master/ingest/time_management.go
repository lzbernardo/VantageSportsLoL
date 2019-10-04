// TimeManagement is a state machine that tracks (simplified) event stream for
// a single champion over the course of a match. We track two orthogonal pieces
// of state: location (area) and activity.
//
// Possible 'areas':      base, top, mid, bot, jungle, roam
// Possible 'activities': farm, fight, baron, dragon, siege, ward, dead
//
// Advanced stats stores both a detailed view of when each location/activity
// event occurred, as well as an aggregated view of total seconds spent in each
// state in each stage of the game.
//
// NOTE: The aggregated view could be calculated rather trivially on the
// frontend.

package ingest

import (
	"strings"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats/baseview"
)

// TimeManagementFrame is the most granular level at which we track time
// management, as it describes the activity and area, as well as the start and
// stop times for each. Each time the activity or location changes, a new
// TimeManagementFrame will be emitted.
type TimeManagementFrame struct {
	Activity string  `json:"activity"`
	AreaName string  `json:"area"`
	Begin    float64 `json:"begin,omitempty"`
	End      float64 `json:"end,omitempty"`
}

// TimeManagementSummary is an aggregated view of an Activity/Location.
type TimeManagementSummary struct {
	Activity string  `json:"activity"`
	AreaName string  `json:"area"`
	Seconds  float64 `json:"seconds,omitempty"`
}

func (tf *TimeManagementFrame) canMerge(o *TimeManagementFrame) bool {
	if tf.Activity != o.Activity {
		return false
	}
	if tf.AreaName != o.AreaName {
		return false
	}
	if tf.End < o.Begin-1 {
		return false
	}
	return true
}

func (tf *TimeManagementFrame) truncate(atSec float64) {
	if atSec < tf.End {
		tf.End = atSec
	}
}

type timeManagementState struct {
	participantID int64
	teamID        int64
	lastAreaName  string
	lastActivity  string
	isDead        bool
	frames        []TimeManagementFrame
}

func NewTimeManagement(participantID int64) *timeManagementState {
	return &timeManagementState{
		participantID: participantID,
		teamID:        baseview.TeamID(participantID),
		frames:        []TimeManagementFrame{},
	}
}

// AllFrames returns the 'official' time management frames. It modifies the
// raw list of frames by:
// (1) Removing 'jitter' which is any activity that lasted < 1 second.
// (2) Adding 'roaming' events.
func (t *timeManagementState) AllFrames() []TimeManagementFrame {
	res := []TimeManagementFrame{}
	for i := range t.frames {
		if t.frames[i].End-t.frames[i].Begin < 1 {
			continue
		}
		res = append(res, t.frames[i])
	}
	return res
}

func (t *timeManagementState) AddAttack(sec float64, targetID int64, targetType baseview.ActorType) {
	if vicTeam := baseview.TeamID(targetID); vicTeam > 0 && vicTeam != t.teamID {
		// We'll add "fight" in the AddSupplemental() method, using team_fights
		return
	} else if targetType == baseview.ActorTurret {
		t.activity("siege", sec, 5)
	} else if targetID == baseview.MonsterRiftHerald || targetID == baseview.MonsterBaron {
		t.activity("baron", sec, 5)
	} else if targetID == baseview.MonsterDragonElemental || targetID == baseview.MonsterDragonElder {
		t.activity("dragon", sec, 5)
	} else if targetType == baseview.ActorMinion || string(targetType) == "" {
		// see: https://github.com/VantageSports/lolstats/pull/184 for
		// discussion regarding reason for making all non-targeted attacks
		// "farm" events.
		t.activity("farm", sec, 2)
	}
}

func (t *timeManagementState) AddDeath(sec float64) {
	if !t.isDead {
		t.isDead = true
		t.activity("dead", sec, 120)
	}
}

func (t *timeManagementState) AddDamage(sec float64, attacker int64, attackerType baseview.ActorType) {
	switch {
	case baseview.TeamID(attacker) > 0:
		// We'll add "fight" in the AddSupplemental() method, using team_fights
	case attacker == baseview.MonsterRiftHerald || attacker == baseview.MonsterBaron:
		t.activity("baron", sec, 5)
	case attacker == baseview.MonsterDragonElemental || attacker == baseview.MonsterDragonElder:
		t.activity("dragon", sec, 5)
	case attackerType == baseview.ActorTurret:
		t.activity("siege", sec, 5)
	}
}

func (t *timeManagementState) AddStateUpdate(sec float64, pos baseview.Position, health float64) {
	if t.isDead && health > 0 {
		t.isDead = false
		t.activity("", sec, 0.1) // will be removed by
	}

	name := areaName(baseview.AreaID(pos))
	t.area(name, sec)
}

func (t *timeManagementState) AddWard(sec float64) {
	t.activity("ward", sec, 5)
}

// UsefulPercent takes an array of TimeManagementFrame and returns a map
// describing the "useful" percentage of all events by game stage.
func UsefulPercent(in []TimeManagementFrame, totalSecs float64) map[string]float64 {
	usefulTotal := map[string]float64{}

	for _, f := range in {
		for _, stageKey := range []string{"all", gameStage(f.Begin)} {
			if isUsefulActivity(f.Activity) {
				usefulTotal[stageKey] += f.End - f.Begin
			}
		}
	}

	usefulPercent := map[string]float64{}
	for stage, total := range usefulTotal {
		usefulPercent[stage] = 100 * total / stageDurationSecs(stage, totalSecs)
	}

	return usefulPercent
}

func isUsefulActivity(activity string) bool {
	switch activity {
	case "baron", "dragon", "farm", "fight", "hide", "siege", "tower", "ward":
		return true
	case "dead", "roam":
		return false
	default:
		log.Warning("unknown activity: " + activity)
		return false
	}
}

// AddSupplemental adds time management frames that are determined outside the
// normal event flow. Currently this includes fight (since that is calculated
// independently anyway) and roam (which we consider the )
func (t *timeManagementState) AddSupplemental(fights []*TeamFight) {
	t.addFighting(fights)
	t.addRoaming()
}

// addFighting adds "fight" frames using the team fights slice computed
// elsewhere in advanced_stats generation. Any other activity occurring during
// a fight (e.g. farm) is removed.
func (t *timeManagementState) addFighting(fights []*TeamFight) {
	newFrames := []TimeManagementFrame{}
	last := &TimeManagementFrame{AreaName: "base", End: 0.0}

	for i := range t.frames {
		frame := t.frames[i]
		// iterate through all the fights I participanted in that started before
		// the first frame
		for len(fights) > 0 && fights[0].Begin <= frame.Begin {
			if !isRelevant(fights[0], t.participantID) {
				fights = fights[1:]
				continue
			}
			if last.End > fights[0].Begin {
				last.End = fights[0].Begin
			}
			newFrames = append(newFrames, TimeManagementFrame{
				Activity: "fight",
				AreaName: last.AreaName,
				Begin:    fights[0].Begin,
				End:      fights[0].End,
			})
			last = &newFrames[len(newFrames)-1]
			fights = fights[1:]
		}

		// Add the original frame if it was a death OR it started after the last event.
		if frame.Activity == "dead" {
			last.truncate(frame.Begin)
		} else if frame.Begin < last.End {
			continue
		}

		newFrames = append(newFrames, frame)
		last = &newFrames[len(newFrames)-1]
	}
	t.frames = newFrames
}

func isRelevant(f *TeamFight, pID int64) bool {
	_, found := f.ParticipantIDs[pID]
	return found
}

// addRoaming adds "roam" frames when a champion has been in the jungle not
// doing anything else for 8+ seconds.
func (t *timeManagementState) addRoaming() {
	newFrames := []TimeManagementFrame{}

	var last *TimeManagementFrame
	for i := range t.frames {
		frame := t.frames[i]
		if last != nil && last.AreaName == "jungle" && frame.AreaName == "jungle" && frame.Begin-last.End > 8 {
			newFrames = append(newFrames, TimeManagementFrame{
				Activity: "roam",
				AreaName: "jungle",
				Begin:    last.End + 3,
				End:      frame.Begin - 3,
			})
		}
		newFrames = append(newFrames, frame)
		last = &frame
	}

	t.frames = newFrames
}

//
// internal / utility functions
//

// activity adds an event signifying that the champ has initiated a new activity
// reusing the same area.
func (t *timeManagementState) activity(activity string, start, duration float64) {
	if last := t.last(); last != nil {
		last.truncate(start)
	}

	f := TimeManagementFrame{
		Activity: activity,
		AreaName: t.lastAreaName,
		Begin:    start,
		End:      start + duration,
	}
	if last := t.last(); last != nil && last.canMerge(&f) {
		last.End = f.End
	} else {
		t.frames = append(t.frames, f)
	}
	t.lastActivity = activity
}

// area adds a zero-duration event signifying that the champ has traveled to a
// new map area, reusing the last activity.
// NOTE: most of these events will be removed in AllFrames() filtering, but
// it's useful in the meantime for updating the lastAreaName state.
func (t *timeManagementState) area(areaName string, start float64) {
	if areaName == t.lastAreaName {
		return
	} else if areaName == "base" {
		// if you've recalled, reset the activity.
		t.lastActivity = ""
		return
	}

	if last := t.last(); last != nil {
		last.truncate(start)
	}

	f := TimeManagementFrame{
		Activity: t.lastActivity,
		AreaName: areaName,
		Begin:    start,
		End:      start,
	}
	t.frames = append(t.frames, f)
	t.lastAreaName = areaName
}

func (t *timeManagementState) last() *TimeManagementFrame {
	if len(t.frames) == 0 {
		return nil
	}
	return &t.frames[len(t.frames)-1]
}

func areaName(areaID int64) string {
	return strings.ToLower(string(baseview.AreaToRegion(areaID)))
}
