package ingest

import (
	"container/list"
	"encoding/json"
	"math"
	"strconv"

	"github.com/VantageSports/lolstats/baseview"
)

/*
Team fight: Look for champions with a large loss of HP
Begins:
  - When any champ takes 30% max HP damage (drop_threshold) in two or more hits in one continuous session (no gaps more than 3 seconds long)
  - When a champion dies
Ends:
  - When no players who were involved in the team fight take any damage for 3 seconds
  - When all players who were involved in the team fight are more than chaseDistance apart
Technically, a 1v1 duel will be counted as a "TeamFight". Maybe Engagement is a better word to use?
*/

const continuousDamageTime = 3
const noDamageTimeForTeamFightEnd = 3

// Team fight participants are separated into two groups. "Participants" and "Influencers"
// Participants are players who are directly involved due to proximity to the start of the fight
// Influencers are players who influence the fight, but less than a participant.
// Currently, influencers are:
// * Players who teleport into the team fight and damage any participant
// * Players who use global ultimates that damage any participant

const participantDistance = 2000 // About 1 screen length in 1080p
const chaseDistance = 1000

// When we detect a team fight, we have to look back to find the actual start.
// This is how much history we store
const windowSeconds = 30
const windowGranularity = 0.25

// A player missing vision in a team fight if they weren't visible 4-8 seconds before the team fight
const visionWindowHiddenMax = 8
const visionWindowHiddenMin = 4

type JsonLongBoolMap map[int64]bool

type TeamFight struct {
	Begin                   float64           `json:"begin"`
	End                     float64           `json:"end"`
	ParticipantIDs          JsonLongBoolMap   `json:"participant_ids"`
	InfluencerIDs           JsonLongBoolMap   `json:"influencer_ids"`
	InitialTarget           int64             `json:"initial_target"`
	TeamKills               JsonLongBoolMap   `json:"team_kills"`
	TeamDeaths              JsonLongBoolMap   `json:"team_deaths"`
	SumTeamHealthPercent    float64           `json:"sum_team_health_percent"`
	SumEnemyHealthPercent   float64           `json:"sum_enemy_health_percent"`
	ParticipantLevel        int               `json:"partcipant_level"`
	EnemyLevelAverage       float64           `json:"enemy_level_average"`
	TeamGoldSpent           int64             `json:"team_gold_spent"`
	EnemyGoldSpent          int64             `json:"enemy_gold_spent"`
	NeutralDamageTaken      float64           `json:"neutral_damage_taken"`
	EnemyNeutralDamageTaken float64           `json:"enemy_neutral_damage_taken"`
	TeamSummonerSpells      int               `json:"team_summoner_spells"`
	EnemySummonerSpells     int               `json:"enemy_summoner_spells"`
	EnemiesInVision         JsonLongBoolMap   `json:"enemies_in_vision"`
	AlliesInVision          JsonLongBoolMap   `json:"allies_in_vision"`
	TeamGoldAtStart         int64             `json:"team_gold_start"`
	EnemyGoldAtStart        int64             `json:"enemy_gold_start"`
	NetGoldDiff             int64             `json:"net_gold_diff"`
	DeathAnalysisTags       map[string]bool   `json:"death_analysis_tags"`
	KillAnalysisTags        map[string]bool   `json:"kill_analysis_tags"`
	DamageDealt             map[int64]float64 `json:"damage_dealt"`
	EnemyDamageTaken        map[int64]float64 `json:"enemy_damage_taken"`
}

func (tf TeamFight) balance(participantID int64) float64 {
	teamPower := map[int64]float64{}
	for id, _ := range tf.ParticipantIDs {
		teamPower[baseview.TeamID(id)] += 1
	}
	for id, _ := range tf.InfluencerIDs {
		teamPower[baseview.TeamID(id)] += 0.5
	}
	myTeam, otherTeam := baseview.TeamID(participantID), baseview.EnemyTeamID(participantID)
	return teamPower[myTeam] - teamPower[otherTeam]
}

// computeNetGoldDiff calculates how much the team fight affected your team's gold totals
// It is: (our_end_gold - our_start_gold) - (their_end_gold - their_start_gold)
func (tf *TeamFight) computeNetGoldDiff(participantID int64, currentState *augmentedState) {
	teamTotalGold, enemyTotalGold := currentState.friendlyAndEnemyTeamGold(baseview.TeamID(participantID))
	tf.NetGoldDiff = teamTotalGold - tf.TeamGoldAtStart - (enemyTotalGold - tf.EnemyGoldAtStart)
}

