package api

import "fmt"

type RecentGames struct {
	Games      []Game `json:"games"`
	SummonerID int64  `json:"summonerId"`
}

type Game struct {
	ChampionID    int64        `json:"championId"`
	CreateDate    int64        `json:"createDate"`
	FellowPlayers []GamePlayer `json:"fellowPlayers"`
	GameID        int64        `json:"gameId"`
	GameMode      string       `json:"gameMode"`
	Invalid       bool         `json:"invalid"`
	IPEarned      int64        `json:"ipEarned"`
	Level         int64        `json:"level"`
	MapId         int64        `json:"mapId"`
	Spell1        int64        `json:"spell1"`
	Spell2        int64        `json:"spell2"`
	Stats         RawStats     `json:"stats"`
	SubType       string       `json:"subType"`
	TeamID        int64        `json:"teamId"`
}

type GamePlayer struct {
	ChampionID int64 `json:"championId"`
	SummonerID int64 `json:"summonerId"`
	TeamID     int64 `json:"teamId"`
}

type RawStats struct {
	Assists                         int64 `json:"assists"`
	BountyLevel                     int64 `json:"bountyLevel"`
	ChampionsKilled                 int64 `json:"championsKilled"`
	GoldEarned                      int64 `json:"goldEarned"`
	GoldSpent                       int64 `json:"goldSpent"`
	Item0                           int64 `json:"item0"`
	Item1                           int64 `json:"item1"`
	Item2                           int64 `json:"item2"`
	Item3                           int64 `json:"item3"`
	Item4                           int64 `json:"item4"`
	Item5                           int64 `json:"item5"`
	Item6                           int64 `json:"item6"`
	KillingSprees                   int64 `json:"killingSprees"`
	LargestKillingSpree             int64 `json:"largestKillingSpree"`
	LargestMultiKill                int64 `json:"largestMultiKill"`
	Level                           int64 `json:"level"`
	MagicDamageDealtPlayer          int64 `json:"magicDamageDealtPlayer"`
	MagicDamageDealtToChampions     int64 `json:"magicDamageDealtToChampions"`
	MagicDamageTaken                int64 `json:"magicDamageTaken"`
	MinionsKilled                   int64 `json:"minionsKilled"`
	NeutralMinionsKilled            int64 `json:"neutralMinionsKilled"`
	NeutralMinionsKilledEnemyJungle int64 `json:"neutralMinionsKilledEnemyJungle"`
	NeutralMinionsKilledYourJungle  int64 `json:"neutralMinionsKilledYourJungle"`
	PhysicalDamageDealtPlayer       int64 `json:"physicalDamageDealtPlayer"`
	PhysicalDamageDealtToChampions  int64 `json:"physicalDamageDealtToChampions"`
	PhysicalDamageTaken             int64 `json:"physicalDamageTaken"`
	PlayerPosition                  int64 `json:"playerPosition"`
	Team                            int64 `json:"team"`
	TimePlayed                      int64 `json:"timePlayed"`
	TotalDamageDealt                int64 `json:"totalDamageDealt"`
	TotalDamageDealtToBuildings     int64 `json:"totalDamageDealtToBuildings"`
	TotalDamageDealtToChampions     int64 `json:"totalDamageDealtToChampions"`
	TotalDamageTaken                int64 `json:"totalDamageTaken"`
	TotalHeal                       int64 `json:"totalHeal"`
	TotalTimeCrowdControlDealt      int64 `json:"totalTimeCrowdControlDealt"`
	TotalUnitsHealed                int64 `json:"totalUnitsHealed"`
	TrueDamageDealtPlayer           int64 `json:"trueDamageDealtPlayer"`
	TrueDamageDealtToChampions      int64 `json:"trueDamageDealtToChampions"`
	TurretsKilled                   int64 `json:"turretsKilled"`
	VisionWardsBought               int64 `json:"visionWardsBought"`
	WardKilled                      int64 `json:"wardKilled"`
	WardPlaced                      int64 `json:"wardPlaced"`
	Win                             bool  `json:"win"`
}

func (a *Api) RecentGames(summonerId int64) (recentGames RecentGames, err error) {
	url, err := a.addParams(fmt.Sprintf("%s/api/lol/%v/v1.3/game/by-summoner/%d/recent",
		a.baseURL(), a.Region(), summonerId))
	err = a.getJSON(url, &recentGames)
	return recentGames, err
}
