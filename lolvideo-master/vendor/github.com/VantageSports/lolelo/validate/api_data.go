// The validate package uses external data sources (riot match details,
// spectator data, etc) to validate/augment elo data.

package validate

import (
	"fmt"
	"math"
	"sort"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolelo/event"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/api"
)

const maxSecondsDiff = 30.0

// CheckTruncation returns an error if the elo-determined match duration is
// not within an acceptable range of the riot-api-determined match duration.
func CheckTruncation(events []event.EloEvent, matchDurationSecs int64) error {
	eloSecs := eloDurationSeconds(events, true)
	matchSecs := float64(matchDurationSecs)

	if math.Abs(eloSecs-matchSecs) > maxSecondsDiff {
		return fmt.Errorf("elo duration is %.1f, but api match duration is %.1f", eloSecs, matchSecs)
	}
	return nil
}

// eloDurationSeconds returns the duration of the match represented by the
// specified events. If skipPings is true, pings are skipped for the purpose of
// determining the 'last' event. This is usually useful because when elodata
// is tracked for a game in which one team surrendered, there is no event
// representing the 'end' of the game, and pings last for several minutes beyond
// the last real game action.
func eloDurationSeconds(events []event.EloEvent, skip bool) float64 {
	if len(events) < 1 {
		return 0.0
	}

	var lastEvent event.EloEvent

	for i := len(events) - 2; lastEvent == nil && i >= 0; i-- {
		e := events[i]
		// skip insignificant events, assume there's a BasicAttack, Damage, ChampKill, etc often enough.
		if skip {
			switch e.(type) {
			case *event.Damage, *event.ChampDie, *event.ChampKill, *event.SpellCast, *event.NexusDestroyed:
				lastEvent = e
			}
		}
	}
	if lastEvent == nil {
		return 0.0
	}

	return lastEvent.Time() - events[0].Time()
}

func AlignAPI(events []event.EloEvent, matchDetail *api.MatchDetail) error {
	apiDeaths, err := apiDeaths(matchDetail)
	if err != nil {
		return err
	}
	nameToParticipantID := map[string]int64{}
	for _, p := range matchDetail.ParticipantIdentities {
		nameToParticipantID[p.Player.SummonerName] = int64(p.ParticipantID)
	}
	eloDeaths, err := eloDeaths(events, nameToParticipantID)
	if err != nil {
		return err
	}
	offset, err := determineEloToAPIOffset(apiDeaths, eloDeaths)
	if err != nil {
		return err
	}
	UpdateMatchSeconds(events, offset)

	return nil
}

func UpdateMatchSeconds(events []event.EloEvent, offset float64) {
	for _, e := range events {
		e.SetMatchSeconds(math.Max(0, e.Time()+offset))
	}
}

// participantEvent is a generic description of something that happened to a
// particular participant.
type participantEvent struct {
	ParticipantID int64
	MatchSeconds  float64
	EloSeconds    float64
}

func apiDeaths(md *api.MatchDetail) ([]participantEvent, error) {
	deaths := []participantEvent{}
	for _, frame := range md.Timeline.Frames {
		for _, event := range frame.Events {
			if event.EventType == "CHAMPION_KILL" {
				deaths = append(deaths, participantEvent{
					ParticipantID: int64(event.VictimID),
					MatchSeconds:  float64(event.Timestamp) / 1000,
				})
			}
		}
	}
	if len(deaths) == 0 {
		return nil, fmt.Errorf("no deaths found in match details")
	}
	return deaths, nil
}

func eloDeaths(eloEvents []event.EloEvent, nameToPID map[string]int64) ([]participantEvent, error) {
	participantByNetworkID := map[int64]int64{}
	deaths := []participantEvent{}
	for _, e := range eloEvents {
		switch ev := e.(type) {
		case *event.NetworkIDMapping:
			if pID, found := nameToPID[ev.SenderName]; found {
				participantByNetworkID[ev.NetworkID] = pID
			}
		case *event.Die:
			// Only look at "DIE" events with a hero network id
			if _, exists := participantByNetworkID[ev.NetworkID]; !exists {
				continue
			}

			death := participantEvent{
				ParticipantID: participantByNetworkID[ev.NetworkID],
				EloSeconds:    ev.Time(),
			}
			if death.ParticipantID < 1 && death.ParticipantID > 10 {
				return nil, fmt.Errorf("unknown champDie participant (%d) at %.2f (p_id: %d)", death.ParticipantID, ev.Time())
			}
			deaths = append(deaths, death)
		}
	}
	return deaths, nil
}

