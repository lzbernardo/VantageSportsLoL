package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/files"
	vjson "github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/api"
)

var (
	apiKey                   = env.SmartString("RIOT_API_KEY")
	projectID                = env.Must("GOOG_PROJECT_ID")
	queueID                  = env.Must("INPUT_QUEUE_ID")
	matchDownloadQueueID     = env.Must("MATCH_DOWNLOAD_QUEUE_ID")
	rateLimit                = api.CallRate{env.MustInt("REQ_PER_10SEC"), time.Second * 10}
	recentGamesStoreLocation = strings.TrimRight(env.Must("RECENT_GAMES_STORE_LOCATION"), "/")
)

func main() {

	creds := google.MustEnvCreds(projectID, pubsub.ScopePubSub)
	log.Debug("Creating pubsub client")
	sub, err := queue.InitClient(creds)
	exitIf(err)

	log.Debug("Creating gcs client")
	filesClient, err := files.InitClient(files.AutoRegisterGCS(projectID, storage.ScopeReadWrite))
	exitIf(err)

	crawl := SummonerCrawl{
		riotKey:                  apiKey,
		riotApis:                 api.NewAPIs(apiKey, rateLimit),
		recentGamesStoreLocation: recentGamesStoreLocation,
		filesClient:              filesClient,
		matchDownloadTopic:       sub.Topic(matchDownloadQueueID),
	}

	tr := queue.NewTaskRunner(sub, queueID, 1, time.Duration(2*time.Minute))

	log.Debug("Created taskRunner")
	tr.Start(context.Background(), crawl.Handle)
}

type SummonerCrawl struct {
	riotKey                  string
	riotApis                 api.APIs
	recentGamesStoreLocation string
	filesClient              *files.Client
	matchDownloadTopic       *pubsub.Topic
}

func (c *SummonerCrawl) Handle(ctx context.Context, m *pubsub.Message) error {
	// Deserialize the message
	msg := messages.LolSummonerCrawl{}
	log.Debug(string(m.Data))
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Error(err)
		return nil
	}
	if err := msg.Valid(); err != nil {
		return err
	}

	platform := riot.PlatformFromString(msg.PlatformId)
	region := riot.RegionFromPlatform(platform)

	api, err := c.riotApis.Platform(msg.PlatformId)
	if err != nil {
		return err
	}

	switch msg.HistoryType {
	case "ranked_history":
		since := time.Now().Add(time.Hour * -24)
		if msg.Since != "" {
			since, err = time.Parse(time.RFC3339, msg.Since)
			if err != nil {
				return err
			}
		}
		matchList, err := rankedHistory(api, since, msg.SummonerId)
		if err != nil {
			return err
		}
		err = c.processMatchList(matchList)
		if err != nil {
			return err
		}
	case "recent":
		recentGames, err := api.RecentGames(msg.SummonerId)
		if err != nil {
			// If the summonerId has no recent games, the api will return a 404.
			// If they have no games, there's nothing to do
			if err.Error() == "api error - code: 404" {
				return nil
			}
			return err
		}
		filterAndModify(&recentGames)
		err = c.saveRecentGames(&recentGames, region)
		if err != nil {
			return err
		}
		err = c.processRecentGames(&recentGames, msg.PlatformId)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Unrecognized history_type:" + msg.HistoryType)
	}
	return nil
}

func rankedHistory(a *api.Api, since time.Time, summonerId int64) (*api.MatchListItem, error) {
	opts := api.NewMatchListOptions()
	opts.RankedQueues(riot.TB_RANKED_SOLO_5x5).BeginTime(since).EndTime(time.Now())

	item, err := a.MatchList(summonerId, opts)
	if err != nil {
		return nil, err
	}

	// This manual filtering is necessary because of a bug in riot's
	// matchlist API, because occasionally the matchrefs that come back
	// have nothing to do with the filters we applied:
	// https://developer.riotgames.com/discussion/community-discussion/show/lEfrrBUT
	validRefs := []api.MatchReference{}
	for _, ref := range item.Matches {
		if opts.MatchesRef(ref) {
			validRefs = append(validRefs, ref)
		}
	}
	if len(validRefs) != len(item.Matches) {
		return nil, fmt.Errorf("riot returned %d refs for %s, but %d were valid.", len(item.Matches), summonerId, len(validRefs))
	}

	return &item, nil
}

