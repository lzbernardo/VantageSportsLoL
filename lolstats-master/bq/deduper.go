package bq

import (
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"

	vsbigquery "github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
)

type BqDefinition struct {
	BqDataset    string
	BqTable      string
	BqSchemaFile string
}

func (def *BqDefinition) Columns() []string {
	var schema interface{}
	err := json.DecodeFile(def.BqSchemaFile, &schema)
	if err != nil {
		log.Error(err)
	}

	names := []string{}
	// The json file is a list of maps
	a := schema.([]interface{})
	for _, v := range a {
		m := v.(map[string]interface{})
		// We only care about the top level name, because we use standardSQL
		n := m["name"].(string)
		names = append(names, n)
	}
	return names
}

type Deduper struct {
	ProjectID    string
	BqClient     *vsbigquery.Client
	DaysToDedupe int
}

func (d *Deduper) Dedupe(def *BqDefinition, t time.Time) error {
	for i := 0; i < d.DaysToDedupe; i++ {
		// Query over target partition and the previous one to account for duplicates that span partitions
		startTime := t.Add(-24 * time.Hour)

		// Run the dedupe query, and save the results in a temp table
		table, err := d.dedupeToTempTable(def, startTime, t)
		if err != nil {
			return err
		}

		// Copy the temp table back to the original tables.
		splitTime := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
		err = d.copyTempTableBackToOriginal(table, def, splitTime)
		if err != nil {
			return err
		}

		// Delete temp table
		log.Info(fmt.Sprintf("Deleting table: %v", table))
		ctx := context.Background()
		err = d.BqClient.Client.Dataset(def.BqDataset).Table(table).Delete(ctx)
		if err != nil {
			return err
		}

		t = t.Add(-24 * time.Hour)
	}

	return nil
}

// dedupeToTempTable queries the table in def, spanning partitions from startTime to endTime,
// and saves the deduped results into a temp table.
// Returns the temp table name
func (d *Deduper) dedupeToTempTable(def *BqDefinition, startTime, endTime time.Time) (string, error) {
	// Get a list of all the columns in the table
	columns := strings.Join(def.Columns(), ",")
	startPartition := fmt.Sprintf("%v-%02d-%02d", startTime.Year(), int(startTime.Month()), startTime.Day())
	endPartition := fmt.Sprintf("%v-%02d-%02d", endTime.Year(), int(endTime.Month()), endTime.Day())
	tempTableName := fmt.Sprintf("%v_%v%02d%02d_%v%02d%02d_dedupe", def.BqTable, startTime.Year(), int(startTime.Month()), startTime.Day(), endTime.Year(), int(endTime.Month()), endTime.Day())

	log.Info(fmt.Sprintf("Deduping %v for %v-%v", def.BqTable, startPartition, endPartition))

	// Select everything from the partition range, including a row number
	// grouped by summoner_id, match_id, and platform_id (our dedupe condition).
	// Then, select everything from that (minus the row number) where the row_number is 1
	queryStr := fmt.Sprintf("SELECT %s FROM ( SELECT *, ROW_NUMBER() OVER (PARTITION BY summoner_id, match_id, platform_id order by last_updated desc) row_number FROM %s.%s WHERE _PARTITIONTIME BETWEEN TIMESTAMP('%s') AND TIMESTAMP('%s')) WHERE row_number = 1", columns, def.BqDataset, def.BqTable, startPartition, endPartition)
	log.Info(queryStr)

	query := d.BqClient.Client.Query(queryStr)
	query.UseStandardSQL = true
	query.Dst = &bigquery.Table{
		ProjectID: d.ProjectID,
		DatasetID: def.BqDataset,
		TableID:   tempTableName,
	}
	query.AllowLargeResults = true
	query.DisableFlattenedResults = true

	ctx := context.Background()
	job, err := query.Run(ctx)
	if err != nil {
		return "", err
	}

	return tempTableName, d.BqClient.WaitForJob(ctx, job)
}

// copyTempTableBackToOriginal takes the temp table of deduped results, and copys them back into the
// two partitions that this data came from. We determine which partition it belongs to based on the last_updated date of the row.
func (d *Deduper) copyTempTableBackToOriginal(tempTable string, def *BqDefinition, split time.Time) error {
	splitString := fmt.Sprintf("%v-%02d-%02d", split.Year(), int(split.Month()), split.Day())
	queryStr := fmt.Sprintf("SELECT * FROM %s.%s WHERE last_updated >= TIMESTAMP('%s')", def.BqDataset, tempTable, splitString)

	err := d.queryIntoOriginalPartition(queryStr, def, split)
	if err != nil {
		return err
	}

	queryStr2 := fmt.Sprintf("SELECT * FROM %s.%s WHERE last_updated < TIMESTAMP('%s') OR last_updated is null", def.BqDataset, tempTable, splitString)
	splitMinusOne := split.Add(-24 * time.Hour)
	return d.queryIntoOriginalPartition(queryStr2, def, splitMinusOne)
}

// queryIntoOriginalPartition runs a query, and saves the results into the original table, for a specific partition
func (d *Deduper) queryIntoOriginalPartition(queryStr string, def *BqDefinition, destPartition time.Time) error {
	destTableName := fmt.Sprintf("%v$%v%02d%02d", def.BqTable, destPartition.Year(), int(destPartition.Month()), destPartition.Day())

	query := d.BqClient.Client.Query(queryStr)
	log.Info(fmt.Sprintf("Running %s to %s", queryStr, destTableName))

	query.UseStandardSQL = true
	query.Dst = &bigquery.Table{
		ProjectID: d.ProjectID,
		DatasetID: def.BqDataset,
		TableID:   destTableName,
	}
	query.AllowLargeResults = true
	query.DisableFlattenedResults = true
	query.WriteDisposition = bigquery.WriteTruncate

	ctx := context.Background()
	job, err := query.Run(ctx)
	if err != nil {
		return err
	}

	return d.BqClient.WaitForJob(ctx, job)
}
