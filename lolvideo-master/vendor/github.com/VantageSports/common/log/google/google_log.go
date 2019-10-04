package google

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"cloud.google.com/go/logging"
	"golang.org/x/net/context"
	"google.golang.org/api/option"

	"github.com/VantageSports/common/credentials/google"
)

// GoogleLogWriter is a writer that can be written to by a json logger in
// common/log (using for any other purpose will probably be difficult), and will
// send those logs to the google cloud logging console. This writer is useful
// for applications that run somewhere where stderr logs aren't automatically
// collected for us (such as managed instance groups or amazon ec2, etc).
//
// Usage might look like:
// creds := parseGoogleCreds("/path/to/creds.json", logging.Scope)
// gWriter, _ := google.New("creds", "mylog", nil, false)
// log.WithWriter(gWriter)
// ...
// log.Error("uh oh!")
type GoogleLogWriter struct {
	logger *logging.Logger
	stdErr bool // if true, each written log entry is also written to stdErr
}

// New returns a GoogleLogWriter that can be used as the writer beneath a
// go-kit json logger.
func New(creds *google.Creds, logName string, labels map[string]string, stdErr bool) (*GoogleLogWriter, error) {
	ctx := context.Background()

	client, err := logging.NewClient(ctx, creds.ProjectID, option.WithTokenSource(creds.TokenSource(ctx)))
	if err != nil {
		return nil, err
	}

	log := fmt.Sprintf("/projects/%s/logs/%s", creds.ProjectID, logName)
	logger := client.Logger(log, logging.CommonLabels(labels))

	return &GoogleLogWriter{logger: logger, stdErr: stdErr}, err
}

// Write receives the output of kit's json-marshaled byte stream, and attempts
// to parse out a severify and a message from the encoded stream. If it can't
// determine either, it writes the bytes as-received.
func (w *GoogleLogWriter) Write(p []byte) (n int, err error) {
	if w.stdErr {
		fmt.Fprintln(os.Stderr, string(p))
	}

	m := map[string]interface{}{}

	if err := json.Unmarshal(p, &m); err != nil {
		w.logger.Log(logging.Entry{
			Payload:  fmt.Sprintf("%s", p),
			Severity: logging.Default,
		})
		return len(p), nil
	}

	entry := getEntry(m)
	w.logger.Log(entry)
	return len(p), nil
}

// getEntry generates a logging.Entry from the parsed json object. It is broken
// out into its own function mostly for test purposes.
func getEntry(m map[string]interface{}) logging.Entry {
	return logging.Entry{
		Payload:  getMessage(m),
		Severity: getSeverity(m["severity"]),
	}
}

// getMessage returns the value of the "message" key in the specified json
// object, otherwise returning the whole object.
func getMessage(m map[string]interface{}) interface{} {
	if message := m["message"]; message != nil {
		return message
	}
	return m
}

// getSeverity translates from the common/log levels to the cloud logging levels.
// See https://github.com/GoogleCloudPlatform/gcloud-golang/blob/master/logging/logging.go#L39
func getSeverity(v interface{}) logging.Severity {
	if str, ok := v.(string); ok {
		switch strings.ToLower(str) {
		case "debug":
			return logging.Debug
		case "notice", "info":
			return logging.Info
		case "warning":
			return logging.Warning
		case "error":
			return logging.Error
		case "critical":
			return logging.Critical
		case "alert":
			return logging.Alert
		default:
			return logging.Default
		}
	}
	return logging.Default
}
