package main

import (
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc/grpclog"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/lolstats/gcd"
	"github.com/VantageSports/lolstats/ingest"
	"github.com/VantageSports/lolstats/server"
)

var (
	projectID = env.Must("GOOG_PROJECT_ID")
	queueID   = env.Must("INPUT_QUEUE_ID")
	statsDir  = env.Must("MATCH_STATS_STORE_LOCATION")
)

func init() {
	grpclog.SetLogger(log.NewGRPCAdapter(log.Quiet))
}

func main() {
	creds := google.MustEnvCreds(projectID, pubsub.ScopePubSub, datastore.ScopeDatastore)
	log.Debug("Creating pubsub client")
	sub, err := queue.InitClient(creds)
	exitIf(err)

	log.Debug("Creating gcd client")
	ctx := context.Background()
	gcdClient, err := datastore.NewClient(ctx, creds.ProjectID, option.WithTokenSource(creds.Conf.TokenSource(ctx)))
	exitIf(err)

	log.Debug("Creating files client")
	gcsProvider, err := files.InitClient(files.AutoRegisterGCS(projectID, storage.ScopeReadOnly))
	exitIf(err)

	goalsUpdater := GoalsUpdater{
		GcdClient:   gcdClient,
		FilesClient: gcsProvider,
		StatsDir:    statsDir,
	}

	tr := queue.NewTaskRunner(sub, queueID, 1, time.Duration(2*time.Minute))

	log.Debug("Created taskRunner")
	tr.Start(context.Background(), goalsUpdater.Handle)
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type GoalsUpdater struct {
	GcdClient   *datastore.Client
	FilesClient *files.Client
	StatsDir    string
}

func (f *GoalsUpdater) Handle(ctx context.Context, m *pubsub.Message) error {
	// Deserialize the message
	msg := messages.LolGoalsUpdate{}
	log.Debug(string(m.Data))
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		return err
	}
	if err := msg.Valid(); err != nil {
		log.Error("Invalid message: " + err.Error())
		return err
	}

	// Check to see if goals exist. Should weed out a bunch of messages
	exists, err := f.hasGoals(ctx, msg)
	if err != nil {
		return err
	}
	if !exists {
		log.Info(fmt.Sprintf("No goals found for %v-%v-%v-%v", msg.SummonerId, msg.PlatformId, msg.RolePosition, msg.ChampionId))
		return nil
	}

	// Get all their non-completed goals
	allGoals, err := f.getAllGoals(ctx, msg)
	if err != nil {
		return err
	}

	goalsToUpdate := []gcd.LolGoal{}
	for _, g := range allGoals {
		// Only consider goals for the same position/champion
		if g.RolePosition == msg.RolePosition && g.ChampionID == msg.ChampionId &&
			// Make sure it's not a custom goal
			server.VPRSQL[g.UnderlyingStat] != "" &&
			// Make sure it's accepted
			g.Status == lolstats.GoalStatus_name[int32(lolstats.GoalStatus_ACCEPTED)] &&
			// Make sure it's not the last match (since pubsub will duplicate messages)
			(g.LastMatchID == 0 || g.LastMatchID != msg.MatchId) {

			goalsToUpdate = append(goalsToUpdate, g)
		}
	}

	log.Info(fmt.Sprintf("Found %v goals to update for %v-%v-%v-%v", len(goalsToUpdate), msg.SummonerId, msg.PlatformId, msg.RolePosition, msg.ChampionId))
	if len(goalsToUpdate) == 0 {
		return nil
	}

	matchDetails, err := f.getMatchStats(msg)
	if err != nil {
		return err
	}

	categoriesToClear := map[string]bool{}
	completedGoals := map[string]bool{}

	for _, g := range goalsToUpdate {
		// Update the last values for each goal.
		g.LastValue, err = server.TranslateStatToPerGameValue(matchDetails[g.UnderlyingStat], g.UnderlyingStat)
		if err != nil {
			return err
		}
		g.LastMatchID = msg.MatchId

		// If lastValue breaches that target
		if isBreach(g, g.LastValue) {
			g.AchievementCount++

			log.Info(fmt.Sprintf("Goal complete for %v-%v-%v-%v - %v is %v %v (%v), Progress: (%v/%v)",
				msg.SummonerId, msg.PlatformId, msg.RolePosition, msg.ChampionId, g.UnderlyingStat,
				g.Comparator, g.TargetValue, g.LastValue, g.AchievementCount, g.TargetAchievementCount))
			// If we were one away from the target achievement count, then it's complete
			if g.AchievementCount == g.TargetAchievementCount {
				g.Status = lolstats.GoalStatus_name[int32(lolstats.GoalStatus_COMPLETED)]
				// Mark categories that need to be cleared
				categoriesToClear[g.Category] = true
				completedGoals[g.GenerateId()] = true
			}
		}

		err := f.writeToGcd(ctx, g)
		if err != nil {
			return err
		}
	}

	// Clear out categories
	for k, _ := range categoriesToClear {
		for _, g := range allGoals {
			if g.Category == k && !completedGoals[g.GenerateId()] {
				log.Info(fmt.Sprintf("Deleting goal: %v", g.GenerateId()))
				err := f.deleteFromGcd(ctx, g.GenerateId())
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (f *GoalsUpdater) hasGoals(ctx context.Context, msg messages.LolGoalsUpdate) (bool, error) {
	prefixStr := fmt.Sprintf("%v-%v-%v-%v", msg.SummonerId, msg.PlatformId, msg.RolePosition, msg.ChampionId)
	keyPrefix := datastore.NameKey(gcd.KindLolGoal, prefixStr, nil)
	keyPrefixEnd := datastore.NameKey(gcd.KindLolGoal, fmt.Sprintf("%v0", prefixStr), nil)
	query := datastore.NewQuery(gcd.KindLolGoal).
		Filter("__key__ >", keyPrefix).
		Filter("__key__ <", keyPrefixEnd).
		KeysOnly()

	it := f.GcdClient.Run(ctx, query)
	row := gcd.LolGoal{}
	_, err := it.Next(&row)
	if err == nil {
		// We found something.
		return true, nil
	} else if err != iterator.Done {
		return false, err
	} else {
		return false, nil
	}
}

func (f *GoalsUpdater) getAllGoals(ctx context.Context, msg messages.LolGoalsUpdate) ([]gcd.LolGoal, error) {
	prefixStr := fmt.Sprintf("%v-%v-%v-%v", msg.SummonerId, msg.PlatformId, msg.RolePosition, msg.ChampionId)
	keyPrefix := datastore.NameKey(gcd.KindLolGoal, prefixStr, nil)
	keyPrefixEnd := datastore.NameKey(gcd.KindLolGoal, fmt.Sprintf("%v0", prefixStr), nil)
	query := datastore.NewQuery(gcd.KindLolGoal).
		Filter("__key__ >", keyPrefix).
		Filter("__key__ <", keyPrefixEnd)

	it := f.GcdClient.Run(ctx, query)

	goals := []gcd.LolGoal{}
	row := gcd.LolGoal{}
	var err error
	for _, err = it.Next(&row); err == nil; _, err = it.Next(&row) {
		// Ignore completed goals. Don't touch them
		if row.Status == lolstats.GoalStatus_name[int32(lolstats.GoalStatus_COMPLETED)] ||
			row.Status == lolstats.GoalStatus_name[int32(lolstats.GoalStatus_COMPLETED_SEEN)] {
			continue
		}
		rowCopy := row
		goals = append(goals, rowCopy)
	}
	if err != iterator.Done {
		return nil, err
	}
	return goals, nil
}

func (f *GoalsUpdater) getMatchStats(msg messages.LolGoalsUpdate) (map[string]float64, error) {
	latestGame := map[string]float64{}

	basicFilename := fmt.Sprintf("%s/%d-%s-%d.basic.json", f.StatsDir, msg.MatchId, msg.PlatformId, msg.SummonerId)
	basicData, err := f.FilesClient.Read(basicFilename)
	if err != nil {
		log.Error("error reading basic stats file: " + basicFilename)
		return nil, err
	}
	basicJson := make(map[string]interface{})
	err = json.Unmarshal(basicData, &basicJson)
	if err != nil {
		log.Error("error unmarshalling basic stats file")
		return nil, err
	}
	for k, v := range basicJson {
		switch t := v.(type) {
		case int64:
			latestGame[k] = float64(t)
		case float64:
			latestGame[k] = t
		}
	}

	advancedFilename := fmt.Sprintf("%s/%d-%s-%d.advanced.json", f.StatsDir, msg.MatchId, msg.PlatformId, msg.SummonerId)
	advancedData, err := f.FilesClient.Read(advancedFilename)
	if err != nil {
		log.Error("error reading advanced stats file: " + advancedFilename)
		return nil, err
	}
	advancedJson := make(map[string]interface{})
	err = json.Unmarshal(advancedData, &advancedJson)
	if err != nil {
		log.Error("error unmarshalling advanced stats file")
		return nil, err
	}
	for k, v := range advancedJson {
		switch t := v.(type) {
		case float64:
			latestGame[k] = t
		}
	}
	// special cases...
	matchDuration := basicJson["match_duration"].(float64)
	advancedStats := ingest.AdvancedStats{}
	err = json.Unmarshal(advancedData, &advancedStats)
	if err != nil {
		log.Error("error unmarshalling advanced stats into struct")
		return nil, err
	}

	latestGame["attacks_per_minute_useful"] = 0
	if advancedStats.UsefulPercent["all"] != 0 {
		latestGame["attacks_per_minute_useful"] = advancedStats.AttacksPerMinute * 100.0 / advancedStats.UsefulPercent["all"]
	}
	latestGame["map_coverages_full"] = advancedStats.MapCoverages["full"]
	latestGame["useful_percent_all"] = advancedStats.UsefulPercent["all"]
	latestGame["live_wards_average_pink"] = advancedStats.LiveWardsAverage["pink"]
	latestGame["live_wards_average_yellow_and_blue"] = advancedStats.LiveWardsAverage["yellow_and_blue"]
	latestGame["bad_deaths_total_per_minute"] = float64(advancedStats.BadDeaths.Total*60) / float64(matchDuration)
	latestGame["good_kills_total_per_minute"] = float64(advancedStats.GoodKills.Total*60) / float64(matchDuration)

	return latestGame, nil
}

func isBreach(goal gcd.LolGoal, value float64) bool {
	switch goal.Comparator {
	case lolstats.GoalComparator_name[int32(lolstats.GoalComparator_GREATER_THAN)]:
		return value > goal.TargetValue
	case lolstats.GoalComparator_name[int32(lolstats.GoalComparator_GREATER_THAN_OR_EQUAL)]:
		return value >= goal.TargetValue
	case lolstats.GoalComparator_name[int32(lolstats.GoalComparator_LESS_THAN_OR_EQUAL)]:
		return value <= goal.TargetValue
	case lolstats.GoalComparator_name[int32(lolstats.GoalComparator_LESS_THAN)]:
		return value < goal.TargetValue
	default:
		log.Error("Invalid comparator: " + goal.Comparator)
		return false
	}
}

func (f *GoalsUpdater) writeToGcd(ctx context.Context, goal gcd.LolGoal) error {
	key := datastore.NameKey(gcd.KindLolGoal, goal.GenerateId(), nil)

	_, err := f.GcdClient.Put(ctx, key, &goal)
	return err
}

func (f *GoalsUpdater) deleteFromGcd(ctx context.Context, id string) error {
	key := datastore.NameKey(gcd.KindLolGoal, id, nil)
	return f.GcdClient.Delete(ctx, key)
}