func (tf *TeamFight) computeInitialParticipants(triggerParticipantID int64, state *augmentedState) {
	for i := int64(1); i <= 10; i++ {
		// Ignore dead players. They don't contribute
		if !state.IsAlive[i] {
			continue
		}

		// If the other player is close to the triggerParticipant, then include them
		if state.isFightParticipant(i, triggerParticipantID) {
			tf.ParticipantIDs[i] = true
		}
	}
}

func (tf *TeamFight) friendlyAndEnemyParticipants(teamID int64) ([]int64, []int64) {
	friend := []int64{}
	enemy := []int64{}
	for k, _ := range tf.ParticipantIDs {
		if baseview.TeamID(k) != teamID {
			enemy = append(enemy, k)
		} else {
			friend = append(friend, k)
		}
	}
	return friend, enemy
}

type balanceCondition func(float64) bool

func FavorableFights(x float64) bool   { return x > 0 }
func BalancedFights(x float64) bool    { return x == 0 }
func UnfavorableFights(x float64) bool { return x < 0 }

type TeamFightAggregate struct {
	Count       int64   `json:"count"`
	TotalKills  int64   `json:"kills"`
	TotalDeaths int64   `json:"deaths"`
	NetKills    float64 `json:"net_kills"` // (kills-deaths)/count
}

type continuousDamage struct {
	StartTime          float64
	LastTime           float64
	TotalPercentDamage float64
	NumDamageEvents    float64
}

// Add processes a new damage taken event.
// It either appends if it's close enough in time, or starts fresh
func (cd *continuousDamage) Add(t *baseview.Damage) {
	if t.Seconds()-cd.LastTime < continuousDamageTime {
		cd.TotalPercentDamage += t.Percent
		// Sometimes there will be lots of damage events with the same timestamp. Don't count those as separate damage events
		if t.Seconds() > cd.LastTime {
			cd.NumDamageEvents++
		}
	} else {
		cd.TotalPercentDamage = t.Percent
		cd.NumDamageEvents = 1
		cd.StartTime = t.Seconds()
	}
	cd.LastTime = t.Seconds()
}

type continuousAttack struct {
	StartTime float64
	LastTime  float64
}

// Add processes a new damage dealt event.
// It either appends if it's close enough, or starts fresh
func (ca *continuousAttack) Add(t *baseview.Damage) {
	// Keep track of players dealing champion damage
	if t.Seconds()-ca.LastTime >= continuousDamageTime {
		ca.StartTime = t.Seconds()
	}
	ca.LastTime = t.Seconds()
}

// Keep track of everything we need to analyze a team fight
type augmentedState struct {
	Seconds                 float64
	IsAlive                 map[int64]bool
	LastDeathTime           map[int64]float64
	ChampPositions          map[int64]*baseview.Position
	ChampLevels             map[int64]int
	ChampHealthPercentages  map[int64]float64
	GoldAmount              map[int64]int64
	GoldSpent               map[int64]int64
	SummonerSpellsAvailable map[int64]map[string]float64
	NeutralDamageTaken      map[int64]float64 // This is not copied over to new states
	LiveTurrets             map[int64]bool
}

func (st *augmentedState) isFightParticipant(victimID int64, participantID int64) bool {
	return st.isCloserThan(victimID, participantID, participantDistance)
}

func (st *augmentedState) isFightChasing(victimID int64, participantID int64) bool {
	return st.isCloserThan(victimID, participantID, chaseDistance)
}

func (st *augmentedState) isCloserThan(victimID int64, participantID int64, distance float64) bool {
	return st.ChampPositions[victimID].DistanceXY(*st.ChampPositions[participantID]) < distance
}

func (st *augmentedState) friendlyAndEnemyTeamGold(teamID int64) (int64, int64) {
	enemyTotalGold := int64(0)
	teamTotalGold := int64(0)
	for i := int64(1); i <= 10; i++ {
		if baseview.TeamID(i) != teamID {
			enemyTotalGold += st.GoldAmount[i] + st.GoldSpent[i]
		} else {
			teamTotalGold += st.GoldAmount[i] + st.GoldSpent[i]
		}
	}
	return teamTotalGold, enemyTotalGold
}

