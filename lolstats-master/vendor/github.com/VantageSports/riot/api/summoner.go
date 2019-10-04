package api

import (
	"fmt"
	"strings"

	"github.com/VantageSports/riot"
)

func (a *Api) SummonerByName(summonerNames ...string) (summoners map[string]riot.Summoner, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v1.4/summoner/by-name/%s",
		a.baseURL(), a.Region(), strings.Join(summonerNames, ","))
	if url, err = a.addParams(url); err != nil {
		return nil, err
	}

	err = a.getJSON(url, &summoners)
	return summoners, err
}

func (a *Api) Summoner(summonerIDs ...int64) (summoners map[string]riot.Summoner, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v1.4/summoner/%s",
		a.baseURL(), a.Region(), joinIDs(summonerIDs...))
	if url, err = a.addParams(url); err != nil {
		return nil, err
	}

	err = a.getJSON(url, &summoners)
	return summoners, err
}

func (a *Api) Masteries(summonerIDs ...int64) (masteryPages map[string]riot.MasteryPages, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v1.4/summoner/%s/masteries",
		a.baseURL(), a.Region(), joinIDs(summonerIDs...))
	if url, err = a.addParams(url); err != nil {
		return nil, err
	}

	err = a.getJSON(url, &masteryPages)
	return masteryPages, err
}

func (a *Api) Name(summonerIDs ...int64) (names map[string]string, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v1.4/summoner/%s/name",
		a.baseURL(), a.Region(), joinIDs(summonerIDs...))
	if url, err = a.addParams(url); err != nil {
		return nil, err
	}

	err = a.getJSON(url, &names)
	return names, err
}

type RunePages struct {
	pages      []RunePage `json:"runePage"`
	summonerID int64      `json:"summonerId"`
}

type RunePage struct {
	Current bool       `json:"current"`
	ID      int64      `json:"id"`
	Name    string     `json:"name"`
	Slots   []RuneSlot `json:"slots"`
}

type RuneSlot struct {
	RuneID     int `json:"runeId"`
	RuneSlotID int `json:"runeSlotId"`
}

func (a *Api) Runes(summonerIDs ...int64) (runePages map[string]RunePages, err error) {
	url := fmt.Sprintf("%s/api/lol/%v/v1.4/summoner/%s/runes",
		a.baseURL(), a.Region(), joinIDs(summonerIDs...))
	if url, err = a.addParams(url); err != nil {
		return nil, err
	}

	err = a.getJSON(url, &runePages)
	return runePages, err
}
