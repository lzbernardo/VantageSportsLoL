package api

import (
	"fmt"

	"github.com/VantageSports/riot"
)

type CurrentGameInfo struct {
	BannedChampions   []BannedChampion         `json:"bannedChampions"`
	GameID            int64                    `json:"gameId"`
	GameLength        int64                    `json:"gameLength"`
	GameMode          string                   `json:"gameMode"`
	GameQueueConfigID int64                    `json:"gameQueueConfigId"`
	GameStartTime     int64                    `json:"gameStartTime"`
	GameType          string                   `json:"gameType"`
	MapID             int64                    `json:"mapId"`
	Observers         Observer                 `json:"observers"`
	Participants      []CurrentGameParticipant `json:"participants"`
	PlatformID        string                   `json:"platformId"`
}

// Returns the GameID as a string (a very common need when making calls to the
// riot API, which always expects Game/MatchID as a string).
func (cgi CurrentGameInfo) GameIDStr() string {
	return fmt.Sprintf("%d", cgi.GameID)
}

type CurrentGameParticipant struct {
	Bot           bool           `json:"bot"`
	ChampionID    int64          `json:"championId"`
	Masteries     []riot.Mastery `json:"masteries"`
	ProfileIconID int64          `json:"profileIconId"`
	Runes         []Rune         `json:"runes"`
	Spell1ID      int64          `json:"spell1Id"`
	Spell2ID      int64          `json:"spell2Id"`
	SummonerID    int64          `json:"summonerId"`
	SummonerName  string         `json:"summonerName"`
	TeamID        int64          `json:"teamId"`
}

type Observer struct {
	EncryptionKey string `json:"encryptionKey"`
}

type CGRune struct {
	Count  int   `json:"count"`
	RuneID int64 `json:"runeId"`
}

func (a *Api) CurrentGame(platformID string, summonerID int64) (gameInfo CurrentGameInfo, err error) {
	url := fmt.Sprintf("%s/observer-mode/rest/consumer/getSpectatorGameInfo/%s/%v",
		a.baseURL(), platformID, summonerID)
	if url, err = a.addParams(url); err != nil {
		return gameInfo, err
	}

	err = a.getJSON(url, &gameInfo)
	return gameInfo, err
}