func NewAugmentedState(stateSeconds float64, previousState *augmentedState) *augmentedState {
	currentState := &augmentedState{
		Seconds:                 stateSeconds,
		IsAlive:                 map[int64]bool{},
		LastDeathTime:           map[int64]float64{},
		ChampPositions:          map[int64]*baseview.Position{},
		ChampLevels:             map[int64]int{},
		ChampHealthPercentages:  map[int64]float64{},
		GoldAmount:              map[int64]int64{},
		GoldSpent:               map[int64]int64{},
		SummonerSpellsAvailable: map[int64]map[string]float64{},
		NeutralDamageTaken:      map[int64]float64{}, // This is not copied over to new states
		LiveTurrets:             map[int64]bool{},
	}

	// Do a deep copy of the previous state to the current state
	if previousState != nil {
		for i := int64(1); i <= 10; i++ {
			currentState.IsAlive[i] = previousState.IsAlive[i]
			currentState.LastDeathTime[i] = previousState.LastDeathTime[i]
			currentState.ChampPositions[i] = previousState.ChampPositions[i]
			currentState.ChampLevels[i] = previousState.ChampLevels[i]
			currentState.ChampHealthPercentages[i] = previousState.ChampHealthPercentages[i]
			currentState.GoldAmount[i] = previousState.GoldAmount[i]
			currentState.GoldSpent[i] = previousState.GoldSpent[i]
			ssMap := map[string]float64{}
			for k, v := range previousState.SummonerSpellsAvailable[i] {
				ssMap[k] = v
			}
			currentState.SummonerSpellsAvailable[i] = ssMap
		}
		for turretID, _ := range baseview.TurretPositions {
			currentState.LiveTurrets[turretID] = previousState.LiveTurrets[turretID]
		}
	} else {
		// For the first state, we need to initialize the maps with default data
		for i := int64(1); i <= 10; i++ {
			currentState.IsAlive[i] = true
			currentState.ChampLevels[i] = 1
			currentState.SummonerSpellsAvailable[i] = map[string]float64{"Summoner1": 0.0, "Summoner2": 0.0}
		}
		for turretID, _ := range baseview.TurretPositions {
			currentState.LiveTurrets[turretID] = true
		}
	}
	return currentState
}

type teamFightAnalysis struct {
	ParticipantID          int64
	RecentStates           *list.List
	PlayerContinuousDamage map[int64]*continuousDamage
	PlayerContinuousAttack map[int64]*continuousAttack
	TeamFights             []*TeamFight
	MatchDuration          float64
	WardAnalysis           *wardAnalysis
}

func NewTeamFightAnalysis(participantID int64, matchDuration float64, wardAnalysis *wardAnalysis) *teamFightAnalysis {
	continuousDamageMap := map[int64]*continuousDamage{}
	continuousAttackMap := map[int64]*continuousAttack{}
	for i := int64(1); i <= 10; i++ {
		continuousDamageMap[i] = &continuousDamage{}
		continuousAttackMap[i] = &continuousAttack{}
	}
	return &teamFightAnalysis{
		ParticipantID:          participantID,
		RecentStates:           list.New(),
		PlayerContinuousDamage: continuousDamageMap,
		PlayerContinuousAttack: continuousAttackMap,
		MatchDuration:          matchDuration,
		WardAnalysis:           wardAnalysis,
	}
}

// findClosest returns the augmentedState closest to the specified time
func (tfa *teamFightAnalysis) findClosest(time float64) *augmentedState {
	stateList := tfa.RecentStates
	closest := 99999.0
	var prevState *augmentedState
	for e := stateList.Back(); e != nil; e = e.Prev() {
		state, _ := e.Value.(*augmentedState)
		if len(state.ChampPositions) != 10 {
			continue
		}
		if math.Abs(time-state.Seconds) < closest {
			closest = math.Abs(time - state.Seconds)
		} else {
			return prevState
		}
		prevState = state
	}
	return prevState
}

// updateRecentHistory adds a new entry to the RecentHistory list if stateSeconds is later than the last entry
func (tfa *teamFightAnalysis) updateRecentHistory(stateSeconds float64) {
	stateList := tfa.RecentStates
	lastElement := stateList.Back()
	var previousState *augmentedState
	if lastElement != nil {
		previousState = lastElement.Value.(*augmentedState)
		if stateSeconds-previousState.Seconds < windowGranularity {
			// The update is too close to the last one. Just reuse that one
			return
		}
	}

	currentState := NewAugmentedState(stateSeconds, previousState)
	stateList.PushBack(currentState)

	// Drop any states that are older than the window duration
	for front := stateList.Front(); front != nil; front = stateList.Front() {
		st := front.Value.(*augmentedState)
		if st.Seconds < stateSeconds-windowSeconds {
			stateList.Remove(front)
		} else {
			break
		}
	}
}

