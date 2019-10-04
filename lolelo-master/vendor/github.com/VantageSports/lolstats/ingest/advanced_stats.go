package ingest

import (
	"fmt"
	"math"
	"time"

	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/riot"
)

// advancedStatsState is the "bucket o' data" that we don't plan on exporting,
// but is useful state for calculating advanced stats.
type advancedStatsState struct {
	AttacksTotal            int64
	DamageTakenPercentTotal float64
	// LastChampDamage is a map from victim participantID to the last champ
	// attack upon that champion. It is used to attribute deaths, which are
	// given to the last champ to cause damage (within 13 seconds).
	LastChampDamage   map[int64]baseview.Damage
	MatchDuration     float64
	Deaths            int64
	MinionStatTracker *MinionStatTracker
	BuddyFinder       *BuddyFinder
}

type AdvancedStats struct {
	ParticipantID int64         `json:"participant_id,omitempty"`
	SummonerID    int64         `json:"summoner_id,omitempty"`
	TeamID        int64         `json:"team_id,omitempty"`
	MatchID       int64         `json:"match_id,omitempty"`
	PlatformID    riot.Platform `json:"platform_id,omitempty"`
	LastUpdated   time.Time     `json:"last_updated,omitempty"`

	Positions    []PositionTime        `json:"champ_positions,omitempty"`
	RolePosition baseview.RolePosition `json:"role_position,omitempty"`

	// These damage numbers include champ-on-champ damage only.
	DamageDealt                DamageMap `json:"damage_dealt,omitempty"`
	DamageTaken                DamageMap `json:"damage_taken,omitempty"`
	DamageTakenPercentPerDeath float64   `json:"damage_taken_percent_per_death"`
	CarryFocusEfficiency       float64   `json:"carry_focus_efficiency"`

	AttacksPerMinute   float64            `json:"attacks_per_minute,omitempty"`
	AbilityCounts      map[string]int64   `json:"ability_counts,omitempty"`
	AbilityCounts0to10 map[string]int64   `json:"ability_counts_zero_to_ten,omitempty"`
	MapCoverages       map[string]float64 `json:"map_coverages,omitempty"`

	TeamComp *TeamComp `json:"team_comp,omitempty"`

	TimeDetail    []TimeManagementFrame `json:"time_detail,omitempty"`
	UsefulPercent map[string]float64    `json:"useful_percent,omitempty"`

	TeamFights            []*TeamFight       `json:"team_fights,omitempty"`
	FavorableTeamFights   TeamFightAggregate `json:"favorable_team_fights,omitempty"`
	BalancedTeamFights    TeamFightAggregate `json:"balanced_team_fights,omitempty"`
	UnfavorableTeamFights TeamFightAggregate `json:"unfavorable_team_fights,omitempty"`

	WardLives             map[int64][]*JsonWardLife `json:"ward_lives,omitempty"`
	RevealsPerWardAverage float64                   `json:"reveals_per_ward_average"`
	LiveWardsAverage      map[string]float64        `json:"live_wards_average,omitempty"`

	FavorableFightPercent float64           `json:"favorable_team_fight_percent"`
	GoodKills             SignificantFights `json:"good_kills"`
	BadDeaths             SignificantFights `json:"bad_deaths"`

	Combos               []*ComboSummary `json:"combos,omitempty"`
	ComboDamagePerMinute float64         `json:"combo_damage_per_minute"`
}

func (s *AdvancedStats) TrimNonStats() {
	s.Positions = nil
	s.DamageDealt = nil
	s.DamageTaken = nil
	s.TeamComp = nil
	s.TimeDetail = nil
	s.TeamFights = nil
	s.WardLives = nil
	s.Combos = nil
}

type PositionTime struct {
	Seconds  float64           `json:"seconds"`
	Position baseview.Position `json:"position"`
	Type     string            `json:"type,omitempty"`
}

