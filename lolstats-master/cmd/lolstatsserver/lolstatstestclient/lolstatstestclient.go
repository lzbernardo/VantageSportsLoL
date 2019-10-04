package main

import (
	"flag"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/VantageSports/common/certs"
	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/users/client"
)

var (
	serverAddr     = flag.String("server_addr", "localhost:50014", "Target of user server")
	serverCertPath = flag.String("server_cert", "", "Path to cert for user server")
	serverName     = flag.String("server_name", "", "Name of server")
	token          = flag.String("token", "_fake_internal_key_", "Auth token")
	insecureGRPC   = false
)

func main() {
	flag.Parse()

	config, err := certs.ClientTLS(*serverCertPath, certs.Insecure(insecureGRPC))
	if err != nil {
		log.Fatalln(err)
	}

	conn, err := grpc.Dial(*serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(config)))
	if err != nil {
		log.Fatalln(err)
	}

	lolstatsClient := lolstats.NewLolstatsClient(conn)
	history(lolstatsClient)
	percentiles(lolstatsClient)
	details(lolstatsClient)
	advanced(lolstatsClient)
	summonerMatches(lolstatsClient)
	search(lolstatsClient)
	teamDetails(lolstatsClient)
	teamAdvanced(lolstatsClient)
}

func history(c lolstats.LolstatsClient) {
	resp, err := c.History(tokenCtx(), &lolstats.HistoryRequest{
		SummonerId: 26337245,
		Platform:   "NA1",
		Limit:      2,
		QueueType:  "TEAM_BUILDER_DRAFT_RANKED_5x5",
	})
	if err != nil {
		log.Fatalln("ERROR IN HISTORY", err)
	}

	log.Println(resp)
}

func percentiles(c lolstats.LolstatsClient) {
	resp, err := c.Percentiles(tokenCtx(), &lolstats.StatsRequest{
		Selects: []string{"kills", "deaths", "assists", "wards_placed",
			"total_damage_dealt_to_champions", "total_damage_taken_per_minute",
			"carry_focus_efficiency", "attacks_per_minute", "favorable_team_fights.count",
			"live_wards_average.yellow", "kda", "q_per_minute", "w_per_minute_zero_to_ten",
			"good_kills_total_per_minute", "bad_deaths_total_per_minute",
		},
		Platform:  "NA1",
		QueueType: string(riot.TB_RANKED_SOLO_5x5),
		LastN:     1000,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Percentiles count:", resp.Count)
	log.Println("Percentiles response:", resp.Result)
}

func details(c lolstats.LolstatsClient) {
	resp, err := c.Details(tokenCtx(), &lolstats.MatchRequest{
		SummonerId: 72976976,
		Platform:   "NA1",
		MatchId:    2371690559,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Details response:", resp)
}

func advanced(c lolstats.LolstatsClient) {
	resp, err := c.Advanced(tokenCtx(), &lolstats.MatchRequest{
		SummonerId: 70551369,
		Platform:   "NA1",
		MatchId:    2368391157,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("Advanced response:", resp)
}

func summonerMatches(c lolstats.LolstatsClient) {
	resp, err := c.SummonerMatches(tokenCtx(), &lolstats.StatsRequest{
		Selects: []string{"kills", "deaths", "assists", "wards_placed",
			"total_damage_dealt_to_champions", "total_damage_taken_per_minute",
			"carry_focus_efficiency", "attacks_per_minute", "favorable_team_fights.count",
			"live_wards_average.yellow", "kda", "q_per_minute", "w_per_minute_zero_to_ten",
		},
		Platform:   "NA1",
		QueueType:  string(riot.TB_RANKED_SOLO_5x5),
		LastN:      10,
		SummonerId: 68811099,
	})
	if err != nil {
		log.Fatalln(err)
	}

	log.Println("SummonerMatches count:", resp.Count)
	log.Println("SummonerMatches response:", resp.Result)
}

func search(c lolstats.LolstatsClient) {
	resp, err := c.Search(tokenCtx(), &lolstats.SearchRequest{
		Platform:     "NA1",
		QueueType:    string(riot.RANKED_FLEX_SR),
		PatchPrefix:  "6.22.",
		LastN:        10,
		ChampionId:   57,
		RolePosition: "top",
		TopStat:      "laning",
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Search Count: ", resp.Count)
	log.Println("Search Response: ", resp.Results)
}

func teamDetails(c lolstats.LolstatsClient) {
	resp, err := c.TeamDetails(tokenCtx(), &lolstats.TeamRequest{
		TeamId:   100,
		Platform: "NA1",
		MatchId:  2368391157,
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Team Details Response: ", resp)
}

func teamAdvanced(c lolstats.LolstatsClient) {
	resp, err := c.TeamAdvanced(tokenCtx(), &lolstats.TeamRequest{
		TeamId:   200,
		Platform: "NA1",
		MatchId:  2414627721,
	})
	if err != nil {
		log.Fatalln(err)
	}
	log.Println("Team Advanced Response: ", resp)
}

func tokenCtx() context.Context {
	return client.SetCtxToken(context.Background(), *token)
}
