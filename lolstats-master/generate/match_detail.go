package generate

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"time"

	"github.com/VantageSports/lolstats/gcd"
	"github.com/VantageSports/riot/api"
)

func ComputeHistory(m *api.MatchDetail, summonerID int64) (*gcd.HistoryRow, error) {
	player, participant, err := playerAndParticipant(m, summonerID)
	if err != nil {
		// No player found. This should have been handled by the match downloader.
		return nil, err
	}

	row := &gcd.HistoryRow{
		MapID:         m.MapID,
		MatchCreation: m.MatchCreation,
		MatchDuration: float64(m.MatchDuration),
		MatchID:       m.MatchID,
		MatchMode:     m.MatchMode,
		MatchType:     m.MatchType,
		MatchVersion:  m.MatchVersion,
		Platform:      m.PlatformID,
		QueueType:     m.QueueType,
		SummonerID:    player.SummonerID,
		SummonerName:  player.SummonerName,

		ChampionID:    participant.ChampionID,
		ChampionIndex: participant.ParticipantID,
		Kills:         participant.Stats.Kills,
		Deaths:        participant.Stats.Deaths,
		Assists:       participant.Stats.Assists,
		Won:           participant.Stats.Winner,
		Lane:          participant.Timeline.Lane,
		Role:          participant.Timeline.Role,
	}

	ourLane, theirLane := computeLaneMatchup(*participant, m.Participants)
	row.OffMeta = len(ourLane) != 1 || len(theirLane) != 1
	if !row.OffMeta {
		row.OpponentChampionID = theirLane[0].ChampionID
	}

	return row, nil
}

func ComputeBasic(m *api.MatchDetail, summonerID int64) (*BasicStats, error) {
	player, participant, err := playerAndParticipant(m, summonerID)
	if err != nil {
		// No player found. This should have been handled by the match downloader.
		return nil, err
	}

	stats := &BasicStats{
		ChampionID:    participant.ChampionID,
		ChampionIndex: participant.ParticipantID,
		Won:           participant.Stats.Winner,
		Lane:          participant.Timeline.Lane,
		MatchCreation: m.MatchCreation,
		MatchDuration: m.MatchDuration,
		MatchID:       m.MatchID,
		MatchVersion:  m.MatchVersion,
		PlatformID:    m.PlatformID,
		QueueType:     m.QueueType,
		Role:          participant.Timeline.Role,
		SummonerID:    player.SummonerID,
		LastUpdated:   time.Now(),
	}
	ourLane, theirLane := computeLaneMatchup(*participant, m.Participants)
	if len(ourLane) == 1 && len(theirLane) == 1 {
		stats.OpponentChampionID = theirLane[0].ChampionID
	}

	stats.TeamSummaries, err = ComputeTeamSummaries(m)
	if err != nil {
		return nil, err
	}

	addParticipantStats(stats, *participant, m.MatchDuration)
	addLaneDiffStats(stats, ourLane, theirLane)
	addLevelStats(stats, m, *participant, ourLane, theirLane)
	addDeathTimePercentage(stats, m, participant)

	return stats, nil
}

// playerAndParticipant returns the player and participant with the given summoner id.
func playerAndParticipant(m *api.MatchDetail, summonerID int64) (*api.Player, *api.Participant, error) {
	ident, err := participantIdentity(m, summonerID)
	if err != nil {
		return nil, nil, err
	}

	pID := ident.ParticipantID
	for _, p := range m.Participants {
		if p.ParticipantID == pID {
			return &ident.Player, &p, nil
		}
	}
	return nil, nil, fmt.Errorf("no participant found for id: %s", summonerID)
}

// participantIdentity looks for a matching ParticipantIdentity object in the matchDetail.
func participantIdentity(m *api.MatchDetail, summonerID int64) (*api.ParticipantIdentity, error) {
	for i := range m.ParticipantIdentities {
		pID := m.ParticipantIdentities[i]
		if pID.Player.SummonerID == summonerID {
			return &pID, nil
		}
	}

	return nil, fmt.Errorf("unable to find participant identity for id: %s", summonerID)
}