func gameStage(seconds float64) string {
	minutes := seconds / 60
	switch {
	case minutes >= 25:
		return "late"
	case minutes >= 10:
		return "mid"
	default:
		return "early"
	}
}

func stageDurationSecs(gameStage string, matchDurationSecs float64) float64 {
	switch gameStage {
	case "early":
		return 600
	case "mid":
		return 900
	case "late":
		return matchDurationSecs - 1500
	}
	return matchDurationSecs
}

func ComputeAdvanced(bv baseview.Baseview, summonerID int64, matchID int64, platformID riot.Platform) (*AdvancedStats, error) {
	byPID, p := buildPIDMap(bv.Participants, summonerID)
	if p == nil {
		return nil, fmt.Errorf("no participant found with summoner id %d", summonerID)
	}

	state := advancedStatsState{
		LastChampDamage:   map[int64]baseview.Damage{},
		MatchDuration:     baseview.MatchDuration(bv.Events),
		MinionStatTracker: NewMinionStatTracker(),
		BuddyFinder:       NewBuddyFinder(),
	}

	res := AdvancedStats{
		ParticipantID: p.ParticipantID,
		SummonerID:    summonerID,
		TeamID:        baseview.TeamID(p.ParticipantID),
		MatchID:       matchID,
		PlatformID:    platformID,
		LastUpdated:   time.Now(),

		AbilityCounts:      map[string]int64{"Q": 0, "W": 0, "E": 0, "R": 0},
		AbilityCounts0to10: map[string]int64{"Q": 0, "W": 0, "E": 0, "R": 0},
		DamageDealt:        DamageMap{},
		DamageTaken:        DamageMap{},
		TeamComp:           &TeamComp{},
	}

	for _, eachP := range byPID {
		res.TeamComp.AddParticipant(*p, *eachP)
	}

	coverageMap := NewCoverageMap(12)
	wardAnalysis := NewWardAnalysis()
	teamFights := NewTeamFightAnalysis(p.ParticipantID, state.MatchDuration, wardAnalysis)
	timeManagement := NewTimeManagement(p.ParticipantID)
	comboAnalysis := NewComboAnalysis(p)

	for i := range bv.Events {
		e := bv.Events[i]
		switch t := e.(type) {

		case *baseview.Attack:
			wardAnalysis.AddAttack(t)
			state.MinionStatTracker.AddAttack(t)
			teamFights.AddAttack(t)
			if t.AttackerID != p.ParticipantID {
				continue
			}
			state.AttacksTotal++
			if t.Slot == "Q" || t.Slot == "W" || t.Slot == "E" || t.Slot == "R" {
				res.AbilityCounts[t.Slot]++
				if t.Seconds() <= 600 {
					res.AbilityCounts0to10[t.Slot]++
				}
			}
			comboAnalysis.AddAttack(t)
			timeManagement.AddAttack(t.Seconds(), t.TargetID, t.TargetType)

		case *baseview.Damage:
			state.MinionStatTracker.AddDamage(t)
			attacker := byPID[t.AttackerID]
			victim := byPID[t.VictimID]

			if victim == nil {
				// this shouldn't happen.
				continue
			}

			if attacker != nil {
				state.LastChampDamage[victim.ParticipantID] = *t
			}
			if attacker != nil && attacker != victim {
				if t.AttackerID == p.ParticipantID {
					res.DamageDealt.AddDamage(victim.ChampionID, t.Seconds(), t.Total, t.Percent)
					comboAnalysis.AddDamage(t)
				}
				if t.VictimID == p.ParticipantID {
					res.DamageTaken.AddDamage(attacker.ChampionID, t.Seconds(), t.Total, t.Percent)
					state.DamageTakenPercentTotal += t.Percent
				}
			}

			if t.VictimID == p.ParticipantID {
				timeManagement.AddDamage(t.Seconds(), t.AttackerID, t.AttackerType)
			}

			// Significant sources of damage to players from heros, turrets, and
			// monsters should be considered for team fights.
			if (attacker != nil && baseview.TeamID(victim.ParticipantID) != baseview.TeamID(attacker.ParticipantID)) ||
				t.AttackerType == baseview.ActorTurret || baseview.IsEpicMonster(t.AttackerID) {
				teamFights.AddDamage(t)
			}

		case *baseview.StateUpdate:
			teamFights.AddStateUpdate(t)
			wardAnalysis.AddStateUpdate(t)
			state.MinionStatTracker.AddStateUpdate(t)
			state.BuddyFinder.AddStateUpdate(t)
			if t.ParticipantID != p.ParticipantID {
				continue
			}
			if (len(res.Positions) == 0 || t.Seconds()-res.Positions[len(res.Positions)-1].Seconds > 3) &&
				t.Health > 0 {
				res.Positions = append(res.Positions, PositionTime{
					Position: t.Position,
					Seconds:  t.Seconds(),
				})
			}
			coverageMap.Add(t.Seconds(), t.Position)
			timeManagement.AddStateUpdate(t.Seconds(), t.Position, t.Health)

		case *baseview.WardPlaced:
			wardAnalysis.AddWardPlaced(t, state.MatchDuration)
			if t.ParticipantID != p.ParticipantID {
				continue
			}
			timeManagement.AddWard(t.Seconds())

		case *baseview.WardDeath:
			if err := wardAnalysis.AddWardDeath(t); err != nil {
				return nil, err
			}
		case *baseview.Death:
			teamFights.AddDeath(t)
			victim := byPID[t.VictimID]

			var killer *baseview.Participant
			if lastDamage, found := state.LastChampDamage[t.VictimID]; found {
				// If a champ took damaged from another champ within the last 13
				// seconds, kill credit goes to that champ.
				if t.Seconds()-lastDamage.Seconds() < 13 {
					killer = byPID[lastDamage.AttackerID]
				}
			}

			if killer != nil && killer.ParticipantID == p.ParticipantID {
				res.DamageDealt.AddDeath(victim.ChampionID, t.Seconds())
			}

			if t.VictimID != p.ParticipantID {
				continue
			}
			if killer != nil {
				res.DamageTaken.AddDeath(killer.ChampionID, t.Seconds())
			}
			state.Deaths++
			timeManagement.AddDeath(t.Seconds())
		case *baseview.LevelUp:
			state.MinionStatTracker.AddLevelUp(t)
			teamFights.AddLevelUp(t)
		case *baseview.BuildingKill:
			if t.BuildingType == "turret" {
				teamFights.AddTurretKill(t)
			}
		}

	}

	res.AttacksPerMinute = float64(state.AttacksTotal) / (state.MatchDuration / 60.0)
	res.FavorableTeamFights = teamFights.Aggregate(FavorableFights)
	res.BalancedTeamFights = teamFights.Aggregate(BalancedFights)
	res.UnfavorableTeamFights = teamFights.Aggregate(UnfavorableFights)
	res.GoodKills, res.BadDeaths = teamFights.SignificantKillsAndDeaths()
	res.TeamFights = teamFights.TeamFights
	res.FavorableFightPercent = 100.0 * float64(res.FavorableTeamFights.Count) / math.Max(float64(len(res.TeamFights)), 1)

	res.WardLives = wardAnalysis.ExportPlayerWards()
	res.RevealsPerWardAverage = wardAnalysis.RevealsPerWardAverage(p.ParticipantID)
	res.LiveWardsAverage = wardAnalysis.LiveWardsAverage(p.ParticipantID, state.MatchDuration)

	rolePositions, err := CalculateRolePosition(state.MinionStatTracker, state.BuddyFinder)
	if err != nil {
		return nil, err
	}

	res.CarryFocusEfficiency = CalculateCarryFocusEfficiency(res.TeamFights, rolePositions, state.MatchDuration)

	res.DamageTakenPercentPerDeath = state.DamageTakenPercentTotal / math.Max(float64(state.Deaths), 1)

	res.MapCoverages = coverageMap.Percents()

	res.TeamComp.ComputeKeyAttributes()

	timeManagement.AddSupplemental(res.TeamFights)
	res.TimeDetail = timeManagement.AllFrames()
	res.UsefulPercent = UsefulPercent(res.TimeDetail, state.MatchDuration)

	res.Combos = comboAnalysis.ComboSummary()
	res.ComboDamagePerMinute = comboAnalysis.ComboDamagePerMinute(state.MatchDuration)

	res.RolePosition = rolePositions[p.ParticipantID]

	return &res, nil
}

