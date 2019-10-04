package server

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	bq "cloud.google.com/go/bigquery"
	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"

	"github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/constants/privileges"
	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/lolstats/gcd"
	"github.com/VantageSports/lolstats/generate"
	"github.com/VantageSports/lolstats/ingest"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/api"
	"github.com/VantageSports/users"
	"github.com/VantageSports/users/client"
)

type StatsServer struct {
	AuthClient      users.AuthCheckClient
	DsClient        *datastore.Client
	BQClient        *bigquery.Client
	FilesClient     *files.Client
	BasicTable      string
	AdvancedTable   string
	MatchDetailsDir string
	StatsDir        string
	LolUsersClient  lolusers.LolUsersClient
	ProjectID       string
}

func (s *StatsServer) History(ctx context.Context, in *lolstats.HistoryRequest) (out *lolstats.HistoryResponse, err error) {
	if _, err := s.validateCtx(ctx, privileges.LOLstatsRead); err != nil {
		return nil, err
	}

	if err := in.Valid(); err != nil {
		return nil, err
	}

	query := datastore.NewQuery(gcd.KindLolHistoryRow).
		Filter("summoner_id =", in.SummonerId).
		Filter("platform =", in.Platform).
		Order("-match_creation").
		Limit(int(in.Limit))
	if in.QueueType != "" {
		query = query.Filter("queue_type =", in.QueueType)
	}

	if in.Cursor != "" {
		cursor, err := datastore.DecodeCursor(in.Cursor)
		if err != nil {
			return nil, err
		}

		query = query.Start(cursor)
	}

	it := s.DsClient.Run(ctx, query)

	var res = &lolstats.HistoryResponse{
		Matches: []*lolstats.MatchSummary{},
	}
	var statRow gcd.HistoryRow
	for _, err = it.Next(&statRow); err == nil; _, err = it.Next(&statRow) {
		rowCopy, err := statRow.ToMatchSummary()
		if err != nil {
			return nil, err
		}
		res.Matches = append(res.Matches, rowCopy)
	}
	if err != iterator.Done {
		return nil, err
	}
	cur, err := it.Cursor()
	if err != nil {
		return nil, err
	}
	res.Cursor = cur.String()

	return res, nil
}

func (s *StatsServer) Details(ctx context.Context, in *lolstats.MatchRequest) (out *lolstats.DetailsResponse, err error) {
	if err := in.Valid(); err != nil {
		return nil, err
	}

	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	pIds, err := s.participantIdentitiesInMatch(ctx, in.MatchId, in.Platform)
	if err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, pIds, in.Platform); err != nil {
		return nil, err
	}

	// Fetch basic stats from gcs
	filename := fmt.Sprintf("%s/%d-%s-%d.basic.json", s.StatsDir, in.MatchId, in.Platform, in.SummonerId)
	data, err := s.FilesClient.Read(filename)

	return &lolstats.DetailsResponse{
		Basic: &lolstats.BasicStats{StatsJson: string(data)},
	}, err
}

func (s *StatsServer) Advanced(ctx context.Context, in *lolstats.MatchRequest) (*lolstats.AdvancedStats, error) {
	if err := in.Valid(); err != nil {
		return nil, err
	}

	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	pIds, err := s.participantIdentitiesInMatch(ctx, in.MatchId, in.Platform)
	if err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, pIds, in.Platform); err != nil {
		return nil, err
	}

	// Fetch advanced stats json from gcs
	filename := fmt.Sprintf("%s/%d-%s-%d.advanced.json", s.StatsDir, in.MatchId, in.Platform, in.SummonerId)
	data, err := s.FilesClient.Read(filename)

	return &lolstats.AdvancedStats{StatsJson: string(data)}, err
}

