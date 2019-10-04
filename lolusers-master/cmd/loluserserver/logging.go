package main

import (
	"fmt"
	"time"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolusers"
	"golang.org/x/net/context"
)

type LolUsersLogger struct {
	next lolusers.LolUsersServer
}

func (u *LolUsersLogger) CrawlSummoners(ctx context.Context, in *lolusers.CrawlSummonersRequest) (*lolusers.SimpleResponse, error) {
	start := time.Now()
	out, err := u.next.CrawlSummoners(ctx, in)
	logRequest("CrawlSummoners", start, err)
	return out, err
}

func (u *LolUsersLogger) Create(ctx context.Context, in *lolusers.LolUserRequest) (*lolusers.LolUserResponse, error) {
	start := time.Now()
	out, err := u.next.Create(ctx, in)
	logRequest("Create", start, err)
	return out, err
}

func (u *LolUsersLogger) Update(ctx context.Context, in *lolusers.LolUserRequest) (*lolusers.LolUserResponse, error) {
	start := time.Now()
	out, err := u.next.Update(ctx, in)
	logRequest("Update", start, err)
	return out, err
}

func (u *LolUsersLogger) List(ctx context.Context, in *lolusers.ListLolUsersRequest) (*lolusers.LolUsersResponse, error) {
	start := time.Now()
	out, err := u.next.List(ctx, in)
	logRequest("List", start, err)
	return out, err
}

func (u *LolUsersLogger) AdjustVantagePoints(ctx context.Context, in *lolusers.VantagePointsRequest) (*lolusers.SimpleResponse, error) {
	start := time.Now()
	out, err := u.next.AdjustVantagePoints(ctx, in)
	logRequest("AdjustVantagePoints", start, err)
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
