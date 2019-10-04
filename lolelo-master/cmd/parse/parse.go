package main

import (
	"flag"
	"log"

	"github.com/VantageSports/common/json"
	"github.com/VantageSports/lolelo/event"
	"github.com/VantageSports/lolelo/parse"
	"github.com/VantageSports/lolelo/validate"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/riot/api"
)

var in = flag.String("i", "/tmp/log.txt", "input log file to parse events from")
var out = flag.String("o", "/tmp/out.json", "output json file to write")
var md = flag.String("md", "", "optional match detail file")
var cg = flag.String("cg", "", "optional current_game file")
var eloOnly = flag.Bool("elo", false, "write the elo file only")

func main() {
	flag.Parse()

	events, err := parse.LogFile(*in)
	exitIf(err)

	var participants []baseview.Participant
	if *md != "" {
		matchDetail := &api.MatchDetail{}
		exitIf(json.DecodeFile(*md, matchDetail))
		exitIf(validate.AlignAPI(events, matchDetail))
		exitIf(validate.CheckTruncation(events, matchDetail.MatchDuration))
		if len(participants) == 0 {
			participants, err = validate.ParticipantMapMD(events, matchDetail)
			exitIf(err)
		}
	}

	var output interface{}
	if len(participants) == 0 || *eloOnly {
		log.Println("empty participants, just writing elo events")
		output = map[string]interface{}{
			"events": events,
		}
	} else {
		log.Println("writing baseview")
		output, err = event.EloToBaseview(events, participants)
		exitIf(err)
	}

	exitIf(json.Write(*out, output, 0600))
}

func exitIf(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
