package generate

import (
	"time"

	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/riot"
)

type BasicStats struct {
	SummonerID    int64     `json:"summoner_id"`
	PlatformID    string    `json:"platform_id"`
	MatchID       int64     `json:"match_id"`
	MatchCreation int64     `json:"match_creation"`
	MatchDuration int64     `json:"match_duration"`
	LastUpdated   time.Time `json:"last_updated,omitempty"`

	TeamSummaries *TeamSummaries `json:"team_summaries,omitempty"`

	// We need these to filter entries for percentiles
	ChampionID         int           `json:"champion_id"`
	Lane               string        `json:"lane"`
	Role               string        `json:"role"`
	OpponentChampionID int           `json:"opponent_champion_id"`
	QueueType          string        `json:"queue_type"`
	MatchVersion       string        `json:"match_version"`
	Tier               riot.Tier     `json:"tier"`
	Division           riot.Division `json:"division"`

	ChampionIndex int   `json:"champion_index"`
	Won           bool  `json:"won"`
	Kills         int64 `json:"kills"`
	Deaths        int64 `json:"deaths"`
	Assists       int64 `json:"assists"`

	CSPerMinuteZeroToTen     float64  `json:"cs_per_minute_zero_to_ten"`
	CSPerMinuteDiffZeroToTen *float64 `json:"cs_per_minute_diff_zero_to_ten,omitempty"`

	NeutralMinionsKilledPerMinute            float64 `json:"neutral_minions_killed_per_minute"`
	NeutralMinionsKilledEnemyJunglePerMinute float64 `json:"neutral_minions_killed_enemy_jungle_per_minute"`

	GoldZeroToTen     float64  `json:"gold_zero_to_ten"`
	GoldDiffZeroToTen *float64 `json:"gold_diff_zero_to_ten,omitempty"`

	GoldTenToTwenty     float64  `json:"gold_ten_to_twenty"`
	GoldDiffTenToTwenty *float64 `json:"gold_diff_ten_to_twenty,omitempty"`

	GoldTwentyToThirty     float64  `json:"gold_twenty_to_thirty"`
	GoldDiffTwentyToThirty *float64 `json:"gold_diff_twenty_to_thirty,omitempty"`

	Level6Seconds     float64  `json:"level_6_seconds"`
	Level6DiffSeconds *float64 `json:"level_6_diff_seconds,omitempty"`

	WardsPlaced          int64   `json:"wards_placed"`
	WardsPlacedPerMinute float64 `json:"wards_placed_per_minute"`
	WardsKilledPerMinute float64 `json:"wards_killed_per_minute"`

	TotalDamageDealt          int64   `json:"total_damage_dealt"`
	TotalDamageDealtPerMinute float64 `json:"total_damage_dealt_per_minute"`

	TotalDamageDealtToChampions          int64   `json:"total_damage_dealt_to_champions"`
	TotalDamageDealtToChampionsPerMinute float64 `json:"total_damage_dealt_to_champions_per_minute"`

	TotalDamageTaken          int64   `json:"total_damage_taken"`
	TotalDamageTakenPerMinute float64 `json:"total_damage_taken_per_minute"`

	TotalHeal          int64   `json:"total_heal"`
	TotalHealPerMinute float64 `json:"total_heal_per_minute"`

	DeadPercentLast5           []TimePercent `json:"dead_percent_last_5,omitempty"`
	AverageDeathTimePercentage float64       `json:"average_death_time_percentage"`
}

type TeamSummaries struct {
	Team1        *TeamSummary       `json:"team1,omitempty"`
	Team2        *TeamSummary       `json:"team2,omitempty"`
	Participants []*ParticipantInfo `json:"participants,omitempty"`
}
type TeamSummary struct {
	Kills   int64 `json:"kills,omitempty"`
	Deaths  int64 `json:"deaths,omitempty"`
	Assists int64 `json:"assists,omitempty"`
	Towers  int32 `json:"towers,omitempty"`
	Barons  int32 `json:"barons,omitempty"`
	Dragons int32 `json:"dragons,omitempty"`
}

