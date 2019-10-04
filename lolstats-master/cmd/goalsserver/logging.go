package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats"
)

type GoalsLogger struct {
	next lolstats.LolgoalsServer
}

func (s *GoalsLogger) CreateStats(ctx context.Context, in *lolstats.GoalCreateStatsRequest) (*lolstats.CountResponse, error) {
	start := time.Now()
	out, err := s.next.CreateStats(ctx, in)
	logRequest("CreateStats", start, err)
	return out, err
}

func (s *GoalsLogger) CreateCustom(ctx context.Context, in *lolstats.GoalCreateCustomRequest) (*lolstats.SimpleResponse, error) {
	start := time.Now()
	out, err := s.next.CreateCustom(ctx, in)
	logRequest("CreateCustom", start, err)
	return out, err
}

func (s *GoalsLogger) Get(ctx context.Context, in *lolstats.GoalGetRequest) (*lolstats.GoalGetResponse, error) {
	start := time.Now()
	out, err := s.next.Get(ctx, in)
	logRequest("Get", start, err)
	return out, err
}

func (s *GoalsLogger) Delete(ctx context.Context, in *lolstats.GoalDeleteRequest) (*lolstats.SimpleResponse, error) {
	start := time.Now()
	out, err := s.next.Delete(ctx, in)
	logRequest("Delete", start, err)
	return out, err
}

func (s *GoalsLogger) UpdateStatus(ctx context.Context, in *lolstats.GoalUpdateStatusRequest) (*lolstats.SimpleResponse, error) {
	start := time.Now()
	out, err := s.next.UpdateStatus(ctx, in)
	logRequest("UpdateStatus", start, err)
	return out, err
}

func logRequest(method string, start time.Time, err error, v ...interface{}) {
	millis := int64(time.Since(start).Seconds() * 1000)
	v = append(v, "method", method, "ms", millis, "err", err)

	// Make a pretty message since there will be a lot of these message.
	prettyMsg := fmt.Sprintf("%s (%d ms)", method, millis)
	if err != nil {
		prettyMsg += " err: " + err.Error()
	}
	v = append(v, "message", prettyMsg)

	log.Info(v...)
}
