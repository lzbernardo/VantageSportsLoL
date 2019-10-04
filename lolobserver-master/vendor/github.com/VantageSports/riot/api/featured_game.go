package api

type FeaturedGames struct {
	ClientRefreshInterval int64              `json:"clientRefreshInterval"`
	GameList              []FeaturedGameInfo `json:"gameList"`
}

type FeaturedGameInfo struct {
	BannedChampions   []BannedChampion `json:"bannedChampions"`
	GameID            int64            `json:"gameId"`
	GameLength        int64            `json:"gameLength"`
	GameMode          string           `json:"gameMode"`
	GameQueueConfigID int64            `json:"gameQueueConfigId"`
	GameStartTime     int64            `json:"gameStartTime"`
	GameType          string           `json:"gameType"`
	MapID             int64            `json:"mapId"`
	Observers         Observer         `json:"observers"`
	Participants      []FGParticipant  `json:"participants"`
	PlatformID        string           `json:"platformId"`
}

type FGParticipant struct {
	Bot           bool   `json:"bot"`
	ChampionID    int64  `json:"championId"`
	ProfileIconID int64  `json:"profileIconId"`
	Spell1ID      int64  `json:"spell1Id"`
	Spell2ID      int64  `json:"spell2Id"`
	SummonerName  string `json:"summonerName"`
	TeamID        int64  `json:"teamId"`
}

func (a *Api) FeaturedGames() (games FeaturedGames, err error) {
	url, err := a.addParams(a.baseURL() + "/observer-mode/rest/featured")
	if err != nil {
		return games, err
	}

	err = a.getJSON(url, &games)
	return games, err
}