func (s *StatsServer) TeamDetails(ctx context.Context, in *lolstats.TeamRequest) (*lolstats.TeamDetailsResponse, error) {
	if err := in.Valid(); err != nil {
		return nil, err
	}

	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	pIds, err := s.participantIdentitiesInMatch(ctx, in.MatchId, in.Platform)
	if err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, pIds, in.Platform); err != nil {
		return nil, err
	}

	results := &lolstats.TeamDetailsResponse{
		Results: map[int64]*lolstats.BasicStats{},
	}

	// Fetch basic stats from gcs
	aggBasicStats := []generate.BasicStats{}
	for _, pId := range pIds {
		if baseview.TeamID(int64(pId.ParticipantID)) == in.TeamId {
			filename := fmt.Sprintf("%s/%d-%s-%d.basic.json", s.StatsDir, in.MatchId, in.Platform, pId.Player.SummonerID)
			data, err := s.FilesClient.Read(filename)
			if err != nil {
				// If we don't have the stats for a particular summoner, just skip it
				log.Warning(fmt.Sprintf("Unable to fetch basic stats %v: %v", filename, err))
				continue
			}
			results.Results[int64(pId.ParticipantID)] = &lolstats.BasicStats{StatsJson: string(data)}

			// Add to aggregate
			basicStats := generate.BasicStats{}
			err = json.Unmarshal(data, &basicStats)
			if err != nil {
				return nil, err
			}
			aggBasicStats = append(aggBasicStats, basicStats)
		}
	}

	if len(aggBasicStats) == 0 {
		return nil, fmt.Errorf("No basic stats found for match %v team %v", in.MatchId, in.TeamId)
	}

	// Average the aggregates
	averagedStats := generate.AverageBasicStats(aggBasicStats)
	averagedStatsStr, err := json.Marshal(averagedStats)
	if err != nil {
		return nil, err
	}

	results.Results[in.TeamId] = &lolstats.BasicStats{StatsJson: string(averagedStatsStr)}

	return results, nil
}

func (s *StatsServer) TeamAdvanced(ctx context.Context, in *lolstats.TeamRequest) (*lolstats.TeamAdvancedStats, error) {
	if err := in.Valid(); err != nil {
		return nil, err
	}

	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	pIds, err := s.participantIdentitiesInMatch(ctx, in.MatchId, in.Platform)
	if err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, pIds, in.Platform); err != nil {
		return nil, err
	}

	results := &lolstats.TeamAdvancedStats{
		Results: map[int64]string{},
	}

	// Fetch advanced stats from gcs
	aggAdvancedStats := []ingest.AdvancedStats{}
	for _, pId := range pIds {
		if baseview.TeamID(int64(pId.ParticipantID)) == in.TeamId {
			filename := fmt.Sprintf("%s/%d-%s-%d.advanced.json", s.StatsDir, in.MatchId, in.Platform, pId.Player.SummonerID)
			data, err := s.FilesClient.Read(filename)
			if err != nil {
				// If we don't have the stats for a particular summoner, just skip it
				log.Warning(fmt.Sprintf("Unable to fetch advanced stats %v: %v", filename, err))
				continue
			}

			results.Results[int64(pId.ParticipantID)] = string(data)

			// Add to aggregate
			advancedStats := ingest.AdvancedStats{}
			err = json.Unmarshal(data, &advancedStats)
			if err != nil {
				return nil, err
			}
			aggAdvancedStats = append(aggAdvancedStats, advancedStats)
		}
	}

	if len(aggAdvancedStats) == 0 {
		return nil, fmt.Errorf("No advanced stats found for match %v team %v", in.MatchId, in.TeamId)
	}

	// Average the aggregates
	averagedStats := ingest.AverageAdvancedStats(aggAdvancedStats)
	averagedStatsStr, err := json.Marshal(averagedStats)
	if err != nil {
		return nil, err
	}

	results.Results[in.TeamId] = string(averagedStatsStr)

	return results, nil
}

