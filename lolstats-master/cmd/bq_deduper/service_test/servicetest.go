package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"

	vsbigquery "github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/lolstats/bq"
)

const (
	ProjectId = "vs-dev"
	Dataset   = "vsd_lol"
	TestTable = "bq_deduper_service_test"
)

var Deduper *bq.Deduper
var AdvTableDef *bq.BqDefinition
var DataTime = time.Date(2016, 12, 20, 0, 0, 0, 0, time.UTC)
var DataTime2 = time.Date(2016, 12, 19, 0, 0, 0, 0, time.UTC)
var DataTime3 = time.Date(2016, 12, 18, 0, 0, 0, 0, time.UTC)

func main() {
	creds := google.MustEnvCreds(ProjectId, bigquery.Scope)
	client, err := vsbigquery.NewClient(creds)
	exitIf(err)

	Deduper = &bq.Deduper{ProjectId, client, 2}
	AdvTableDef = &bq.BqDefinition{Dataset, TestTable, "../advanced_stats_schema.json"}

	exitIf(CheckTestTable(client))
	exitIf(TestBasic(client))
	exitIf(TestCopyBackToOriginal(client))
	exitIf(TestTwoPartitionsOverlap(client))
}

func exitIf(e error) {
	if e != nil {
		log.Fatal(e)
	}
}

func CheckTestTable(client *vsbigquery.Client) error {
	_, err := client.Client.Dataset(Dataset).Table(TestTable).Metadata(context.Background())
	if err != nil {
		if strings.HasPrefix(err.Error(), "googleapi: Error 404: Not found:") {
			return fmt.Errorf("Test table not found please run the following:\n"+
				"bq mk --project_id %s --time_partitioning_type=DAY --schema ../advanced_stats_schema.json %s.%s\n", ProjectId, Dataset, TestTable)
		} else {
			return err
		}
	}
	return nil
}

func TestBasic(client *vsbigquery.Client) error {
	// Create 4 duplicate rows in the same partition.
	partition1 := fmt.Sprintf("%v%v%v", DataTime.Year(), int(DataTime.Month()), DataTime.Day())
	exitIf(insertRow(client, "advanced1.json", partition1))
	exitIf(insertRow(client, "advanced1.json", partition1))
	exitIf(insertRow(client, "advanced2.json", partition1))
	exitIf(insertRow(client, "advanced2.json", partition1))

	numRows, err := countRows(client, TestTable, partition1)
	exitIf(err)
	fmt.Printf("Rows before: %v\n", numRows)

	exitIf(Deduper.Dedupe(AdvTableDef, DataTime))

	numRows, err = countRows(client, TestTable, partition1)
	exitIf(err)

	fmt.Printf("Rows after: %v\n", numRows)

	if numRows != 2 {
		return fmt.Errorf("unexpected result.")
	}
	return nil
}

func TestCopyBackToOriginal(client *vsbigquery.Client) error {
	// Create 4 rows, with the partition not quite matching up to the last_updated date
	partition1 := fmt.Sprintf("%v%v%v", DataTime.Year(), int(DataTime.Month()), DataTime.Day())
	partition2 := fmt.Sprintf("%v%v%v", DataTime2.Year(), int(DataTime2.Month()), DataTime2.Day())
	partition3 := fmt.Sprintf("%v%v%v", DataTime3.Year(), int(DataTime3.Month()), DataTime3.Day())

	exitIf(insertRow(client, "advanced1.json", partition1))
	exitIf(insertRow(client, "advanced2.json", partition2)) // This should be moved to partition1
	exitIf(insertRow(client, "advanced3.json", partition3)) // This should be moved to partition2
	exitIf(insertRow(client, "advanced4.json", partition1)) // This should be moved to partition3

	numRows1, err := countRows(client, TestTable, partition1)
	exitIf(err)
	numRows2, err := countRows(client, TestTable, partition2)
	exitIf(err)
	numRows3, err := countRows(client, TestTable, partition3)
	exitIf(err)
	fmt.Printf("Rows before, partition1: %v, partition2: %v, partition3: %v\n", numRows1, numRows2, numRows3)

	exitIf(Deduper.Dedupe(AdvTableDef, DataTime))

	numRows1, err = countRows(client, TestTable, partition1)
	exitIf(err)
	numRows2, err = countRows(client, TestTable, partition2)
	exitIf(err)
	numRows3, err = countRows(client, TestTable, partition3)
	exitIf(err)

	fmt.Printf("Rows after, partition1: %v, partition2: %v, partition3: %v\n", numRows1, numRows2, numRows3)
	if numRows1 != 2 || numRows2 != 1 || numRows3 != 1 {
		return fmt.Errorf("unexpected output")
	}
	return nil
}

func TestTwoPartitionsOverlap(client *vsbigquery.Client) error {
	// Create duplicates of each row in both partitions
	partition1 := fmt.Sprintf("%v%v%v", DataTime.Year(), int(DataTime.Month()), DataTime.Day())
	partition2 := fmt.Sprintf("%v%v%v", DataTime2.Year(), int(DataTime2.Month()), DataTime2.Day())

	exitIf(insertRow(client, "advanced1.json", partition1))
	exitIf(insertRow(client, "advanced1.json", partition2))
	exitIf(insertRow(client, "advanced2.json", partition1))
	exitIf(insertRow(client, "advanced2.json", partition2))
	exitIf(insertRow(client, "advanced3.json", partition1))
	exitIf(insertRow(client, "advanced3.json", partition2))

	numRows1, err := countRows(client, TestTable, partition1)
	exitIf(err)
	numRows2, err := countRows(client, TestTable, partition2)
	exitIf(err)
	fmt.Printf("Rows before, partition1: %v, partition2: %v\n", numRows1, numRows2)

	exitIf(Deduper.Dedupe(AdvTableDef, DataTime))

	numRows1, err = countRows(client, TestTable, partition1)
	exitIf(err)
	numRows2, err = countRows(client, TestTable, partition2)
	exitIf(err)

	fmt.Printf("Rows after, partition1: %v, partition2: %v\n", numRows1, numRows2)
	if numRows1 != 2 || numRows2 != 1 {
		return fmt.Errorf("unexpected output")
	}
	return nil
}

func insertRow(client *vsbigquery.Client, filename, partition string) error {
	f, err := os.Open(filename)
	defer f.Close()
	if err != nil {
		return err
	}
	reader := bufio.NewReader(f)
	readerSource := bigquery.NewReaderSource(reader)
	readerSource.SourceFormat = bigquery.JSON

	tablePartition := fmt.Sprintf("%v$%v", TestTable, partition)
	fmt.Printf("Inserting %v into %v\n", filename, tablePartition)
	job, err := client.Client.Dataset(Dataset).Table(tablePartition).LoaderFrom(readerSource).Run(context.Background())
	if err != nil {
		return err
	}

	return client.WaitForJob(context.Background(), job)
}

func countRows(client *vsbigquery.Client, tableName, partition string) (int64, error) {
	ctx := context.Background()
	queryStr := fmt.Sprintf("SELECT count(*) from %s.%s$%s", Dataset, tableName, partition)
	job, err := client.Query(ctx, queryStr, false)
	exitIf(err)

	it, err := job.Read(ctx)
	exitIf(err)

	rows := []map[string]interface{}{}
	parser := vsbigquery.RowParser{}

	for {
		err = it.Next(&parser)
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, err
		}
		rows = append(rows, parser.LastRow())
	}

	for _, v := range rows[0] {
		count := v.(int64)
		return count, nil
	}
	return 0, fmt.Errorf("cannot find count")
}
