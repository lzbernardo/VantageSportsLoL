package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/VantageSports/common/certs"
	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/files"
	vjson "github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/lolstats/gcd"
	"github.com/VantageSports/lolstats/generate"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/api"
	"github.com/VantageSports/riot/service"
	"github.com/VantageSports/users/client"
)

var (
	projectID               = env.Must("GOOG_PROJECT_ID")
	queueID                 = env.Must("INPUT_QUEUE_ID")
	basicStatsStoreLocation = strings.TrimRight(env.Must("BASIC_STATS_STORE_LOCATION"), "/")
	bqImportLocation        = strings.TrimRight(env.Must("BQ_IMPORT_LOCATION"), "/")
	serverCertPath          = os.Getenv("RIOT_PROXY_CERT_PATH")
	serverAddr              = env.Must("RIOT_PROXY_SERVER_ADDR")
	signKeyInternal         = env.SmartString("SIGN_KEY_INTERNAL")
	insecureGRPC            = os.Getenv("INSECURE_GRPC") != ""
)

func init() {
	grpclog.SetLogger(log.NewGRPCAdapter(log.Quiet))
}

func main() {
	creds := google.MustEnvCreds(projectID, pubsub.ScopePubSub, datastore.ScopeDatastore)
	log.Debug("Creating pubsub client")
	sub, err := queue.InitClient(creds)
	exitIf(err)

	log.Debug("Creating gcs client")
	filesClient, err := files.InitClient(files.AutoRegisterGCS(projectID, storage.ScopeReadWrite))
	exitIf(err)

	log.Debug("Creating gcd client")
	ctx := context.Background()
	gcdClient, err := datastore.NewClient(ctx, creds.ProjectID, option.WithTokenSource(creds.Conf.TokenSource(ctx)))
	exitIf(err)

	log.Debug("Creating riot proxy client")
	riotProxy := mustRiotProxy(serverCertPath, serverAddr)

	basicStats := BasicStatsIngester{
		RiotProxy:               riotProxy,
		SignKeyInternal:         signKeyInternal,
		BasicStatsStoreLocation: basicStatsStoreLocation,
		BqImportLocation:        bqImportLocation,
		FilesClient:             filesClient,
		GcdClient:               gcdClient,
	}

	tr := queue.NewTaskRunner(sub, queueID, 1, time.Duration(2*time.Minute))

	log.Debug("Created taskRunner")
	tr.Start(context.Background(), basicStats.Handle)
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func mustRiotProxy(serverCertPath, serverAddr string) service.RiotClient {
	config, err := certs.ClientTLS(serverCertPath, certs.Insecure(insecureGRPC))
	exitIf(err)

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(config)))
	exitIf(err)

	return service.NewRiotClient(conn)
}

type BasicStatsIngester struct {
	RiotProxy               service.RiotClient
	SignKeyInternal         string
	BasicStatsStoreLocation string
	BqImportLocation        string
	FilesClient             *files.Client
	GcdClient               *datastore.Client
}

func (f *BasicStatsIngester) Handle(ctx context.Context, m *pubsub.Message) error {
	// Deserialize the message
	msg := messages.LolBasicStatsIngest{}
	log.Debug(string(m.Data))
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		return err
	}
	if err := msg.Valid(); err != nil {
		log.Error("Invalid message: " + err.Error())
		return err
	}

	// Create one temp dir per task
	tempDir, err := ioutil.TempDir("", fmt.Sprintf("%d-%s", msg.MatchId, msg.PlatformId))
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	// Download the match details
	matchDetails, err := f.FetchCachedMatch(msg.MatchDetailsPath, tempDir)
	if err != nil {
		return err
	}

	// Generate basic stats object.
	historyRow, err := generate.ComputeHistory(&matchDetails, msg.SummonerId)
	if err != nil {
		return err
	}

	platform := riot.PlatformFromString(msg.PlatformId)
	tier, division, err := f.FetchRank(ctx, msg.SummonerId, platform)
	if err != nil {
		return fmt.Errorf("couldn't get rank info for %d (%s): %v", msg.SummonerId, msg.PlatformId, err)
	}

	found, err := f.writeToGCD(ctx, &matchDetails, historyRow, msg.SummonerId)
	if err != nil {
		return err
	}

	if found {
		log.Info(fmt.Sprintf("found existing match history for %d-%s (summoner: %s), override=%t", msg.MatchId, msg.PlatformId, msg.SummonerId, msg.Override))
		if !msg.Override {
			return ctx.Err()
		}
	}

	stats, err := generate.ComputeBasic(&matchDetails, msg.SummonerId)
	if err != nil {
		return fmt.Errorf("cant generate basic stats in %d for %d (%s): %v", msg.MatchId, msg.SummonerId, msg.PlatformId, err)
	}
	stats.Division, stats.Tier = division, tier

	// Upload basic stats to gcs
	err = f.UploadBasicStats(stats, f.BasicStatsStoreLocation, tempDir)
	if err != nil {
		return err
	}

	// Trim off non-stat data
	stats.TrimNonStats()

	// Upload basic stats to baseview directory
	err = f.UploadBasicStats(stats, f.BqImportLocation, tempDir)
	if err != nil {
		return err
	}

	return nil
}

