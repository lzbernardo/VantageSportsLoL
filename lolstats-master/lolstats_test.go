package lolstats

import (
	"testing"
)

func TestStatsRequestValid(t *testing.T) {
	r := &StatsRequest{}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for empty request")
	}

	r = &StatsRequest{
		Selects: []string{"kills"},
	}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for no filter request")
	}

	r = &StatsRequest{
		Platform: "NA1",
	}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for no selects request")
	}

	r = &StatsRequest{
		Selects:     []string{"kills"},
		Platform:    "NA1",
		PatchPrefix: "6.8",
	}
	if err := r.Valid(); err != nil {
		t.Error("Unexpected error for valid request")
	}
}

func TestHistoryRequestValid(t *testing.T) {
	r := &HistoryRequest{}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for empty request")
	}

	r = &HistoryRequest{
		SummonerId: 123,
	}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for just summonerId")
	}

	r = &HistoryRequest{
		SummonerId: 123,
		Platform:   "NA1",
	}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for no limit")
	}

	r = &HistoryRequest{
		Platform: "NA1",
		Limit:    1,
	}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for no summonerId")
	}

	r = &HistoryRequest{
		SummonerId: 123,
		Platform:   "NA1",
		Limit:      1,
	}
	if err := r.Valid(); err != nil {
		t.Error("Unexpected error for valid request")
	}
}

func TestMatchRequestValid(t *testing.T) {
	r := &MatchRequest{}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for empty request")
	}

	r = &MatchRequest{
		SummonerId: 123,
	}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for just summonerId")
	}

	r = &MatchRequest{
		SummonerId: 123,
		Platform:   "NA1",
	}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for no matchId")
	}

	r = &MatchRequest{
		Platform: "NA1",
		MatchId:  345,
	}
	if err := r.Valid(); err == nil {
		t.Error("Expected error for no summonerId")
	}

	r = &MatchRequest{
		SummonerId: 123,
		Platform:   "NA1",
		MatchId:    345,
	}
	if err := r.Valid(); err != nil {
		t.Error("Unexpected error for valid request")
	}
}
