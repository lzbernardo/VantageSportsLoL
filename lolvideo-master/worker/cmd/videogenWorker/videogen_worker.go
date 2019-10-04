package main

import (
	"flag"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/logging"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"
	"google.golang.org/api/option"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/log"
	gLog "github.com/VantageSports/common/log/google"
	"github.com/VantageSports/common/queue"
	"github.com/VantageSports/lolvideo/worker"
)

var (
	queueID        = flag.String("queueID", "", "The task queue to read from")
	coordinatorURL = flag.String("coordinatorURL", "", "The url to send work requests to")
	clientVersion  = flag.String("clientVersion", "", "The version of the lol client that is on this machine")
	devMode        = flag.Bool("devMode", false, "Stub out instance/region urls. Use this if you're not running on a windows ec2 box")

	googProjectID   = env.Must("GOOG_PROJECT_ID")
	videoOutputPath = env.Must("VIDEO_OUTPUT_PATH")
)

func main() {
	flag.Parse()

	if *queueID == "" || *coordinatorURL == "" || *clientVersion == "" {
		flag.Usage()
		log.Fatal("missing required flag")
	}

	creds := google.MustEnvCreds(googProjectID, pubsub.ScopePubSub, datastore.ScopeDatastore, logging.WriteScope)
	sub, err := queue.InitClient(creds)
	exitIf(err)

	files, err := files.InitClient(files.AutoRegisterGCS(googProjectID, storage.ScopeReadWrite))
	exitIf(err)

	ctx := context.Background()
	datastoreClient, err := datastore.NewClient(ctx, creds.ProjectID, option.WithTokenSource(creds.Conf.TokenSource(ctx)))

	instanceID, region, err := worker.GetInstanceAndRegion(*devMode)
	exitIf(err)

	logLabels := map[string]string{"instance_id": instanceID, "region": region}
	gWriter, err := gLog.New(creds, "lol-videogen-worker-v1", logLabels, true)
	exitIf(err)

	log.WithWriter(gWriter)

	workerObj := worker.VideogenWorker{
		InstanceID:      instanceID,
		Region:          region,
		CoordinatorURL:  *coordinatorURL,
		ClientVersion:   *clientVersion,
		VideoOutputPath: videoOutputPath,
		Files:           files,
		DatastoreClient: datastoreClient,
		DevServerMode:   googProjectID == "vs-dev",
	}

	tr := queue.NewTaskRunner(sub, *queueID, 1, time.Duration(2*time.Hour))
	exitIf(err)
	tr.Start(ctx, workerObj.Handle)
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	flag.Usage()
	log.Fatal("missing required flag")
}