func determineEloToAPIOffset(apiDeaths, eloDeaths []participantEvent) (float64, error) {
	if err := validateDeathCounts(apiDeaths, eloDeaths); err != nil {
		return 0.0, err
	}

	// Compute the diffs.
	diffs := []float64{}

	for i := range apiDeaths {
		diff := float64(apiDeaths[i].MatchSeconds) - eloDeaths[i].EloSeconds
		diffs = append(diffs, diff)
	}

	// Take the median value.
	sort.Float64Slice(diffs).Sort()
	offset := diffs[len(diffs)/2]

	// Warn us if any of the values are wildly different than the chosen offset.
	for _, diff := range diffs {
		if math.Abs(diff-offset) > maxSecondsDiff {
			log.Warning(fmt.Sprintf("death diff: %.1f, offset: %.1f", diff, offset))
		}
	}
	return offset, nil
}

// TODO(Cameron): This should just use baseview events or should take the netIDs map
func validateDeathCounts(apiDeaths, eloDeaths []participantEvent) error {
	deathCountsByParticipant := map[int64]int{}
	for i := range apiDeaths {
		deathCountsByParticipant[apiDeaths[i].ParticipantID]++
	}
	for i := range eloDeaths {
		deathCountsByParticipant[eloDeaths[i].ParticipantID]--
	}
	for pID, diff := range deathCountsByParticipant {
		if pID > 0 && pID <= 10 && diff != 0 {
			return fmt.Errorf("api and elo differ in deaths for participant %d by %d", pID, diff)
		}
	}
	return nil
}

// ParticipantMapMD builds a list of participants sorted by participant id.
func ParticipantMapMD(events []event.EloEvent, md *api.MatchDetail) ([]baseview.Participant, error) {
	if md == nil || md.IsParticipantIdentitiesEmpty() {
		return []baseview.Participant{}, fmt.Errorf("no participants found")
	}

	byPID := map[int64]baseview.Participant{}
	for _, pi := range md.Participants {
		p := baseview.Participant{
			ChampionID:    int64(pi.ChampionID),
			ParticipantID: int64(pi.ParticipantID),
			Spell1:        riot.SpellNamesByID[int64(pi.Spell1ID)],
			Spell2:        riot.SpellNamesByID[int64(pi.Spell2ID)],
		}
		if p.Spell1 == "" || p.Spell2 == "" {
			log.Warning(fmt.Sprintf("participant %d in match %d (%s) missing spell: %s and %s", pi.ParticipantID, md.MatchID, md.PlatformID, p.Spell1, p.Spell2))
		}
		byPID[int64(pi.ParticipantID)] = p
	}

	for _, pi := range md.ParticipantIdentities {
		p := byPID[int64(pi.ParticipantID)]
		p.SummonerID = pi.Player.SummonerID
		p.SummonerName = pi.Player.SummonerName // Note: may not be present.
		byPID[int64(pi.ParticipantID)] = p
	}

	return sortParticipants(byPID)
}

type participants []baseview.Participant

func (p participants) Len() int           { return len(p) }
func (p participants) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }
func (p participants) Less(i, j int) bool { return p[i].ParticipantID < p[j].ParticipantID }

func sortParticipants(m map[int64]baseview.Participant) ([]baseview.Participant, error) {
	participants := participants{}
	for _, p := range m {
		participants = append(participants, p)
	}
	sort.Sort(participants)
	for i := range participants {
		if participants[i].ParticipantID != int64(i+1) {
			return nil, fmt.Errorf("participant at index %d has participant id %d", i, participants[i].ParticipantID)
		}
	}
	return []baseview.Participant(participants), nil
}
