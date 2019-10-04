package api

import (
	"fmt"
	"strconv"

	"github.com/VantageSports/riot"
)

type MatchDetail struct {
	MapID                 int                   `json:"mapId"`
	MatchCreation         int64                 `json:"matchCreation"`
	MatchDuration         int64                 `json:"matchDuration"`
	MatchID               int64                 `json:"matchId"`
	MatchMode             string                `json:"matchMode"`
	MatchType             string                `json:"matchType"`
	MatchVersion          string                `json:"matchVersion"`
	ParticipantIdentities []ParticipantIdentity `json:"participantIdentities"`
	Participants          []Participant         `json:"participants"`
	PlatformID            string                `json:"platformId"`
	QueueType             string                `json:"queueType"`
	Region                string                `json:"region"`
	Season                string                `json:"season"`
	Teams                 []Team                `json:"teams"`
	Timeline              Timeline              `json:"timeline,omitempty"`
}

type Participant struct {
	ChampionID                int                 `json:"championId,omitempty"`
	HighestAchievedSeasonTier string              `json:"highestAchievedSeasonTier,omitempty"` // CHALLENGER, MASTER, DIAMOND, PLATINUM, GOLD, SILVER, BRONZE, UNRANKED
	Masteries                 []riot.Mastery      `json:"masteries,omitempty"`
	ParticipantID             int                 `json:"participantId,omitempty"`
	Runes                     []Rune              `json:"runes,omitempty"`
	Spell1ID                  int                 `json:"spell1Id,omitempty"`
	Spell2ID                  int                 `json:"spell2Id,omitempty"`
	Stats                     ParticipantStats    `json:"stats,omitempty"`
	TeamID                    int                 `json:"teamId,omitempty"`
	Timeline                  ParticipantTimeline `json:"timeline,omitempty"`
}

type ParticipantIdentity struct {
	ParticipantID int    `json:"participantId,omitempty"`
	Player        Player `json:"player,omitempty"`
}

type Team struct {
	Bans                 []BannedChampion `json:"bans,omitempty"`
	BaronKills           int              `json:"baronKills,omitempty"`
	DominionVictoryScore int64            `json:"dominionVictoryScore,omitempty"`
	DragonKills          int              `json:"dragonKills,omitempty"`
	FirstBaron           bool             `json:"firstBaron,omitempty"`
	FirstBlood           bool             `json:"firstBlood,omitempty"`
	FirstDragon          bool             `json:"firstDragon,omitempty"`
	FirstInhibitor       bool             `json:"firstInhibitor,omitempty"`
	FirstTower           bool             `json:"firstTower,omitempty"`
	InhibitorKills       int              `json:"inhibitorKills,omitempty"`
	TeamID               int              `json:"teamId,omitempty"`
	TowerKills           int              `json:"towerKills,omitempty"`
	VilemawKills         int              `json:"vilemawKills,omitempty"`
	Winner               bool             `json:"winner,omitempty"`
}

type Timeline struct {
	FrameInterval int64   `json:"frameInterval,omitempty"`
	Frames        []Frame `json:"frames,omitempty"`
}

