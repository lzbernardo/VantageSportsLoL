package main

import (
	"fmt"
	"time"

	"golang.org/x/net/context"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats"
)

type StatsLogger struct {
	next lolstats.LolstatsServer
}

func (s *StatsLogger) History(ctx context.Context, in *lolstats.HistoryRequest) (*lolstats.HistoryResponse, error) {
	start := time.Now()
	out, err := s.next.History(ctx, in)
	logRequest("History", start, err)
	return out, err
}

func (s *StatsLogger) Percentiles(ctx context.Context, in *lolstats.StatsRequest) (*lolstats.StatListResponse, error) {
	start := time.Now()
	out, err := s.next.Percentiles(ctx, in)
	logRequest("Percentiles", start, err)
	return out, err
}

func (s *StatsLogger) Details(ctx context.Context, in *lolstats.MatchRequest) (*lolstats.DetailsResponse, error) {
	start := time.Now()
	out, err := s.next.Details(ctx, in)
	logRequest("Details", start, err)
	return out, err
}

func (s *StatsLogger) Advanced(ctx context.Context, in *lolstats.MatchRequest) (*lolstats.AdvancedStats, error) {
	start := time.Now()
	out, err := s.next.Advanced(ctx, in)
	logRequest("Advanced", start, err)
	return out, err
}

func (s *StatsLogger) SummonerMatches(ctx context.Context, in *lolstats.StatsRequest) (*lolstats.StatListResponse, error) {
	start := time.Now()
	out, err := s.next.SummonerMatches(ctx, in)
	logRequest("SummonerMatches", start, err)
	return out, err
}

func (s *StatsLogger) Search(ctx context.Context, in *lolstats.SearchRequest) (*lolstats.SearchResponse, error) {
	start := time.Now()
	out, err := s.next.Search(ctx, in)
	logRequest("Search", start, err)
	return out, err
}

func (s *StatsLogger) TeamDetails(ctx context.Context, in *lolstats.TeamRequest) (*lolstats.TeamDetailsResponse, error) {
	start := time.Now()
	out, err := s.next.TeamDetails(ctx, in)
	logRequest("TeamDetails", start, err)
	return out, err
}

func (s *StatsLogger) TeamAdvanced(ctx context.Context, in *lolstats.TeamRequest) (*lolstats.TeamAdvancedStats, error) {
	start := time.Now()
	out, err := s.next.TeamAdvanced(ctx, in)
	logRequest("TeamAdvanced", start, err)
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