func (s *StatsServer) Percentiles(ctx context.Context, in *lolstats.StatsRequest) (out *lolstats.StatListResponse, err error) {
	if err := in.Valid(); err != nil {
		return nil, err
	}

	if _, err := s.validateCtx(ctx, privileges.LOLstatsRead); err != nil {
		return nil, err
	}

	if err := checkInvalidCharacters(append([]string{in.Platform, in.QueueType, in.PatchPrefix, in.Tier, in.Division, in.Lane, in.Role}, in.Selects...)); err != nil {
		return nil, err
	}

	for i := range in.Selects {
		// Quantiles returns a sequence of values. Concatenate the values together in order to return multiple columns in one query
		in.Selects[i] = fmt.Sprintf("APPROX_QUANTILES(%s, 100) as %s", translateExpression(in.Selects[i]), strings.Replace(in.Selects[i], ".", "_", -1))
	}

	var queryString string
	// If we're only fetching the last N games, then use a subquery to get the recent games, and then join it to the advanced table
	if in.LastN != 0 {
		queryString = fmt.Sprintf("SELECT count(*) as count, %s FROM (SELECT * FROM `%s` basic LEFT OUTER JOIN `%s` advanced ON basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id AND basic.match_id = advanced.match_id WHERE %s AND %s ORDER BY match_creation desc LIMIT %d)", strings.Join(in.Selects, ","), s.BasicTable, s.AdvancedTable, in.BuildWhereClause(), in.BuildAdvancedWhereClause(), in.LastN)
	} else {
		queryString = fmt.Sprintf("SELECT count(*) as count, %s FROM `%s` basic LEFT OUTER JOIN `%s` advanced ON basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id AND basic.match_id = advanced.match_id WHERE %s AND %s", strings.Join(in.Selects, ","), s.BasicTable, s.AdvancedTable, in.BuildWhereClause(), in.BuildAdvancedWhereClause())

	}

	results, err := DoQuery(ctx, s.BQClient, queryString)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// There should be 1 row returned, with a column for each select specified
	if len(results) != 1 {
		return nil, fmt.Errorf("expected 1 result. Found %v instead", len(results))
	}
	result1 := results[0]

	out = &lolstats.StatListResponse{
		Result: make(map[string]*lolstats.StatListResponse_StatList),
	}

	// Loop through each column
	for k, v := range result1 {
		if k == "count" {
			switch t := v.(type) {
			case int:
				out.Count = int64(t)
			case int64:
				out.Count = t
			}
		} else {
			// The value in each column should be an array of values
			values := v.([]bq.Value)
			out.Result[k] = &lolstats.StatListResponse_StatList{
				Values: make([]float64, len(values)),
			}
			for j := range values {
				switch t := values[j].(type) {
				case float64:
					out.Result[k].Values[j] = t
				case int:
					out.Result[k].Values[j] = float64(t)
				case int64:
					out.Result[k].Values[j] = float64(t)
				default:
					return nil, fmt.Errorf("unhandled type: %v", t)
				}
			}
		}

	}

	return out, nil
}

func (s *StatsServer) SummonerMatches(ctx context.Context, in *lolstats.StatsRequest) (out *lolstats.StatListResponse, err error) {
	if err := in.Valid(); err != nil {
		return nil, err
	}
	if in.LastN == 0 || in.SummonerId == 0 || in.Platform == "" {
		return nil, fmt.Errorf("Missing required parameters 'last_n', 'summoner_id' and 'platform'")
	}

	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	pId := api.ParticipantIdentity{Player: api.Player{SummonerID: in.SummonerId}}
	if err = s.validateSummonerAccess(ctx, claims, []api.ParticipantIdentity{pId}, in.Platform); err != nil {
		return nil, err
	}

	if err := checkInvalidCharacters(append([]string{in.Platform, in.QueueType, in.PatchPrefix, in.Tier, in.Division, in.Lane, in.Role}, in.Selects...)); err != nil {
		return nil, err
	}

	selectsQuery := make([]string, len(in.Selects))
	for i := range in.Selects {
		// Casting everything to floats makes it easier to build the response
		selectsQuery[i] = fmt.Sprintf("%s as %s", translateExpression(in.Selects[i]), strings.Replace(in.Selects[i], ".", "_", -1))
	}
	// Add match_id to give context to each row
	selectsQuery = append(selectsQuery, "basic_match_id as match_id")

	queryString := fmt.Sprintf("SELECT %s FROM (SELECT *,basic.match_id as basic_match_id FROM `%s` basic LEFT OUTER JOIN `%s` advanced ON basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id AND basic.match_id = advanced.match_id WHERE %s AND %s ORDER BY match_creation desc LIMIT %d) ORDER BY match_creation DESC", strings.Join(selectsQuery, ","), s.BasicTable, s.AdvancedTable, in.BuildWhereClause(), in.BuildAdvancedWhereClause(), in.LastN)

	results, err := DoQuery(ctx, s.BQClient, queryString)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	out = &lolstats.StatListResponse{
		Count:  int64(len(results)),
		Result: make(map[string]*lolstats.StatListResponse_StatList),
	}

	for i, row := range results {
		for k, v := range row {
			if _, ok := out.Result[k]; !ok {
				out.Result[k] = &lolstats.StatListResponse_StatList{
					Values: make([]float64, len(results)),
				}
			}
			switch t := v.(type) {
			case float64:
				out.Result[k].Values[i] = t
			case int:
				out.Result[k].Values[i] = float64(t)
			case int64:
				out.Result[k].Values[i] = float64(t)
			default:
				return nil, fmt.Errorf("unhandled type: %v", t)
			}
		}
	}

	return out, nil
}