type ParticipantStats struct {
	Assists                         int64 `json:"assists,omitempty"`
	ChampLevel                      int64 `json:"champLevel,omitempty"`
	CombatPlayerScore               int64 `json:"combatPlayerScore,omitempty"`
	Deaths                          int64 `json:"deaths,omitempty"`
	DoubleKills                     int64 `json:"doubleKills,omitempty"`
	FirstBloodAssist                bool  `json:"firstBloodAssist,omitempty"`
	FirstBloodKill                  bool  `json:"firstBloodKill,omitempty"`
	FirstInhibitorAssist            bool  `json:"firstInhibitorAssist,omitempty"`
	FirstInhibitorKill              bool  `json:"firstInhibitorKill,omitempty"`
	FirstTowerAssist                bool  `json:"firstTowerAssist,omitempty"`
	FirstTowerKill                  bool  `json:"firstTowerKill,omitempty"`
	GoldEarned                      int64 `json:"goldEarned,omitempty"`
	GoldSpent                       int64 `json:"goldSpent,omitempty"`
	InhibitorKills                  int64 `json:"inhibitorKills,omitempty"`
	Item0                           int64 `json:"item0,omitempty"`
	Item1                           int64 `json:"item1,omitempty"`
	Item2                           int64 `json:"item2,omitempty"`
	Item3                           int64 `json:"item3,omitempty"`
	Item4                           int64 `json:"item4,omitempty"`
	Item5                           int64 `json:"item5,omitempty"`
	Item6                           int64 `json:"item6,omitempty"`
	KillingSprees                   int64 `json:"killingSprees,omitempty"`
	Kills                           int64 `json:"kills,omitempty"`
	LargestCriticalStrike           int64 `json:"largestCriticalStrike,omitempty"`
	LargestKillingSpree             int64 `json:"largestKillingSpree,omitempty"`
	LargestMultiKill                int64 `json:"largestMultiKill,omitempty"`
	MagicDamageDealt                int64 `json:"magicDamageDealt,omitempty"`
	MagicDamageDealtToChanpions     int64 `json:"magicDamageDealtToChampions,omitempty"`
	MagicDamageTaken                int64 `json:"magicDamageTaken,omitempty"`
	MinionsKilled                   int64 `json:"minionsKilled,omitempty"`
	NeutralMinionsKilled            int64 `json:"neutralMinionsKilled,omitempty"`
	NeutralMinionsKilledEnemyJungle int64 `json:"neutralMinionsKilledEnemyJungle,omitempty"`
	NeutralMinionsKilledTeamJungle  int64 `json:"neutralMinionsKilledTeamJungle,omitempty"`
	NodeCapture                     int64 `json:"nodeCapture,omitempty"`
	NodeCaptureAssist               int64 `json:"nodeCaptureAssist,omitempty"`
	NodeNeutralize                  int64 `json:"nodeNeutralize,omitempty"`
	NodeNeutralizeAssist            int64 `json:"nodeNeutralizeAssist,omitempty"`
	ObjectivePlayerScore            int64 `json:"objectivePlayerScore,omitempty"`
	PentaKills                      int64 `json:"pentaKills,omitempty"`
	PhysicalDamageDealt             int64 `json:"physicalDamageDealt,omitempty"`
	PhysicalDamageDealtToChampions  int64 `json:"physicalDamageDealtToChampions,omitempty"`
	PhysicalDamageTaken             int64 `json:"physicalDamageTaken,omitempty"`
	QuadraKills                     int64 `json:"quadraKills,omitempty"`
	SightWardsBoughtInGame          int64 `json:"sightWardsBoughtInGame,omitempty"`
	TeamObjective                   int64 `json:"teamObjective,omitempty"`
	TotalDamageDealt                int64 `json:"totalDamageDealt,omitempty"`
	TotalDamageDealtToChampions     int64 `json:"totalDamageDealtToChampions,omitempty"`
	TotalDamageTaken                int64 `json:"totalDamageTaken,omitempty"`
	TotalHeal                       int64 `json:"totalHeal,omitempty"`
	TotalPlayerScore                int64 `json:"totalPlayerScore,omitempty"`
	TotalScoreRank                  int64 `json:"totalScoreRank,omitempty"`
	TotalTimeCrowdControlDealt      int64 `json:"totalTimeCrowdControlDealt,omitempty"`
	TotalUnitsHealed                int64 `json:"totalUnitsHealed,omitempty"`
	TowerKills                      int64 `json:"towerKills,omitempty"`
	TripleKills                     int64 `json:"tripleKills,omitempty"`
	TrueDamageDealt                 int64 `json:"trueDamageDealt,omitempty"`
	TrueDamageDealtToChampions      int64 `json:"trueDamageDealtToChampions,omitempty"`
	TrueDamageTaken                 int64 `json:"trueDamageTaken,omitempty"`
	UnrealKills                     int64 `json:"unrealKills,omitempty"`
	VisionWardsBoughtInGame         int64 `json:"visionWardsBoughtInGame,omitempty"`
	WardsKilled                     int64 `json:"wardsKilled,omitempty"`
	WardsPlaced                     int64 `json:"wardsPlaced,omitempty"`
	Winner                          bool  `json:"winner,omitempty"`
}

