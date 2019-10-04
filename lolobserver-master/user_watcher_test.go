package lolobserver

import (
	"testing"

	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot/api"
)

func TestLolUsersInGame(t *testing.T) {
	lolUsers := map[string][]*lolusers.LolUser{
		"123": []*lolusers.LolUser{
			{},
			{},
		},
		"345": []*lolusers.LolUser{
			{},
		},
	}

	participants := []api.CurrentGameParticipant{
		{SummonerID: 900, SummonerName: "nil", Bot: true},
		{SummonerID: 940, SummonerName: "not", Bot: false},
		{SummonerID: 960, SummonerName: "nope", Bot: false},
		{SummonerID: 345, SummonerName: "Weee", Bot: false},
		{SummonerID: 998, SummonerName: "boo", Bot: false},
		{SummonerID: 921, SummonerName: "no", Bot: false},
		{SummonerID: 123, SummonerName: "Yay", Bot: false},
	}

	users := lolUsersInGame(participants, lolUsers)
	if len(users) != 3 {
		t.Errorf("expected 3 lolusers in the game, found %d", len(users))
	}
}