type ParticipantInfo struct {
	ChampID                     int32                 `json:"champ_id,omitempty"`
	ChampLevel                  int64                 `json:"champ_level,omitempty"`
	Spell1ID                    int32                 `json:"spell1_id,omitempty"`
	Spell2ID                    int32                 `json:"spell2_id,omitempty"`
	Masteries                   map[string]int64      `json:"masteries,omitempty"`
	Kills                       int64                 `json:"kills,omitempty"`
	Deaths                      int64                 `json:"deaths,omitempty"`
	Assists                     int64                 `json:"assists,omitempty"`
	Gold                        int64                 `json:"gold,omitempty"`
	TotalDamageDealtToChampions int64                 `json:"total_damage_dealt_to_champions,omitempty"`
	WardsPlaced                 int64                 `json:"wards_placed,omitempty"`
	WardKills                   int64                 `json:"ward_kills,omitempty"`
	Item0                       int64                 `json:"item0,omitempty"`
	Item1                       int64                 `json:"item1,omitempty"`
	Item2                       int64                 `json:"item2,omitempty"`
	Item3                       int64                 `json:"item3,omitempty"`
	Item4                       int64                 `json:"item4,omitempty"`
	Item5                       int64                 `json:"item5,omitempty"`
	Item6                       int64                 `json:"item6,omitempty"`
	Role                        baseview.RolePosition `json:"role,omitempty"`
	SummonerID                  int64                 `json:"summoner_id,omitempty"`
	SummonerName                string                `json:"summoner_name,omitempty"`
}

type TimePercent struct {
	Time    float64 `json:"time"`
	Percent float64 `json:"percent"`
}

// There are only 2 types of things we need in bigquery: percentile stats and filters
// Remove things that aren't in these categories
func (s *BasicStats) TrimNonStats() {
	s.TeamSummaries = nil
	s.DeadPercentLast5 = nil
}