func (tfa *teamFightAnalysis) updateState(t *baseview.StateUpdate) {
	if tfa.RecentStates.Back() == nil {
		return
	}

	st := tfa.RecentStates.Back().Value.(*augmentedState)
	st.ChampPositions[t.ParticipantID] = &t.Position

	// Any time a player's gold goes down, add that to their gold spent.
	// NOTE: This currently doesn't account for shop undo, which can screw with this number
	if t.Gold < st.GoldAmount[t.ParticipantID] {
		st.GoldSpent[t.ParticipantID] += st.GoldAmount[t.ParticipantID] - t.Gold
	}
	st.GoldAmount[t.ParticipantID] = t.Gold

	// Check to see if a player has revived.
	// Make sure the status update is at least 2 seconds after the death event, to account for possible out-of-order events
	if !st.IsAlive[t.ParticipantID] && t.Health > 0 && t.Seconds() > st.LastDeathTime[t.ParticipantID]-2 {
		st.IsAlive[t.ParticipantID] = true
	}

	st.ChampHealthPercentages[t.ParticipantID] = 100 * t.Health / t.HealthMax
}

func (tfa *teamFightAnalysis) updateDeath(t *baseview.Death) {
	if tfa.RecentStates.Back() == nil {
		return
	}

	st := tfa.RecentStates.Back().Value.(*augmentedState)

	// Setting alive to false at the end allows us to capture team fights where we died
	st.IsAlive[t.VictimID] = false
	st.LastDeathTime[t.VictimID] = t.Seconds()
}

func (tfa *teamFightAnalysis) AddDamage(t *baseview.Damage) {
	tfa.PlayerContinuousDamage[t.VictimID].Add(t)

	// Keep track of players dealing champion damage
	if baseview.TeamID(t.AttackerID) != 0 {
		tfa.PlayerContinuousAttack[t.AttackerID].Add(t)
	} else {
		tfa.findClosest(t.Seconds()).NeutralDamageTaken[t.VictimID] += t.Percent
	}

	firstDamage := false
	// See if this creates a team fight
	// If someone takes 30% hp damage in multiple hits over a short time period, then start a team fight
	if (tfa.TeamFights == nil || tfa.TeamFights[len(tfa.TeamFights)-1].End != tfa.MatchDuration) &&
		tfa.PlayerContinuousDamage[t.VictimID].TotalPercentDamage > 30 && tfa.PlayerContinuousDamage[t.VictimID].NumDamageEvents > 1 {
		tfa.startTeamFight(t.VictimID, t.Seconds())
		firstDamage = true

		// Continue to include this damage as part of the fight
	}

	// If you're already in a team fight, don't start a new team fight
	curFight := tfa.currentFight()
	if curFight != nil {
		// Count the initial damage against the team fight's health percentages
		if firstDamage {
			if baseview.TeamID(tfa.ParticipantID) == baseview.TeamID(t.VictimID) {
				curFight.SumTeamHealthPercent -= math.Floor(t.Percent + 0.5)
			} else {
				curFight.SumEnemyHealthPercent -= math.Floor(t.Percent + 0.5)
			}
		}

		// Check to see if we have new influencers in the team fight that weren't there originally.
		// Adding them to the "Influencers" list makes them eligible for the kill/death count
		// This also accounts for players teleporting into the fight, or using global ultimates
		_, isAttackerInFight := curFight.ParticipantIDs[t.AttackerID]
		_, isVictimInFight := curFight.ParticipantIDs[t.VictimID]

		// Did someone not in the original participants damage the original participants? If so, add them
		// Note: We don't track the inverse (non participants who took damage from a participant),
		// because if someone really far away takes damage or gets hit by a really long range ability,
		// but they didn't damage anyone, then they didn't contribute anything. Think Karthus ult. Don't add the
		// entire enemy team to the influencers list.
		if !isAttackerInFight && isVictimInFight {
			if baseview.TeamID(t.AttackerID) != 0 {
				curFight.InfluencerIDs[t.AttackerID] = true
			}
		}

		// Add damage from the participant to give credit for kills
		if t.AttackerID == tfa.ParticipantID {
			curFight.DamageDealt[t.VictimID] += t.Percent
		}

		// Count up all damage against the enemy team, for carry focus efficiency
		if baseview.TeamID(t.VictimID) != baseview.TeamID(tfa.ParticipantID) &&
			(isVictimInFight || curFight.InfluencerIDs[t.VictimID]) {
			curFight.EnemyDamageTaken[t.VictimID] += t.Percent
		}

		// Track ongoing neutral and tower damage
		if baseview.TeamID(t.AttackerID) == 0 && isVictimInFight {
			if baseview.TeamID(t.VictimID) == baseview.TeamID(tfa.ParticipantID) {
				tfa.TeamFights[len(tfa.TeamFights)-1].NeutralDamageTaken += t.Percent
			} else {
				tfa.TeamFights[len(tfa.TeamFights)-1].EnemyNeutralDamageTaken += t.Percent
			}
		}
	}
}