func (s *StatsServer) Search(ctx context.Context, in *lolstats.SearchRequest) (out *lolstats.SearchResponse, err error) {
	if err := in.Valid(); err != nil {
		return nil, err
	}
	if _, err := s.validateCtx(ctx, privileges.LOLstatsRead); err != nil {
		return nil, err
	}

	orderBy := ""
	if in.TopStat == "laning" {
		orderBy = "gold_diff_zero_to_ten DESC"
	} else if in.TopStat == "engagements" {
		orderBy = "good_kills.total DESC"
	} else if in.TopStat == "cs_diff" {
		orderBy = "cs_per_minute_diff_zero_to_ten DESC"
	} else {
		return nil, fmt.Errorf("invalid top_stat")
	}

	if err := checkInvalidCharacters(append([]string{in.Platform, in.QueueType, in.PatchPrefix, in.RolePosition}, append(in.Tiers, in.GoodKills...)...)); err != nil {
		return nil, err
	}

	queryString := fmt.Sprintf("SELECT basic.match_id, basic.platform_id, basic.opponent_champion_id, basic.tier, basic.gold_diff_zero_to_ten, advanced.good_kills.*, basic.cs_per_minute_diff_zero_to_ten FROM `%s` basic INNER JOIN `%s` advanced ON basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id AND basic.match_id = advanced.match_id WHERE %s ORDER BY %s LIMIT %d", s.BasicTable, s.AdvancedTable, in.BuildWhereClause(), orderBy, in.LastN)

	results, err := DoQuery(ctx, s.BQClient, queryString)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	out = &lolstats.SearchResponse{}
	out.Count = int64(len(results))

	for _, row := range results {
		matchId := row["match_id"].(int64)
		platformId := row["platform_id"].(string)
		goldDiffZeroToTen := row["gold_diff_zero_to_ten"].(float64)
		opponentChampId := row["opponent_champion_id"].(int64)
		tier := row["tier"].(string)
		csDiff := row["cs_per_minute_diff_zero_to_ten"].(float64)

		// These are the good_kill columns
		goodKillsMap := map[string]int64{}
		for _, v := range []string{"total", "numbers_difference", "numbers_difference_lacking_vision",
			"health_difference", "gold_spent_difference", "neutral_damage_difference", "other"} {
			val, ok := row[v].(int64)
			// These columns can be null, if they were generated before we added good_kill logic
			if ok {
				goodKillsMap[v] = int64(val)
			}
		}

		entry := &lolstats.ReplayEntry{
			MatchId:            int64(matchId),
			Platform:           platformId,
			GoldDiffZeroToTen:  goldDiffZeroToTen,
			GoodKills:          goodKillsMap,
			OpponentChampionId: int64(opponentChampId),
			Tier:               tier,
			CsDiff:             csDiff,
		}
		out.Results = append(out.Results, entry)
	}

	return out, nil
}

func (s *StatsServer) validateCtx(ctx context.Context, claims string) (*users.Claims, error) {
	// Don't validate tokens in dev
	if s.ProjectID == "vs-dev" {
		return nil, nil
	}
	return client.ValidateCtxClaims(ctx, s.AuthClient, claims)
}

func (s *StatsServer) participantIdentitiesInMatch(ctx context.Context, matchID int64, platformID string) ([]api.ParticipantIdentity, error) {
	region := riot.RegionFromPlatform(riot.PlatformFromString(platformID))

	// Fetch the match details
	filename := fmt.Sprintf("%s/%d-%s.json", s.MatchDetailsDir, matchID, region)
	data, err := s.FilesClient.Read(filename)
	if err != nil {
		return nil, fmt.Errorf("Unable to fetch match details json for %v: %v", filename, err)
	}

	// Transform it into a MatchDetail object
	matchDetail := api.MatchDetail{}
	err = json.Unmarshal(data, &matchDetail)
	if err != nil {
		return nil, err
	}

	return matchDetail.ParticipantIdentities, nil
}

// In order to request data for a summoner, the user
func (s *StatsServer) validateSummonerAccess(ctx context.Context, claims *users.Claims, pIds []api.ParticipantIdentity, platformID string) error {
	// Don't validate summoner access in dev
	if s.ProjectID == "vs-dev" {
		return nil
	}

	resp, err := s.LolUsersClient.List(ctx, &lolusers.ListLolUsersRequest{UserId: claims.Sub})
	if err != nil {
		return err
	}

	region := riot.RegionFromPlatform(riot.PlatformFromString(platformID))

	for _, pId := range pIds {
		summonerStr := fmt.Sprintf("%d", pId.Player.SummonerID)
		for _, u := range resp.LolUsers {
			if u.SummonerId == summonerStr && u.Region == region.String() {
				return nil
			}
		}
	}

	return fmt.Errorf("user not allowed to request data for match")
}

