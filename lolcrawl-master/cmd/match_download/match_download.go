package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
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
	inputSubID               = env.Must("INPUT_SUB_ID")
	rateLimiter              = api.CallRate{env.MustInt("REQ_PER_10SEC"), time.Second * 10}
	matchStoreLocation       = strings.TrimRight(env.Must("MATCH_STORE_LOCATION"), "/")
	observerStoreLocation    = strings.TrimRight(env.Must("OBSERVER_STORE_LOCATION"), "/")
	recentGamesStoreLocation = strings.TrimRight(env.Must("RECENT_GAMES_STORE_LOCATION"), "/")
	outputTopicID            = env.Must("OUTPUT_TOPIC_ID")
)

const Timeout404 = -8 * time.Hour

func main() {

	creds := google.MustEnvCreds(projectID, pubsub.ScopePubSub)
	sub, err := queue.InitClient(creds)
	exitIf(err)

	filesClient, err := files.InitClient(files.AutoRegisterGCS(projectID, storage.ScopeReadWrite))
	exitIf(err)

	downloader := MatchDownloader{
		riotApis:                 api.NewAPIs(apiKey, rateLimiter),
		matchStoreLocation:       matchStoreLocation,
		observerStoreLocation:    observerStoreLocation,
		recentGamesStoreLocation: recentGamesStoreLocation,
		filesClient:              filesClient,
		outputTopic:              sub.Topic(outputTopicID),
	}

	tr := queue.NewTaskRunner(sub, inputSubID, 1, time.Duration(2*time.Minute))

	log.Info("starting handler")
	tr.Start(context.Background(), downloader.Handle)
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

type MatchDownloader struct {
	riotApis                 api.APIs
	matchStoreLocation       string
	observerStoreLocation    string
	recentGamesStoreLocation string
	filesClient              *files.Client
	outputTopic              *pubsub.Topic
}

func (f *MatchDownloader) Handle(ctx context.Context, m *pubsub.Message) error {
	// Deserialize the message
	msg := messages.LolMatchDownload{}
	log.Debug(string(m.Data))
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Error(err)
		return err
	}
	if err := msg.Valid(); err != nil {
		log.Error("invalid message: " + err.Error())
		return err
	}

	platform := riot.PlatformFromString(msg.PlatformId)
	region := riot.RegionFromPlatform(platform)

	// Create one temp dir per task
	tempDir, err := ioutil.TempDir("", fmt.Sprintf("%d-%s", msg.MatchId, msg.PlatformId))
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Look to see if the match details exist already
	needToCache := false
	matchDetails, err := f.GetCachedMatch(region, msg.MatchId, tempDir)
	if err != nil {
		return err
	}

	if len(matchDetails.Timeline.Frames) == 0 {
		log.Info(fmt.Sprintf("not cached or timeline missing, fetching riot match %d-%s", msg.MatchId, region))
		needToCache = true
		api, err := f.riotApis.Platform(msg.PlatformId)
		if err != nil {
			return err
		}

		matchDetails, err = api.Match(strconv.FormatInt(msg.MatchId, 10), true)
		if err != nil {
			// Riot api will sometimes never give results for certain matches.
			// Manually time these out after a few hours, in order to keep the queue healthy
			if err.Error() == "api error - code: 404" {
				if m.PublishTime.Before(time.Now().Add(Timeout404)) {
					log.Warning(fmt.Sprintf("skipping match download for %d-%s due to timeout: msg was created at %v", msg.MatchId, region, m.PublishTime))
					return nil
				}
			}

			return err
		}
		if len(matchDetails.Timeline.Frames) == 0 {
			// Some matches never get timeline info
			if m.PublishTime.Before(time.Now().Add(Timeout404)) {
				log.Warning(fmt.Sprintf("skipping match download for %d-%s due to timeout: msg was created at %v", msg.MatchId, region, m.PublishTime))
				return nil
			}
			return fmt.Errorf("timeline missing from match details")
		}
	}

	// If we have the match data, but it's just missing the player info,
	// we don't need to fetch it again from riot
	if matchDetails.IsParticipantIdentitiesEmpty() {
		log.Debug("cached match is missing participant identities")
		needToCache = true
		// Non-ranked match details don't have participant identities.
		// First try to use current_game.json.
		// If that doesn't work, then try to use the recent_game.json
		remoteCurrentGameFile := fmt.Sprintf("%s/%d-%s/current_game.json", f.observerStoreLocation, matchDetails.MatchID, strings.ToLower(matchDetails.PlatformID))
		remoteGameFile := fmt.Sprintf("%s/%d-%s.recent.json", f.recentGamesStoreLocation, matchDetails.MatchID, region)
		exists, err := f.filesClient.Exists(remoteCurrentGameFile, remoteGameFile)
		if err != nil {
			return err
		}
		if exists[0] {
			err = f.FixParticipantIdentitiesWithCurrentGame(&matchDetails, platform, tempDir)
		} else if exists[1] {
			err = f.FixParticipantIdentitiesWithRecentGames(&matchDetails, region, tempDir)
		} else {
			return fmt.Errorf("no current_game or match details for match %d-%s. how did we get here!?!", msg.MatchId, msg.PlatformId)
		}
		if err != nil {
			return err
		}
	}

	if needToCache {
		// Save match
		err = f.CacheMatch(&matchDetails, tempDir)
		if err != nil {
			return fmt.Errorf("error saving match: %v", err)
		}
	}

	// Send the same message to Ingester
	tctx, _ := context.WithTimeout(context.Background(), time.Second*time.Duration(30))
	_, err = f.outputTopic.Publish(tctx, m)
	return err
}

