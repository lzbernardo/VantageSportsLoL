package server

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	bq "cloud.google.com/go/bigquery"
	"golang.org/x/net/context"

	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/lolstats/gcd"
	"github.com/VantageSports/riot"
)

var VPRWeights map[string]map[string]float64 = map[string]map[string]float64{
	string(baseview.RoleTop): {
		"attacks_per_minute_useful":          15.0,
		"gold_zero_to_ten":                   25.0,
		"gold_diff_zero_to_ten":              25.0,
		"level_6_seconds":                    5.0,
		"cs_per_minute_zero_to_ten":          5.0,
		"cs_per_minute_diff_zero_to_ten":     5.0,
		"damage_taken_percent_per_death":     30.0,
		"wards_killed_per_minute":            5.0,
		"favorable_team_fight_percent":       20.0,
		"map_coverages_full":                 30.0,
		"useful_percent_all":                 30.0,
		"live_wards_average_pink":            15.0,
		"live_wards_average_yellow_and_blue": 15.0,
		"bad_deaths_total_per_minute":        100.0,
		"good_kills_total_per_minute":        100.0,
	},
	string(baseview.RoleJungle): {
		"attacks_per_minute_useful":                      15.0,
		"gold_zero_to_ten":                               25.0,
		"gold_diff_zero_to_ten":                          25.0,
		"level_6_seconds":                                5.0,
		"neutral_minions_killed_per_minute":              5.0,
		"neutral_minions_killed_enemy_jungle_per_minute": 5.0,
		"damage_taken_percent_per_death":                 30.0,
		"wards_killed_per_minute":                        5.0,
		"favorable_team_fight_percent":                   20.0,
		"map_coverages_full":                             50.0,
		"useful_percent_all":                             30.0,
		"live_wards_average_pink":                        15.0,
		"live_wards_average_yellow_and_blue":             15.0,
		"bad_deaths_total_per_minute":                    100.0,
		"good_kills_total_per_minute":                    100.0,
	},
	string(baseview.RoleMid): {
		"attacks_per_minute_useful":          15.0,
		"gold_zero_to_ten":                   25.0,
		"gold_diff_zero_to_ten":              25.0,
		"level_6_seconds":                    5.0,
		"cs_per_minute_zero_to_ten":          5.0,
		"cs_per_minute_diff_zero_to_ten":     5.0,
		"damage_taken_percent_per_death":     30.0,
		"wards_killed_per_minute":            5.0,
		"favorable_team_fight_percent":       20.0,
		"map_coverages_full":                 30.0,
		"useful_percent_all":                 30.0,
		"live_wards_average_pink":            15.0,
		"live_wards_average_yellow_and_blue": 15.0,
		"bad_deaths_total_per_minute":        100.0,
		"good_kills_total_per_minute":        100.0,
	},
	string(baseview.RoleAdc): {
		"attacks_per_minute_useful":          20.0,
		"gold_zero_to_ten":                   30.0,
		"gold_diff_zero_to_ten":              30.0,
		"level_6_seconds":                    5.0,
		"cs_per_minute_zero_to_ten":          5.0,
		"cs_per_minute_diff_zero_to_ten":     5.0,
		"damage_taken_percent_per_death":     30.0,
		"wards_killed_per_minute":            5.0,
		"favorable_team_fight_percent":       20.0,
		"map_coverages_full":                 30.0,
		"useful_percent_all":                 30.0,
		"live_wards_average_pink":            15.0,
		"live_wards_average_yellow_and_blue": 15.0,
		"bad_deaths_total_per_minute":        100.0,
		"good_kills_total_per_minute":        100.0,
	},
	string(baseview.RoleSupport): {
		"attacks_per_minute_useful":          20.0,
		"gold_zero_to_ten":                   15.0,
		"gold_diff_zero_to_ten":              15.0,
		"level_6_seconds":                    5.0,
		"damage_taken_percent_per_death":     30.0,
		"wards_killed_per_minute":            5.0,
		"favorable_team_fight_percent":       20.0,
		"map_coverages_full":                 30.0,
		"useful_percent_all":                 30.0,
		"live_wards_average_pink":            20.0,
		"live_wards_average_yellow_and_blue": 20.0,
		"bad_deaths_total_per_minute":        100.0,
		"good_kills_total_per_minute":        100.0,
	},
}
var VPRWeightsLowerIsBetter map[string]bool = map[string]bool{
	"level_6_seconds":               true,
	"bad_deaths_total_per_minute":   true,
	"average_death_time_percentage": true,
}
var VPRSQL map[string]string = map[string]string{
	"attacks_per_minute_useful":      "attacks_per_minute/useful_percent.all*100 AS attacks_per_minute_useful",
	"gold_zero_to_ten":               "gold_zero_to_ten AS gold_zero_to_ten",
	"gold_diff_zero_to_ten":          "gold_diff_zero_to_ten AS gold_diff_zero_to_ten",
	"level_6_seconds":                "level_6_seconds AS level_6_seconds",
	"cs_per_minute_zero_to_ten":      "cs_per_minute_zero_to_ten AS cs_per_minute_zero_to_ten",
	"cs_per_minute_diff_zero_to_ten": "cs_per_minute_diff_zero_to_ten AS cs_per_minute_diff_zero_to_ten",

	"neutral_minions_killed_per_minute":              "neutral_minions_killed_per_minute AS neutral_minions_killed_per_minute",
	"neutral_minions_killed_enemy_jungle_per_minute": "neutral_minions_killed_enemy_jungle_per_minute AS neutral_minions_killed_enemy_jungle_per_minute",

	"damage_taken_percent_per_death":     "damage_taken_percent_per_death AS damage_taken_percent_per_death",
	"wards_killed_per_minute":            "wards_killed_per_minute AS wards_killed_per_minute",
	"favorable_team_fight_percent":       "favorable_team_fight_percent AS favorable_team_fight_percent",
	"map_coverages_full":                 "map_coverages.`full` AS map_coverages_full",
	"useful_percent_all":                 "useful_percent.all AS useful_percent_all",
	"live_wards_average_pink":            "live_wards_average.pink AS live_wards_average_pink",
	"live_wards_average_yellow_and_blue": "live_wards_average.yellow_and_blue AS live_wards_average_yellow_and_blue",
	"bad_deaths_total_per_minute":        "bad_deaths.total*60/match_duration AS bad_deaths_total_per_minute",
	"good_kills_total_per_minute":        "good_kills.total*60/match_duration AS good_kills_total_per_minute",

	// Legacy VPR stats
	"average_death_time_percentage": "average_death_time_percentage AS average_death_time_percentage",
	"reveals_per_ward_average":      "reveals_per_ward_average AS reveals_per_ward_average",
	"carry_focus_efficiency":        "carry_focus_efficiency AS carry_focus_efficiency",
	"attacks_per_minute":            "attacks_per_minute AS attacks_per_minute",
}

