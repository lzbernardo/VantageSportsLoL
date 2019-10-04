package main

import (
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	vsbigquery "github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/certs"
	"github.com/VantageSports/common/cli"
	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot"
	riotservice "github.com/VantageSports/riot/service"
	"github.com/VantageSports/users"
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

var clToken = flag.String("token", "", "Reuse an existing jwt token")
var clSummonerIdList = flag.String("summonerIdList", "", "Use a hard coded list of summoner ids")
var clClean = flag.Bool("clean", false, "Force regenerate table")

var tlsCert string

var statDefs = [][]string{
	{"CS per minute zero to ten", "basic_cs_per_minute_zero_to_ten"},
	{"CS per minute diff zero to ten", "basic_cs_per_minute_diff_zero_to_ten"},
	{"Neutral minions killed per minute", "basic_neutral_minions_killed_per_minute"},
	{"Enemy jungle neutral minions killed per minute", "basic_neutral_minions_killed_enemy_jungle_per_minute"},
	{"Gold at 10 minutes", "(basic_gold_zero_to_ten*10+500)"},
	{"Gold diff at 10 minutes", "basic_gold_diff_zero_to_ten*10"},
	{"Gold between 10-20 minutes", "basic_gold_ten_to_twenty*10"},
	{"Gold diff between 10-20 minutes", "basic_gold_diff_ten_to_twenty*10"},
	{"Gold between 20-30 minutes", "basic_gold_twenty_to_thirty*10"},
	{"Gold diff between 20-30 minutes", "basic_gold_diff_twenty_to_thirty*10"},
	{"Time to level 6", "basic_level_6_seconds"},
	{"Time diff to level 6", "basic_level_6_diff_seconds"},
	{"Wards placed per minute", "basic_wards_placed_per_minute"},
	{"Wards killed per minute", "basic_wards_killed_per_minute"},
	{"Damage to champions per minute", "basic_total_damage_dealt_to_champions_per_minute"},
	{"Average death time percentage", "basic_average_death_time_percentage"},
	{"Survivability ratio", "advanced_damage_taken_percent_per_death"},
	{"Carry focus efficiency", "advanced_carry_focus_efficiency"},
	{"Map coverage", "advanced_map_coverages_full"},
	{"Sight wards uptime", "COALESCE(advanced_live_wards_average_yellow_and_blue,0)"},
	{"Control wards uptime", "advanced_live_wards_average_pink"},
	{"Reveals per ward average", "advanced_reveals_per_ward_average"},
	{"Favorable fight percentage", "advanced_favorable_team_fight_percent"},
	{"KDA", "CASE WHEN basic_deaths = 0 THEN (basic_kills+basic_assists)/1.0 ELSE (basic_kills+basic_assists)/basic_deaths END"},
	{"Smart kills per minute", "advanced_good_kills_total*60/basic_match_duration"},
	{"Worthless deaths per minute", "CASE WHEN advanced_bad_deaths_total IS NULL THEN 99999 ELSE advanced_bad_deaths_total*60/basic_match_duration END"},
}

var lowerIsBetter = map[string]bool{
	"Average death time percentage": true,
	"Time to level 6":               true,
	"Worthless deaths per minute":   true,
}

type StatEntry struct {
	SummonerId   int64   `json:"summoner_id"`
	PlatformId   string  `json:"platform_id"`
	ChampionId   int64   `json:"champion_id"`
	ChampionIcon string  `json:"champion_icon"`
	SummonerName string  `json:"summoner_name"`
	MatchId      int64   `json:"match_id"`
	StatValue    float64 `json:"stat_value"`
	Count        int64   `json:"count"`
}

type StatCollection map[string][]*StatEntry

func main() {
	flag.Parse()

	creds := google.MustEnvCreds(ProjectID, bigquery.Scope)
	log.Notice("Creating BigQuery service")
	bqClient, err := vsbigquery.NewClient(creds)
	exitIf(err)

	tlsCert, _ = certs.MustWriteDevCerts()

	riotClient := riotservice.NewRiotClient(getConn(RiotAddr))

	hasAllCached := true
	if *clClean {
		hasAllCached = false
	}
	topStats, err := loadCachedStats("cache_topStats.json")
	if err != nil {
		hasAllCached = false
	}
	averageStats, err := loadCachedStats("cache_avgStats.json")
	if err != nil {
		hasAllCached = false
	}

	if !hasAllCached {
		var summoners []string
		if *clSummonerIdList == "" {
			// Get all the summoners in Vantage League
			token := getTokenString(*clToken)
			usersClient := users.NewUsersClient(getConn(UsersAddr))
			lolusersClient := lolusers.NewLolUsersClient(getConn(LolUsersAddr))
			summoners = getSummonerIdsFromGroup(usersClient, lolusersClient, token, "5678724825481216")
		} else {
			var summonerInts []int64
			exitIf(json.DecodeFile(*clSummonerIdList, &summonerInts))
			for _, v := range summonerInts {
				s := strconv.FormatInt(v, 10)
				summoners = append(summoners, s)
			}
		}
		// Look for vantage league games using those summoners
		matchIds := getVantageLeagueGames(bqClient, summoners, 60, 6)

		// Update the vantage league table
		exitIf(updateVantageLeagueTable(bqClient, matchIds))

		// Get top of each stat
		topStats = getStatCollection(bqClient, "singleGame", 10)

		// Get average of each stat
		averageStats = getStatCollection(bqClient, "average", 10)

		// Save the raw data, in case the riot calls fail, and we have to re-run
		exitIf(json.Write("cache_topStats.json", topStats, 0666))
		exitIf(json.Write("cache_avgStats.json", averageStats, 0666))
	}

	// Fill in the summoner names
	fillSummonerNames(riotClient, topStats)
	fillSummonerNames(riotClient, averageStats)

	// Fill in champ icons
	fillChampIcons(topStats)

	// Generate the html
	generateHtml(statDefs, topStats, "top10v2.html")
	generateHtml(statDefs, averageStats, "avg10v2.html")
}

func loadCachedStats(filename string) (StatCollection, error) {
	stats := StatCollection{}
	err := json.DecodeFile(filename, &stats)
	return stats, err
}

func getTokenString(clToken string) string {
	if clToken == "" {
		email := cli.ReadString("Enter admin credentials\nEmail")
		password, err := cli.ReadPassword("Password")
		exitIf(err)

		token := getToken(UsersAddr, email, string(password))
		fmt.Println("Token (can pass in with the -token arg)\n" + token)
		return token
	} else {
		return clToken
	}
}

func getToken(usersServerAddr, email, password string) string {
	authGenClient := users.NewAuthGenClient(getConn(UsersAddr))

	resp, err := authGenClient.GenerateToken(tokenCtx(""), &users.LoginRequest{
		Email:    email,
		Password: password,
	})
	exitIf(err)
	return resp.Token
}

func getSummonerIdsFromGroup(usersClient users.UsersClient, lolusersClient lolusers.LolUsersClient, token, groupId string) []string {
	fmt.Println("Looking up users in group: " + groupId)
	resp, err := usersClient.ListUsers(tokenCtx(token), &users.ListUsersRequest{
		GroupId: groupId,
	})
	exitIf(err)
	fmt.Printf("Found %v users\n", len(resp.Users))

	result := []*lolusers.LolUser{}

	fmt.Println("Looking up loluser for each user")
	for _, user := range resp.Users {
		loluserResp, err := lolusersClient.List(tokenCtx(token), &lolusers.ListLolUsersRequest{
			UserId: user.Id,
		})
		exitIf(err)

		result = append(result, loluserResp.LolUsers...)
	}
	fmt.Printf("Found %v lolusers\n", len(result))

	summonerIdList := []string{}
	for _, s := range result {
		summonerIdList = append(summonerIdList, s.SummonerId)
	}

	return summonerIdList
}

func getVantageLeagueGames(client *vsbigquery.Client, summonerIdList []string, daysBack int, minLeaguePlayers int) []int64 {
	ctx := context.Background()

	startTime := time.Now().Add(time.Duration(daysBack) * -24 * time.Hour)
	timeStr := fmt.Sprintf("%v-%v-%v", startTime.Year(), int(startTime.Month()), startTime.Day())

	queryStr := fmt.Sprintf(
		"SELECT	match_id, MAX(last_updated), SUM(CASE WHEN summoner_id IN (%s) THEN 1 ELSE 0 END) as num_in_league FROM [%s.%s]"+
			" WHERE match_id in ( SELECT match_id FROM [%s.%s] WHERE summoner_id in (%s) AND queue_type = 'CUSTOM'"+
			" AND platform_id = '%s' AND last_updated > TIMESTAMP('%s') AND _PARTITIONTIME > TIMESTAMP('%s') GROUP BY match_id)"+
			" AND _PARTITIONTIME > TIMESTAMP('%s') GROUP BY match_id HAVING num_in_league >= %d ORDER BY match_id DESC",
		strings.Join(summonerIdList, ","), BqDataset, BqAdvancedTable,
		BqDataset, BqBasicTable, strings.Join(summonerIdList, ","), PlatformID,
		timeStr, timeStr, timeStr, minLeaguePlayers,
	)
	fmt.Println("Looking up all vantage league games:\n" + queryStr)

	job, err := client.Query(ctx, queryStr, false)
	exitIf(err)

	it, err := job.Read(ctx)
	exitIf(err)

	rows := []int64{}
	parser := vsbigquery.RowParser{}

	for {
		err = it.Next(&parser)
		if err == iterator.Done {
			break
		}
		exitIf(err)

		row := parser.LastRow()

		match_id := row["match_id"].(int64)
		rows = append(rows, match_id)
	}

	fmt.Printf("Found %v matches\n", len(rows))
	return rows
}

func updateVantageLeagueTable(client *vsbigquery.Client, matches []int64) error {

	ctx := context.Background()
	matchStrings := []string{}
	for _, id := range matches {
		matchStrings = append(matchStrings, strconv.FormatInt(id, 10))
	}

	queryStr := fmt.Sprintf("SELECT * FROM %s.%s basic JOIN %s.%s advanced"+
		" ON basic.match_id = advanced.match_id	AND basic.summoner_id = advanced.summoner_id AND basic.platform_id = advanced.platform_id"+
		" WHERE	basic.platform_id = '%s' AND basic.match_id in (%s)", BqDataset, BqBasicTable, BqDataset, BqAdvancedTable, PlatformID, strings.Join(matchStrings, ","))

	fmt.Printf("Copying all game data to %s:\n%v\n", VantageLeagueTable, queryStr)

	q := client.Client.Query(queryStr)
	q.UseStandardSQL = false
	q.Dst = &bigquery.Table{
		ProjectID: ProjectID,
		DatasetID: BqDataset,
		TableID:   VantageLeagueTable,
	}
	q.WriteDisposition = bigquery.WriteTruncate

	job, err := q.Run(ctx)
	exitIf(err)

	return client.WaitForJob(ctx, job)
}

func getStatCollection(client *vsbigquery.Client, statType string, topN int) StatCollection {
	ctx := context.Background()

	result := StatCollection{}
	for _, ent := range statDefs {
		n, q := ent[0], ent[1]

		sortDirection := "DESC"
		if lowerIsBetter[n] {
			sortDirection = "ASC"
		}

		var queryStr string
		if statType == "singleGame" {
			queryStr = fmt.Sprintf("SELECT * FROM (SELECT basic_summoner_id, basic_platform_id, basic_match_id, basic_champion_id,"+
				" ROW_NUMBER() OVER (PARTITION BY basic_summoner_id, basic_platform_id ORDER BY stat %s) AS r,"+
				" %s AS stat FROM [%s.%s] )"+
				" WHERE r = 1 ORDER BY stat %s LIMIT %d", sortDirection, q, BqDataset, VantageLeagueTable, sortDirection, topN)
		} else if statType == "average" {
			queryStr = fmt.Sprintf("SELECT basic_summoner_id, basic_platform_id, COUNT(*) AS num, AVG(%s) AS stat"+
				" FROM [%s.%s] GROUP BY basic_summoner_id, basic_platform_id ORDER BY stat %s LIMIT %d",
				q, BqDataset, VantageLeagueTable, sortDirection, topN)
		} else {
			log.Fatal("Unknown type: " + statType)
		}
		fmt.Printf("Looking up %v stats for %v\n%v\n", statType, n, queryStr)

		job, err := client.Query(ctx, queryStr, false)
		exitIf(err)

		it, err := job.Read(ctx)
		exitIf(err)

		rows := []*StatEntry{}
		parser := vsbigquery.RowParser{}

		for {
			err = it.Next(&parser)
			if err == iterator.Done {
				break
			}
			exitIf(err)

			row := parser.LastRow()

			entry := &StatEntry{
				SummonerId: row["basic_summoner_id"].(int64),
				PlatformId: row["basic_platform_id"].(string),
				StatValue:  row["stat"].(float64),
			}
			if statType == "singleGame" {
				entry.ChampionId = row["basic_champion_id"].(int64)
				entry.MatchId = row["basic_match_id"].(int64)
			} else if statType == "average" {
				entry.Count = row["num"].(int64)
			}

			fmt.Printf("%v %v %v, %v %v %v\n", entry.SummonerId, entry.PlatformId, entry.ChampionId, entry.MatchId, entry.StatValue, entry.Count)
			rows = append(rows, entry)
		}
		result[n] = rows
	}
	return result
}

func fillSummonerNames(client riotservice.RiotClient, topStats StatCollection) {
	summonerIds := map[string]bool{}

	// Get a list of unique summoner/platform id combos
	for _, stats := range topStats {
		for _, stat := range stats {
			summonerIds[fmt.Sprintf("%v-%v", stat.SummonerId, stat.PlatformId)] = true
		}
	}

	// Max based on https://developer.riotgames.com/api/methods#!/1208/4681
	maxIdsPerRequest := 40
	batches := map[string][]int64{}
	// Do this in batches to save api calls
	for id, _ := range summonerIds {
		parts := strings.Split(id, "-")
		sumId, _ := strconv.ParseInt(parts[0], 10, 64)
		platformId := parts[1]

		// Group up ids by platform
		batches[platformId] = append(batches[platformId], sumId)

		// If we find a platform that's full, then send the request and empty it
		for plat, batch := range batches {
			if len(batch) == maxIdsPerRequest {
				resp := sendSummonerIdRequest(client, plat, batch)

				fillTopStats(plat, topStats, resp)
				batches[plat] = []int64{}
			}
		}
	}

	// Clear out our batches at the end
	for plat, batch := range batches {
		if len(batch) > 0 {
			resp := sendSummonerIdRequest(client, plat, batch)

			fillTopStats(plat, topStats, resp)
			batches[plat] = []int64{}
		}
	}
}

func fillTopStats(plat string, topStats StatCollection, resp map[int64]string) {
	for _, entries := range topStats {
		for _, entry := range entries {
			if entry.PlatformId == plat {
				val, found := resp[entry.SummonerId]
				if found {
					entry.SummonerName = val
				}
			}
		}
	}
}

func sendSummonerIdRequest(client riotservice.RiotClient, platform string, batch []int64) map[int64]string {
	ctx := context.Background()

	resp, err := client.SummonersById(ctx, &riotservice.SummonerIDRequest{
		Region: riot.RegionFromPlatform(riot.Platform(platform)).String(),
		Ids:    batch,
	})
	exitIf(err)

	result := map[int64]string{}
	for _, sum := range resp.Summoners {
		result[sum.Id] = sum.Name
		fmt.Printf("%v = %v\n", sum.Id, sum.Name)
	}
	return result
}

func fillChampIcons(topStats StatCollection) {
	resp, err := http.Get(fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/%s/data/en_US/champion.json", LatestGameVersion))
	exitIf(err)
	defer resp.Body.Close()

	var data map[string]interface{}
	exitIf(json.DecodeIf(resp.Body, 0, &data))

	champs := data["data"].(map[string]interface{})
	for champName, val := range champs {
		valMap := val.(map[string]interface{})
		key := valMap["key"].(string)
		keyInt, _ := strconv.ParseInt(key, 10, 64)

		for _, stats := range topStats {
			for _, stat := range stats {
				if stat.ChampionId == keyInt {
					stat.ChampionIcon = fmt.Sprintf("http://ddragon.leagueoflegends.com/cdn/%s/img/champion/%s.png", LatestGameVersion, champName)
				}
			}
		}

	}
}

func generateHtml(statDefs [][]string, topStats StatCollection, outFile string) {
	t, err := template.ParseFiles("html.template")
	if err != nil {
		fmt.Println(err)
		return
	}

	context := struct {
		StatDefs [][]string
		TopStats StatCollection
	}{
		StatDefs: statDefs,
		TopStats: topStats,
	}

	var output bytes.Buffer
	err = t.Execute(&output, context)
	if err != nil {
		fmt.Println(err)
		return
	}

	ioutil.WriteFile(outFile, output.Bytes(), 0666)
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
