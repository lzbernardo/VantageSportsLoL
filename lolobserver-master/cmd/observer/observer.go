package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/grpclog"

	"github.com/VantageSports/common/certs"
	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue"
	"github.com/VantageSports/common/queue/messages"
	obs "github.com/VantageSports/lolobserver"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot/api"
)

var (
	addrLolUsers         = env.Must("ADDR_LOL_USERS")
	googProjectId        = env.Must("GOOG_PROJECT_ID")
	internalKey          = env.SmartString("SIGN_KEY_INTERNAL")
	outputPath           = env.Must("OUTPUT_MATCHES_PATH")
	minimumVantagePoints = env.MustInt("MINIMUM_VANTAGE_POINTS")
	requestsPer10Sec     = env.MustInt("RIOT_REQ_PER_10SEC")
	replayServer         = env.Must("REPLAY_SERVER")
	riotKey              = env.SmartString("API_KEY")
	tLolMatchDownload    = env.Must("TOPIC_LOL_MATCH_DOWNLOAD")
	tlsCertPath          = os.Getenv("TLS_CERT")
)

func init() {
	api.Verbose = false
	grpclog.SetLogger(log.NewGRPCAdapter(log.Quiet))
}

func main() {
	pubClient := mustPubClient()
	toDownload := mustUserWatcher().Start()
	toIngest := mustStartDownloader(mustFilesClient(), toDownload)
	for ms := range toIngest {
		isOk := false
		for !isOk {
			err := emitIngestionTask(pubClient, ms)
			if err == nil {
				isOk = true
			} else {
				log.Error(err)
			}
		}
	}
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

//
// User Watching
//

func mustUserWatcher() *obs.UserWatcher {
	rate := api.CallRate{CallsPer: requestsPer10Sec, Dur: time.Second * 10}
	api := api.NewAPIs(riotKey, rate)
	return &obs.UserWatcher{
		LolUsers: mustUserLister(),
		Api:      api,
	}
}

func mustUserLister() *obs.LolUsersBySummoners {
	log.Notice("connecting to lolusers at " + addrLolUsers)

	tlsConfig, err := certs.ClientTLS(tlsCertPath)
	exitIf(err)
	creds := credentials.NewTLS(tlsConfig)

	conn, err := grpc.Dial(addrLolUsers, grpc.WithTransportCredentials(creds))
	exitIf(err)

	lister := &obs.LolUsersVPLister{
		Client:           lolusers.NewLolUsersClient(conn),
		InternalKey:      internalKey,
		MinVantagePoints: int64(minimumVantagePoints),
	}
	return &obs.LolUsersBySummoners{
		Lister:   lister,
		CacheDur: time.Minute * 30,
	}
}

//
// Match Downloading
//

// mustStartDownloader starts the downloader loop in its own goroutine,
// returning a channel of matches that have been successfully downloaded.
func mustStartDownloader(fc *files.Client, in <-chan *obs.MatchUsers) <-chan *obs.MatchUsers {
	out := make(chan *obs.MatchUsers, 20)
	go downloadLoop(fc, in, out)
	return out
}

func mustFilesClient() *files.Client {
	creds := google.MustEnvCreds(googProjectId, storage.ScopeReadWrite)
	filesClient, err := files.InitClient(files.RegisterGCS("gs://", creds))
	exitIf(err)
	return filesClient
}

// downloadLoop listens on the in channel, and attempts to download each match
// that passes through iff ShouldDownload() returns true. Otherwise the match
// is discarded. Match downloading happens in its own goroutine, and if no
// download errors are encountered, the match is pushed out through the out
// channel.
// NOTE: this function assumes control of the out channel, closing it after the
// in channel is closed.
func downloadLoop(fc *files.Client, in <-chan *obs.MatchUsers, out chan<- *obs.MatchUsers) {
	for m := range in {
		usersSummary := obs.Summary(m.LolUsers...)
		if !obs.ShouldDownload(m.CurrentGame) {
			log.Info(fmt.Sprintf("skipping match %d on behalf of %s", m.CurrentGame.GameID, usersSummary))
			continue
		}

		log.Info(fmt.Sprintf("downloading match %d (%s) on behalf of %s", m.CurrentGame.GameID, m.CurrentGame.PlatformID, usersSummary))
		go downloadMatch(fc, *m, out)
	}
	close(out)
}

// downloadMatch initializes a new file saver, constructs a new replay save
// state (attempting to load the partially download match), and begins (or
// resumes) the download from riot's spectator server (the empty string default,
// below).
func downloadMatch(fc *files.Client, m obs.MatchUsers, out chan<- *obs.MatchUsers) {
	matchDir := fmt.Sprintf("%s/%d-%s", strings.TrimSuffix(outputPath, "/"), m.CurrentGame.GameID, strings.ToLower(m.CurrentGame.PlatformID))

	saver, err := obs.NewFileSaver(fc, matchDir)
	if err != nil {
		log.Error(err)
		return
	}

	existing, err := fc.List(matchDir)
	if err != nil {
		log.Warning(fmt.Sprintf("unable to list contents of %s. starting from scratch", matchDir))
	}

	saveState, err := obs.NewReplaySaveState(&m, existing)
	if err == nil {
		err = saveState.Save("", saver)
	}
	m.Observed = err == nil
	out <- &m
	log.Info(fmt.Sprintf("match %d finished. err: %v", m.CurrentGame.GameID, err))
}

//
// Match Ingest functionality
//

// mustPubClient initializes a new google pubsub client and verifies that the
// configured topic exists.
func mustPubClient() *pubsub.Client {
	creds := google.MustEnvCreds(googProjectId, pubsub.ScopePubSub)
	pub, err := queue.InitClient(creds)
	exitIf(err)

	_, err = queue.EnsureTopic(pub, tLolMatchDownload)
	exitIf(err)

	return pub
}

// emitIngestionTask constructs and publishes a matchIngestTopic for a given
// match, referencing all of the registered lolusers that we downloaded this
// match on behalf of.
func emitIngestionTask(pub *pubsub.Client, ms *obs.MatchUsers) error {
	log.Info(fmt.Sprintf("emitting match %d (observed:%v)", ms.CurrentGame.GameID, ms.Observed))

	ingest := messages.LolMatchDownload{
		MatchId:    ms.CurrentGame.GameID,
		PlatformId: ms.CurrentGame.PlatformID,
	}
	if ms.Observed {
		ingest.Key = ms.CurrentGame.Observers.EncryptionKey
		ingest.ReplayServer = replayServer
		ingest.ObservedSummonerIds = summoners(ms.LolUsers)
	}

	data, err := json.Marshal(ingest)
	if err != nil {
		return err
	}
	msg := &pubsub.Message{Data: data}

	ctx, _ := context.WithTimeout(context.Background(), time.Second*time.Duration(30))
	_, err = pub.Topic(tLolMatchDownload).Publish(ctx, msg)
	return err
}

// taskLolUsers converts the list of lolobserver.LolUser to a list of
// summoner ids.
func summoners(users []lolusers.LolUser) []int64 {
	res := []int64{}
	seen := map[int64]bool{}

	for _, s := range users {
		sID, err := strconv.ParseInt(s.SummonerId, 10, 64)
		if err != nil {
			continue
		}
		if !seen[sID] {
			seen[sID] = true
			res = append(res, sID)
		}
	}

	return res
}