func (tfa *teamFightAnalysis) AddStateUpdate(t *baseview.StateUpdate) {
	tfa.updateRecentHistory(t.Seconds())
	tfa.updateState(t)

	// If we're currently in a team fight, then check to see whether it's over or not
	curFight := tfa.currentFight()
	if curFight != nil {
		currentState := tfa.findClosest(t.Seconds())

		// Update ongoing stats of the team fight
		curFight.computeNetGoldDiff(tfa.ParticipantID, currentState)

		// Check for fight end
		fightEnd := tfa.fightEnd(curFight.ParticipantIDs, currentState, t.Seconds())
		if fightEnd != 0 {
			curFight.End = fightEnd
		}
	}
}

func (tfa *teamFightAnalysis) AddDeath(t *baseview.Death) {
	curFight := tfa.currentFight()

	// If we're not in a team fight, then start a team fight (provided we're alive and close)
	if curFight == nil {
		tfa.startTeamFight(t.VictimID, t.Seconds())
		curFight = tfa.currentFight()
	}

	// If we're currently in a team fight, then tally up the deaths
	if curFight != nil {
		allParticipantIDs := map[int64]bool{}
		for k, _ := range curFight.ParticipantIDs {
			allParticipantIDs[k] = true
		}
		for k, _ := range curFight.InfluencerIDs {
			allParticipantIDs[k] = true
		}

		if _, exists := allParticipantIDs[t.VictimID]; exists {
			if baseview.TeamID(t.VictimID) == baseview.TeamID(tfa.ParticipantID) {
				curFight.TeamDeaths[t.VictimID] = true
			} else if baseview.TeamID(t.VictimID) == baseview.EnemyTeamID(tfa.ParticipantID) {
				curFight.TeamKills[t.VictimID] = true
			}
		}
	}

	tfa.updateDeath(t)
}

