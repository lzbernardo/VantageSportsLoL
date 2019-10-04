package main

import (
	"flag"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/sheets/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	vsbigquery "github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/certs"
	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats/server"
	"github.com/VantageSports/riot"
	riotservice "github.com/VantageSports/riot/service"
	"github.com/VantageSports/users/client"
)

const (
	LatestGameVersion  = "6.24.1"
	ProjectID          = "vs-main"
	PlatformID         = "NA1"
	LolUsersAddr       = "130.211.166.215:443"
	UsersAddr          = "130.211.174.99:443"
	RiotAddr           = "130.211.154.47:443"
	BqDataset          = "lol"
	BqBasicTable       = "lolstats_basic"
	BqAdvancedTable    = "lolstats_advanced"
	VantageLeagueTable = "vantageleague"
)

var LegacyVPRStats = []string{
	"favorable_team_fight_percent",
	"good_kills_total_per_minute",
	"average_death_time_percentage",
	"bad_deaths_total_per_minute",
	"live_wards_average_pink",
	"live_wards_average_yellow_and_blue",
	"reveals_per_ward_average",
	"wards_killed_per_minute",
	"carry_focus_efficiency",
	"map_coverages_full",
	"useful_percent_all",
	"attacks_per_minute",
	"gold_diff_zero_to_ten",
	"gold_zero_to_ten",
	"level_6_seconds",
	"cs_per_minute_zero_to_ten",
	"cs_per_minute_diff_zero_to_ten",
	"damage_taken_percent_per_death",
}

var tlsCert string

func main() {
	flag.Parse()
	creds := google.MustEnvCreds(ProjectID, bigquery.Scope, sheets.SpreadsheetsReadonlyScope)
	log.Notice("Creating BigQuery service")
	bqClient, err := vsbigquery.NewClient(creds)
	exitIf(err)

	tlsCert, _ = certs.MustWriteDevCerts()
	riotClient := riotservice.NewRiotClient(getConn(RiotAddr))

	log.Notice("Creating sheets service")
	ctx := context.Background()
	srv, err := sheets.New(creds.Conf.Client(ctx))
	exitIf(err)

	matchIdsByDivision, err := getMatchIdsFromSpreadsheet(ctx, srv, "1ifsGvHXf4sSoUhxaPONwF2PwLYGN_MBifyY5Q9SM62w")

	// Get vpr percentiles by role_position
	vprPercentiles, err := getVprPercentiles(ctx, bqClient)
	if err != nil {
		log.Fatal(err)
	}

	for division, matchIds := range matchIdsByDivision {
		log.Notice("Processing division: " + division)
		// For each match
		for _, matchId := range matchIds {
			log.Notice(fmt.Sprintf("Processing match: %v", matchId))

			// Get each summoner+role in each match
			summonerIdStats, rolePositions, err := getSummonerIdStatsFromMatch(ctx, bqClient, matchId)
			if err != nil {
				log.Fatal(err)
			}

			if len(summonerIdStats) == 0 {
				log.Error("No data found for match")
				continue
			}

			// Translate the summoner ids to names
			summonerIdList := []int64{}
			for summonerId, _ := range summonerIdStats {
				summonerIdList = append(summonerIdList, summonerId)
			}
			nameMap := map[int64]string{}
			resp, err := riotClient.SummonersById(ctx, &riotservice.SummonerIDRequest{
				Region: riot.RegionFromPlatform(riot.Platform(PlatformID)).String(),
				Ids:    summonerIdList,
			})
			exitIf(err)
			for _, sum := range resp.Summoners {
				nameMap[sum.Id] = sum.Name
			}

			// Calculate VPRs
			for summonerId, stats := range summonerIdStats {
				vpr := getVPR(stats, vprPercentiles[rolePositions[summonerId]])
				fmt.Printf("%v,%v,%v\n", nameMap[summonerId], matchId, vpr)
			}
		}
	}
}

