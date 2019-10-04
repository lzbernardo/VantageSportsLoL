package main

import (
	"flag"
	"log"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/VantageSports/common/certs"
	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/users/client"
)

var (
	serverAddr     = flag.String("server_addr", "localhost:50015", "Target of server")
	serverCertPath = flag.String("server_cert", "", "Path to cert for server")
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

	goalsClient := lolstats.NewLolgoalsClient(conn)
	createStats(goalsClient)
	// createCustom(goalsClient)
	get(goalsClient)
	// delete(goalsClient)
	// updateStatus(goalsClient)
}

func createStats(g lolstats.LolgoalsClient) {
	resp, err := g.CreateStats(tokenCtx(), &lolstats.GoalCreateStatsRequest{
		SummonerId:             26337245,
		Platform:               "NA1",
		RolePosition:           "jng",
		NumGoals:               4,
		TargetAchievementCount: 5,
		Categories:             []lolstats.GoalCategory{lolstats.GoalCategory_DECISION_MAKING, lolstats.GoalCategory_MECHANICS},
	})
	if err != nil {
		log.Fatalln("ERROR IN CREATE STATS", err)
	}

	log.Println(resp)
}

func createCustom(g lolstats.LolgoalsClient) {
	resp, err := g.CreateCustom(tokenCtx(), &lolstats.GoalCreateCustomRequest{
		// SummonerId:   26337245,
		SummonerId:     26337245,
		Platform:       "NA1",
		RolePosition:   "jng",
		UnderlyingStat: "Pass the pop quiz",
	})
	if err != nil {
		log.Fatalln("ERROR IN CREATE CUSTOM", err)
	}

	log.Println(resp)
}

func get(g lolstats.LolgoalsClient) {
	resp, err := g.Get(tokenCtx(), &lolstats.GoalGetRequest{
		SummonerId: 26337245,
		Platform:   "NA1",
		Status:     lolstats.GoalStatus_NEW,
	})
	if err != nil {
		log.Fatalln("ERROR IN GET", err)
	}

	for _, val := range resp.Goals {
		log.Printf("Id: %v, Created: %v, LastUpdated: %v, status: %v, summonerId: %v, platform: %v, underlyingStat: %v, targetValue: %v, comparator: %v, achievement_count: %v, importance_weight: %v, category: %v, last_value: %v\n",
			val.Id, val.Created, val.LastUpdated, val.Status,
			val.SummonerId, val.Platform, val.UnderlyingStat,
			val.TargetValue, val.Comparator, val.AchievementCount,
			val.ImportanceWeight, val.Category, val.LastValue,
		)
	}
}

func delete(g lolstats.LolgoalsClient) {
	_, err := g.Delete(tokenCtx(), &lolstats.GoalDeleteRequest{
		SummonerId: 26337245,
		Platform:   "NA1",
		GoalId:     "26337245-NA1-jng-0-favorable_team_fight_percent-1",
	})
	if err != nil {
		log.Fatalln("ERROR IN DELETE", err)
	}
}

func updateStatus(g lolstats.LolgoalsClient) {
	_, err := g.UpdateStatus(tokenCtx(), &lolstats.GoalUpdateStatusRequest{
		SummonerId: 26337245,
		Platform:   "NA1",
		GoalId:     "26337245-NA1-jng-0-favorable_team_fight_percent-5",
		Status:     lolstats.GoalStatus_ACCEPTED,
	})
	if err != nil {
		log.Fatalln("ERROR IN UPDATE_STATUS", err)
	}
}

func tokenCtx() context.Context {
	return client.SetCtxToken(context.Background(), *token)
}
