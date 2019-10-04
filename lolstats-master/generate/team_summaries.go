package generate

import (
	"strconv"

	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/riot/api"
)

func ComputeTeamSummaries(matchDetails *api.MatchDetail) (*TeamSummaries, error) {
	summaries := &TeamSummaries{
		Team1: &TeamSummary{
			Towers:  int32(matchDetails.Teams[0].TowerKills),
			Barons:  int32(matchDetails.Teams[0].BaronKills),
			Dragons: int32(matchDetails.Teams[0].DragonKills),
		},
		Team2: &TeamSummary{
			Towers:  int32(matchDetails.Teams[1].TowerKills),
			Barons:  int32(matchDetails.Teams[1].BaronKills),
			Dragons: int32(matchDetails.Teams[1].DragonKills),
		},
	}

	for i, participant := range matchDetails.Participants {
		if participant.TeamID == 100 {
			summaries.Team1.Kills += participant.Stats.Kills
			summaries.Team1.Deaths += participant.Stats.Deaths
			summaries.Team1.Assists += participant.Stats.Assists
		} else if participant.TeamID == 200 {
			summaries.Team2.Kills += participant.Stats.Kills
			summaries.Team2.Deaths += participant.Stats.Deaths
			summaries.Team2.Assists += participant.Stats.Assists
		}

		masteryMap := map[string]int64{}
		for _, mastery := range participant.Masteries {
			masteryMap[strconv.FormatInt(mastery.MasteryId, 10)] = mastery.Rank
		}

		p := &ParticipantInfo{
			ChampID:    int32(participant.ChampionID),
			ChampLevel: participant.Stats.ChampLevel,
			Spell1ID:   int32(participant.Spell1ID),
			Spell2ID:   int32(participant.Spell2ID),
			Masteries:  masteryMap,
			Kills:      participant.Stats.Kills,
			Deaths:     participant.Stats.Deaths,
			Assists:    participant.Stats.Assists,
			Gold:       participant.Stats.GoldEarned,
			TotalDamageDealtToChampions: participant.Stats.TotalDamageDealtToChampions,
			WardsPlaced:                 participant.Stats.WardsPlaced,
			WardKills:                   participant.Stats.WardsKilled,
			Item0:                       participant.Stats.Item0,
			Item1:                       participant.Stats.Item1,
			Item2:                       participant.Stats.Item2,
			Item3:                       participant.Stats.Item3,
			Item4:                       participant.Stats.Item4,
			Item5:                       participant.Stats.Item5,
			Item6:                       participant.Stats.Item6,
			Role:                        lolstats.GetMetaRole(participant.Timeline.Lane, participant.Timeline.Role),
			SummonerID:                  matchDetails.ParticipantIdentities[i].Player.SummonerID,
			SummonerName:                matchDetails.ParticipantIdentities[i].Player.SummonerName,
		}
		summaries.Participants = append(summaries.Participants, p)
	}

	return summaries, nil
}