func buildPIDMap(participants []baseview.Participant, summonerID int64) (map[int64]*baseview.Participant, *baseview.Participant) {
	byPID := map[int64]*baseview.Participant{}
	var p *baseview.Participant

	for i := range participants {
		tmpP := participants[i]
		if tmpP.SummonerID == summonerID {
			p = &tmpP
		}
		byPID[tmpP.ParticipantID] = &tmpP
	}

	return byPID, p
}

func CalculateRolePosition(minionStatTracker *MinionStatTracker, buddyFinder *BuddyFinder) (map[int64]baseview.RolePosition, error) {
	roles := map[int64]baseview.RolePosition{}

	midCS := map[int64]int64{}
	totalCS := map[int64]int64{}
	for p, levelMap := range minionStatTracker.MinionsKilled {
		for level, regionMap := range levelMap {
			// Role positions only look at cs up to level 6
			if level > 6 {
				continue
			}
			if val, ok := regionMap[baseview.RegionMid]; ok {
				midCS[p] += val
			}
			// Loop over all the regions, and add them to the total
			for _, cs := range regionMap {
				totalCS[p] += cs
			}
		}
	}
	jungleAttacks := map[int64]int64{}
	for p, levelMap := range minionStatTracker.JungleMinionInteractions {
		for level, attacks := range levelMap {
			if level > 6 {
				continue
			}
			jungleAttacks[p] += attacks
		}
	}

	// Identify the jungles - most jungle minions minions attacked up to level 6
	roles[findCs(max, jungleAttacks, baseview.BlueTeam, roles)] = baseview.RoleJungle
	roles[findCs(max, jungleAttacks, baseview.RedTeam, roles)] = baseview.RoleJungle

	// Identify the supports - least cs up to level 6
	t1Support := findCs(min, totalCS, baseview.BlueTeam, roles)
	t2Support := findCs(min, totalCS, baseview.RedTeam, roles)
	roles[t1Support] = baseview.RoleSupport
	roles[t2Support] = baseview.RoleSupport

	// Identify the mids - most mid cs up to level 6
	roles[findCs(max, midCS, baseview.BlueTeam, roles)] = baseview.RoleMid
	roles[findCs(max, midCS, baseview.RedTeam, roles)] = baseview.RoleMid

	// The bot role goes to the player who spends the most time with the support in the early game
	roles[buddyFinder.BuddyFor(t1Support, roles)] = baseview.RoleAdc
	roles[buddyFinder.BuddyFor(t2Support, roles)] = baseview.RoleAdc

	// We should have 8 unique assignments (no overlaps). If there are overlaps, then it's an error
	if _, found := roles[-1]; found || len(roles) != 8 {
		// These games are really unstructured, and are typically low level games where people don't know about roles
		return roles, fmt.Errorf("error in role position calculation. expected 8 roles, but only got %d %v", len(roles), roles)
	}
	// Fill in the last 2 roles as "top".
	for i := int64(1); i <= 10; i++ {
		if _, ok := roles[i]; !ok {
			roles[i] = baseview.RoleTop
		}
	}
	return roles, nil
}

