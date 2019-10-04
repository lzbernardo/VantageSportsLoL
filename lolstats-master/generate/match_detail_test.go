package generate

import (
	"bytes"
	js "encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"testing"
	"time"

	"gopkg.in/tylerb/is.v1"

	"github.com/VantageSports/common/json"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/lolstats/testutil"
	"github.com/VantageSports/riot/api"
)

func TestGetDeadTimeEnd(t *testing.T) {
	// These times were pulled from elo data on match 2291255478-NA1.
	// Elo times can be off due to the sampling rate, but should be within 1 second
	within1(t, 212.08, deathEndSeconds(199.337, 2))
	within1(t, 248.05, deathEndSeconds(229.950, 4))
	within1(t, 267.89, deathEndSeconds(252.515, 3))
	within1(t, 639.39, deathEndSeconds(614.046, 7))
	within1(t, 1200.94, deathEndSeconds(1169.487, 9))
	within1(t, 1222.19, deathEndSeconds(1187.796, 10))
	within1(t, 1239.85, deathEndSeconds(1202.715, 11))
	within1(t, 1736.34, deathEndSeconds(1692.005, 13))
	within1(t, 2114.10, deathEndSeconds(2061.854, 15))
	within1(t, 1960.92, deathEndSeconds(1903.320, 17))
	within1(t, 2240.08, deathEndSeconds(2178.591, 18))
}

var (
	testMatchID    int64 = 2344262088
	testSummonerID int64 = 26337245
	rewriteGolden        = false
)

func TestComputeHistory(t *testing.T) {
	is := is.New(t)

	md := api.MatchDetail{}
	is.Nil(json.DecodeFile(fmt.Sprintf("../testdata/%d-na.json", testMatchID), &md))

	history, err := ComputeHistory(&md, testSummonerID)
	is.Nil(err)

	// make sure the created/last_updated times match the golden file.
	when := time.Date(2016, 1, 2, 3, 4, 5, 6, time.UTC)
	history.Created, history.LastUpdated = when, when

	historyPath := fmt.Sprintf("../testdata/%d-NA1-%d.history.json", testMatchID, testSummonerID)
	// change to true if you want to regenerate the golden file.
	if rewriteGolden {
		is.Nil(json.WriteIndent(historyPath, history, 0600))
		t.Error("rewrote golden file. test is invalid")
	}

	goldenData, err := ioutil.ReadFile(historyPath)
	is.Nil(err)

	ourData, err := js.MarshalIndent(history, "", "  ")
	is.Nil(err)

	if !bytes.Equal(goldenData, ourData) {
		t.Errorf("found diff between golden file (-) and history (+)...\n")
		t.Errorf(testutil.DiffLines(goldenData, ourData, t))
	}
}

func TestComputeBasic(t *testing.T) {
	is := is.New(t)

	md := api.MatchDetail{}
	is.Nil(json.DecodeFile(fmt.Sprintf("../testdata/%d-na.json", testMatchID), &md))

	stats, err := ComputeBasic(&md, testSummonerID)
	is.Nil(err)

	basicPath := fmt.Sprintf("../testdata/%d-NA1-%d.basic.json", testMatchID, testSummonerID)
	// change to true if you want to regenerate the golden file.
	if rewriteGolden {
		is.Nil(json.WriteIndent(basicPath, stats, 0600))
		t.Error("rewrote golden file. test is invalid")
	}

	goldenData, err := ioutil.ReadFile(basicPath)
	is.Nil(err)

	// Set the date to the golden file date to prevent diffs from happening every time
	goldenStruct := BasicStats{}
	is.Nil(js.Unmarshal(goldenData, &goldenStruct))
	stats.LastUpdated = goldenStruct.LastUpdated

	ourData, err := js.MarshalIndent(stats, "", "  ")
	is.Nil(err)

	if !bytes.Equal(goldenData, ourData) {
		t.Errorf("found diff between golden file (-) and basic (+) stats...\n")
		t.Errorf(testutil.DiffLines(goldenData, ourData, t))
	}
}
func TestComputeBasicBqStats(t *testing.T) {
	is := is.New(t)
	md := api.MatchDetail{}
	is.Nil(json.DecodeFile(fmt.Sprintf("../testdata/%d-na.json", testMatchID), &md))

	stats, err := ComputeBasic(&md, testSummonerID)
	is.Nil(err)
	stats.TrimNonStats()

	basicPath := fmt.Sprintf("../testdata/%d-NA1-%d.basic-bq.json", testMatchID, testSummonerID)
	// change to true if you want to regenerate the golden file.
	if rewriteGolden {
		is.Nil(json.WriteIndent(basicPath, stats, 0600))
		t.Error("rewrote golden file. test is invalid")
	}

	goldenData, err := ioutil.ReadFile(basicPath)
	is.Nil(err)

	// Set the date to the golden file date to prevent diffs from happening every time
	goldenStruct := BasicStats{}
	is.Nil(js.Unmarshal(goldenData, &goldenStruct))
	stats.LastUpdated = goldenStruct.LastUpdated

	ourData, err := js.MarshalIndent(stats, "", "  ")
	is.Nil(err)

	if !bytes.Equal(goldenData, ourData) {
		t.Errorf("found diff in bq json between golden file (-) and basic (+) stats...\n")
		t.Errorf(testutil.DiffLines(goldenData, ourData, t))
	}

}

func TestComputeBasicAverageStats(t *testing.T) {
	is := is.New(t)
	md := api.MatchDetail{}
	is.Nil(json.DecodeFile(fmt.Sprintf("../testdata/%d-na.json", testMatchID), &md))

	aggBasicStats := []BasicStats{}
	for _, pId := range md.ParticipantIdentities {
		if baseview.TeamID(int64(pId.ParticipantID)) == 100 {
			stats, err := ComputeBasic(&md, pId.Player.SummonerID)
			is.Nil(err)

			aggBasicStats = append(aggBasicStats, *stats)
		}
	}
	averagedStats := AverageBasicStats(aggBasicStats)

	aggPath := fmt.Sprintf("../testdata/%d-NA1-agg.basic.json", testMatchID)
	// change to true if you want to regenerate the golden file.
	if rewriteGolden {
		is.Nil(json.WriteIndent(aggPath, averagedStats, 0600))
		t.Error("rewrote golden file. test is invalid")
	}

	goldenData, err := ioutil.ReadFile(aggPath)
	is.Nil(err)

	// Set the date to the golden file date to prevent diffs from happening every time
	goldenStruct := BasicStats{}
	is.Nil(js.Unmarshal(goldenData, &goldenStruct))
	averagedStats.LastUpdated = goldenStruct.LastUpdated

	ourData, err := js.MarshalIndent(averagedStats, "", "  ")
	is.Nil(err)

	if !bytes.Equal(goldenData, ourData) {
		t.Errorf("found diff in averaged json between golden file (-) and basic (+) stats...\n")
		t.Errorf(testutil.DiffLines(goldenData, ourData, t))
	}
}

func within1(t *testing.T, a, b float64) {
	diff := math.Abs(a - b)
	is.New(t).Msg("Expected %f and %f to be within 1, but diff is %f", a, b, diff).True(diff < 1)
}
