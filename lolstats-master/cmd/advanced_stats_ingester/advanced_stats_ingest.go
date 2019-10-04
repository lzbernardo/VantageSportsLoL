package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/files/util"
	vsjson "github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/lolstats/ingest"
	"github.com/VantageSports/riot"
)

var (
	outputDir     = env.Must("OUTPUT_DIR")
	bqOutputDir   = env.Must("BQ_OUTPUT_DIR")
	googProjectID = env.Must("GOOG_PROJECT_ID")
	pubsubInput   = env.Must("PUBSUB_INPUT_SUB")
	pubsubOutput  = env.Must("PUBSUB_OUTPUT_SUB")
)

func main() {
	pubsubClient := mustPubsubClient()

	ingester := &Ingester{
		outputDir:   outputDir,
		bqOutputDir: bqOutputDir,
		fc:          mustFilesClient(),
		outputTopic: pubsubClient.Topic(pubsubOutput),
	}

	runner := queue.NewTaskRunner(pubsubClient, pubsubInput, 1, time.Minute*2)

	log.Notice("starting new task runner")
	runner.Start(context.Background(), ingester.Handle)
}

type Ingester struct {
	outputDir   string
	bqOutputDir string
	fc          *files.Client
	outputTopic *pubsub.Topic
}

func (i *Ingester) Handle(ctx context.Context, m *pubsub.Message) error {
	msg := messages.LolAdvancedStatsIngest{}
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Error(err)
		return err
	}
	log.Debug(fmt.Sprintf("message: %s", m.Data))
	if err := msg.Valid(); err != nil {
		return err
	}

	// Sometimes when eloprocess fails (because the game was truncated or messed
	// up somehow), we want to regenerate the game data, so we send a new
	// message to elogen. Elogen doesnt need to know which summoner the game is
	// being processed for, so our script is lazy and just uses an obvious dummy
	// summoner id (-32). However, since the original message (with the real
	// summoner id) is still sitting in the eloprocess queue (waiting for valid
	// elo data), eventually BOTH messages will make it here to the advanced
	// stats ingester. All this is to say, if we ever see a dummy summoner id,
	// we can silently ack it since we know that we generated it knowing that it
	// is superfluous.
	if msg.SummonerId == -32 {
		log.Info("debug summoner id found. skipping")
		return nil
	}

	platform := riot.PlatformFromString(msg.PlatformId)

	statsFilename := fmt.Sprintf("%d-%v-%d.advanced.json", msg.MatchId, platform, msg.SummonerId)
	remotePath := fmt.Sprintf("%s/%s", i.outputDir, statsFilename)

	exists, err := i.fc.Exists(remotePath)
	if err != nil {
		return err
	}
	if exists[0] {
		log.Info(fmt.Sprintf("advanced stats file (%s) already exists, override=%t", remotePath, msg.Override))
		if !msg.Override {
			return ctx.Err()
		}
	}

	workDir, err := ioutil.TempDir("", "lol_advanced_stats")
	if err != nil {
		log.Error(err)
		return err
	}
	defer os.RemoveAll(workDir)

	_, filename := filepath.Split(msg.BaseviewPath)
	baseviewLocal := filepath.Join(workDir, filename)

	if err := i.fc.Copy(msg.BaseviewPath, baseviewLocal); err != nil {
		return err
	}
	bv := baseview.Baseview{}
	if err := vsjson.DecodeFile(baseviewLocal, &bv); err != nil {
		return err
	}

	stats, err := ingest.ComputeAdvanced(bv, msg.SummonerId, msg.MatchId, platform)
	if err != nil {
		// Errors in role position calculation represent outliers where players don't stick to the meta.
		// Since we want to have stats to help people play in the meta, these games don't make sense to include
		if strings.HasPrefix(err.Error(), "error in role position calculation") {
			log.Error(err.Error())
			return nil
		} else {
			return err
		}
	}

	if err := i.zipAndUpload(stats, remotePath, workDir, statsFilename); err != nil {
		return err
	}

	bqRemotePath := fmt.Sprintf("%s/%s", i.bqOutputDir, statsFilename)

	stats.TrimNonStats()
	if err := i.zipAndUpload(stats, bqRemotePath, workDir, statsFilename); err != nil {
		return err
	}

	// After each advanced stat task, update the summoner's goals
	if err := i.sendGoalUpdateMessage(ctx, stats); err != nil {
		return err
	}

	return ctx.Err()
}

func (i *Ingester) zipAndUpload(stats *ingest.AdvancedStats, remotePath string, workDir string, statsFilename string) error {

	localPath := filepath.Join(workDir, statsFilename)
	if err := vsjson.Write(localPath, stats, 0664); err != nil {
		return err
	}
	if err := util.GzipFile(localPath, localPath+".zip", 0664); err != nil {
		return err
	}
	if err := i.fc.Copy(localPath+".zip", remotePath,
		files.ContentType("application/json"),
		files.ContentEncoding("gzip")); err != nil {
		return err
	}
	return nil
}

func (i *Ingester) sendGoalUpdateMessage(ctx context.Context, stats *ingest.AdvancedStats) error {
	outMsg := messages.LolGoalsUpdate{
		MatchId:      stats.MatchID,
		PlatformId:   string(stats.PlatformID),
		SummonerId:   stats.SummonerID,
		RolePosition: string(stats.RolePosition),
	}

	log.Notice(fmt.Sprintf("Adding goals update message: %v", outMsg))
	b, err := json.Marshal(outMsg)
	if err != nil {
		log.Error(fmt.Sprintf("Error marshalling msg: %v", err))
		return err
	}

	pubsubMsg := &pubsub.Message{Data: b}
	_, err = i.outputTopic.Publish(ctx, pubsubMsg)
	return err
}

func mustFilesClient() *files.Client {
	client, err := files.InitClient(
		files.AutoRegisterGCS(googProjectID, storage.ScopeReadWrite))
	exitIf(err)
	return client
}

func mustPubsubClient() *pubsub.Client {
	creds := google.MustEnvCreds(googProjectID, pubsub.ScopePubSub)
	client, err := queue.InitClient(creds)
	exitIf(err)
	return client
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