// computeLaneMatchup returns the participant's "partners" and "opponents" for
// the participant's lane. In a multi-participant lane, we only match
// participants with single-opponents if it is absolutely clear who the
// lane-opponent is (e.g. DUO_CARRY vs DUO_CARRY and DUO_SUPPORT vs
// DUO_SUPPORT). Otherwise, we group all the participants in the lane by team
// and expect that stats compute diffs by averaging all participans per team.
func computeLaneMatchup(p api.Participant, all []api.Participant) ([]api.Participant, []api.Participant) {
	ourLane, theirLane, perfectMatch := []api.Participant{}, []api.Participant{}, []api.Participant{}

	for _, candidate := range all {
		if candidate.Timeline.Lane != p.Timeline.Lane {
			continue
		}
		if candidate.TeamID == p.TeamID {
			ourLane = append(ourLane, candidate)
		} else {
			theirLane = append(theirLane, candidate)
			if candidate.Timeline.Role == p.Timeline.Role {
				perfectMatch = append(perfectMatch, candidate)
			}
		}
	}

	// NOTE: we're relying a bit on the current meta with this check, assuming
	// that if there is one perfect match in the lane, all matches will be
	// perfect. If that doesn't hold, we're going to misclassify some off-meta
	// games as on-meta.
	if len(perfectMatch) == 1 && len(ourLane) == len(theirLane) {
		return []api.Participant{p}, perfectMatch
	}
	return ourLane, theirLane
}

// addParticipantStats adds all the stats that can be derived from a single
// participant object, including some of the solo timeline stats.
func addParticipantStats(stats *BasicStats, p api.Participant, matchSeconds int64) {
	matchMinutes := float64(matchSeconds) / 60.0

	stats.Assists = p.Stats.Assists
	stats.CSPerMinuteZeroToTen = p.Timeline.CreepsPerMinDeltas.ZeroToTen
	stats.Deaths = p.Stats.Deaths
	stats.GoldTenToTwenty = p.Timeline.GoldPerMinDeltas.TenToTwenty
	stats.GoldTwentyToThirty = p.Timeline.GoldPerMinDeltas.TwentyToThirty
	stats.GoldZeroToTen = p.Timeline.GoldPerMinDeltas.ZeroToTen
	stats.Kills = p.Stats.Kills
	stats.NeutralMinionsKilledEnemyJunglePerMinute = float64(p.Stats.NeutralMinionsKilledEnemyJungle) / matchMinutes
	stats.NeutralMinionsKilledPerMinute = float64(p.Stats.NeutralMinionsKilled) / matchMinutes
	stats.TotalDamageDealt = p.Stats.TotalDamageDealt
	stats.TotalDamageDealtPerMinute = float64(p.Stats.TotalDamageDealt) / matchMinutes
	stats.TotalDamageDealtToChampions = p.Stats.TotalDamageDealtToChampions
	stats.TotalDamageDealtToChampionsPerMinute = float64(p.Stats.TotalDamageDealtToChampions) / matchMinutes
	stats.TotalDamageTaken = p.Stats.TotalDamageTaken
	stats.TotalDamageTakenPerMinute = float64(p.Stats.TotalDamageTaken) / matchMinutes
	stats.TotalHeal = p.Stats.TotalHeal
	stats.TotalHealPerMinute = float64(p.Stats.TotalHeal) / matchMinutes
	stats.WardsPlacedPerMinute = float64(p.Stats.WardsPlaced) / matchMinutes
	stats.WardsKilledPerMinute = float64(p.Stats.WardsKilled) / matchMinutes
	stats.WardsPlaced = p.Stats.WardsPlaced
}

