package scrape

import (
	"testing"

	"gopkg.in/tylerb/is.v1"
)

var testPickBans []PickBan

func init() {
	testPickBans = []PickBan{
		PickBan{
			Season:    "Test Season 1",
			Round:     "Test Week 1",
			Team1:     "Test Team 1",
			Team2:     "Test Team 2",
			Team1Won:  false,
			Team1Bans: []string{"Poppy", "Alistar", "Fiora"},
			Team2Bans: []string{"Gangplank", "Tahm Kench", "Ryze"},
			Team1Picks: []Pick{
				Pick{Champion: "Graves", Position: "Top Lane"},
				Pick{Champion: "Elise", Position: "Jungle"},
				Pick{Champion: "Braum", Position: "Support"},
				Pick{Champion: "Lucian", Position: "AD Carry"},
				Pick{Champion: "Lissandra", Position: "Mid Lane"},
			},
			Team2Picks: []Pick{
				Pick{Champion: "Lulu", Position: "Top Lane"},
				Pick{Champion: "Corki", Position: "Mid Lane"},
				Pick{Champion: "Thresh", Position: "Support"},
				Pick{Champion: "Kalista", Position: "AD Carry"},
				Pick{Champion: "Gragas", Position: "Jungle"},
			},
		},
		PickBan{
			Season:    "Test Season 1",
			Round:     "Test Week 2",
			Team1:     "Test Team 1",
			Team2:     "Test Team 3",
			Team1Won:  false,
			Team1Bans: []string{"Graves", "Corki", "Alistar"},
			Team2Bans: []string{"Ryze", "Lulu", "Poppy"},
			Team1Picks: []Pick{
				Pick{Champion: "Gangplank", Position: "Top Lane"},
				Pick{Champion: "Elise", Position: "Jungle"},
				Pick{Champion: "Nautilus", Position: "Support"},
				Pick{Champion: "Kalista", Position: "AD Carry"},
				Pick{Champion: "Ahri", Position: "Mid Lane"},
			},
			Team2Picks: []Pick{
				Pick{Champion: "Fiora", Position: "Top Lane"},
				Pick{Champion: "LeBlanc", Position: "Mid Lane"},
				Pick{Champion: "Janna", Position: "Support"},
				Pick{Champion: "Ezreal", Position: "AD Carry"},
				Pick{Champion: "Nidalee", Position: "Jungle"},
			},
		},
	}
}

func TestGetDraftDates(t *testing.T) {
	is := is.New(t)

	dates := getDraftDates(testPickBans)
	is.NotNil(dates)
	is.Equal(len(dates), 2)
	is.Equal(dates[0].Season, "Test Season 1")
}

func TestGetTeamGameCounts(t *testing.T) {
	is := is.New(t)

	counts := getTeamGameCounts(testPickBans)
	is.NotNil(counts)
	is.Equal(len(counts), 3)

	team1Count := 0
	for _, c := range counts {
		if c.TeamName == "Test Team 1" {
			team1Count = c.GameCount
		}
	}
	is.Equal(team1Count, 2)
}

func TestGetPositionCounts(t *testing.T) {
	is := is.New(t)

	counts := getPositionCounts(testPickBans)
	is.NotNil(counts)

	is.Equal(counts["Alistar"].Bans, 2)
	is.Equal(counts["Braum"].Bans, 0)

	is.Equal(counts["Corki"].Picks, 1)
	is.Equal(counts["Ryze"].Picks, 0)

	is.Equal(counts["Kalista"].PositionCounts["AD Carry"], 2)
}
