package event

import (
	"strings"

	"github.com/VantageSports/lolstats/baseview"
)

type EloEvent interface {
	Type() string
	Time() float64
	SetTime(float64)
	MatchSeconds() float64
	SetMatchSeconds(float64)
}

type baseEvent struct {
	Event         string  `json:"event" mapstructure:"event"`
	Time_         float64 `json:"time" mapstructure:"time"`
	MatchSeconds_ float64 `json:"match_seconds" mapstructure:"match_seconds"`
}

func (be *baseEvent) Type() string              { return be.Event }
func (be *baseEvent) Time() float64             { return be.Time_ }
func (be *baseEvent) SetTime(f float64)         { be.Time_ = f }
func (be *baseEvent) MatchSeconds() float64     { return be.MatchSeconds_ }
func (be *baseEvent) SetMatchSeconds(f float64) { be.MatchSeconds_ = f }

type BasicAttack struct {
	baseEvent       `mapstructure:",squash"`
	SenderName      string            `json:"sender" mapstructure:"sender"`
	NetworkID       int64             `json:"network_id" mapstructure:"network_id"`
	TargetName      string            `json:"target" mapstructure:"target"`
	TargetNetworkID int64             `json:"target_network_id" mapstructure:"target_network_id"`
	TargetPosition  baseview.Position `json:"target_position" mapstructure:"target_position"`
}

type ChampDie struct {
	baseEvent `mapstructure:",squash"`
	NetworkID int64 `json:"network_id" mapstructure:"network_id"`
}

type ChampKill struct {
	baseEvent `mapstructure:",squash"`
	NetworkID int64 `json:"network_id" mapstructure:"network_id"`
}

type Damage struct {
	baseEvent       `mapstructure:",squash"`
	Damage          float64 `json:"damage" mapstructure:"damage"`
	SenderName      string  `json:"sender" mapstructure:"sender"`
	NetworkID       int64   `json:"network_id" mapstructure:"network_id"`
	TargetName      string  `json:"target" mapstructure:"target"`
	TargetNetworkID int64   `json:"target_network_id" mapstructure:"target_network_id"`
	DamageType      string  `json:"type" mapstructure:"type"`
}

type Die struct {
	baseEvent `mapstructure:",squash"`
	NetworkID int64 `json:"network_id" mapstructure:"network_id"`
}

type EpicMonsterDeath struct {
	baseEvent  `mapstructure:",squash"`
	SenderName string `json:"sender" mapstructure:"sender"` // monster name
}

type EpicMonsterKill struct {
	baseEvent `mapstructure:",squash"`
	SenderID  int64 `json:"sender_id" mapstructure:"sender_id"` // monster id
	KillerID  int64 `json:"killer_id" mapstructure:"killer_id"`
}

type GameEnd struct {
	baseEvent `mapstructure:",squash"`
}

type NetworkIDMapping struct {
	baseEvent  `mapstructure:",squash"`
	SenderName string `json:"name" mapstructure:"name"`
	NetworkID  int64  `json:"network_id" mapstructure:"network_id"`
	EloType    string `json:"elo_type" mapstructure:"elo_type"`
}

type Kill struct {
	baseEvent `mapstructure:",squash"`
	NetworkID int64 `json:"network_id" mapstructure:"network_id"`
}

type LevelUp struct {
	baseEvent  `mapstructure:",squash"`
	Level      int64  `json:"level" mapstructure:"level"`
	NetworkID  int64  `json:"network_id" mapstructure:"network_id"`
	SenderName string `json:"sender" mapstructure:"sender"`
}

type NexusDestroyed struct {
	baseEvent `mapstructure:",squash"`
	Nexus     string `json:"nexus" mapstructure:"nexus"`
}

type OnCreate struct {
	baseEvent  `mapstructure:",squash"`
	NetworkID  int64             `json:"network_id" mapstructure:"network_id"`
	SenderName string            `json:"sender" mapstructure:"sender"`
	TeamID     int64             `json:"team_id" mapstructure:"team_id"`
	Position   baseview.Position `json:"position" mapstructure:"position"`
}

