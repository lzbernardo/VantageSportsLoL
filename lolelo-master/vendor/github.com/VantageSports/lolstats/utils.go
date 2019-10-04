package lolstats

import (
	"fmt"
	"math"
	"strings"

	"github.com/VantageSports/lolstats/baseview"
)

func (r *StatsRequest) Valid() error {
	if len(r.Selects) == 0 {
		return fmt.Errorf("no selects specified")
	}

	if r.Platform == "" && r.PatchPrefix == "" && r.Tier == "" && r.Division == "" && r.Lane == "" && r.Role == "" && r.ChampionId == 0 && r.OpponentChampionId == 0 && r.SummonerId == 0 && r.QueueType == "" && r.RolePosition == "" {
		return fmt.Errorf("no filters specified")
	}

	return nil
}

func (r *HistoryRequest) Valid() error {
	if r.SummonerId == 0 {
		return fmt.Errorf("no summoner_id specified")
	}
	if r.Platform == "" {
		return fmt.Errorf("no platform specified")
	}
	if r.Limit == 0 {
		return fmt.Errorf("no limit specified")
	}
	return nil
}

func (r *MatchRequest) Valid() error {
	if r.SummonerId == 0 {
		return fmt.Errorf("no summoner_id specified")
	}
	if r.Platform == "" {
		return fmt.Errorf("no platform specified")
	}
	if r.MatchId == 0 {
		return fmt.Errorf("no match_id specified")
	}
	return nil
}

func (r *TeamRequest) Valid() error {
	if r.TeamId != 100 && r.TeamId != 200 {
		return fmt.Errorf("team_id must be 100 or 200")
	}
	if r.Platform == "" {
		return fmt.Errorf("no platform specified")
	}
	if r.MatchId == 0 {
		return fmt.Errorf("no match_id specified")
	}
	return nil
}

func (r *SearchRequest) Valid() error {
	if r.ChampionId == 0 {
		return fmt.Errorf("no champion_id specified")
	}
	if r.LastN == 0 {
		return fmt.Errorf("must specify last_n")
	}
	if r.RolePosition == "" {
		return fmt.Errorf("must specify role_position")
	}
	if r.TopStat == "" {
		return fmt.Errorf("must specify top_stat")
	}
	return nil
}

func (r *GoalCreateStatsRequest) Valid() error {
	if r.SummonerId == 0 || r.Platform == "" || r.RolePosition == "" || r.NumGoals == 0 || r.TargetAchievementCount == 0 || len(r.Categories) == 0 {
		return fmt.Errorf("must specify summoner_id, platform, role_position, num_goals, target_achievement_count, at least 1 value in categories")
	}

	return nil
}

func (r *GoalCreateCustomRequest) Valid() error {
	if r.SummonerId == 0 || r.Platform == "" || r.RolePosition == "" || r.UnderlyingStat == "" {
		return fmt.Errorf("must specify summoner_id, platform, role_position, underlying_stat")
	}

	return nil
}

func (r *GoalGetRequest) Valid() error {
	if r.SummonerId == 0 || r.Platform == "" {
		return fmt.Errorf("must specify summoner_id, platform")
	}
	return nil
}

func (r *GoalDeleteRequest) Valid() error {
	if r.SummonerId == 0 || r.Platform == "" || r.GoalId == "" {
		return fmt.Errorf("must specify summoner_id, platform, and goal_id")
	}
	return nil
}

func (r *GoalUpdateStatusRequest) Valid() error {
	if r.SummonerId == 0 || r.Platform == "" || r.GoalId == "" || r.Status == 0 {
		return fmt.Errorf("must specify summoner_id, platform, goal_id, status")
	}
	return nil
}