// startTeamFight creates a team fight centered around a particular participant
func (tfa *teamFightAnalysis) startTeamFight(pID int64, triggerTime float64) {
	// Rewind to the point where the decision to team fight was made
	// TODO: Include flank positions. Maybe check when you have vision of all the participants?
	decisionTime := tfa.PlayerContinuousDamage[pID].StartTime

	decisionState := tfa.findClosest(decisionTime)
	triggerState := tfa.findClosest(triggerTime)
	// Ignore team fights that happen when you're dead, and team fights that you weren't near
	if !triggerState.IsAlive[tfa.ParticipantID] || !triggerState.isFightParticipant(pID, tfa.ParticipantID) {
		return
	}

	teamTotalGold, enemyTotalGold := decisionState.friendlyAndEnemyTeamGold(baseview.TeamID(tfa.ParticipantID))
	tf := &TeamFight{
		Begin:             tfa.PlayerContinuousDamage[pID].StartTime,
		End:               tfa.MatchDuration,
		ParticipantIDs:    map[int64]bool{},
		InfluencerIDs:     map[int64]bool{},
		InitialTarget:     pID,
		TeamKills:         map[int64]bool{},
		TeamDeaths:        map[int64]bool{},
		ParticipantLevel:  decisionState.ChampLevels[tfa.ParticipantID],
		EnemiesInVision:   map[int64]bool{},
		AlliesInVision:    map[int64]bool{},
		DeathAnalysisTags: map[string]bool{},
		KillAnalysisTags:  map[string]bool{},
		DamageDealt:       map[int64]float64{},
		EnemyDamageTaken:  map[int64]float64{},
		TeamGoldAtStart:   teamTotalGold,
		EnemyGoldAtStart:  enemyTotalGold,
	}

	// Find the people involved in the fight, based around pID, according to triggerState
	tf.computeInitialParticipants(pID, triggerState)

	// Break out the friendly and enemy participants
	friends, enemies := tf.friendlyAndEnemyParticipants(baseview.TeamID(tfa.ParticipantID))
	// If you take lots of damage from turrets, dragon, baron, but no enemy is near, then it's not a team fight
	if len(enemies) == 0 {
		return
	}

	for _, p := range friends {
		tf.SumTeamHealthPercent += math.Floor(decisionState.ChampHealthPercentages[p] + 0.5)
		tf.TeamGoldSpent += decisionState.GoldSpent[p]
		// Add up the neutral damage between decisionTime -> triggerTime
		for e := tfa.RecentStates.Front(); e != nil; e = e.Next() {
			ns := e.Value.(*augmentedState)
			if ns.Seconds >= decisionTime && ns.Seconds <= triggerTime {
				tf.NeutralDamageTaken += ns.NeutralDamageTaken[p]
			}
		}
		for _, spellCooldown := range decisionState.SummonerSpellsAvailable[p] {
			if spellCooldown <= triggerTime {
				// TODO: Exclude smite and teleport
				tf.TeamSummonerSpells++
			}
		}
		tf.AlliesInVision[p] = tfa.IsInVision(p, decisionTime)
	}

	sumEnemyLevel := 0.0
	for _, p := range enemies {
		tf.SumEnemyHealthPercent += math.Floor(decisionState.ChampHealthPercentages[p] + 0.5)
		sumEnemyLevel += float64(decisionState.ChampLevels[p])
		tf.EnemyGoldSpent += decisionState.GoldSpent[p]
		// Add up the neutral damage between decisionTime -> triggerTime
		for e := tfa.RecentStates.Front(); e != nil; e = e.Next() {
			ns := e.Value.(*augmentedState)
			if ns.Seconds >= decisionTime && ns.Seconds <= triggerTime {
				tf.EnemyNeutralDamageTaken += ns.NeutralDamageTaken[p]
			}
		}
		for _, spellCooldown := range decisionState.SummonerSpellsAvailable[p] {
			if spellCooldown <= triggerTime {
				// TODO: Exclude smite and teleport
				tf.EnemySummonerSpells++
			}
		}
		tf.EnemiesInVision[p] = tfa.IsInVision(p, decisionTime)
	}
	tf.EnemyLevelAverage = sumEnemyLevel / float64(len(enemies))

	tfa.TeamFights = append(tfa.TeamFights, tf)
}

func (tfa *teamFightAnalysis) AddLevelUp(t *baseview.LevelUp) {
	if tfa.RecentStates.Back() == nil {
		return
	}

	st := tfa.RecentStates.Back().Value.(*augmentedState)
	st.ChampLevels[t.ParticipantID]++
}

func (tfa *teamFightAnalysis) AddAttack(t *baseview.Attack) {
	if tfa.RecentStates.Back() == nil {
		return
	}

	st := tfa.RecentStates.Back().Value.(*augmentedState)
	if t.Slot == "Summoner1" || t.Slot == "Summoner2" {
		st.SummonerSpellsAvailable[t.AttackerID][t.Slot] = t.CooldownExpires
	}
}

func (tfa *teamFightAnalysis) AddTurretKill(t *baseview.BuildingKill) {
	if tfa.RecentStates.Back() == nil {
		return
	}

	st := tfa.RecentStates.Back().Value.(*augmentedState)
	st.LiveTurrets[t.BuildingID] = false
}

func (tfa *teamFightAnalysis) Aggregate(cond balanceCondition) TeamFightAggregate {
	tfas := TeamFightAggregate{}

	for _, tf := range tfa.TeamFights {
		balance := tf.balance(tfa.ParticipantID)
		if cond(balance) {
			tfas.Count++
			tfas.TotalKills += int64(len(tf.TeamKills))
			tfas.TotalDeaths += int64(len(tf.TeamDeaths))
		}
	}
	tfas.NetKills = float64(tfas.TotalKills-tfas.TotalDeaths) / math.Max(float64(tfas.Count), 1)
	return tfas
}

