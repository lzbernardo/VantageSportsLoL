package validate

import (
	"strings"
	"testing"

	"github.com/VantageSports/lolelo/event"
	"github.com/VantageSports/riot/api"

	"gopkg.in/tylerb/is.v1"
)

func TestCheckTruncation(t *testing.T) {
	is := is.New(t)

	md := &api.MatchDetail{MatchDuration: 200}
	events := []event.EloEvent{}
	is.Msg("empty events = error").Err(CheckTruncation(events, md.MatchDuration))

	events = []event.EloEvent{eventAt("P", 0.5), eventAt("D", 10), eventAt("D", 150)}
	is.Msg("last event should be game_end").Err(CheckTruncation(events, md.MatchDuration))

	events = append(events, eventAt("ND", 199.3))
	is.Msg("game_end shouldn't be enough to match time").Err(CheckTruncation(events, md.MatchDuration))

	// replace last event with a nexus_destroyed and then a game_end
	events = append(events[0:3], eventAt("ND", 199.3), eventAt("GE", 199.4))
	is.NotErr(CheckTruncation(events, md.MatchDuration))
}

func eventAt(t string, seconds float64) event.EloEvent {
	var e event.EloEvent
	switch t {
	case "D":
		e = &event.Damage{}
	case "GE":
		e = &event.GameEnd{}
	case "ND":
		e = &event.NexusDestroyed{}
	default:
		e = &event.Ping{}
	}
	e.SetTime(seconds)
	return e
}

func TestDetermineApiToEloOffset(t *testing.T) {
	is := is.New(t)

	apiDeaths := []participantEvent{
		{ParticipantID: 2, MatchSeconds: 100.0},
		{ParticipantID: 1, MatchSeconds: 200.0},
		{ParticipantID: 6, MatchSeconds: 300.0},
		{ParticipantID: 1, MatchSeconds: 400.0},
		{ParticipantID: 2, MatchSeconds: 500.0},
		{ParticipantID: 7, MatchSeconds: 600.0},
	}

	eloDeaths := []participantEvent{
		{ParticipantID: 2, EloSeconds: 110.0},
		{ParticipantID: 1, EloSeconds: 200.0},
		{ParticipantID: 6, EloSeconds: 320.0},
		{ParticipantID: 1, EloSeconds: 490.0},
		{ParticipantID: 2, EloSeconds: 500.0},
		{ParticipantID: 2, EloSeconds: 610.0}, // should be 7 to match API!
	}

	offset, err := determineEloToAPIOffset(apiDeaths, eloDeaths)
	is.Err(err)
	errMatches := strings.Contains(err.Error(), "2 by -1") || strings.Contains(err.Error(), "7 by 1")
	is.Msg("expected error for pid 2 or 7, got: " + err.Error()).True(errMatches)

	// Fix the elo participant and re-run.
	eloDeaths[5].ParticipantID = 7
	offset, err = determineEloToAPIOffset(apiDeaths, eloDeaths)
	is.NotErr(err)
	is.Equal(offset, -10.0) // median of diffs
}
