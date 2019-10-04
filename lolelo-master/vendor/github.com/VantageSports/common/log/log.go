// log provides some utility methods wrapping go-kit's structured json logger
// with severity levels (so that we can better-harness cloud-logging to parse
// through the cruft)

package log

import (
	"io"
	"os"

	kitlog "github.com/go-kit/kit/log"
)

var ctx *kitlog.Context

func init() {
	WithWriter(os.Stderr)
}

func WithWriter(w io.Writer) {
	ctx = kitlog.NewContext(kitlog.NewJSONLogger(w)).With("ts", kitlog.DefaultTimestampUTC, "caller", kitlog.Caller(5))
}

// Debug describes trace-level information. Useful for understanding the flow
// of a specific request-path.
func Debug(v ...interface{}) {
	log(ctx, "debug", v...)
}

// Info is the default log level. routine information, such as ongoing status
// or performance.
func Info(v ...interface{}) {
	log(ctx, "info", v...)
}

// Notice is for normal but significant events, such as start up, shut down, or
// configuration.
func Notice(v ...interface{}) {
	log(ctx, "notice", v...)
}

// Warning events might cause problems.
func Warning(v ...interface{}) {
	log(ctx, "warning", v...)
}

// Error events are likely to cause problems.
func Error(v ...interface{}) {
	log(ctx, "error", v...)
}

// Critical events cause more severe problems or brief outages.
func Critical(v ...interface{}) {
	log(ctx, "critical", v...)
}

// Alert events require a person to take an action immediately.
func Alert(v ...interface{}) {
	log(ctx, "alert", v...)
}

// Fatal events exit the process after logging critical. This should only be
// called from main() packages, and only during startup/shutdown (not in
// response) to requets / user input, etc.
func Fatal(v ...interface{}) {
	log(ctx, "critical", v...)
	os.Exit(1)
}

func log(ctx *kitlog.Context, severity string, v ...interface{}) {
	ctx = ctx.With("severity", severity)
	if len(v) == 1 {
		ctx.Log("message", v[0])
	} else {
		ctx.Log(v...)
	}
}