func getVprPercentiles(ctx context.Context, bqClient *vsbigquery.Client) (map[string]map[string]interface{}, error) {
	monthAgo := time.Now().Add(-30 * 24 * time.Hour)
	monthAgoStr := fmt.Sprintf("%v-%02d-%02d", monthAgo.Year(), int(monthAgo.Month()), monthAgo.Day())

	percentileSelects := []string{}
	for _, k := range LegacyVPRStats {
		withAlias := strings.Index(server.VPRSQL[k], " AS ")
		if withAlias != -1 {
			percentileSelects = append(percentileSelects, fmt.Sprintf("APPROX_QUANTILES(%s, 100)%s", server.VPRSQL[k][0:withAlias], server.VPRSQL[k][withAlias:]))
		} else {
			percentileSelects = append(percentileSelects, fmt.Sprintf("APPROX_QUANTILES(%s, 100)", server.VPRSQL[k]))
		}
	}

	percentilesByRole := make(map[string]map[string]interface{})
	// Get the percentiles for the VPR stats for that role
	for rolePosition, _ := range server.VPRWeights {
		fmt.Println("Position: " + rolePosition)
		queryString := fmt.Sprintf("SELECT count(*) as count, %s FROM `%s.%s` basic LEFT OUTER JOIN `%s.%s` advanced ON basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id AND basic.match_id = advanced.match_id WHERE basic.platform_id = \"%s\" and basic.queue_type in (%s) AND role_position = \"%s\" AND basic._PARTITIONTIME >= TIMESTAMP('%s')", strings.Join(percentileSelects, ","), BqDataset, BqBasicTable, BqDataset, BqAdvancedTable, PlatformID, strings.Join(server.VPRQueueTypes, ","), rolePosition, monthAgoStr)

		results, err := server.DoQuery(ctx, bqClient, queryString)
		if err != nil {
			return nil, err
		}

		percentilesByRole[rolePosition] = results[0]
	}

	return percentilesByRole, nil
}

func getSummonerIdStatsFromMatch(ctx context.Context, bqClient *vsbigquery.Client, matchId int64) (map[int64]map[string]float64, map[int64]string, error) {

	selects := []string{}
	for _, k := range LegacyVPRStats {
		selects = append(selects, server.VPRSQL[k])
	}

	queryString := fmt.Sprintf("SELECT basic.summoner_id as basic_summoner_id, role_position, %s FROM `%s.%s` basic LEFT OUTER JOIN `%s.%s` advanced ON basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id AND basic.match_id = advanced.match_id WHERE advanced.platform_id = \"%s\" and advanced.match_id = %d", strings.Join(selects, ","), BqDataset, BqBasicTable, BqDataset, BqAdvancedTable, PlatformID, matchId)

	results, err := server.DoQuery(ctx, bqClient, queryString)
	if err != nil {
		return nil, nil, err
	}

	retResults := map[int64]map[string]float64{}
	rolePositions := map[int64]string{}

	for _, row := range results {
		rowMap := map[string]float64{}

		sumId := row["basic_summoner_id"].(int64)
		position := row["role_position"].(string)
		rolePositions[sumId] = position

		for k, v := range row {
			if k == "basic_summoner_id" || k == "role_position" {
				continue
			}

			switch t := v.(type) {
			case int64:
				rowMap[k] = float64(t)
			case float64:
				rowMap[k] = t
			}
		}
		retResults[sumId] = rowMap
	}
	return retResults, rolePositions, nil
}

func getVPR(stats map[string]float64, percentiles map[string]interface{}) float64 {
	vprTotal := 0.0
	for k, myValue := range stats {
		myPercentile := 0

		percentileValues := percentiles[k].([]bigquery.Value)
		if server.VPRWeightsLowerIsBetter[k] {
			for j := range percentileValues {
				pValue := percentileValues[len(percentileValues)-1-j].(float64)
				if myValue >= pValue {
					myPercentile = j
					break
				}
			}
		} else {
			for j := range percentileValues {
				pValue := percentileValues[j].(float64)
				if myValue <= pValue {
					myPercentile = j
					break
				}
			}
		}

		vprTotal += float64(myPercentile)
	}
	return vprTotal / float64(len(stats))
}

func getConn(address string) *grpc.ClientConn {
	conn, err := grpc.Dial(address, getDialOpts()...)
	exitIf(err)

	return conn
}

func getDialOpts() []grpc.DialOption {
	c, _ := certs.ClientTLS(tlsCert, certs.Insecure(true))
	creds := credentials.NewTLS(c)
	return []grpc.DialOption{grpc.WithTransportCredentials(creds)}
}

func tokenCtx(token string) context.Context {
	return client.SetCtxToken(context.Background(), token)
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}
