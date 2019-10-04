// Translation functions from service types to internal types

package gcd

import (
	"encoding/json"

	"github.com/VantageSports/lolstats"
)

func FromMatchSummary(stats *lolstats.MatchSummary) (*HistoryRow, error) {
	statsRow := &HistoryRow{}
	if err := remarshalAs(stats, statsRow); err != nil {
		return nil, err
	}
	return statsRow, nil
}

func (r *HistoryRow) ToMatchSummary() (*lolstats.MatchSummary, error) {
	ms := &lolstats.MatchSummary{}
	if err := remarshalAs(r, ms); err != nil {
		return nil, err
	}
	return ms, nil
}

func remarshalAs(from, to interface{}) error {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, to)
}
