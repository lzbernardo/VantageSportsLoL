package api

import (
	"fmt"

	"github.com/VantageSports/riot"
)

func (a *Api) Champions(dataById bool) (champMeta riot.ChampMeta, err error) {
	url := fmt.Sprintf("%s/api/lol/static-data/%v/v1.2/champion", a.globalURL(), a.Region())
	if url, err = a.addParams(url, "dataById", fmt.Sprintf("%v", dataById)); err != nil {
		return champMeta, err
	}
	if url, err = a.addParams(url, "champData", "all"); err != nil {
		return champMeta, err
	}
	err = a.getJSON(url, &champMeta)
	return champMeta, err
}

func (a *Api) SummonerSpells(dataById bool) (spellList riot.SummonerSpellList, err error) {
	url := fmt.Sprintf("%s/api/lol/static-data/%v/v1.2/summoner-spell", a.globalURL(), a.Region())
	if url, err = a.addParams(url, "dataById", fmt.Sprintf("%v", dataById)); err != nil {
		return spellList, err
	}
	if url, err = a.addParams(url, "spellData", "all"); err != nil {
		return spellList, err
	}
	err = a.getJSON(url, &spellList)
	return spellList, err
}
