package lolqueue

import "testing"

func TestRankCompare(t *testing.T) {
	cases := []struct {
		A          Rank
		B          Rank
		IsAGreater bool
	}{
		{r("Silver", "ii"), r("Silver", "iii"), true},
		{r("Silver", "iii"), r("Silver", "ii"), false},
		{r("Silver", "iii"), r("bronze", "i"), true},
		{r("Silver", "iii"), r("platinum", "v"), false},
		{r("Silver", "ii"), r("Silver", "ii"), false},
		{r("challenger", "i"), r("master", "i"), true},
	}

	for i, c := range cases {
		gt := c.A.Val() > c.B.Val()
		if gt != c.IsAGreater {
			t.Errorf("case %d, found unexpected rank comparison", i+1)
		}
	}
}

func TestResolvePositions(t *testing.T) {
	positions := resolvePositions([]position{}, [][]position{
		{any, Jungle},
		{Jungle, any},
		{Top, any},
		{any},
		{Jungle},
	})
	if len(positions) != 5 {
		t.Error("expected any, any, (any|top), any, jungle. got:", positions)
	}

	positions = resolvePositions([]position{}, [][]position{
		{Top, Jungle},
		{Top, Jungle},
	})
	if len(positions) != 2 {
		t.Error("expected top, jungle. got:", positions)
	}

	positions = resolvePositions([]position{}, [][]position{
		{Top, Jungle},
		{Top, Jungle},
		{Top, Jungle},
	})
	if len(positions) > 0 {
		t.Error("cannot assign 3 players to 2 positions. got:", positions)
	}

	positions = resolvePositions([]position{}, [][]position{
		{Top, Jungle},
		{Top, Jungle},
		{Top, Jungle, any},
	})
	if len(positions) != 3 {
		t.Error("expected top, jungle any. got:", positions)
	}

	positions = resolvePositions([]position{}, [][]position{
		{Top, Jungle},
		{Top, Jungle},
		{Top, Jungle, any},
	})
	if len(positions) != 3 {
		t.Error("expected top, jungle, any. got: ", positions)
	}
}

func TestFindMatch(t *testing.T) {
	p1 := &Player{
		SummonerName: "p1",
		Region:       "na",
		Rank:         r("Gold", "iii"),
		Criteria: Criteria{
			MaxRank:     r("gold", "i"),
			MinRank:     r("silver", "iii"),
			MinPlayers:  3,
			Region:      "na",
			MyPositions: []position{Top, Jungle},
		},
	}

	p2 := &Player{
		SummonerName: "p2",
		Region:       "na",
		Rank:         r("silver", "iii"),
		Criteria: Criteria{
			MaxRank:     r("challenger", "i"),
			MinRank:     r("bronze", "v"),
			MinPlayers:  2,
			Region:      "na",
			MyPositions: []position{Top, Jungle},
		},
	}

	p3 := &Player{
		// too position-restricted
		SummonerName: "p3",
		Region:       "na",
		Rank:         r("Gold", "ii"),
		Criteria: Criteria{
			MaxRank:     r("gold", "ii"),
			MinRank:     r("silver", "ii"),
			MinPlayers:  2,
			Region:      "na",
			MyPositions: []position{Jungle},
		},
	}

	p4 := &Player{
		// too low ranked
		SummonerName: "p4",
		Region:       "na",
		Rank:         r("Bronze", "v"),
		Criteria: Criteria{
			MaxRank:     r("gold", "ii"),
			MinRank:     r("silver", "ii"),
			MinPlayers:  2,
			Region:      "na",
			MyPositions: []position{Top, Jungle, Support},
		},
	}

	p5 := &Player{
		SummonerName: "p5",
		Region:       "na",
		Rank:         r("Silver", "ii"),
		Criteria: Criteria{
			MaxRank:     r("gold", "ii"),
			MinRank:     r("silver", "ii"),
			MinPlayers:  2,
			Region:      "na",
			MyPositions: []position{Top, Jungle, Support},
		},
	}

	p6 := &Player{
		SummonerName: "p6",
		Region:       "na",
		Rank:         r("Silver", "ii"),
		Criteria: Criteria{
			MaxRank:     r("gold", "ii"),
			MinRank:     r("silver", "ii"),
			MinPlayers:  2,
			Region:      "na",
			MyPositions: []position{any},
		},
	}

	var match *Match
	if match = Make(p1); match != nil {
		t.Error("match found even though only 1 player specified.")
	}

	if match = Make(p1, p2); match != nil {
		t.Error("match found even though p1 requires 3 players")
	}

	if match = Make(p1, p2, p3, p4); match != nil {
		t.Error("match should not be made since p3 cannot be jungler and p4 is ranked too low")
	}

	if match = Make(p1, p2, p3, p4, p5); match == nil {
		t.Error("expected to find match for players [p1, p2, p5]")
	}
	if len(match.Invited) != 3 {
		t.Errorf("expected 3 invitees to match. found %d", len(match.Invited))
	}

	// Now verify 'the more the merrier'
	if match = Make(p1, p2, p3, p4, p5, p6); match == nil {
		t.Error("expected to find match for players [p1, p2, p5, p6]")
	}
	if len(match.Invited) != 4 {
		t.Errorf("expected 4 invitees to match. found %d", len(match.Invited))
	}

}

func r(tier, division string) Rank {
	return Rank{Tier: tier, Division: division}
}
