package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/VantageSports/riot"
)

type MatchListItem struct {
	EndIndex   int              `json:"endIndex"`
	Matches    []MatchReference `json:"matches"`
	StartIndex int              `json:"startIndex"`
	TotalGames int              `json:"totalGames"`
}

type MatchReference struct {
	Champion   int64  `json:"champion"`
	Lane       string `json:"lane"`
	MatchID    int64  `json:"matchId"`
	PlatformID string `json:"platformId"`
	Queue      string `json:"queue"`
	Region     string `json:"region"`
	Role       string `json:"role"`
	Season     string `json:"season"`
	Timestamp  int64  `json:"timestamp"`
}

type MatchListOptions struct {
	championIDs  []string
	rankedQueues []riot.QueueType
	seasons      []string
	beginTime    time.Time
	endTime      time.Time
	beginIndex   int
	endIndex     int
}

func NewMatchListOptions() *MatchListOptions {
	return &MatchListOptions{
		championIDs:  []string{},
		rankedQueues: []riot.QueueType{},
		seasons:      []string{},
		beginIndex:   -1,
		endIndex:     -1,
	}
}

func (m *MatchListOptions) Champions(championIDs ...string) *MatchListOptions {
	m.championIDs = append(m.championIDs, championIDs...)
	return m
}

func (m *MatchListOptions) RankedQueues(queues ...riot.QueueType) *MatchListOptions {
	m.rankedQueues = append(m.rankedQueues, queues...)
	return m
}

func (m *MatchListOptions) Seasons(seasons ...string) *MatchListOptions {
	m.seasons = append(m.seasons, seasons...)
	return m
}

func (m *MatchListOptions) BeginTime(t time.Time) *MatchListOptions {
	m.beginTime = t
	return m
}

func (m *MatchListOptions) EndTime(t time.Time) *MatchListOptions {
	m.endTime = t
	return m
}

func (m *MatchListOptions) BeginIndex(i int) *MatchListOptions {
	m.beginIndex = i
	return m
}

func (m *MatchListOptions) EndIndex(i int) *MatchListOptions {
	m.endIndex = i
	return m
}

func (m *MatchListOptions) pairs() []string {
	res := []string{}
	if len(m.championIDs) > 0 {
		res = append(res, "championIds", strings.Join(m.championIDs, ","))
	}
	if len(m.rankedQueues) > 0 {
		strs := []string{}
		for _, queue := range m.rankedQueues {
			strs = append(strs, string(queue))
		}
		res = append(res, "rankedQueues", strings.Join(strs, ","))
	}
	if len(m.seasons) > 0 {
		res = append(res, "seasons", strings.Join(m.seasons, ","))
	}
	if !m.beginTime.IsZero() {
		res = append(res, "beginTime", fmt.Sprintf("%v", m.beginTime.Unix()*1000))
	}
	if !m.endTime.IsZero() {
		res = append(res, "endTime", fmt.Sprintf("%v", m.endTime.Unix()*1000))
	}
	if m.beginIndex >= 0 {
		res = append(res, "beginIndex", fmt.Sprintf("%d", m.beginIndex))
	}
	if m.endIndex >= 0 {
		res = append(res, "endIndex", fmt.Sprintf("%d", m.endIndex))
	}
	return res
}

// There is a bug in the matchlist endpoint, which sometimes causes all the
// options you specify to be ignored:
// https://developer.riotgames.com/discussion/community-discussion/show/lEfrrBUT
//
// Emperically, only about 1 out of every ~400 or so requests seem to do this,
// but when the bug does rear its head, its pretty bad. We expect like 2 or 3
// matches, and receive like 5000 matchrefs (which obviously slows down the
// crawl.
//
// This function can be used in such circumstances, by verifying that a given
// matchref matches the matchlist options. If not, it should be ignored.
// The match ids crawler uses this by requesting match lists for each summoner,
// and then calling MatchesRef(ref) for each returned ref.
func (m *MatchListOptions) MatchesRef(ref MatchReference) bool {
	if !m.beginTime.IsZero() && ref.Timestamp < (m.beginTime.Unix()*1000) {
		return false
	}
	if !m.endTime.IsZero() && ref.Timestamp > (m.endTime.Unix()*1000) {
		return false
	}
	if len(m.seasons) > 0 {
		found := false
		for _, s := range m.seasons {
			if found = ref.Season == s; found {
				break
			}
		}
		if !found {
			return false
		}
	}
	if len(m.rankedQueues) > 0 {
		found := false
		for _, s := range m.rankedQueues {
			if found = string(s) == ref.Queue; found {
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// MatchList returns a list of matches according to the specified criteria. A
// number of optional parameters are provided for filtering. It is up to the
// caller to ensure that the combination of filter parameters provided is valid
// for the requested summoner, otherwise, no matches may be returned. If either
// of the beginTime or endTime parameters is set, they must both be set,
// although there is no maximum limit on their range. If the beginTime
// parameter is specified on its own, endTime is assumed to be the current
// time. If the endTime parameter is specified on its own, beginTime is assumed
// to be the start of the summoner's match history.
func (a *Api) MatchList(summonerID int64, options *MatchListOptions) (matchList MatchListItem, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v2.2/matchlist/by-summoner/%v",
		a.baseURL(), a.Region(), summonerID)
	if url, err = a.addParams(url, options.pairs()...); err != nil {
		return matchList, err
	}

	err = a.getJSON(url, &matchList)
	return matchList, err
}