// This returns an array of maps representing each result row
func DoQuery(ctx context.Context, bqClient *bigquery.Client, queryString string) ([]map[string]interface{}, error) {
	log.Debug(queryString)

	job, err := bqClient.Query(ctx, queryString, true)
	if err != nil {
		return nil, err
	}

	it, err := job.Read(ctx)
	if err != nil {
		return nil, err
	}

	rows := []map[string]interface{}{}
	parser := bigquery.RowParser{}

	for {
		err = it.Next(&parser)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		rows = append(rows, parser.LastRow())
	}

	return rows, nil
}

// translateExpression converts special "select" strings into formulas
func translateExpression(expr string) string {
	switch expr {
	case "kda":
		return "CASE WHEN deaths = 0 THEN (kills+assists)/1.0 ELSE (kills+assists)/deaths END"
	case "q_per_minute":
		return "ability_counts.Q*60/match_duration"
	case "w_per_minute":
		return "ability_counts.W*60/match_duration"
	case "e_per_minute":
		return "ability_counts.E*60/match_duration"
	case "r_per_minute":
		return "ability_counts.R*60/match_duration"
	case "q_per_minute_zero_to_ten":
		return "ability_counts_zero_to_ten.Q/10"
	case "w_per_minute_zero_to_ten":
		return "ability_counts_zero_to_ten.W/10"
	case "e_per_minute_zero_to_ten":
		return "ability_counts_zero_to_ten.E/10"
	case "r_per_minute_zero_to_ten":
		return "ability_counts_zero_to_ten.R/10"
	case "good_kills_total_per_minute":
		return "good_kills.total*60/match_duration"
	case "good_kills_numbers_difference_per_minute":
		return "good_kills.numbers_difference*60/match_duration"
	case "good_kills_numbers_difference_lacking_vision_per_minute":
		return "good_kills.numbers_difference_lacking_vision*60/match_duration"
	case "good_kills_health_difference_per_minute":
		return "good_kills.health_difference*60/match_duration"
	case "good_kills_gold_spent_difference_per_minute":
		return "good_kills.gold_spent_difference*60/match_duration"
	case "good_kills_neutral_damage_difference_per_minute":
		return "good_kills.neutral_damage_difference*60/match_duration"
	case "good_kills_other_per_minute":
		return "good_kills.other*60/match_duration"
	case "bad_deaths_total_per_minute":
		return "bad_deaths.total*60/match_duration"
	case "bad_deaths_numbers_difference_per_minute":
		return "bad_deaths.numbers_difference*60/match_duration"
	case "bad_deaths_numbers_difference_lacking_vision_per_minute":
		return "bad_deaths.numbers_difference_lacking_vision*60/match_duration"
	case "bad_deaths_health_difference_per_minute":
		return "bad_deaths.health_difference*60/match_duration"
	case "bad_deaths_gold_spent_difference_per_minute":
		return "bad_deaths.gold_spent_difference*60/match_duration"
	case "bad_deaths_neutral_damage_difference_per_minute":
		return "bad_deaths.neutral_damage_difference*60/match_duration"
	case "bad_deaths_other_per_minute":
		return "bad_deaths.other*60/match_duration"
	case "good_kills_other_minus_bad_deaths_other_per_minute":
		return "(good_kills.other-bad_deaths.other)*60/match_duration"
	// Using standard SQL (not leagcy SQL) forces you to escape keywords if they're column names
	case "map_coverages.full":
		return "map_coverages.`full`"
	case "attacks_per_minute_useful":
		return "attacks_per_minute/useful_percent.all*100"
	case "combo_damage_per_minute_useful":
		return "combo_damage_per_minute/useful_percent.all*100"
	}

	return expr
}

func checkInvalidCharacters(fields []string) error {
	// Prevent SQL injection
	sanitize := regexp.MustCompile("^[a-zA-Z0-9_.]*$")
	for _, val := range fields {
		if !sanitize.MatchString(val) {
			return fmt.Errorf("one or more selects has invalid characters in it: %v", val)
		}
	}
	return nil
}