// addLaneDiffStats computes the lane differential stats for the lane occupied
// by the participant.
func addLaneDiffStats(row *BasicStats, us, them []api.Participant) {
	// If we can't figure out the lane opponent, then leave the diff numbers as null.
	// This will exclude them from percentiles
	if len(them) == 0 {
		return
	}

	numUs := math.Max(1.0, float64(len(us)))
	numThem := math.Max(1.0, float64(len(them)))
	usTimeline := combineParticipantTimelines(us...)
	themTimeline := combineParticipantTimelines(them...)

	csDiff := float64(usTimeline.CreepsPerMinDeltas.ZeroToTen)/numUs - float64(themTimeline.CreepsPerMinDeltas.ZeroToTen)/numThem
	goldDiffZeroToTen := float64(usTimeline.GoldPerMinDeltas.ZeroToTen)/numUs - float64(themTimeline.GoldPerMinDeltas.ZeroToTen)/numThem
	goldDiffTenToTwenty := float64(usTimeline.GoldPerMinDeltas.TenToTwenty)/numUs - float64(themTimeline.GoldPerMinDeltas.TenToTwenty)/numThem
	goldDiffTwentyToThirty := float64(usTimeline.GoldPerMinDeltas.TwentyToThirty)/numUs - float64(themTimeline.GoldPerMinDeltas.TwentyToThirty)/numThem

	row.CSPerMinuteDiffZeroToTen = &csDiff
	row.GoldDiffZeroToTen = &goldDiffZeroToTen
	row.GoldDiffTenToTwenty = &goldDiffTenToTwenty
	row.GoldDiffTwentyToThirty = &goldDiffTwentyToThirty
}

// combineParticipantTimelines returns a new participant timeline containing the
// sums of all values of each participants timeline data.
func combineParticipantTimelines(all ...api.Participant) api.ParticipantTimeline {
	combined := api.ParticipantTimeline{}
	for _, a := range all {
		addToTimeline(&combined, a.Timeline)
	}
	return combined
}

// addToTimeline uses reflection to find all the fields of type
// ParticipantTimelineData in src and copies each of the timeline data's values
// over to the equivalent location in dest. Because there are over 20 different
// timeline data object in each timeline, using reflection saves lots of l.o.c.
func addToTimeline(dest *api.ParticipantTimeline, src api.ParticipantTimeline) {
	// In order to see if fields within the timeline are timelinedata structs,
	// we find the type of the first one (thouch could be any of them) to
	// compare with the others.
	tlDataType := reflect.TypeOf(dest.AncientGolemAssistsPerMinCounts)

	destVal := reflect.ValueOf(dest).Elem()
	srcVal := reflect.ValueOf(src)

	// For each field in the Timeline object, add the fields subfields
	// (zeroToTen, tenToTwenty, etc) to the equivalent subfield in dest.
	for i := 0; i < srcVal.NumField(); i++ {
		destField := destVal.Field(i)
		if destField.Type() != tlDataType {
			continue
		}
		srcField := srcVal.Field(i)

		operands := []api.ParticipantTimelineData{
			destField.Interface().(api.ParticipantTimelineData),
			srcField.Interface().(api.ParticipantTimelineData),
		}
		sum := api.ParticipantTimelineData{}
		for _, p := range operands {
			sum.ZeroToTen += p.ZeroToTen
			sum.TenToTwenty += p.TenToTwenty
			sum.TwentyToThirty += p.TwentyToThirty
			sum.ThirtyToEnd += p.ThirtyToEnd
		}

		destField.Set(reflect.ValueOf(sum))
	}
}

// addSkillStats adds the skill level up stats.
func addLevelStats(row *BasicStats, m *api.MatchDetail, p api.Participant, us, them []api.Participant) {
	level6ByParticipant := map[int]api.Event{}
	curLevelByParticipant := map[int]int{}

	for _, frame := range m.Timeline.Frames {
		for _, e := range frame.Events {
			if e.EventType != "SKILL_LEVEL_UP" || e.LevelUpType != "NORMAL" {
				continue
			}
			prevChampLevel := curLevelByParticipant[e.ParticipantID]
			if prevChampLevel == 5 {
				level6ByParticipant[e.ParticipantID] = e
			}
			curLevelByParticipant[e.ParticipantID]++
		}
	}

	matchSecs := float64(m.MatchDuration)
	row.Level6Seconds = avgEventSeconds(matchSecs, level6ByParticipant, p)

	if len(them) != 0 {
		usUltSecs := avgEventSeconds(matchSecs, level6ByParticipant, us...)
		themUltSecs := avgEventSeconds(matchSecs, level6ByParticipant, them...)
		ultDiff := themUltSecs - usUltSecs
		row.Level6DiffSeconds = &ultDiff
	}
}

