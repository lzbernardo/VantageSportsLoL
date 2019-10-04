package log

import (
	"fmt"
	"os"
	"strings"

	kitlog "github.com/go-kit/kit/log"
)

type Verbosity string

const (
	quiet  Verbosity = "quiet"
	silent Verbosity = "silent"
)

type option func(*gRPCAdapter)

func Quiet(g *gRPCAdapter) {
	g.verbosity = quiet
}

func Silent(g *gRPCAdapter) {
	g.verbosity = silent
}

type gRPCAdapter struct {
	ctx       *kitlog.Context
	verbosity Verbosity
}

func NewGRPCAdapter(opts ...option) *gRPCAdapter {
	adapter := &gRPCAdapter{
		ctx: ctx.With("caller", kitlog.Caller(7)),
	}
	for _, opt := range opts {
		opt(adapter)
	}
	return adapter
}

func (a *gRPCAdapter) Fatal(args ...interface{}) {
	a.fatal(fmt.Sprint(args...))
}

func (a *gRPCAdapter) Fatalf(format string, args ...interface{}) {
	a.fatal(fmt.Sprintf(format, args...))
}

func (a *gRPCAdapter) Fatalln(args ...interface{}) {
	a.fatal(fmt.Sprintln(args...))
}

func (a *gRPCAdapter) Print(args ...interface{}) {
	a.debug(fmt.Sprint(args...))
}

func (a *gRPCAdapter) Printf(format string, args ...interface{}) {
	a.debug(fmt.Sprintf(format, args...))
}

func (a *gRPCAdapter) Println(args ...interface{}) {
	a.debug(fmt.Sprintln(args...))
}

func (a *gRPCAdapter) fatal(message string) {
	log(a.ctx, "critical", message)
	os.Exit(1)
}

func (a *gRPCAdapter) debug(message string) {
	if a.verbosity == silent {
		return
	}
	if a.verbosity == quiet && isNoise(message) {
		return
	}
	log(a.ctx, "debug", message)
}

func isNoise(message string) bool {
	if strings.Contains(message, "transport: http2Client.notifyError got notified that the client transport was broken") {
		return true
	}
	return false
}
