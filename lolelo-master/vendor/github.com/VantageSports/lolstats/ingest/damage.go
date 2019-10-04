package ingest

import "strconv"

// DamageMap maps from champion id to gameStage to DamageSummary. And yes,
// champion id is an integer, but json objects are required to have string keys,
// so we suffer silently.
type DamageMap map[string]map[string]*DamageSummary

func (dm DamageMap) AddDamage(champID int64, seconds, total, percent float64) {
	dm.addSummary(champID, 0, seconds, total, percent)
}

func (dm DamageMap) AddDeath(champID int64, seconds float64) {
	dm.addSummary(champID, 1, seconds, 0, 0)
}

func (dm DamageMap) addSummary(champID, deaths int64, seconds, damageTotal, damagePercent float64) {
	champKey := strconv.FormatInt(champID, 10)
	stageSummary := dm[champKey]
	if stageSummary == nil {
		stageSummary = map[string]*DamageSummary{}
		dm[champKey] = stageSummary
	}

	stage := gameStage(seconds)
	summary := stageSummary[stage]
	if summary == nil {
		summary = &DamageSummary{}
		stageSummary[stage] = summary
	}

	summary.Deaths += deaths
	summary.DamageTotal += damageTotal
	summary.DamagePercent += damagePercent
}

type DamageSummary struct {
	// DamageTotal is the absolute amount of damage done. E.g. if some
	// damage caused a champion's health to go from 88 to 64, DamageTotal
	// would be 24.0.
	DamageTotal float64 `json:"damage_total"`

	// DamagePercent is the percent of damage done. E.g. if some damage
	// caused a champion's health to go from 88 to 64, and their max health
	// at the time they received the damage was 300, the DamagePercent
	// value would be 8.0.
	DamagePercent float64 `json:"damage_percent"`

	// Deaths is total deaths
	Deaths int64 `json:"deaths,omitempty"`
}