type ParticipantTimeline struct {
	AncientGolemAssistsPerMinCounts ParticipantTimelineData `json:"ancientGolemAssistsPerMinCounts,omitempty"`
	AncientGolemKillsPerMinCounts   ParticipantTimelineData `json:"ancientGolemKillsPerMinCounts,omitempty"`
	AssistedLaneDeathsPerMinDeltas  ParticipantTimelineData `json:"assistedLaneDeathsPerMinDeltas,omitempty"`
	AssistedLaneKillsPerMinDeltas   ParticipantTimelineData `json:"assistedLaneKillsPerMinDeltas,omitempty"`
	BaronAssistsPerMinCounts        ParticipantTimelineData `json:"baronAssistsPerMinCounts,omitempty"`
	BaronKillsPerMinCounts          ParticipantTimelineData `json:"baronKillsPerMinCounts,omitempty"`
	CreepsPerMinDeltas              ParticipantTimelineData `json:"creepsPerMinDeltas,omitempty"`
	CsDiffPerMinDeltas              ParticipantTimelineData `json:"csDiffPerMinDeltas,omitempty"`
	DamageTakenDiffPerMinDeltas     ParticipantTimelineData `json:"damageTakenDiffPerMinDeltas,omitempty"`
	DamageTakenPerMinDeltas         ParticipantTimelineData `json:"damageTakenPerMinDeltas,omitempty"`
	DragonAssistsPerMinCounts       ParticipantTimelineData `json:"dragonAssistsPerMinCounts,omitempty"`
	DragonKillsPerMinCounts         ParticipantTimelineData `json:"dragonKillsPerMinCounts,omitempty"`
	ElderLizardAssistsPerMinCounts  ParticipantTimelineData `json:"elderLizardAssistsPerMinCounts,omitempty"`
	ElderLizardKillsPerMinCounts    ParticipantTimelineData `json:"elderLizardKillsPerMinCounts,omitempty"`
	GoldPerMinDeltas                ParticipantTimelineData `json:"goldPerMinDeltas,omitempty"`
	InhibitorAssistsPerMinCounts    ParticipantTimelineData `json:"inhibitorAssistsPerMinCounts,omitempty"`
	InhibitorKillsPerMinCounts      ParticipantTimelineData `json:"inhibitorKillsPerMinCounts,omitempty"`
	Lane                            string                  `json:"lane,omitempty"`
	Role                            string                  `json:"role,omitempty"`
	TowerAssistsPerMinCounts        ParticipantTimelineData `json:"towerAssistsPerMinCounts,omitempty"`
	TowerKillsPerMinCounts          ParticipantTimelineData `json:"towerKillsPerMinCounts,omitempty"`
	TowerKillsPerMinDeltas          ParticipantTimelineData `json:"towerKillsPerMinDeltas,omitempty"`
	VilemawAssistsPerMinCounts      ParticipantTimelineData `json:"vilemawAssistsPerMinCounts,omitempty"`
	VilemawKillsPerMinCounts        ParticipantTimelineData `json:"vilemawKillsPerMinCounts,omitempty"`
	WardsPerMinDeltas               ParticipantTimelineData `json:"wardsPerMinDeltas,omitempty"`
	XpDiffPerMinDeltas              ParticipantTimelineData `json:"xpDiffPerMinDeltas,omitempty"`
	XpPerMinDeltas                  ParticipantTimelineData `json:"xpPerMinDeltas,omitempty"`
}

type Rune struct {
	RuneID int64 `json:"runeId,omitempty"`
	Rank   int64 `json:"rank,omitempty"`
}

type Player struct {
	MatchHistoryURI string `json:"matchHistoryUri,omitempty"`
	ProfileIcon     int    `json:"profileIcon,omitempty"`
	SummonerID      int64  `json:"summonerId,omitempty"`
	SummonerName    string `json:"summonerName,omitempty"`
}

type BannedChampion struct {
	ChampionID int `json:"championId,omitempty"`
	PickTurn   int `json:"pickTurn,omitempty"`
	TeamID     int `json:"teamId,omitempty"` // Only specified in CurrentGame call
}

type Frame struct {
	Events            []Event                     `json:"events,omitempty"`
	ParticipantFrames map[string]ParticipantFrame `json:"participantFrames,omitempty"`
	Timestamp         int64                       `json:"timestamp,omitempty"`

	// ParticipantFrameList is a construct of our own (not riots) to make it
	// easier to put data into bigquery, which doesn't like maps.
	ParticipantFrameList []ParticipantFrame `json:"participantFrameList,omitempty"`
}

func (f *Frame) ToBQ() {
	f.ParticipantFrameList = []ParticipantFrame{}
	for _, pf := range f.ParticipantFrames {
		f.ParticipantFrameList = append(f.ParticipantFrameList, pf)
	}
	f.ParticipantFrames = nil
}

func (f *Frame) FromBQ() {
	f.ParticipantFrames = map[string]ParticipantFrame{}
	for i := range f.ParticipantFrameList {
		pf := f.ParticipantFrameList[i]
		idStr := strconv.Itoa(pf.ParticipantID)
		f.ParticipantFrames[idStr] = pf
	}
	f.ParticipantFrameList = nil
}

