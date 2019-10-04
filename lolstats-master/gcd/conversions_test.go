package gcd

import (
	"reflect"
	"testing"
	"time"

	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/riot"
)

func TestFromMatchSummary(t *testing.T) {
	ms := &lolstats.MatchSummary{
		SummonerId: 123,
		Platform:   "NA1",
		Lane:       "myLane",
		Tier:       "DIAMOND",
		Division:   "I",
		ChampionId: 222,
	}

	statRow, err := FromMatchSummary(ms)
	if err != nil {
		t.Error("Unexpected error for valid request", err)
	}

	expectedStatRow := &HistoryRow{
		SummonerID: 123,
		Platform:   "NA1",
		Lane:       "myLane",
		Tier:       riot.DIAMOND,
		Division:   riot.DIV_I,
		ChampionID: 222,
	}

	if !reflect.DeepEqual(expectedStatRow, statRow) {
		t.Error("value mismatch.")
	}
}

func TestToMatchSummary(t *testing.T) {
	now := time.Now()
	statRow := HistoryRow{
		SummonerID: 123,
		Platform:   "NA1",
		Lane:       "myLane",
		OffMeta:    true,
		Tier:       riot.DIAMOND,
		Division:   riot.DIV_I,
		Created:    now,
		ChampionID: 222,
	}

	ms, err := statRow.ToMatchSummary()
	if err != nil {
		t.Error(err)
	}

	expectedMatchSummary := &lolstats.MatchSummary{
		SummonerId: 123,
		Platform:   "NA1",
		Lane:       "myLane",
		Tier:       "DIAMOND",
		Division:   "I",
		ChampionId: 222,
	}

	if !reflect.DeepEqual(expectedMatchSummary, ms) {
		t.Error("value mismatch.")
	}
}