// avgEventSeconds returns the average timestamp (in seconds) of the event
// in the map for the specified participants. If no event is present in the map
// for the participant, the defaultSeconds value is used instead.
func avgEventSeconds(defaultSeconds float64, eventsByParticipant map[int]api.Event, participants ...api.Participant) float64 {
	if len(participants) == 0 {
		return defaultSeconds
	}
	totalSecs := 0.0
	for _, p := range participants {
		if e, found := eventsByParticipant[p.ParticipantID]; found {
			totalSecs += float64(e.Timestamp) / 1000.0
		} else {
			totalSecs += defaultSeconds
		}
	}
	return totalSecs / float64(len(participants))
}

// Create a history of percentage of time spent dead in a sliding window
func addDeathTimePercentage(row *BasicStats, m *api.MatchDetail, p *api.Participant) {
	deathSpawnTimes := []float64{}

	for _, frame := range m.Timeline.Frames {
		for _, event := range frame.Events {
			// Add the start death time and the end death time to the deathSpawnTimes array.
			// The end death time is a function of game time and champion level.
			// The champion level is an approximation because we only get snapshots of their level every minute
			if event.EventType == "CHAMPION_KILL" && event.VictimID == p.ParticipantID {
				deathSeconds := float64(event.Timestamp) / 1000.0
				level := frame.ParticipantFrames[strconv.Itoa(p.ParticipantID)].Level
				deathSpawnTimes = append(deathSpawnTimes, deathSeconds, deathEndSeconds(deathSeconds, level))
			}
		}
	}

	sumDeathPercentage := 0.0
	numDataPoints := 0.0

	// Data point once every 60 seconds.
	// For each data point, look back 5 minutes
	interval := 60.0
	lookback := 300.0
	for end := interval; end <= float64(m.MatchDuration); end += interval {
		start := math.Max(0, float64(end-lookback))
		deathDuration := 0.0
		for j := 0; j < len(deathSpawnTimes); j += 2 {
			if deathSpawnTimes[j] >= end {
				break
			}
			if deathSpawnTimes[j] < end && deathSpawnTimes[j+1] > start {
				// Add the overlapping interval amount to the total for this interval
				deathDuration += math.Min(end, deathSpawnTimes[j+1]) - math.Max(start, deathSpawnTimes[j])
			}
		}
		row.DeadPercentLast5 = append(row.DeadPercentLast5, TimePercent{
			Time:    end,
			Percent: 100 * deathDuration / lookback,
		})

		sumDeathPercentage += 100 * deathDuration / lookback
		numDataPoints++
	}
	row.AverageDeathTimePercentage = sumDeathPercentage / numDataPoints
}

// Based off of http://leagueoflegends.wikia.com/wiki/Death
// Doesn't currently count quintessence of revivals, but that should be extremely rare
func deathEndSeconds(seconds float64, level int) float64 {
	base := (float64(level) * 2.5) + 7.5
	minutes := seconds / 60.0

	endTime := base

	if minutes > 15 {
		endTime += ((base / 100.0) * (minutes - 15) * 2 * 0.375)
	}
	if minutes > 30 {
		endTime += ((base / 100) * (minutes - 30) * 2 * 0.2)
	}
	if minutes > 45 {
		endTime += ((base / 100) * (minutes - 45) * 2 * 1.45)
	}
	if minutes > 53.5 {
		endTime = base * 1.5
	}
	return seconds + endTime
}
