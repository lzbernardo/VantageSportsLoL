package main

import (
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"cloud.google.com/go/storage"
	"golang.org/x/net/context"

	vsbigquery "github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/env"
	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/log"
)

// Loader encapsulates the configuration required for a bigquery data transfer
type Loader struct {
	BqClient    *vsbigquery.Client
	BqDataset   string
	BqTable     string
	FilesClient *files.Client
	JsonDir     string
	ProjectID   string
	Suffix      string
}

// BigQuery quota limit
const MaxFilesInLoad = 10000

func main() {
	var (
		projectID             = env.Must("GOOG_PROJECT_ID")
		loadIntervalMinutes   = env.MustInt("LOAD_INTERVAL_MINUTES")
		bigQueryDataset       = env.Must("BIG_QUERY_DATASET")
		bigQueryTableBasic    = env.Must("BIG_QUERY_TABLE_BASIC")
		bigQueryTableAdvanced = env.Must("BIG_QUERY_TABLE_ADVANCED")
		JsonDirBasic          = env.Must("JSON_DIR_BASIC")
		JsonDirAdvanced       = env.Must("JSON_DIR_ADVANCED")
	)

	creds := google.MustEnvCreds(projectID, bigquery.Scope, storage.ScopeFullControl)

	log.Notice("Creating BigQuery service")
	bqClient, err := vsbigquery.NewClient(creds)
	exitIf(err)

	log.Debug("Creating gcs client")
	filesClient, err := files.InitClient(files.AutoRegisterGCS(projectID, storage.ScopeFullControl))
	exitIf(err)

	loaderBasic := &Loader{bqClient, bigQueryDataset, bigQueryTableBasic, filesClient, JsonDirBasic, projectID, ".basic.json"}
	loaderAdvanced := &Loader{bqClient, bigQueryDataset, bigQueryTableAdvanced, filesClient, JsonDirAdvanced, projectID, ".advanced.json"}

	for {
		err := loaderBasic.load()
		if err == nil {
			err = loaderAdvanced.load()
		}
		if err != nil {
			log.Error(err.Error())
		}
		time.Sleep(time.Minute * time.Duration(loadIntervalMinutes))
	}
}

func (ld *Loader) load() error {
	// Look for files in bq folder
	files.MaxListResults = 10000
	files, err := ld.FilesClient.List(ld.JsonDir)

	matchingFiles := []string{}
	for _, f := range files {
		if strings.HasSuffix(f, ld.Suffix) {
			matchingFiles = append(matchingFiles, f)
		}
		if len(matchingFiles) >= MaxFilesInLoad {
			break
		}
	}

	log.Debug(fmt.Sprintf("found %d rows to insert into %s", len(matchingFiles), ld.BqTable))
	if len(matchingFiles) == 0 {
		return nil
	}

	err = ld.appendToTable(ld.BqTable, matchingFiles)
	if err != nil {
		return err
	}
	log.Debug("successfully appended to " + ld.BqTable)

	// Delete the matchingFiles from gcs
	for _, file := range matchingFiles {
		for i := 0; i < 3; i++ {
			if err = ld.FilesClient.ManagerFor(file).Remove(file); err != nil {
				continue
			}
			if err == nil {
				break
			}
		}
		if err != nil {
			// If the file we're trying to delete doesn't exist, then skip it
			if err.Error() == "storage: object doesn't exist" {
				log.Warning(fmt.Sprintf("skipping file %v - not found", file))
			} else {
				return fmt.Errorf("failed to delete file: %v - %v", file, err)
			}
		}
	}
	log.Debug("successfully deleted gcs files for suffix " + ld.Suffix)
	return nil
}

func (ld *Loader) appendToTable(tableName string, jsonUris []string) error {
	// Build a bigquery load request
	// From https://cloud.google.com/bigquery/quota-policy:
	// Maximum size per load job: 12 TB across all input files for CSV and JSON
	// Maximum number of files per load job: 10,000
	ctx := context.Background()
	job, err := ld.BqClient.Load(ctx, ld.BqDataset, tableName, "", bigquery.JSON, false, jsonUris...)
	if err != nil {
		return err
	}

	if err = ld.BqClient.WaitForJob(ctx, job); err != nil {
		return err
	}

	return nil
}

func exitIf(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}