// GetCachedMatch returns the copy of the match details that we have stored.
// An error returned from this function will result in failing the task.
func (f *MatchDownloader) GetCachedMatch(region riot.Region, matchID int64, tempDir string) (api.MatchDetail, error) {
	details := api.MatchDetail{}
	fileName := fmt.Sprintf("%d-%s.json", matchID, region)
	cachedMatchFile := filepath.Join(tempDir, fileName)

	exists, err := f.filesClient.Exists(fmt.Sprintf("%s/%s", f.matchStoreLocation, fileName))
	if err != nil {
		return details, err
	}
	if !exists[0] {
		return details, nil
	}

	err = f.filesClient.Copy(fmt.Sprintf("%s/%s", f.matchStoreLocation, fileName), cachedMatchFile)
	if err != nil {
		return details, err
	}
	log.Debug(fmt.Sprintf("found cached match %d-%s", matchID, region))

	err = vjson.DecodeFile(cachedMatchFile, &details)
	if err != nil {
		log.Warning(fmt.Sprintf("failed to decode json match details: %v", err))
		// don't return an error here so that the details are fetched from
		// riot again.
	}

	return details, nil
}

func (f *MatchDownloader) CacheMatch(match *api.MatchDetail, tempDir string) error {
	fileName := fmt.Sprintf("%d-%s.json", match.MatchID, strings.ToLower(match.Region))

	data, err := vjson.Compress(match)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(tempDir, fileName), data, 0666)
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("upload %s/%s to %s/%s", tempDir, fileName, f.matchStoreLocation, fileName))
	err = f.filesClient.Copy(
		fmt.Sprintf("%s/%s", tempDir, fileName),
		fmt.Sprintf("%s/%s", f.matchStoreLocation, fileName),
		files.ContentType("application/json"),
		files.ContentEncoding("gzip"),
	)
	return err
}

// FixParticipantIdentities fills in the participantIdentites in the matchDetail if they're missing by using the current Game info
func (f *MatchDownloader) FixParticipantIdentitiesWithCurrentGame(m *api.MatchDetail, platform riot.Platform, tempDir string) error {
	// Fetch current_game json from observer
	currentGame := api.CurrentGameInfo{}
	localCurrentGameFile := filepath.Join(tempDir, "current_game.json")
	platformStr := string(platform)
	remoteCurrentGameFile := fmt.Sprintf("%s/%d-%s/current_game.json", f.observerStoreLocation, m.MatchID, strings.ToLower(platformStr))
	err := f.filesClient.Copy(remoteCurrentGameFile, localCurrentGameFile)
	if err != nil {
		// If the source file doesn't exist, then error out
		log.Warning("unable to download file from gcs: " + remoteCurrentGameFile)
		return err
	}
	err = vjson.DecodeFile(localCurrentGameFile, &currentGame)
	if err != nil {
		return err
	}

	// Overwrite the ParticipantIdentities of the MatchDetails
	for i, participant := range currentGame.Participants {
		m.ParticipantIdentities[i] = api.ParticipantIdentity{
			ParticipantID: i + 1,
			Player: api.Player{
				SummonerID:   participant.SummonerID,
				SummonerName: participant.SummonerName,
			},
		}
	}
	return nil
}

func (f *MatchDownloader) FixParticipantIdentitiesWithRecentGames(m *api.MatchDetail, region riot.Region, tempDir string) error {
	// Fetch recent games json from gcs
	game := api.Game{}
	localGameFile := filepath.Join(tempDir, "recent_game.json")
	remoteGameFile := fmt.Sprintf("%s/%d-%s.recent.json", f.recentGamesStoreLocation, m.MatchID, region)
	err := f.filesClient.Copy(remoteGameFile, localGameFile)
	if err != nil {
		// If the source file doesn't exist, then error out
		log.Warning("unable to download file from gcs: " + remoteGameFile)
		return err
	}
	err = vjson.DecodeFile(localGameFile, &game)
	if err != nil {
		return err
	}

	for i, participant := range m.Participants {
		foundMatch := false
		for _, gamePlayer := range game.FellowPlayers {
			if gamePlayer.ChampionID == int64(participant.ChampionID) &&
				gamePlayer.TeamID == int64(participant.TeamID) {
				foundMatch = true
				m.ParticipantIdentities[i] = api.ParticipantIdentity{
					ParticipantID: i + 1,
					Player: api.Player{
						SummonerID: gamePlayer.SummonerID,
					},
				}
				break
			}
		}
		if !foundMatch {
			log.Warning("Unable to find summonerID for participant")
			return fmt.Errorf("unable to find summonerid for match %d, champid: %d, teamid: %d", m.MatchID, participant.ChampionID, participant.TeamID)
		}
	}
	return nil
}
