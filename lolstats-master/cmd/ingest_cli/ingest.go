// This is a little command-line stats generator, useful for testing stats
// creation locally.

package main

import (
	"flag"
	"log"

	"github.com/VantageSports/common/json"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/lolstats/generate"
	"github.com/VantageSports/lolstats/ingest"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/api"
)

var advanced = flag.Bool("advanced", true, "if true, do advanced, else do basic")
var matchDetailsFile = flag.String("md", "", "path to match details file (for basic)")
var baseviewFile = flag.String("bv", "", "path to baseview file (for advanced)")
var summonerID = flag.Int64("s", 0, "summoner id")
var matchID = flag.Int64("m", 0, "match id")
var platformID = flag.String("p", "NA1", "platform id")
var outFile = flag.String("o", "/tmp/stats.json", "path to baseview file")

func main() {
	flag.Parse()

	var stats interface{}
	if *advanced {
		stats = computeAdvanced(*baseviewFile, *summonerID, *matchID, *platformID)
	} else {
		stats = computeBasic(*matchDetailsFile, *summonerID)
	}

	err := json.WriteIndent(*outFile, stats, 0660)
	exitIf(err)
}

func computeBasic(detailsFile string, summonerID int64) *generate.BasicStats {
	md := api.MatchDetail{}
	err := json.DecodeFile(detailsFile, &md)
	exitIf(err)

	stats, err := generate.ComputeBasic(&md, summonerID)
	exitIf(err)
	return stats
}

func computeAdvanced(baseviewFile string, summonerID, matchID int64, platformID string) *ingest.AdvancedStats {
	bv := baseview.Baseview{}
	err := json.DecodeFile(baseviewFile, &bv)
	exitIf(err)

	platform := riot.PlatformFromString(platformID)
	stats, err := ingest.ComputeAdvanced(bv, summonerID, matchID, platform)
	exitIf(err)

	return stats
}

func exitIf(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