// AverageBasicStats takes in a bunch of BasicStats, and returns an average.
func AverageBasicStats(basicStats []BasicStats) BasicStats {
	average := BasicStats{}

	if len(basicStats) == 0 {
		return average
	}

	for _, basic := range basicStats {
		// Set these every time, since they should be the same
		average.PlatformID = basic.PlatformID
		average.MatchID = basic.MatchID
		average.MatchCreation = basic.MatchCreation
		average.MatchDuration = basic.MatchDuration
		average.LastUpdated = basic.LastUpdated
		average.TeamSummaries = basic.TeamSummaries
		average.QueueType = basic.QueueType
		average.MatchVersion = basic.MatchVersion
		average.Won = basic.Won

		if average.CSPerMinuteDiffZeroToTen == nil {
			zero := 0.0
			average.CSPerMinuteDiffZeroToTen = &zero
		}
		if average.GoldDiffZeroToTen == nil {
			zero := 0.0
			average.GoldDiffZeroToTen = &zero
		}
		if average.GoldDiffTenToTwenty == nil {
			zero := 0.0
			average.GoldDiffTenToTwenty = &zero
		}
		if average.GoldDiffTwentyToThirty == nil {
			zero := 0.0
			average.GoldDiffTwentyToThirty = &zero
		}
		if average.Level6DiffSeconds == nil {
			zero := 0.0
			average.Level6DiffSeconds = &zero
		}

		// KDA is tough to average because they're defined as int types, so it doesn't allow for decimals.
		average.Kills += basic.Kills
		average.Deaths += basic.Deaths
		average.Assists += basic.Assists

		// Average these
		average.CSPerMinuteZeroToTen += basic.CSPerMinuteZeroToTen
		*average.CSPerMinuteDiffZeroToTen += *basic.CSPerMinuteDiffZeroToTen

		average.NeutralMinionsKilledPerMinute += basic.NeutralMinionsKilledPerMinute
		average.NeutralMinionsKilledEnemyJunglePerMinute += basic.NeutralMinionsKilledEnemyJunglePerMinute

		average.GoldZeroToTen += basic.GoldZeroToTen
		*average.GoldDiffZeroToTen += *basic.GoldDiffZeroToTen

		average.GoldTenToTwenty += basic.GoldTenToTwenty
		*average.GoldDiffTenToTwenty += *basic.GoldDiffTenToTwenty

		average.GoldTwentyToThirty += basic.GoldTwentyToThirty
		*average.GoldDiffTwentyToThirty += *basic.GoldDiffTwentyToThirty

		average.Level6Seconds += basic.Level6Seconds
		*average.Level6DiffSeconds += *basic.Level6DiffSeconds

		average.WardsPlaced += basic.WardsPlaced
		average.WardsPlacedPerMinute += basic.WardsPlacedPerMinute
		average.WardsKilledPerMinute += basic.WardsKilledPerMinute

		average.TotalDamageDealt += basic.TotalDamageDealt
		average.TotalDamageDealtPerMinute += basic.TotalDamageDealtPerMinute

		average.TotalDamageDealtToChampions += basic.TotalDamageDealtToChampions
		average.TotalDamageDealtToChampionsPerMinute += basic.TotalDamageDealtToChampionsPerMinute

		average.TotalDamageTaken += basic.TotalDamageTaken
		average.TotalDamageTakenPerMinute += basic.TotalDamageTakenPerMinute

		average.TotalHeal += basic.TotalHeal
		average.TotalHealPerMinute += basic.TotalHealPerMinute

		average.AverageDeathTimePercentage += basic.AverageDeathTimePercentage
		if average.DeadPercentLast5 == nil {
			average.DeadPercentLast5 = basic.DeadPercentLast5
		} else {
			for i, timePercent := range basic.DeadPercentLast5 {
				average.DeadPercentLast5[i].Percent += timePercent.Percent
			}
		}
	}

	average.Kills = lolstats.Round(float64(average.Kills) / float64(len(basicStats)))
	average.Deaths = lolstats.Round(float64(average.Deaths) / float64(len(basicStats)))
	average.Assists = lolstats.Round(float64(average.Assists) / float64(len(basicStats)))

	average.CSPerMinuteZeroToTen = average.CSPerMinuteZeroToTen / float64(len(basicStats))
	*average.CSPerMinuteDiffZeroToTen = *average.CSPerMinuteDiffZeroToTen / float64(len(basicStats))

	average.NeutralMinionsKilledPerMinute = average.NeutralMinionsKilledPerMinute / float64(len(basicStats))
	average.NeutralMinionsKilledEnemyJunglePerMinute = average.NeutralMinionsKilledEnemyJunglePerMinute / float64(len(basicStats))

	average.GoldZeroToTen = average.GoldZeroToTen / float64(len(basicStats))
	*average.GoldDiffZeroToTen = *average.GoldDiffZeroToTen / float64(len(basicStats))

	average.GoldTenToTwenty = average.GoldTenToTwenty / float64(len(basicStats))
	*average.GoldDiffTenToTwenty = *average.GoldDiffTenToTwenty / float64(len(basicStats))

	average.GoldTwentyToThirty = average.GoldTwentyToThirty / float64(len(basicStats))
	*average.GoldDiffTwentyToThirty = *average.GoldDiffTwentyToThirty / float64(len(basicStats))

	average.Level6Seconds = average.Level6Seconds / float64(len(basicStats))
	*average.Level6DiffSeconds = *average.Level6DiffSeconds / float64(len(basicStats))

	average.WardsPlaced = lolstats.Round(float64(average.WardsPlaced) / float64(len(basicStats)))
	average.WardsPlacedPerMinute = average.WardsPlacedPerMinute / float64(len(basicStats))
	average.WardsKilledPerMinute = average.WardsKilledPerMinute / float64(len(basicStats))

	average.TotalDamageDealt = lolstats.Round(float64(average.TotalDamageDealt) / float64(len(basicStats)))
	average.TotalDamageDealtPerMinute = average.TotalDamageDealtPerMinute / float64(len(basicStats))

	average.TotalDamageDealtToChampions = lolstats.Round(float64(average.TotalDamageDealtToChampions) / float64(len(basicStats)))
	average.TotalDamageDealtToChampionsPerMinute = average.TotalDamageDealtToChampionsPerMinute / float64(len(basicStats))

	average.TotalDamageTaken = lolstats.Round(float64(average.TotalDamageTaken) / float64(len(basicStats)))
	average.TotalDamageTakenPerMinute = average.TotalDamageTakenPerMinute / float64(len(basicStats))

	average.TotalHeal = lolstats.Round(float64(average.TotalHeal) / float64(len(basicStats)))
	average.TotalHealPerMinute = average.TotalHealPerMinute / float64(len(basicStats))

	average.AverageDeathTimePercentage = average.AverageDeathTimePercentage / float64(len(basicStats))
	for i := range average.DeadPercentLast5 {
		average.DeadPercentLast5[i].Percent = average.DeadPercentLast5[i].Percent / float64(len(basicStats))
	}

	return average
}
