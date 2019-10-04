package scrape

import (
	"log"
)

type DraftSummary struct {
	Positions []string          `json:"positions"`
	Dates     []DraftDate       `json:"dates"`
	Teams     []DraftTeam       `json:"teams"`
	Champions []ChampionPickBan `json:"champions"`
}

type DraftDate struct {
	ID     int    `json:"i"`
	Season string `json:"season"`
	Round  string `json:"round"`
}

type DraftTeam struct {
	TeamName  string `json:"team"`
	GameCount int    `json:"games"`
}

type ChampionPickBan struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Shortname      string         `json:"shortname"`
	Picks          float64        `json:"picks"`
	Bans           float64        `json:"bans"`
	PickBanRate    float64        `json:"pbr"`
	PositionCounts map[string]int `json:"positioncounts"`
}

func GetDraftSummary(pbs []PickBan, metas *ChampionsMeta) (*DraftSummary, error) {
	ds := &DraftSummary{
		Positions: []string{"Top Lane", "AD Carry", "Mid Lane", "Support", "Jungle"},
	}

	ds.Dates = getDraftDates(pbs)
	ds.Teams = getTeamGameCounts(pbs)

	pbrs := getPickBanRates(pbs)
	pickBanRates, err := mergeMetaDataForChampions(pbrs, metas)
	if err != nil {
		return nil, err
	}

	ds.Champions = pickBanRates

	return ds, nil
}

func getDraftDates(pbs []PickBan) []DraftDate {
	dates := map[string]map[string]bool{}

	for _, pb := range pbs {
		season, ok := dates[pb.Season]
		if !ok {
			season = map[string]bool{}
		}

		season[pb.Round] = true
		dates[pb.Season] = season
	}

	i := 0
	draftDates := []DraftDate{}
	for season, roundMap := range dates {
		for round := range roundMap {
			draftDates = append(draftDates, DraftDate{
				ID:     i,
				Season: season,
				Round:  round,
			})
			i++
		}
	}

	return draftDates
}

func getTeamGameCounts(pbs []PickBan) []DraftTeam {
	draftTeams := map[string]DraftTeam{}

	for _, pb := range pbs {
		dt1, ok := draftTeams[pb.Team1]
		if !ok {
			dt1 = DraftTeam{
				TeamName:  pb.Team1,
				GameCount: 0,
			}
		}
		dt1.GameCount++
		draftTeams[pb.Team1] = dt1

		dt2, ok := draftTeams[pb.Team2]
		if !ok {
			dt2 = DraftTeam{
				TeamName:  pb.Team2,
				GameCount: 0,
			}
		}
		dt2.GameCount++
		draftTeams[pb.Team2] = dt2
	}

	dts := []DraftTeam{}
	for _, dt := range draftTeams {
		dts = append(dts, dt)
	}

	return dts
}

func getPositionCounts(pbs []PickBan) map[string]ChampionPickBan {
	pbds := map[string]ChampionPickBan{}
	for _, pb := range pbs {
		for _, ban := range pb.Team1Bans {
			champPickBan, ok := pbds[ban]
			if !ok {
				pbds[ban] = ChampionPickBan{}
			}
			champPickBan.Bans++
			pbds[ban] = champPickBan
		}

		for _, ban := range pb.Team2Bans {
			champPickBan, ok := pbds[ban]
			if !ok {
				pbds[ban] = ChampionPickBan{}
			}
			champPickBan.Bans++
			pbds[ban] = champPickBan
		}

		for _, pick := range pb.Team1Picks {
			champPickBan, ok := pbds[pick.Champion]
			if !ok {
				champPickBan = ChampionPickBan{}

			}
			_, ok = champPickBan.PositionCounts[pick.Position]
			if !ok {
				champPickBan.PositionCounts = map[string]int{
					pick.Position: 0,
				}
			}
			champPickBan.PositionCounts[pick.Position]++
			champPickBan.Picks++
			pbds[pick.Champion] = champPickBan
		}

		for _, pick := range pb.Team2Picks {
			champPickBan, ok := pbds[pick.Champion]
			if !ok {
				champPickBan = ChampionPickBan{}

			}
			_, ok = champPickBan.PositionCounts[pick.Position]
			if !ok {
				champPickBan.PositionCounts = map[string]int{
					pick.Position: 0,
				}
			}
			champPickBan.PositionCounts[pick.Position]++
			champPickBan.Picks++
			pbds[pick.Champion] = champPickBan
		}
	}
	return pbds
}

func getPickBanRates(pbs []PickBan) []ChampionPickBan {
	champPickBans := getPositionCounts(pbs)
	totalGames := float64(len(pbs))

	cpbs := []ChampionPickBan{}
	for champName, champPickBan := range champPickBans {
		champPickBan.Name = champName
		champPickBan.PickBanRate = (champPickBan.Picks + champPickBan.Bans) / totalGames

		cpbs = append(cpbs, champPickBan)
	}

	return cpbs
}

func mergeMetaDataForChampions(champs []ChampionPickBan, metas *ChampionsMeta) ([]ChampionPickBan, error) {

	mergedChamps := []ChampionPickBan{}
	for _, champ := range champs {
		mergedChamp := champ

		champMeta, ok := metas.Data[getChampNameMetaKey(champ.Name)]
		if !ok {
			log.Printf("metadata not found for champion: %s", champ.Name)
		} else {
			mergedChamp.ID = champMeta.Key
			mergedChamp.Shortname = champMeta.ID
		}

		mergedChamps = append(mergedChamps, mergedChamp)
	}

	return mergedChamps, nil
}