func (f *BasicStatsIngester) FetchCachedMatch(matchStoreLocation string, tempDir string) (api.MatchDetail, error) {
	details := api.MatchDetail{}
	cachedMatchFile := filepath.Join(tempDir, "match.json")

	err := f.FilesClient.Copy(matchStoreLocation, cachedMatchFile)
	if err != nil {
		return details, err
	}
	err = vjson.DecodeFile(cachedMatchFile, &details)
	return details, err
}

func (f *BasicStatsIngester) UploadBasicStats(stats *generate.BasicStats, gcdDestination string, tempDir string) error {
	fileName := fmt.Sprintf("%d-%s-%d.basic.json", stats.MatchID, stats.PlatformID, stats.SummonerID)

	data, err := vjson.Compress(stats)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(tempDir, fileName), data, 0666)
	if err != nil {
		return err
	}
	log.Debug(fmt.Sprintf("Upload %s/%s to %s/%s", tempDir, fileName, gcdDestination, fileName))
	err = f.FilesClient.Copy(
		fmt.Sprintf("%s/%s", tempDir, fileName),
		fmt.Sprintf("%s/%s", gcdDestination, fileName),
		files.ContentType("application/json"),
		files.ContentEncoding("gzip"),
	)
	if err != nil {
		return err
	}
	return nil
}

func (f *BasicStatsIngester) FetchRank(ctx context.Context, summonerID int64, platform riot.Platform) (riot.Tier, riot.Division, error) {
	region := riot.RegionFromPlatform(platform)

	authCtx := client.SetCtxToken(ctx, f.SignKeyInternal)
	leagues, err := f.RiotProxy.LeagueEntryBySummoner(authCtx,
		&service.SummonerIDRequest{
			Region: region.String(),
			Ids:    []int64{summonerID},
		})

	if err != nil {
		if strings.Contains(err.Error(), "code: 404") {
			// this user isn't ranked, so treat them as the lowest possible
			// ranked player, bronze 5.
			return riot.BRONZE, riot.DIV_V, nil
		}
		return "", "", err
	}

	var tier riot.Tier
	var division riot.Division
	// There should only be one key in the leagues map because we only pass in one summonerID
	for _, v := range leagues.NamedLeagues {
		// Loop through all their leagues. Currently there's only one, but there might be more in the future
		for _, league := range v.Leagues {
			// Only count solo queue and flex queue rank
			if league.Queue != string(riot.RANKED_SOLO_5x5) && league.Queue != string(riot.RANKED_FLEX_SR) {
				continue
			}
			curTier := riot.Tier(league.Tier)
			var curDivision riot.Division

			// There should only be one entry, which is that of the target summonerID, since we called the "entry" endpoint
			for _, ent := range league.Entries {
				curDivision = riot.Division(ent.Division)
			}

			if (tier == "" && division == "") || isHigherRank(curTier, curDivision, tier, division) {
				tier = curTier
				division = curDivision
			}
		}
	}
	return tier, division, nil
}

func (f *BasicStatsIngester) writeToGCD(ctx context.Context, m *api.MatchDetail, row *gcd.HistoryRow, summonerID int64) (bool, error) {
	// See if an entry exists already.
	keyStr := fmt.Sprintf("%d-%v-%d", m.MatchID, strings.ToLower(m.Region), summonerID)
	key := datastore.NameKey(gcd.KindLolHistoryRow, keyStr, nil)

	// Issue Get to see if entry exists.
	err := f.GcdClient.Get(ctx, key, &gcd.HistoryRow{})

	if err == nil {
		return true, nil
	}
	if err != datastore.ErrNoSuchEntity {
		return false, err
	}

	log.Info("Calling put in GCD for: " + keyStr)
	_, err = f.GcdClient.Put(ctx, key, row)
	return false, err
}

func isHigherRank(t1 riot.Tier, d1 riot.Division, t2 riot.Tier, d2 riot.Division) bool {
	if t1.Ord() > t2.Ord() {
		return true
	} else if t1.Ord() == t2.Ord() && d1.Ord() > d2.Ord() {
		return true
	} else {
		return false
	}
}
