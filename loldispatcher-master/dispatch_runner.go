package main

import (
	"os"
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
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue"
	"github.com/VantageSports/lolusers"
)

func init() {
	grpclog.SetLogger(log.NewGRPCAdapter(log.Quiet))
}

var (
	addrLOLUsers     = env.Must("ADDR_LOL_USERS")
	googProjectID    = env.Must("GOOG_PROJECT_ID")
	internalKey      = env.SmartString("SIGN_KEY_INTERNAL")
	matchDir         = strings.TrimSuffix(env.Must("MATCH_DATA_DIR"), "/")
	matchCostVP      = env.MustInt("MATCH_COST")
	subDispatch      = env.Must("SUB_LOL_MATCH_DISPATCH")
	topicElogen      = env.Must("TOPIC_LOL_ELOGEN")
	topicBasicIngest = env.Must("TOPIC_LOL_BASIC_STATS")
	tlsCert          = os.Getenv("TLS_CERT")
)

func main() {
	if (matchCostVP) < 0 {
		log.Fatal("match cost must be >= 0")
	}

	psClient := mustPubsubClient()
	dispatcher := &DispatchHandler{
		fClient:          mustFilesClient(),
		luClient:         mustLolUsers(),
		gcdClient:        mustDatastoreClient(),
		internalKey:      internalKey,
		matchDir:         matchDir,
		eloTopic:         mustPubTopic(psClient, topicElogen),
		basicIngestTopic: mustPubTopic(psClient, topicBasicIngest),
	}

	runner := queue.NewTaskRunner(psClient, subDispatch, 1, time.Minute*2)

	log.Notice("starting new task runner")
	runner.Start(context.Background(), dispatcher.Handle)
}

func mustDatastoreClient() *datastore.Client {
	ctx := context.Background()
	creds := google.MustEnvCreds(googProjectID, datastore.ScopeDatastore)

	client, err := datastore.NewClient(ctx, googProjectID, option.WithTokenSource(creds.TokenSource(ctx)))
	exitIf(err)
	return client
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

	if os.Getenv("PUBSUB_EMULATOR_HOST") != "" {
		queue.EnsureSubscription(client, subDispatch, subDispatch, time.Minute, nil)
		queue.EnsureTopic(client, topicElogen)
		queue.EnsureTopic(client, topicBasicIngest)
	}

	return client
}

func mustPubTopic(client *pubsub.Client, name string) *pubsub.Topic {
	topicHandle, err := queue.EnsureTopic(client, name)
	exitIf(err)
	return topicHandle
}

func mustLolUsers() lolusers.LolUsersClient {
	c, err := certs.ClientTLS(tlsCert)
	exitIf(err)

	conn, err := grpc.Dial(addrLOLUsers, grpc.WithTransportCredentials(credentials.NewTLS(c)))
	exitIf(err)

	return lolusers.NewLolUsersClient(conn)
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