type OnDelete struct {
	baseEvent  `mapstructure:",squash"`
	NetworkID  int64             `json:"network_id" mapstructure:"network_id"`
	SenderName string            `json:"sender" mapstructure:"sender"`
	TeamID     int64             `json:"team_id" mapstructure:"team_id"`
	Position   baseview.Position `json:"position" mapstructure:"position"`
}

type Ping struct {
	baseEvent            `mapstructure:",squash"`
	ChampionID           int64             `json:"champion_id" mapstructure:"champion_id"`
	Dead                 bool              `json:"dead" mapstructure:"dead"`
	Gold                 int64             `json:"gold" mapstructure:"gold"`
	Health               float64           `json:"health" mapstructure:"health"`
	HealthMax            float64           `json:"health_max" mapstructure:"health_max"`
	InGrass              bool              `json:"in_grass" mapstructure:"in_grass"`
	NetworkID            int64             `json:"network_id" mapstructure:"network_id"`
	HeroIndex            int64             `json:"hero_index" mapstructure:"hero_index"`
	Level                int64             `json:"level" mapstructure:"level"`
	Mana                 float64           `json:"mana" mapstructure:"mana"`
	ManaMax              float64           `json:"mana_max" mapstructure:"mana_max"`
	MinionsKilled        int64             `json:"minions_killed" mapstructure:"minions_killed"`
	NeutralMinionsKilled int64             `json:"neutral_minions_killed" mapstructure:"neutral_minions_killed"`
	Name                 string            `json:"name" mapstructure:"name"`
	Position             baseview.Position `json:"position" mapstructure:"position"`
	SlotQLevel           int64             `json:"q_level" mapstructure:"q_level"`
	SlotQCooldownExp     float64           `json:"q_exp" mapstructure:"q_exp"`
	SlotWLevel           int64             `json:"w_level" mapstructure:"w_level"`
	SlotWCooldownExp     float64           `json:"w_exp" mapstructure:"w_exp"`
	SlotELevel           int64             `json:"e_level" mapstructure:"e_level"`
	SlotECooldownExp     float64           `json:"e_exp" mapstructure:"e_exp"`
	SlotRLevel           int64             `json:"r_level" mapstructure:"r_level"`
	SlotRCooldownExp     float64           `json:"r_exp" mapstructure:"r_exp"`
	SlotS1Level          int64             `json:"s1_level" mapstructure:"s1_level"`
	SlotS1CooldownExp    float64           `json:"s1_exp" mapstructure:"s1_exp"`
	SlotS2Level          int64             `json:"s2_level" mapstructure:"s2_level"`
	SlotS2CooldownExp    float64           `json:"s2_exp" mapstructure:"s2_exp"`
	UnderTurret          bool              `json:"under_turret" mapstructure:"under_turret"`
	UnderEnemyTurret     bool              `json:"under_enemy_turret" mapstructure:"under_enemy_turret"`
}

type SpellCast struct {
	baseEvent       `mapstructure:",squash"`
	PositionStart   baseview.Position `json:"start_position" mapstructure:"start_position"`
	PositionEnd     baseview.Position `json:"end_position" mapstructure:"end_position"`
	Level           int64             `json:"level" mapstructure:"level"`
	SpellName       string            `json:"name" mapstructure:"name"`
	SenderName      string            `json:"sender" mapstructure:"sender"`
	NetworkID       int64             `json:"network_id" mapstructure:"network_id"`
	Slot            string            `json:"slot" mapstructure:"slot"`
	TargetName      string            `json:"target" mapstructure:"target"`
	TargetNetworkID int64             `json:"target_network_id" mapstructure:"target_network_id"`
}

func (s *SpellCast) WardItemType() *baseview.WardItemType {
	res := baseview.WardItemType("")
	switch strings.ToLower(s.SpellName) {
	case "itemghostward":
		res = baseview.SightWard
	case "trinketorblvl3":
		res = baseview.BlueTrinket
	case "trinkettotemlvl1":
		res = baseview.YellowTrinket
	case "jammerdevice":
		res = baseview.VisionWard
	default:
		return nil
	}
	return &res
}
