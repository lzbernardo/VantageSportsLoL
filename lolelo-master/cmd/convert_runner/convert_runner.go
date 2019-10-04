// The convert_runner handles messages that deal with the lol ocr processing.

package main

import (
	"strings"
	"time"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
)

var (
	outputDir     = env.Must("OUTPUT_DIR")
	googProjectID = env.Must("GOOG_PROJECT_ID")
	replayDataDir = strings.TrimSuffix(env.Must("REPLAY_DATA_DIR"), "/")
	pubsubInput   = env.Must("PUBSUB_INPUT_SUB")
	pubsubOutput  = env.Must("PUBSUB_OUTPUT_TOPIC")
)

func main() {
	pubsubClient := mustPubsubClient()

	lolOCRExtractor := &LolOCRExtractor{
		OutputDir:     outputDir,
		Files:         mustFilesClient(),
		PubTopic:      mustPubTopic(pubsubClient),
		ReplayDataDir: replayDataDir,
	}

	runner := queue.NewTaskRunner(pubsubClient, pubsubInput, 1, time.Minute*2)

	log.Notice("starting new task runner")
	runner.Start(context.Background(), lolOCRExtractor.Handle)
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

func mustPubTopic(client *pubsub.Client) *pubsub.Topic {
	topicHandle, err := queue.EnsureTopic(client, pubsubOutput)
	exitIf(err)
	return topicHandle
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
