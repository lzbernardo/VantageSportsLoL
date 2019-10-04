package ingest

import (
	"testing"

	"gopkg.in/tylerb/is.v1"
)

func TestAggregate(t *testing.T) {
	is := is.New(t)
	tfa := teamFightAnalysis{ParticipantID: 2}

	unfav := tfa.Aggregate(UnfavorableFights)
	is.Equal(0, unfav.TotalDeaths)
	is.Equal(0, unfav.TotalKills)
	is.Equal(0, unfav.Count)
	is.Equal(0.0, unfav.NetKills)

	tfa.TeamFights = []*TeamFight{
		// Favorable
		&TeamFight{
			Begin:          100.0,
			End:            125.0,
			InitialTarget:  9,
			InfluencerIDs:  JsonLongBoolMap{1: true},
			ParticipantIDs: JsonLongBoolMap{2: true, 4: true, 8: true, 9: true},
			TeamDeaths:     JsonLongBoolMap{},
			TeamKills:      JsonLongBoolMap{8: true, 9: true},
		},
		// Favorable
		&TeamFight{
			Begin:          200.0,
			End:            225.0,
			InitialTarget:  9,
			InfluencerIDs:  JsonLongBoolMap{1: true},
			ParticipantIDs: JsonLongBoolMap{2: true, 9: true},
			TeamDeaths:     JsonLongBoolMap{2: true},
			TeamKills:      JsonLongBoolMap{9: true, 10: true},
		},
		// Unfavorable
		&TeamFight{
			Begin:          300.0,
			End:            325.0,
			InitialTarget:  4,
			InfluencerIDs:  JsonLongBoolMap{6: true},
			ParticipantIDs: JsonLongBoolMap{7: true, 2: true, 8: true, 1: true},
			TeamDeaths:     JsonLongBoolMap{1: true, 2: true},
			TeamKills:      JsonLongBoolMap{7: true},
		},
	}

	unfav = tfa.Aggregate(UnfavorableFights)
	is.Equal(2, unfav.TotalDeaths)
	is.Equal(1, unfav.TotalKills)
	is.Equal(1, unfav.Count)
	is.Equal(-1, unfav.NetKills)

	fav := tfa.Aggregate(FavorableFights)
	is.Equal(1, fav.TotalDeaths)
	is.Equal(4, fav.TotalKills)
	is.Equal(2, fav.Count)
	is.Equal(1.5, fav.NetKills)
}
