// baseview format defines the raw, detailed match data that we extract from
// methods such as ocr, manual, or other programmatic means. We use the structs
// defined here to calculate advanced statistics for summoners.
//
// Right now the baseview format is closely tied to the process(es) we use to
// generate the data (e.b.), but it should exist (and evolve) independently of
// the mechanism we use to generate it. E.g. whether we generate data via ocr,
// manual tagging, or decoding raw data packets, we should be populating the
// events defined here so that downstream processes don't need to know _how_
// we're collecting the data, and aggregate stats etc. without knowledge of
// those details.
//
// Note: the baseview package is imported by clients outside this repo (e.g. lolocr),
// so no imports should be added to this package that we wouldn't want
// those external clients to depend on.

package baseview

import (
	"encoding/json"
	"fmt"
	"math"
	"time"
)

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// Distance returns the euclidian distance between two points.
func (p *Position) Distance(o Position) float64 {
	return math.Sqrt(math.Pow(p.X-o.X, 2) + math.Pow(p.Y-o.Y, 2) + math.Pow(p.Z-o.Z, 2))
}

// DistanceXY returns the euclidian distance on the XY plane (ignoring Z values)
func (p *Position) DistanceXY(o Position) float64 {
	return math.Sqrt(math.Pow(p.X-o.X, 2) + math.Pow(p.Y-o.Y, 2))
}

type Participant struct {
	ParticipantID int64  `json:"participant_id"`
	SummonerID    int64  `json:"summoner_id"`
	SummonerName  string `json:"summoner_name"`
	ChampionID    int64  `json:"champion_id"`
	Tier          string `json:"tier"`
	Division      string `json:"division"`
	Spell1        string `json:"spell_1"`
	Spell2        string `json:"spell_2"`
}

type Baseview struct {
	LastUpdated  time.Time     `json:"last_updated"`
	Participants []Participant `json:"participants"`
	Events       []Event       `json:"events"`
}

func (b *Baseview) UnmarshalJSON(data []byte) error {
	in := struct {
		Updated      time.Time         `json:"last_updated"`
		Participants []Participant     `json:"participants"`
		Events       []json.RawMessage `json:"events"`
	}{}
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}
	events := []Event{}
	for _, msg := range in.Events {
		event, err := parseEvent(msg)
		if err != nil {
			return err
		}
		events = append(events, event)
	}
	b.LastUpdated = in.Updated
	b.Participants = in.Participants
	b.Events = events
	return nil
}

func parseEvent(msg json.RawMessage) (Event, error) {
	m := map[string]interface{}{}
	if err := json.Unmarshal(msg, &m); err != nil {
		return nil, err
	}
	var e Event
	switch m["event"].(string) {
	case "attack":
		e = &Attack{}
	case "building_kill":
		e = &BuildingKill{}
	case "damage":
		e = &Damage{}
	case "death":
		e = &Death{}
	case "epic_monster_kill":
		e = &EpicMonsterKill{}
	case "game_end":
		e = &GameEnd{}
	case "item_used":
		e = &ItemUsed{}
	case "level_up":
		e = &LevelUp{}
	case "slot_upgrade":
		e = &SlotUpgrade{}
	case "state_update":
		e = &StateUpdate{}
	case "spawn":
		e = &Spawn{}
	case "ward_placed":
		e = &WardPlaced{}
	case "ward_death":
		e = &WardDeath{}
	default:
		return nil, fmt.Errorf("unknown event type: %v", m["event"])
	}
	err := json.Unmarshal(msg, e)
	return e, err
}

func MatchDuration(events []Event) float64 {
	for i := len(events) - 1; i >= 0; i-- {
		switch t := events[i].(type) {
		case *StateUpdate, *GameEnd:
			continue
		default:
			return t.Seconds()
		}
	}
	return 0.0
}

type Event interface {
	Type() string
	SetType(string)
	Seconds() float64
	SetSeconds(float64)
}

type baseEvent struct {
	EventType_ string  `json:"event" mapstructure:"event"`
	Seconds_   float64 `json:"seconds" mapstructure:"seconds"`
}

func (be *baseEvent) Type() string         { return be.EventType_ }
func (be *baseEvent) SetType(s string)     { be.EventType_ = s }
func (be *baseEvent) Seconds() float64     { return be.Seconds_ }
func (be *baseEvent) SetSeconds(s float64) { be.Seconds_ = s }

type Attack struct {
	baseEvent
	CooldownExpires float64   `json:"cooldown_expires,omitempty"`
	Start           *Position `json:"start,omitempty"`
	End             *Position `json:"end,omitempty"`
	AttackerID      int64     `json:"attacker_id,omitempty"`
	AttackerType    ActorType `json:"attacker_type,omitempty"`
	Slot            string    `json:"slot"`
	TargetID        int64     `json:"target_id,omitempty"`
	TargetType      ActorType `json:"target_type,omitempty"`
	TargetPosition  *Position `json:"target_position,omitempty"`
}

type BuildingKill struct {
	baseEvent
	BuildingType ActorType `json:"building_type"`
	BuildingID   int64     `json:"building_id"`
}

type Damage struct {
	baseEvent
	Total        float64   `json:"total"`
	Percent      float64   `json:"percent"`
	AttackerID   int64     `json:"attacker_id,omitempty"`
	AttackerType ActorType `json:"attacker_type,omitempty"`
	VictimID     int64     `json:"victim_id"`
	VictimType   ActorType `json:"victim_type"`
}

type Death struct {
	baseEvent
	Position Position `json:"position"`
	VictimID int64    `json:"victim_id"`
}

type EpicMonsterKill struct {
	baseEvent
	VictimID int64 `json:"victim_id"`
	KillerID int64 `json:"killer_id"`
}

type GameEnd struct {
	baseEvent
	Reason string `json:"reason"`
}

type ItemUsed struct {
	baseEvent
	Slot            string  `json:"slot"`
	CooldownExpires float64 `json:"cooldown_expires"`
}

type LevelUp struct {
	baseEvent
	Level         int64 `json:"level"`
	ParticipantID int64 `json:"participant_id"`
}

type SlotUpgrade struct {
	baseEvent
	Level         int64  `json:"level"`
	Slot          string `json:"slot"`
	ParticipantID int64  `json:"participant_id"`
}

type StateUpdate struct {
	baseEvent
	Gold                 int64    `json:"gold"`
	Health               float64  `json:"health"`
	HealthMax            float64  `json:"health_max"`
	InGrass              bool     `json:"in_grass"`
	MinionsKilled        int64    `json:"minions_killed"`
	NeutralMinionsKilled int64    `json:"neutral_minions_killed"`
	Mana                 float64  `json:"mana"`
	ManaMax              float64  `json:"mana_max"`
	ParticipantID        int64    `json:"participant_id"`
	Position             Position `json:"position"`
	UnderOwnTurret       bool     `json:"under_own_turret"`
	UnderEnemyTurret     bool     `json:"under_enemy_turret"`
}

type Spawn struct {
	baseEvent
	ParticipantID int64    `json:"participant_id"`
	Position      Position `json:"position"`
}

type WardPlaced struct {
	baseEvent
	WardType      string       `json:"type"`
	WardItemType  WardItemType `json:"item_type"`
	ParticipantID int64        `json:"participant_id"`
	Position      Position     `json:"position"`
	TeamID        int64        `json:"team_id"`
	WardID        int64        `json:"ward_id"`
}

type WardDeath struct {
	baseEvent
	WardID int64 `json:"ward_id"`
}