func (tfa *teamFightAnalysis) IsInVision(pID int64, time float64) bool {
	// Enumerate all champions, turrets, and wards on the enemy team.
	// Check whether we are within vision range of any of those
	// Champions: 1200
	// Towers: 1095
	// Wards: 1100 vision
	// Blue wards: 500
	// This can still get things wrong, since it doesn't take into account minions, terrain, and brushes.

	for e := tfa.RecentStates.Front(); e != nil; e = e.Next() {
		st := e.Value.(*augmentedState)
		// Only check the states between [time-windowMax, time-windowMin]
		if st.Seconds < time-visionWindowHiddenMax {
			continue
		} else if st.Seconds > time-visionWindowHiddenMin {
			return false
		}

		// Only count champs in the jungle as being out of vision.
		// This is an oversimplification, but it's closer to being correct
		if baseview.AreaToRegion(baseview.AreaID(*st.ChampPositions[pID])) != baseview.RegionJungle {
			return true
		}

		// Track Enemy Champions
		for i := int64(1); i <= 10; i++ {
			if i == pID {
				continue
			}
			if st.IsAlive[i] && st.IsAlive[pID] && baseview.TeamID(i) != baseview.TeamID(pID) {
				if st.isCloserThan(i, pID, 1200) {
					return true
				}
			}
		}

		// Track Towers
		for turretID, turretPosition := range baseview.TurretPositions {
			if turretID/100 != baseview.TeamID(pID)/100 && st.LiveTurrets[turretID] {
				if st.ChampPositions[pID].DistanceXY(turretPosition) < 1095 {
					return true
				}
			}
		}

		// Track Wards
		for i := int64(1); i <= 10; i++ {
			if baseview.TeamID(i) != baseview.TeamID(pID) {
				for _, ward := range tfa.WardAnalysis.PlayerWards[i] {
					if ward.Begin <= st.Seconds && ward.End >= st.Seconds && st.ChampPositions[pID].DistanceXY(*ward.Position) < ward.ItemType.Ward().SightRange() {
						return true
					}
				}
			}
		}
	}
	return false
}

func (tfa *teamFightAnalysis) currentFight() *TeamFight {
	if tfa.TeamFights != nil && tfa.TeamFights[len(tfa.TeamFights)-1].End == tfa.MatchDuration {
		return tfa.TeamFights[len(tfa.TeamFights)-1]
	}
	return nil
}

func (tfa *teamFightAnalysis) fightEnd(fightParticipants JsonLongBoolMap, currentState *augmentedState, lastEventTime float64) float64 {
	lastDamageTime := 0.0
	// Loop over each participant in the team fight, looking for signs of the fight still happening
	for participant, _ := range fightParticipants {
		// If they took damage recently, or damaged a player recently, then the team fight is still on
		if tfa.PlayerContinuousDamage[participant].LastTime > lastEventTime-noDamageTimeForTeamFightEnd ||
			tfa.PlayerContinuousAttack[participant].LastTime > lastEventTime-noDamageTimeForTeamFightEnd {
			return 0
		}

		// Find the smallest distance between a participant and an enemy.
		// If it's in chase distance, then the fight is still on.
		// Note: This doesn't extend the TeamFight's End value unless damage is done following the chase
		for i := int64(1); i <= 10; i++ {
			if currentState.IsAlive[i] && currentState.IsAlive[participant] &&
				baseview.TeamID(i) != baseview.TeamID(participant) && currentState.isFightChasing(i, participant) {
				return 0
			}
		}

		// Keep track of the last damage event, so we can use that to mark the end
		if tfa.PlayerContinuousDamage[participant].LastTime > lastDamageTime {
			lastDamageTime = tfa.PlayerContinuousDamage[participant].LastTime
		}
	}
	// None of the above signs happened, so the team fight is over
	return lastDamageTime
}

type SignificantFights struct {
	Total                          int64 `json:"total"`
	NumbersDifference              int64 `json:"numbers_difference"`
	NumbersDifferenceLackingVision int64 `json:"numbers_difference_lacking_vision"`
	HealthDifference               int64 `json:"health_difference"`
	GoldSpentDifference            int64 `json:"gold_spent_difference"`
	NeutralDamageDifference        int64 `json:"neutral_damage_difference"`

	// These will be labeled as positional or outplayed, because we don't have a way to determine those yet
	Other int64 `json:"other"`
}

