package main

import (
	"log"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	vshttp "github.com/VantageSports/common/http"

	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"google.golang.org/api/option"
)

func main() {

	var (
		gcsReplayPrefix = env.Must("GCS_REPLAY_PREFIX")
		googProjectId   = env.Must("GOOG_PROJECT_ID")
	)

	creds := google.MustEnvCreds(googProjectId, storage.ScopeFullControl)
	ctx := context.Background()
	gcsClient, err := storage.NewClient(ctx, option.WithTokenSource(creds.Conf.TokenSource(ctx)))
	exitIf(err)
	svc := NewFakeRiotService(gcsReplayPrefix, gcsClient)
	router := mux.NewRouter()

	// Register plain-old HTTP
	Attach(router, svc)
	vshttp.StartFromEnv(router)

}

func exitIf(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