// filterAndModify removes games that we aren't interested in. It also adds the target summoner
// information to the "FellowPlayers" array so it has 10 entries instead of 9
func filterAndModify(recentGames *api.RecentGames) {
	filteredGames := []api.Game{}
	for i := 0; i < len(recentGames.Games); i++ {
		game := recentGames.Games[i]
		// Filter out games based on type
		if game.MapId != 11 {
			continue
		} else if game.SubType != "NORMAL" &&
			game.SubType != string(riot.RANKED_SOLO_5x5) &&
			game.SubType != string(riot.RANKED_TEAM_5x5) &&
			game.SubType != string(riot.RANKED_FLEX_SR) &&
			game.SubType != string(riot.TB_RANKED_SOLO) &&
			game.SubType != "NONE" {
			continue
		}

		// In order to make things easier, add "this" player to the FellowPlayers array.
		game.FellowPlayers = append(game.FellowPlayers, api.GamePlayer{
			ChampionID: game.ChampionID,
			SummonerID: recentGames.SummonerID,
			TeamID:     game.TeamID,
		})
		if len(game.FellowPlayers) != 10 {
			log.Info(fmt.Sprintf("skipping game %d for having %d players", game.GameID, len(game.FellowPlayers)))
			continue
		}

		filteredGames = append(filteredGames, game)
	}
	recentGames.Games = filteredGames
}

func (c *SummonerCrawl) saveRecentGames(recentGames *api.RecentGames, region riot.Region) error {
	workDir, err := ioutil.TempDir("", "writeRecentGames")
	if err != nil {
		return err
	}
	defer os.RemoveAll(workDir)

	for _, game := range recentGames.Games {
		fileName := fmt.Sprintf("%d-%s.recent.json", game.GameID, region)

		data, err := vjson.Compress(game)
		if err != nil {
			return err
		}

		err = ioutil.WriteFile(fmt.Sprintf("%s/%s", workDir, fileName), data, 0666)
		if err != nil {
			return err
		}
		log.Debug(fmt.Sprintf("Upload %s/%s to %s/%s", workDir, fileName, c.recentGamesStoreLocation, fileName))

		err = c.filesClient.Copy(
			fmt.Sprintf("%s/%s", workDir, fileName),
			fmt.Sprintf("%s/%s", c.recentGamesStoreLocation, fileName),
			[]files.FileOption{
				files.ContentType("application/json"),
				files.ContentEncoding("gzip"),
			}...,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *SummonerCrawl) processMatchList(matchList *api.MatchListItem) error {
	for _, matchRef := range matchList.Matches {
		err := c.sendMatchDownloadRequest(&messages.LolMatchDownload{
			MatchId:    matchRef.MatchID,
			PlatformId: matchRef.PlatformID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *SummonerCrawl) processRecentGames(recentGames *api.RecentGames, platformID string) error {
	for _, game := range recentGames.Games {
		err := c.sendMatchDownloadRequest(&messages.LolMatchDownload{
			MatchId:    game.GameID,
			PlatformId: platformID,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *SummonerCrawl) sendMatchDownloadRequest(msg *messages.LolMatchDownload) error {
	matchDownloadBytes, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	log.Debug("Adding MatchDownload message: " + string(matchDownloadBytes))
	psMsg := &pubsub.Message{
		Data: matchDownloadBytes,
	}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*time.Duration(30))
	_, err = c.matchDownloadTopic.Publish(ctx, psMsg)
	return err
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