func (sf *SignificantFights) ProcessFight(tf *TeamFight, pID int64, cond balanceCondition, tagMap map[string]bool, visionMap map[int64]bool) {
	sf.Total++

	if cond(tf.balance(pID)) {
		// Numbers difference
		haveVision := true
		for _, inVision := range visionMap {
			if !inVision {
				haveVision = false
				break
			}
		}
		if !haveVision {
			sf.NumbersDifferenceLackingVision++
			tagMap["numbers_difference_lacking_vision"] = true
		} else {
			sf.NumbersDifference++
			tagMap["numbers_difference"] = true
		}
	} else {
		// Balanced or unfavorable kill/favorable death. Look for secondary reasons

		numOnTeam := 0.0
		numOnEnemyTeam := 0.0
		for p, _ := range tf.ParticipantIDs {
			if baseview.TeamID(pID) == baseview.TeamID(p) {
				numOnTeam++
			} else {
				numOnEnemyTeam++
			}
		}
		numOnSmallerTeam := math.Min(numOnTeam, numOnEnemyTeam)
		hasSecondaryReason := false

		// Health Diff > 20% per person
		healthDiff := (tf.SumTeamHealthPercent / numOnTeam) - (tf.SumEnemyHealthPercent / numOnEnemyTeam)
		if cond(healthDiff) && math.Abs(healthDiff) > 20 {
			sf.HealthDifference++
			tagMap["health_difference"] = true
			hasSecondaryReason = true
		}
		// Gold Diff > 500 per person. Cap at 20k gold, where we assume full build and gold doesn't matter
		enemyGoldSpent := math.Min(float64(tf.EnemyGoldSpent)/numOnSmallerTeam, 20000.0)
		teamGoldSpent := math.Min(float64(tf.TeamGoldSpent)/numOnSmallerTeam, 20000.0)
		if cond(teamGoldSpent-enemyGoldSpent) && math.Abs(teamGoldSpent-enemyGoldSpent) > 500 {
			sf.GoldSpentDifference++
			tagMap["gold_spent_difference"] = true
			hasSecondaryReason = true
		}
		// Tower/Neutral Damage taken > 30% max hp.
		neutralDamageDiff := tf.EnemyNeutralDamageTaken - tf.NeutralDamageTaken
		if cond(neutralDamageDiff) && math.Abs(neutralDamageDiff) > 30 {
			sf.NeutralDamageDifference++
			tagMap["neutral_damage_difference"] = true
			hasSecondaryReason = true
		}

		if !hasSecondaryReason {
			sf.Other++
			tagMap["other"] = true
		}
	}
}

// SignificantKillsAndDeaths calculates significant kills, and significant deaths.
// This requires the TeamFights to be fully processed
func (tfa *teamFightAnalysis) SignificantKillsAndDeaths() (SignificantFights, SignificantFights) {
	kills := &SignificantFights{}
	deaths := &SignificantFights{}
	for _, tf := range tfa.TeamFights {
		// If there were kills in this fight, and there was a net gold diff
		if len(tf.TeamKills) > 0 && tf.NetGoldDiff > 300 {
			// See if you contributed to the kill
			contributed := false
			for pID, _ := range tf.TeamKills {
				if tf.DamageDealt[pID] > 0 {
					contributed = true
					break
				}

			}
			if contributed {
				kills.ProcessFight(tf, tfa.ParticipantID, FavorableFights, tf.KillAnalysisTags, tf.AlliesInVision)
			}
		}

		// If you died in this fight, and there was a net gold deficit
		if tf.TeamDeaths[tfa.ParticipantID] && tf.NetGoldDiff < -300 {
			deaths.ProcessFight(tf, tfa.ParticipantID, UnfavorableFights, tf.DeathAnalysisTags, tf.EnemiesInVision)
		}
	}
	return *kills, *deaths
}

// Since a map[int64]bool cannot be serialized directly,
// we define custom type and serialize the ints as strings
func (jlm *JsonLongBoolMap) MarshalJSON() ([]byte, error) {
	res := map[string]bool{}

	for key, val := range *jlm {
		res[strconv.FormatInt(key, 10)] = val
	}

	return json.Marshal(res)
}

func (jlm *JsonLongBoolMap) UnmarshalJSON(data []byte) error {
	source := map[string]bool{}
	if err := json.Unmarshal(data, &source); err != nil {
		return err
	}
	res := JsonLongBoolMap{}
	for str, val := range source {
		id, err := strconv.ParseInt(str, 10, 64)
		if err != nil {
			return err
		}
		res[id] = val
	}
	*jlm = res
	return nil
}