type comparator func(int64, int64) bool

func max(a, b int64) bool { return a >= b }
func min(a, b int64) bool { return a <= b }
func findCs(comp comparator, csByPid map[int64]int64, team int64, skip map[int64]baseview.RolePosition) int64 {
	bestP := int64(-1)

	for p := 1; p <= 10; p++ {
		candidateP := int64(p)
		if _, found := skip[candidateP]; found {
			continue
		}
		if baseview.TeamID(candidateP) != team {
			continue
		}
		if bestP == -1 || comp(csByPid[candidateP], csByPid[bestP]) {
			bestP = candidateP
		}
	}
	return bestP
}

// AverageAdvancedStats takes in a bunch of AdvancedStats, and returns an average.
func AverageAdvancedStats(advancedStats []AdvancedStats) AdvancedStats {
	average := AdvancedStats{}

	if len(advancedStats) == 0 {
		return average
	}

	for _, adv := range advancedStats {
		// Set these every time, since they should always be the same
		average.TeamID = adv.TeamID
		average.MatchID = adv.MatchID
		average.PlatformID = adv.PlatformID
		average.LastUpdated = adv.LastUpdated

		if average.DamageDealt == nil {
			average.DamageDealt = DamageMap{}
		}
		if average.DamageTaken == nil {
			average.DamageTaken = DamageMap{}
		}

		// Add up the damageMaps
		for champId, stageMap := range adv.DamageDealt {
			if average.DamageDealt[champId] == nil {
				average.DamageDealt[champId] = stageMap
			} else {
				for stage, summary := range stageMap {
					if average.DamageDealt[champId][stage] == nil {
						average.DamageDealt[champId][stage] = summary
					} else {
						average.DamageDealt[champId][stage].DamageTotal += summary.DamageTotal
						average.DamageDealt[champId][stage].DamagePercent += summary.DamagePercent
						average.DamageDealt[champId][stage].Deaths += summary.Deaths
					}
				}
			}

		}
		for champId, stageMap := range adv.DamageTaken {
			if average.DamageTaken[champId] == nil {
				average.DamageTaken[champId] = stageMap
			} else {
				for stage, summary := range stageMap {
					if average.DamageTaken[champId][stage] == nil {
						average.DamageTaken[champId][stage] = summary
					} else {
						average.DamageTaken[champId][stage].DamageTotal += summary.DamageTotal
						average.DamageTaken[champId][stage].DamagePercent += summary.DamagePercent
						average.DamageTaken[champId][stage].Deaths += summary.Deaths
					}
				}
			}

		}

		average.DamageTakenPercentPerDeath += adv.DamageTakenPercentPerDeath
		average.CarryFocusEfficiency += adv.CarryFocusEfficiency
		average.AttacksPerMinute += adv.AttacksPerMinute

		// Ability Counts
		if average.AbilityCounts == nil {
			average.AbilityCounts = map[string]int64{}
		}
		if average.AbilityCounts0to10 == nil {
			average.AbilityCounts0to10 = map[string]int64{}
		}

		for ability, count := range adv.AbilityCounts {
			average.AbilityCounts[ability] += count
		}
		for ability, count := range adv.AbilityCounts0to10 {
			average.AbilityCounts0to10[ability] += count
		}

		// Map Coverage
		if average.MapCoverages == nil {
			average.MapCoverages = map[string]float64{}
		}
		for stage, coverage := range adv.MapCoverages {
			average.MapCoverages[stage] += coverage
		}

		// Useful Percent
		if average.UsefulPercent == nil {
			average.UsefulPercent = map[string]float64{}
		}
		for stage, useful := range adv.UsefulPercent {
			average.UsefulPercent[stage] += useful
		}

		// Team Fights
		average.FavorableTeamFights.Count += adv.FavorableTeamFights.Count
		average.FavorableTeamFights.TotalKills += adv.FavorableTeamFights.TotalKills
		average.FavorableTeamFights.TotalDeaths += adv.FavorableTeamFights.TotalDeaths
		// Recompute net kills at the end
		// average.FavorableTeamFights.NetKills += adv.FavorableTeamFights.NetKills

		average.BalancedTeamFights.Count += adv.BalancedTeamFights.Count
		average.BalancedTeamFights.TotalKills += adv.BalancedTeamFights.TotalKills
		average.BalancedTeamFights.TotalDeaths += adv.BalancedTeamFights.TotalDeaths

		average.UnfavorableTeamFights.Count += adv.UnfavorableTeamFights.Count
		average.UnfavorableTeamFights.TotalKills += adv.UnfavorableTeamFights.TotalKills
		average.UnfavorableTeamFights.TotalDeaths += adv.UnfavorableTeamFights.TotalDeaths

		// Wards
		average.RevealsPerWardAverage += adv.RevealsPerWardAverage

		if average.LiveWardsAverage == nil {
			average.LiveWardsAverage = map[string]float64{}
		}
		for wardType, liveWards := range adv.LiveWardsAverage {
			average.LiveWardsAverage[wardType] += liveWards
		}

		// Fights
		average.FavorableFightPercent += adv.FavorableFightPercent

		average.GoodKills.Total += adv.GoodKills.Total
		average.GoodKills.NumbersDifference += adv.GoodKills.NumbersDifference
		average.GoodKills.NumbersDifferenceLackingVision += adv.GoodKills.NumbersDifferenceLackingVision
		average.GoodKills.HealthDifference += adv.GoodKills.HealthDifference
		average.GoodKills.GoldSpentDifference += adv.GoodKills.GoldSpentDifference
		average.GoodKills.NeutralDamageDifference += adv.GoodKills.NeutralDamageDifference
		average.GoodKills.Other += adv.GoodKills.Other

		average.BadDeaths.Total += adv.BadDeaths.Total
		average.BadDeaths.NumbersDifference += adv.BadDeaths.NumbersDifference
		average.BadDeaths.NumbersDifferenceLackingVision += adv.BadDeaths.NumbersDifferenceLackingVision
		average.BadDeaths.HealthDifference += adv.BadDeaths.HealthDifference
		average.BadDeaths.GoldSpentDifference += adv.BadDeaths.GoldSpentDifference
		average.BadDeaths.NeutralDamageDifference += adv.BadDeaths.NeutralDamageDifference
		average.BadDeaths.Other += adv.BadDeaths.Other

		// Combos
		average.ComboDamagePerMinute += adv.ComboDamagePerMinute
	}

	for _, damageMap := range []DamageMap{average.DamageDealt, average.DamageTaken} {
		for champId, stageMap := range damageMap {
			for stage, entry := range stageMap {
				entry.DamageTotal /= float64(len(advancedStats))
				entry.DamagePercent /= float64(len(advancedStats))
				entry.Deaths = lolstats.Round(float64(damageMap[champId][stage].Deaths) / float64(len(advancedStats)))
			}
		}
	}

	average.DamageTakenPercentPerDeath /= float64(len(advancedStats))
	average.CarryFocusEfficiency /= float64(len(advancedStats))
	average.AttacksPerMinute /= float64(len(advancedStats))
	average.RevealsPerWardAverage /= float64(len(advancedStats))
	average.FavorableFightPercent /= float64(len(advancedStats))
	average.ComboDamagePerMinute /= float64(len(advancedStats))

	for _, intMap := range []map[string]int64{average.AbilityCounts, average.AbilityCounts0to10} {
		for k, _ := range intMap {
			intMap[k] = lolstats.Round(float64(intMap[k]) / float64(len(advancedStats)))
		}
	}

	for _, floatMap := range []map[string]float64{average.MapCoverages, average.UsefulPercent, average.LiveWardsAverage} {
		for k, _ := range floatMap {
			floatMap[k] /= float64(len(advancedStats))
		}
	}

	for _, tfa := range []*TeamFightAggregate{&average.FavorableTeamFights, &average.BalancedTeamFights, &average.UnfavorableTeamFights} {
		tfa.NetKills = float64(tfa.TotalKills-tfa.TotalDeaths) / math.Max(float64(tfa.Count), 1)
	}
	return average
}
