package bigquery

import (
	"fmt"
	"time"

	"cloud.google.com/go/bigquery"
	"golang.org/x/net/context"
	"google.golang.org/api/option"

	"github.com/VantageSports/common/credentials/google"
	"github.com/VantageSports/common/json"
)

// Client is a light wrapper for the bigquery library to make queries
// with proper context.
type Client struct {
	ProjectID string
	Client    *bigquery.Client
}

func NewClient(creds *google.Creds) (*Client, error) {
	ctx := context.Background()
	client, err := bigquery.NewClient(ctx, creds.ProjectID, option.WithTokenSource(creds.TokenSource(ctx)))
	return &Client{
		ProjectID: creds.ProjectID,
		Client:    client,
	}, err
}

// Query submits a query job to BigQuery from the query string provided.
// e.g. "SELECT * FROM [dataset.table]"
func (q *Client) Query(ctx context.Context, queryStr string, useStandardSQL bool) (*bigquery.Job, error) {
	query := q.Client.Query(queryStr)
	query.UseStandardSQL = useStandardSQL
	return query.Run(ctx)
}

// InsertAll 'streams' inserts into a table.
func (q *Client) InsertAll(ctx context.Context, datasetID, tableID string, rows ...bigquery.ValueSaver) error {
	t := q.Client.Dataset(datasetID).Table(tableID)
	return t.Uploader().Put(ctx, rows)
}

// Load opens files from GCS and uploads those json records to the provided
// table in the given dataset.
//  - uri is file path in GCS to be loaded into BigQuery table.
//  - dataset is the container for the table being queried.
//  - table is the table containing the data for the query.
//  - schemaFile is the JSON file path of the list of schema fields. If the
//      table exists, there is no need to have the schemaFile parameter set.
//  - ignoreUnknown indicates if BigQuery should allow extra values that are
//      not represented in the table schema
// NOTE: json files can either all be compressed or all be uncompressed.
func (q *Client) Load(ctx context.Context, dataset, table, schemaPath string, format bigquery.DataFormat, ignoreUnknown bool, uris ...string) (*bigquery.Job, error) {
	bqTable := q.Client.Dataset(dataset).Table(table)

	gcs := bigquery.NewGCSReference(uris...)
	gcs.MaxBadRecords = 1
	gcs.IgnoreUnknownValues = ignoreUnknown
	gcs.SourceFormat = format

	if schemaPath != "" {
		schema := bigquery.Schema{}
		if err := json.DecodeFile(schemaPath, &schema); err != nil {
			return nil, err
		}
		gcs.Schema = schema
	}

	loader := bqTable.LoaderFrom(gcs)
	return loader.Run(ctx)
}

// WaitForJob continually checks the status of the specified job, returning only
// once the job has "DONE" state. NOTE: done doesn't indicate success.
func (q *Client) WaitForJob(ctx context.Context, j *bigquery.Job) (err error) {
	var status *bigquery.JobStatus
	for range time.Tick(time.Second * 5) {
		status, err = j.Status(ctx)
		if err != nil {
			return fmt.Errorf("unable to determine job status: %v", err)
		}
		if !status.Done() {
			continue
		}
		break
	}
	return status.Err()
}

// RowParser uses a Schema to parse rows into generic map objects (key->value).
type RowParser struct {
	lastRow map[string]interface{}
}

func (rp *RowParser) LastRow() map[string]interface{} {
	return rp.lastRow
}

// Load is the implementation of bigquery.ValueLoader.
func (rp *RowParser) Load(vals []bigquery.Value, schema bigquery.Schema) error {
	fields := map[int]*bigquery.FieldSchema{}
	for i, field := range schema {
		fields[i] = field
	}

	res := map[string]interface{}{}
	for i, val := range vals {
		field := fields[i]
		if val == nil && !field.Required {
			continue
		}
		res[field.Name] = val
	}

	rp.lastRow = res
	return nil
}