func (r *StatsRequest) BuildWhereClause() string {
	whereClause := "1=1"
	if r.Platform != "" {
		whereClause += fmt.Sprintf(` AND basic.platform_id="%s"`, r.Platform)
	}
	if r.QueueType != "" {
		whereClause += fmt.Sprintf(` AND basic.queue_type="%s"`, r.QueueType)
	}
	if r.PatchPrefix != "" {
		whereClause += fmt.Sprintf(` AND match_version LIKE "%s%%"`, r.PatchPrefix)
	}
	if r.Tier != "" {
		whereClause += fmt.Sprintf(` AND tier = "%s"`, r.Tier)
	}
	if r.Division != "" {
		whereClause += fmt.Sprintf(` AND division = "%s"`, r.Division)
	}
	if r.Lane != "" {
		whereClause += fmt.Sprintf(` AND lane = "%s"`, r.Lane)
	}
	if r.Role != "" {
		whereClause += fmt.Sprintf(` AND role = "%s"`, r.Role)
	}
	if r.ChampionId != 0 {
		whereClause += fmt.Sprintf(` AND champion_id = %d`, r.ChampionId)
	}
	if r.OpponentChampionId != 0 {
		whereClause += fmt.Sprintf(` AND opponent_champion_id = %d`, r.OpponentChampionId)
	}
	if r.SummonerId != 0 {
		whereClause += fmt.Sprintf(` AND basic.summoner_id = %d`, r.SummonerId)
	}

	return whereClause
}

// The filters that live in the advanced table should go here
func (r *StatsRequest) BuildAdvancedWhereClause() string {
	whereClause := "1=1"
	if r.RolePosition != "" {
		whereClause += fmt.Sprintf(` AND role_position = "%s"`, r.RolePosition)
	}

	return whereClause
}

func (r *SearchRequest) BuildWhereClause() string {
	whereClause := "basic.opponent_champion_id != 0"

	if r.Platform != "" {
		whereClause += fmt.Sprintf(` AND basic.platform_id = "%s"`, r.Platform)
	}
	if r.QueueType != "" {
		whereClause += fmt.Sprintf(` AND basic.queue_type = "%s"`, r.QueueType)
	}
	if r.PatchPrefix != "" {
		whereClause += fmt.Sprintf(` AND match_version like "%s%%"`, r.PatchPrefix)
	}
	if r.Tiers != nil {
		quotedStrings := make([]string, len(r.Tiers))
		for i, s := range r.Tiers {
			quotedStrings[i] = fmt.Sprintf(`"%s"`, s)
		}
		whereClause += fmt.Sprintf(` AND basic.tier in (%s)`, strings.Join(quotedStrings, ","))
	}
	if r.RolePosition != "" {
		whereClause += fmt.Sprintf(` AND role_position = "%s"`, r.RolePosition)
	}
	if r.ChampionId != 0 {
		whereClause += fmt.Sprintf(` AND basic.champion_id = %d`, r.ChampionId)
	}
	if r.OpponentChampionId != 0 {
		whereClause += fmt.Sprintf(` AND basic.opponent_champion_id = %d`, r.OpponentChampionId)
	}
	if r.GoodKills != nil {
		for _, k := range r.GoodKills {
			whereClause += fmt.Sprintf(` AND good_kills.%s > 0`, k)
		}
	}
	if r.ExcludeSummonerId != 0 {
		whereClause += fmt.Sprintf(` AND basic.summoner_id != %d`, r.ExcludeSummonerId)
	}

	return whereClause
}

func FromMetaRole(metaRole baseview.RolePosition) (lane string, role string) {
	switch metaRole {
	case baseview.RoleTop:
		return "TOP", "SOLO"
	case baseview.RoleMid:
		return "MIDDLE", "SOLO"
	case baseview.RoleJungle:
		return "JUNGLE", "NONE"
	case baseview.RoleSupport:
		return "BOTTOM", "DUO_SUPPORT"
	case baseview.RoleAdc:
		return "BOTTOM", "DUO_CARRY"
	}
	return "", ""
}

func GetMetaRole(lane, role string) baseview.RolePosition {
	for _, metaRole := range []baseview.RolePosition{
		baseview.RoleTop,
		baseview.RoleMid,
		baseview.RoleJungle,
		baseview.RoleSupport,
		baseview.RoleAdc} {
		if l, r := FromMetaRole(metaRole); l == lane && r == role {
			return metaRole
		}
	}
	return ""
}

func Round(f float64) int64 {
	return int64(math.Floor(f + .5))
}