var VPRQueueTypes = []string{
	fmt.Sprintf("\"%s\"", string(riot.TB_RANKED_SOLO)),
	fmt.Sprintf("\"%s\"", string(riot.RANKED_FLEX_SR)),
	fmt.Sprintf("\"%s\"", string(riot.RANKED_TEAM_5x5)),
	"\"CUSTOM\"",
}

var VPRStatCategories = map[string]string{
	"attacks_per_minute_useful":      lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],
	"gold_zero_to_ten":               lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],
	"gold_diff_zero_to_ten":          lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],
	"level_6_seconds":                lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],
	"cs_per_minute_zero_to_ten":      lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],
	"cs_per_minute_diff_zero_to_ten": lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],

	"neutral_minions_killed_per_minute":              lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],
	"neutral_minions_killed_enemy_jungle_per_minute": lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],

	"damage_taken_percent_per_death":     lolstats.GoalCategory_name[int32(lolstats.GoalCategory_MECHANICS)],
	"wards_killed_per_minute":            lolstats.GoalCategory_name[int32(lolstats.GoalCategory_DECISION_MAKING)],
	"favorable_team_fight_percent":       lolstats.GoalCategory_name[int32(lolstats.GoalCategory_DECISION_MAKING)],
	"map_coverages_full":                 lolstats.GoalCategory_name[int32(lolstats.GoalCategory_DECISION_MAKING)],
	"useful_percent_all":                 lolstats.GoalCategory_name[int32(lolstats.GoalCategory_DECISION_MAKING)],
	"live_wards_average_pink":            lolstats.GoalCategory_name[int32(lolstats.GoalCategory_DECISION_MAKING)],
	"live_wards_average_yellow_and_blue": lolstats.GoalCategory_name[int32(lolstats.GoalCategory_DECISION_MAKING)],
	"bad_deaths_total_per_minute":        lolstats.GoalCategory_name[int32(lolstats.GoalCategory_DECISION_MAKING)],
	"good_kills_total_per_minute":        lolstats.GoalCategory_name[int32(lolstats.GoalCategory_DECISION_MAKING)],
}

// Set the goal target to be this much better than their current average
const VPRGoalPercentileTarget = 5

// VPRPerGameLengthMinutes is the estimated game length when translating from stat -> per/game goal
const VPRPerGameLengthMinutes = 30

