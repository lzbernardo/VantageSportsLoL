package main

import (
	"flag"
	"os"
	"path/filepath"
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
	outQueueID     = flag.String("outQueueID", "", "The queue to write to")
	coordinatorURL = flag.String("coordinatorURL", "", "The url to send work requests to")
	clientVersion  = flag.String("clientVersion", "", "The version of the lol client that is on this machine")
	devMode        = flag.Bool("devMode", false, "Stub out instance/region urls. Use this if you're not running on a windows ec2 box")

	googProjectID      = env.Must("GOOG_PROJECT_ID")
	dataOutputPath     = env.Must("DATA_OUTPUT_PATH")
	pluginDownloadPath = env.Must("PLUGIN_DOWNLOAD_PATH")
)

func main() {
	flag.Parse()

	if *queueID == "" || *coordinatorURL == "" || *clientVersion == "" || *outQueueID == "" {
		usage()
	}

	creds := google.MustEnvCreds(googProjectID, pubsub.ScopePubSub, datastore.ScopeDatastore, logging.WriteScope)
	sub, err := queue.InitClient(creds)
	exitIf(err)

	filesClient, err := files.InitClient(files.AutoRegisterGCS(googProjectID, storage.ScopeReadWrite))
	exitIf(err)

	// The files client doesn't support windows paths, so we have to manually add a handler for it
	localProvider, err := files.NewLocalProvider(os.TempDir())
	exitIf(err)
	filesClient.Register(`C:\`, localProvider, nil)

	ctx := context.Background()
	datastoreClient, err := datastore.NewClient(ctx, creds.ProjectID, option.WithTokenSource(creds.Conf.TokenSource(ctx)))

	instanceID, region, err := worker.GetInstanceAndRegion(*devMode)
	exitIf(err)

	logLabels := map[string]string{"instance_id": instanceID, "region": region, "process": "datagen"}
	gWriter, err := gLog.New(creds, "lol-datagen-worker-v1", logLabels, true)
	exitIf(err)

	log.WithWriter(gWriter)

	gameWait := worker.LolDatagenWorkerNumbers{}
	gwKey := datastore.NameKey("LolDatagenWorkerNumbers", "constants", nil)
	gwErr := datastoreClient.Get(context.Background(), gwKey, &gameWait)
	exitIf(gwErr)

	workerObj := worker.DatagenWorker{
		InstanceID:      instanceID,
		Region:          region,
		CoordinatorURL:  *coordinatorURL,
		ClientVersion:   *clientVersion,
		DataOutputPath:  dataOutputPath,
		Files:           filesClient,
		DatastoreClient: datastoreClient,
		OutTopic:        sub.Topic(*outQueueID),
		RetryTopic:      sub.Topic(*queueID),
		DevServerMode:   googProjectID == "vs-dev",
		MaxRetries:      2,
		Constants:       gameWait,
	}

	if googProjectID != "vs-dev" {
		remotePlugin := pluginDownloadPath + "DataSpectator.zip"
		localPlugin := filepath.Join(`C:\Users\Administrator\Downloads`, "DataSpectator.zip")
		err = filesClient.Copy(remotePlugin, localPlugin)
		exitIf(err)

		err = unzip(localPlugin, `C:\Users\Administrator\Downloads`)
		exitIf(err)
	}

	err = workerObj.Bootstrap()
	exitIf(err)

	tr := queue.NewTaskRunner(sub, *queueID, 1, time.Duration(time.Minute*30))
	tr.Start(ctx, workerObj.Handle)
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func usage() {
	log.Fatal("Missing -queueID, -coordinatorURL, -clientVersion, -outQueueID parameters")
}
