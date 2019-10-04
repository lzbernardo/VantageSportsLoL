package ingest

import (
	"bytes"
	js "encoding/json"
	"fmt"
	"io/ioutil"
	"testing"

	"gopkg.in/tylerb/is.v1"

	"github.com/VantageSports/common/json"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/lolstats/testutil"
)

const (
	matchID    = int64(2344262088)
	summonerID = int64(26337245)
	// change to true temporarily when you want to rewrite golden file...
	rewriteGolden = false
)

func TestComputeAdvancedStats(t *testing.T) {
	is := is.New(t)

	bv := baseview.Baseview{}
	is.Nil(json.DecodeFile(fmt.Sprintf("../testdata/%d-NA1.baseview.json", matchID), &bv))

	advancedStats, err := ComputeAdvanced(bv, summonerID, matchID, "NA1")
	is.Nil(err)

	if rewriteGolden {
		is.Nil(json.WriteIndent(fmt.Sprintf("../testdata/%d-NA1-%d.advanced.json", matchID, summonerID), advancedStats, 0600))
		t.Error("rewrote golden file. test is invalid")
	}

	goldenData, err := ioutil.ReadFile(fmt.Sprintf("../testdata/%d-NA1-%d.advanced.json", matchID, summonerID))
	is.Nil(err)

	// Set the date to the golden file date to prevent diffs from happening every time
	goldenStruct := AdvancedStats{}
	is.Nil(js.Unmarshal(goldenData, &goldenStruct))
	advancedStats.LastUpdated = goldenStruct.LastUpdated

	ourData, err := js.MarshalIndent(advancedStats, "", "  ")
	is.Nil(err)

	if !bytes.Equal(goldenData, ourData) {
		t.Errorf("found diff between golden file (-) and advanced (+) stats...\n")
		t.Errorf(testutil.DiffLines(goldenData, ourData, t))
	}
}

func TestComputeAdvancedBqStats(t *testing.T) {
	is := is.New(t)

	bv := baseview.Baseview{}
	is.Nil(json.DecodeFile(fmt.Sprintf("../testdata/%d-NA1.baseview.json", matchID), &bv))

	advancedStats, err := ComputeAdvanced(bv, summonerID, matchID, "NA1")
	is.Nil(err)
	advancedStats.TrimNonStats()

	if rewriteGolden {
		is.Nil(json.WriteIndent(fmt.Sprintf("../testdata/%d-NA1-%d.advanced-bq.json", matchID, summonerID), advancedStats, 0600))
		t.Error("rewrote golden file. test is invalid")
	}

	goldenData, err := ioutil.ReadFile(fmt.Sprintf("../testdata/%d-NA1-%d.advanced-bq.json", matchID, summonerID))
	is.Nil(err)

	// Set the date to the golden file date to prevent diffs from happening every time
	goldenStruct := AdvancedStats{}
	is.Nil(js.Unmarshal(goldenData, &goldenStruct))
	advancedStats.LastUpdated = goldenStruct.LastUpdated

	ourData, err := js.MarshalIndent(advancedStats, "", "  ")
	is.Nil(err)

	if !bytes.Equal(goldenData, ourData) {
		t.Errorf("found diff in bq json between golden file (-) and advanced (+) stats...\n")
		t.Errorf(testutil.DiffLines(goldenData, ourData, t))
	}
}

func TestComputeAverageAdvancedStats(t *testing.T) {
	is := is.New(t)

	bv := baseview.Baseview{}
	is.Nil(json.DecodeFile(fmt.Sprintf("../testdata/%d-NA1.baseview.json", matchID), &bv))

	aggAdvancedStats := []AdvancedStats{}
	for _, participant := range bv.Participants {
		if baseview.TeamID(participant.ParticipantID) == 100 {
			advancedStats, err := ComputeAdvanced(bv, participant.SummonerID, matchID, "NA1")
			is.Nil(err)

			aggAdvancedStats = append(aggAdvancedStats, *advancedStats)
		}
	}
	averaged := AverageAdvancedStats(aggAdvancedStats)

	if rewriteGolden {
		is.Nil(json.WriteIndent(fmt.Sprintf("../testdata/%d-NA1-agg.advanced.json", matchID), averaged, 0600))
		t.Error("rewrote golden file. test is invalid")
	}

	goldenData, err := ioutil.ReadFile(fmt.Sprintf("../testdata/%d-NA1-agg.advanced.json", matchID))
	is.Nil(err)

	// Set the date to the golden file date to prevent diffs from happening every time
	goldenStruct := AdvancedStats{}
	is.Nil(js.Unmarshal(goldenData, &goldenStruct))
	averaged.LastUpdated = goldenStruct.LastUpdated

	ourData, err := js.MarshalIndent(averaged, "", "  ")
	is.Nil(err)

	if !bytes.Equal(goldenData, ourData) {
		t.Errorf("found diff between golden file (-) and advanced (+) stats...\n")
		t.Errorf(testutil.DiffLines(goldenData, ourData, t))
	}
}