func (s *GoalsServer) percentileResults(ctx context.Context, summonerId int64, platform, rolePosition string) ([]map[string]interface{}, error) {
	monthAgo := time.Now().Add(-30 * 24 * time.Hour)
	monthAgoStr := fmt.Sprintf("%v-%02d-%02d", monthAgo.Year(), int(monthAgo.Month()), monthAgo.Day())

	percentileSelects := []string{}
	for k := range VPRWeights[rolePosition] {
		withAlias := strings.Index(VPRSQL[k], " AS ")
		if withAlias != -1 {
			percentileSelects = append(percentileSelects, fmt.Sprintf("APPROX_QUANTILES(%s, 100)%s", VPRSQL[k][0:withAlias], VPRSQL[k][withAlias:]))
		} else {
			percentileSelects = append(percentileSelects, fmt.Sprintf("APPROX_QUANTILES(%s, 100)", VPRSQL[k]))
		}
	}

	// Get the percentiles for the VPR stats for that role
	// TODO: Maybe this should call the lolstats service's Percentile endpoint instead?
	queryString := fmt.Sprintf("SELECT count(*) as count, %s FROM `%s` basic LEFT OUTER JOIN `%s` advanced ON basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id AND basic.match_id = advanced.match_id WHERE basic.platform_id = \"%s\" and basic.queue_type in (%s) AND role_position = \"%s\" AND basic._PARTITIONTIME >= TIMESTAMP('%s') AND advanced._PARTITIONTIME >= TIMESTAMP('%s')", strings.Join(percentileSelects, ","), s.BasicTable, s.AdvancedTable, platform, strings.Join(VPRQueueTypes, ","), rolePosition, monthAgoStr, monthAgoStr)

	return DoQuery(ctx, s.BqClient, queryString)
}

func (s *GoalsServer) findGoalTargets(ctx context.Context, summonerId int64, platform, rolePosition string, gameWindow int64, targetAchievementCount int64, percentileResults []map[string]interface{}) ([]*gcd.LolGoal, error) {
	selects := []string{}
	for k := range VPRWeights[rolePosition] {
		selects = append(selects, VPRSQL[k])
	}

	monthAgo := time.Now().Add(-30 * 24 * time.Hour)
	monthAgoStr := fmt.Sprintf("%v-%02d-%02d", monthAgo.Year(), int(monthAgo.Month()), monthAgo.Day())

	// Get the VPR stats for the last n games for that role
	queryString := fmt.Sprintf("SELECT %s FROM (SELECT *,basic.match_id as basic_match_id FROM `%s` basic LEFT OUTER JOIN `%s` advanced ON basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id AND basic.match_id = advanced.match_id WHERE basic.summoner_id = %d AND basic.platform_id = \"%s\" AND basic.queue_type in (%s) AND role_position = \"%s\" AND basic._PARTITIONTIME >= TIMESTAMP('%s') AND advanced._PARTITIONTIME >= TIMESTAMP('%s') ORDER BY match_creation desc LIMIT %d) ORDER BY match_creation DESC", strings.Join(selects, ","), s.BasicTable, s.AdvancedTable, summonerId, platform, strings.Join(VPRQueueTypes, ","), rolePosition, monthAgoStr, monthAgoStr, gameWindow)

	summonerResults, err := DoQuery(ctx, s.BqClient, queryString)
	if err != nil {
		return nil, err
	}

	// Get the average of all the stats
	summonerStatAverages := map[string]float64{}
	summonerStatValues := map[string][]float64{}
	for _, row := range summonerResults {
		for k, v := range row {
			val, ok := v.(float64)
			if ok {
				summonerStatAverages[k] += val
				summonerStatValues[k] = append(summonerStatValues[k], val)
			}
		}
	}

	myGoals := []*gcd.LolGoal{}
	for k, v := range summonerStatAverages {
		myValue := v / float64(len(summonerResults))
		myStdDev := stdDev(summonerStatValues[k], myValue)
		myLastValue := 0.0
		// Try to get the most recent value for myLastValue
		// However, sometimes it's null, in which case we look at the next game
		for i := 0; i < len(summonerResults); i++ {
			val, ok := summonerResults[i][k].(float64)
			if ok {
				myLastValue = val
				break
			}
		}

		percentileValues := percentileResults[0][k].([]bq.Value)
		myPercentile := getMyPercentile(percentileValues, myValue, k)

		// Don't set goals for really good performances.
		if myPercentile >= 100-VPRGoalPercentileTarget {
			continue
		}

		myGoalValue, myGoalPercentile := getGoalValue(percentileValues, myValue, myPercentile, k)
		// If we couldn't find a goal better than myValue, then skip this stat.
		// It means they have the best possible score
		if myGoalPercentile < 0 || myGoalPercentile > 100 {
			continue
		}

		goalComparator := lolstats.GoalComparator_name[int32(lolstats.GoalComparator_GREATER_THAN_OR_EQUAL)]
		if VPRWeightsLowerIsBetter[k] {
			goalComparator = lolstats.GoalComparator_name[int32(lolstats.GoalComparator_LESS_THAN_OR_EQUAL)]
		}

		// Convert the goal value (stat) into a per-game value
		goalPerGameValue, err := TranslateStatToPerGameValue(myGoalValue, k)
		if err != nil {
			return nil, err
		}

		lastValuePerGameValue, err := TranslateStatToPerGameValue(myLastValue, k)
		if err != nil {
			return nil, err
		}

		myGoals = append(myGoals, &gcd.LolGoal{
			Created:                time.Now(),
			SummonerID:             summonerId,
			Platform:               platform,
			UnderlyingStat:         k,
			TargetValue:            goalPerGameValue,
			Comparator:             goalComparator,
			AchievementCount:       0,
			TargetAchievementCount: targetAchievementCount,
			// Divide the weight by the percentile. More important stats are weighted higher.
			// Lower percentile scores also weight higher
			ImportanceWeight: VPRWeights[rolePosition][k] * probabilityToReach(myGoalValue, myValue, myStdDev),
			Status:           lolstats.GoalStatus_name[int32(lolstats.GoalStatus_NEW)],
			RolePosition:     rolePosition,
			ChampionID:       0,
			LastValue:        lastValuePerGameValue,
			Category:         VPRStatCategories[k],
		})
	}

	// Sort the stats to get the highest importance weights
	sort.Stable(gcd.LolGoalArray(myGoals))

	return myGoals, nil
}