type ParticipantTimelineData struct {
	TenToTwenty    float64 `json:"tenToTwenty,omitempty"`
	ThirtyToEnd    float64 `json:"thirtyToEnd,omitempty"`
	TwentyToThirty float64 `json:"twentyToThirty,omitempty"`
	ZeroToTen      float64 `json:"zeroToTen,omitempty"`
}

type Event struct {
	AscendedType            string   `json:"ascendedType,omitempty"`
	AssistingParticipantIDs []int    `json:"assistingParticipantIds,omitempty"`
	BuildingType            string   `json:"buildingType,omitempty"` // INHIBITOR_BUILDING, TOWER_BUILDING
	CreatorID               int      `json:"creatorId,omitempty"`
	EventType               string   `json:"eventType,omitempty"` // ASCENDED_EVENT, BUILDING_KILL, CAPTURE_POINT, CHAMPION_KILL, ELITE_MONSTER_KILL, ITEM_DESTROYED, ITEM_PURCHASED, ITEM_SOLD, ITEM_UNDO, PORO_KING_SUMMON, SKILL_LEVEL_UP, WARD_KILL, WARD_PLACED
	ItemAfter               int      `json:"itemAfter,omitempty"`
	ItemBefore              int      `json:"itemBefore,omitempty"`
	ItemID                  int      `json:"itemId,omitempty"`
	KillerID                int      `json:"killerId,omitempty"`
	LaneType                string   `json:"laneType,omitempty"`    // BOT_LANE, MID_LANE, TOP_LANE
	LevelUpType             string   `json:"levelUpType,omitempty"` // EVOLVE, NORMAL
	MonsterType             string   `json:"monsterType,omitempty"` // BARON_NASHOR, BLUE_GOLEM, DRAGON, RED_LIZARD, VILEMAW
	ParticipantID           int      `json:"participantId,omitempty"`
	PointCaptured           string   `json:"pointCaptured,omitempty"` //  POINT_A, POINT_B, POINT_C, POINT_D, POINT_E)
	Position                Position `json:"position,omitempty"`
	SkillSlot               int      `json:"skillSlot,omitempty"`
	TeamID                  int      `json:"teamId,omitempty"`
	Timestamp               int64    `json:"timestamp,omitempty"`
	TowerType               string   `json:"towerType,omitempty"` // BASE_TURRET, FOUNTAIN_TURRET, INNER_TURRET, NEXUS_TURRET, OUTER_TURRET, UNDEFINED_TURRET
	VictimID                int      `json:"victimId,omitempty"`
	WardType                string   `json:"wardType,omitempty"` // SIGHT_WARD, TEEMO_MUSHROOM, UNDEFINED, VISION_WARD, YELLOW_TRINKET, YELLOW_TRINKET_UPGRADE
}

type ParticipantFrame struct {
	CurrentGold         int      `json:"currentGold,omitempty"`
	DominionScore       int      `json:"dominionScore,omitempty"`
	JungleMinionsKilled int      `json:"jungleMinionsKilled,omitempty"`
	Level               int      `json:"level,omitempty"`
	MinionsKilled       int      `json:"minionsKilled,omitempty"`
	ParticipantID       int      `json:"participantId,omitempty"`
	Position            Position `json:"position,omitempty"`
	TeamScore           int      `json:"teamScore,omitempty"`
	TotalGold           int      `json:"totalGold,omitempty"`
	XP                  int      `json:"xp,omitempty"`
}

type Position struct {
	X int `json:"y"`
	Y int `json:"x"`
}

func (m *MatchDetail) ToBQ() {
	for fi := range m.Timeline.Frames {
		m.Timeline.Frames[fi].ToBQ()
	}
}

func (m *MatchDetail) FromBQ() {
	for fi := range m.Timeline.Frames {
		m.Timeline.Frames[fi].FromBQ()
	}
}

func (m *MatchDetail) IsParticipantIdentitiesEmpty() bool {
	return len(m.ParticipantIdentities) == 0 ||
		m.ParticipantIdentities[0].Player.SummonerID == 0
}

// Not all matches have timeline data. If timeline data is requested, but doesn't exist, then the response won't include it.
func (a *Api) Match(matchID string, includeTimeline bool) (matchDetail MatchDetail, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v2.2/match/%v",
		a.baseURL(), a.Region(), matchID)
	if url, err = a.addParams(url, "includeTimeline", fmt.Sprintf("%v", includeTimeline)); err != nil {
		return matchDetail, err
	}

	err = a.getJSON(url, &matchDetail)
	return matchDetail, err
}
