package gcd

import (
	"time"

	"cloud.google.com/go/datastore"

	"github.com/VantageSports/riot"
)

const KindLolHistoryRow = "LolHistoryRow"

// MatchStatRow contains per-match summoner stats.
type HistoryRow struct {
	// Administrative columns
	Created     time.Time `json:"created" datastore:"created"`
	LastUpdated time.Time `json:"last_updated" datastore:"last_updated"`

	// These are in a composite index for gcd match history call
	SummonerID    int64  `json:"summoner_id" datastore:"summoner_id"`
	Platform      string `json:"platform" datastore:"platform"`
	QueueType     string `json:"queue_type" datastore:"queue_type"`
	MatchCreation int64  `json:"match_creation" datastore:"match_creation"`

	// Dimensional information
	ChampionID         int     `json:"champion_id" datastore:"champion_id,noindex"`
	Lane               string  `json:"lane" datastore:"lane,noindex"`
	ChampionIndex      int     `json:"champion_index" datastore:"champion_index,noindex"`
	MapID              int     `json:"map_id" datastore:"map_id,noindex"`
	MatchDuration      float64 `json:"match_duration" datastore:"match_duration,noindex"`
	MatchID            int64   `json:"match_id" datastore:"match_id,noindex"`
	MatchMode          string  `json:"match_mode" datastore:"match_mode,noindex"`
	MatchType          string  `json:"match_type" datastore:"match_type,noindex"`
	MatchVersion       string  `json:"match_version" datastore:"match_version,noindex"`
	OffMeta            bool    `json:"off_meta" datastore:"off_meta,noindex"`
	OpponentChampionID int     `json:"opponent_champion_id" datastore:"opponent_champion_id,noindex"`
	Role               string  `json:"role" datastore:"role,noindex"`
	SummonerName       string  `json:"summoner_name" datastore:"summoner_name,noindex"`
	Won                bool    `json:"winner" datastore:"winner,noindex"`

	// Rank information
	Tier     riot.Tier     `json:"tier" datastore:"tier,noindex"`
	Division riot.Division `json:"division" datastore:"division,noindex"`

	// Metric(al?) information

	Kills   int64 `json:"kills" datastore:"kills,noindex"`
	Deaths  int64 `json:"deaths" datastore:"deaths,noindex"`
	Assists int64 `json:"assists" datastore:"assists,noindex"`
}

func (h *HistoryRow) Save() ([]datastore.Property, error) {
	now := time.Now()
	if h.Created.IsZero() {
		h.Created = now
	}
	h.LastUpdated = now
	return datastore.SaveStruct(h)
}

func (h *HistoryRow) Load(p []datastore.Property) error {
	return datastore.LoadStruct(h, p)
}
