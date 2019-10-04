package main

import (
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"

	vsbigquery "github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats/bq"
)

func main() {
	var (
		projectID                  = env.Must("GOOG_PROJECT_ID")
		bigQueryDataset            = env.Must("BIG_QUERY_DATASET")
		bigQueryTableBasic         = env.Must("BIG_QUERY_TABLE_BASIC")
		bigQueryTableAdvanced      = env.Must("BIG_QUERY_TABLE_ADVANCED")
		bigQuerySchemaFileBasic    = env.Must("BIG_QUERY_SCHEMA_FILE_BASIC")
		bigQuerySchemaFileAdvanced = env.Must("BIG_QUERY_SCHEMA_FILE_ADVANCED")
		daysToDedupe               = env.MustInt("DAYS_TO_DEDUPE")
		waitTimeMinutes            = env.MustInt("WAIT_TIME_MINUTES")
	)

	creds := google.MustEnvCreds(projectID, bigquery.Scope)

	log.Notice("Creating BigQuery service")
	bqClient, err := vsbigquery.NewClient(creds)
	exitIf(err)

	deduper := &bq.Deduper{projectID, bqClient, daysToDedupe}
	basicTable := &bq.BqDefinition{bigQueryDataset, bigQueryTableBasic, bigQuerySchemaFileBasic}
	advancedTable := &bq.BqDefinition{bigQueryDataset, bigQueryTableAdvanced, bigQuerySchemaFileAdvanced}

	for {
		// Start a day ago to prevent deduping a table that's currently updating
		t := time.Now().Add(-24 * time.Hour)
		err := deduper.Dedupe(basicTable, t)
		if err == nil {
			err = deduper.Dedupe(advancedTable, t)
		}

		if err != nil {
			log.Error(err.Error())
		}
		log.Info(fmt.Sprintf("Done. Sleeping for %v minutes", waitTimeMinutes))
		time.Sleep(time.Minute * time.Duration(waitTimeMinutes))
	}
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}