func getMyPercentile(percentileValues []bq.Value, myValue float64, stat string) int {
	j := 0
	if VPRWeightsLowerIsBetter[stat] {
		// Walk the values from the worst to the best.
		// Stop when we find a value that's better than yours.
		for j = range percentileValues {
			if percentileValues[len(percentileValues)-1-j].(float64) < myValue {
				return j
			}
		}
	} else {
		// Do the same thing, but for higher is better stats
		for j = range percentileValues {
			if percentileValues[j].(float64) > myValue {
				return j
			}
		}
	}
	return j
}
func getGoalValue(percentileValues []bq.Value, myValue float64, myPercentile int, stat string) (float64, int) {
	myGoalValue := 0.0
	percentileIndex := 0
	if VPRWeightsLowerIsBetter[stat] {
		percentileIndex = len(percentileValues) - 1 - myPercentile - VPRGoalPercentileTarget
		myGoalValue = percentileValues[percentileIndex].(float64)
		// If the goal value is the same, then keep walking down until we find something different
		for myGoalValue == myValue && percentileIndex >= 0 {
			myGoalValue = percentileValues[percentileIndex].(float64)
			percentileIndex--
		}
	} else {
		percentileIndex = myPercentile + VPRGoalPercentileTarget
		myGoalValue = percentileValues[percentileIndex].(float64)
		// If the goal value is the same, then keep walking up until we find something different
		for myGoalValue == myValue && percentileIndex < len(percentileValues)-1 {
			myGoalValue = percentileValues[percentileIndex].(float64)
			percentileIndex++
		}
	}

	return myGoalValue, percentileIndex
}

func TranslateStatToPerGameValue(value float64, stat string) (float64, error) {
	switch stat {
	case "neutral_minions_killed_per_minute",
		"neutral_minions_killed_enemy_jungle_per_minute",
		"wards_killed_per_minute",
		"live_wards_average_pink",
		"live_wards_average_yellow_and_blue",
		"bad_deaths_total_per_minute",
		"good_kills_total_per_minute":
		return Round(value * VPRPerGameLengthMinutes), nil
	case "gold_zero_to_ten":
		return Round(value*10 + 500), nil
	case "gold_diff_zero_to_ten",
		"cs_per_minute_zero_to_ten",
		"cs_per_minute_diff_zero_to_ten":
		return Round(value * 10), nil
	case "favorable_team_fight_percent":
		// Ran a Bigquery on average favorable + balanced + unfavorable team fight count
		return Round(value * 24 / 100.0), nil
	case "useful_percent_all":
		return Round(value * VPRPerGameLengthMinutes / 100.0), nil
	case "attacks_per_minute_useful",
		"level_6_seconds",
		"damage_taken_percent_per_death",
		"map_coverages_full":
		return Round(value), nil
	default:
		return 0, fmt.Errorf("Cannot translate stat: %v", stat)
	}
}

func Round(f float64) float64 {
	return math.Floor(f + .5)
}

func stdDev(numbers []float64, mean float64) float64 {
	if len(numbers) <= 1 {
		return 0
	}

	total := 0.0
	for _, number := range numbers {
		total += math.Pow(number-mean, 2)
	}
	variance := total / float64(len(numbers)-1)
	return math.Sqrt(variance)
}

func probabilityToReach(target, mean, stddev float64) float64 {
	// Assume a default 50% when we don't have enough data
	if stddev == 0 {
		return 0.5
	}

	return 0.5 * math.Erfc((target-mean)/stddev/math.Sqrt(2))
}
