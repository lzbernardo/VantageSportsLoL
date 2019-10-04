package api

import (
	"fmt"

	"github.com/VantageSports/riot"
)

// LeaguesBySummoner returns all leagues for specified summoners and summoners'
// teams. Entries for each requested participant (i.e., each summoner and
// related teams) will be included in the returned leagues data, whether or not
// the participant is inactive. However, no entries for other inactive summoners
// or teams in the leagues will be included.
func (a *Api) LeagueBySummoner(summonerIDs ...int64) (leagues map[string][]riot.League, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v2.5/league/by-summoner/%v",
		a.baseURL(), a.Region(), joinIDs(summonerIDs...))
	if url, err = a.addParams(url); err != nil {
		return leagues, err
	}

	leagues = map[string][]riot.League{}
	err = a.getJSON(url, &leagues)
	return leagues, err
}

func (a *Api) LeagueEntryBySummoner(summonerIDs ...int64) (leagues map[string][]riot.League, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v2.5/league/by-summoner/%v/entry",
		a.baseURL(), a.Region(), joinIDs(summonerIDs...))
	if url, err = a.addParams(url); err != nil {
		return leagues, err
	}

	leagues = map[string][]riot.League{}
	err = a.getJSON(url, &leagues)
	return leagues, err
}

// LeagueChallenger returns the league information for the Challenger tier
func (a *Api) LeagueChallenger(queueType riot.QueueType) (league riot.League, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v2.5/league/challenger",
		a.baseURL(), a.Region())
	if url, err = a.addParams(url, "type", string(queueType)); err != nil {
		return league, err
	}

	err = a.getJSON(url, &league)
	return
}

// LeagueMaster returns the league information for the Master tier
func (a *Api) LeagueMaster(queueType riot.QueueType) (league riot.League, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v2.5/league/master",
		a.baseURL(), a.Region())
	if url, err = a.addParams(url, "type", string(queueType)); err != nil {
		return league, err
	}

	err = a.getJSON(url, &league)
	return
}
