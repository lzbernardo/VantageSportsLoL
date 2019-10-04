package riot

import "strings"

type Region string

const (
	R_BR   Region = "BR"
	R_EUNE Region = "EUNE"
	R_EUW  Region = "EUW"
	R_JP   Region = "JP"
	R_KR   Region = "KR"
	R_LAN  Region = "LAN"
	R_LAS  Region = "LAS"
	R_NA   Region = "NA"
	R_OCE  Region = "OCE"
	R_RU   Region = "RU"
	R_TR   Region = "TR"
	R_PBE  Region = "PBE"
)

func (r Region) String() string {
	return strings.ToLower(string(r))
}

func RegionFromString(r string) Region {
	return Region(strings.ToUpper(r))
}

type Platform string

const (
	P_BR1  Platform = "BR1"
	P_EUN1 Platform = "EUN1"
	P_EUW1 Platform = "EUW1"
	P_JP1  Platform = "JP1"
	P_KR   Platform = "KR"
	P_LA1  Platform = "LA1"
	P_LA2  Platform = "LA2"
	P_NA1  Platform = "NA1"
	P_OC1  Platform = "OC1"
	P_RU   Platform = "RU"
	P_TR1  Platform = "TR1"
	P_PBE1 Platform = "PBE1"
)

var regionsToPlatforms = map[Region]Platform{
	R_BR:   P_BR1,
	R_EUNE: P_EUN1,
	R_EUW:  P_EUW1,
	R_JP:   P_JP1,
	R_KR:   P_KR,
	R_LAN:  P_LA1,
	R_LAS:  P_LA2,
	R_NA:   P_NA1,
	R_OCE:  P_OC1,
	R_RU:   P_RU,
	R_TR:   P_TR1,
	R_PBE:  P_PBE1,
}

var platformsToRegions = map[Platform]Region{
	P_BR1:  R_BR,
	P_EUN1: R_EUNE,
	P_EUW1: R_EUW,
	P_JP1:  R_JP,
	P_KR:   R_KR,
	P_LA1:  R_LAN,
	P_LA2:  R_LAS,
	P_NA1:  R_NA,
	P_OC1:  R_OCE,
	P_RU:   R_RU,
	P_TR1:  R_TR,
	P_PBE1: R_PBE,
}

func RegionFromPlatform(p Platform) Region {
	return platformsToRegions[p]
}

func PlatformFromRegion(r Region) Platform {
	return regionsToPlatforms[r]
}

func PlatformFromString(p string) Platform {
	return Platform(strings.ToUpper(p))
}

func AllRegions() []Region {
	all := []Region{}
	for k := range regionsToPlatforms {
		all = append(all, k)
	}
	return all
}

func AllPlatforms() []Platform {
	all := []Platform{}
	for k := range platformsToRegions {
		all = append(all, k)
	}
	return all
}

type QueueType string

const (
	RANKED_FLEX_SR     QueueType = "RANKED_FLEX_SR"                // Rank Flex Queue (Season 7+)
	RANKED_SOLO_5x5    QueueType = "RANKED_SOLO_5x5"               // Rank Solo Queue (Pre-Dynamic Queue Deprecated)
	RANKED_TEAM_3x3    QueueType = "RANKED_TEAM_3x3"               // Deprecated
	RANKED_TEAM_5x5    QueueType = "RANKED_TEAM_5x5"               // Ranked 5v5 Queue
	TB_RANKED_SOLO_5x5 QueueType = "TEAM_BUILDER_DRAFT_RANKED_5x5" // Dynamic Queue (Season 6 Deprecated)
	TB_RANKED_SOLO     QueueType = "TEAM_BUILDER_RANKED_SOLO"      // Rank Solo Queue (Season 7+)
)

type Tier string

const (
	CHALLENGER Tier = "CHALLENGER"
	MASTER     Tier = "MASTER"
	DIAMOND    Tier = "DIAMOND"
	PLATINUM   Tier = "PLATINUM"
	GOLD       Tier = "GOLD"
	SILVER     Tier = "SILVER"
	BRONZE     Tier = "BRONZE"
)

func (t Tier) Equals(compareT Tier) bool {
	return strings.EqualFold(string(t), string(compareT))
}

func (t Tier) Ord() int {
	switch t {
	case BRONZE:
		return 1
	case SILVER:
		return 2
	case GOLD:
		return 3
	case PLATINUM:
		return 4
	case DIAMOND:
		return 5
	case MASTER:
		return 6
	case CHALLENGER:
		return 7
	default:
		return 0
	}
}

type Division string

const (
	DIV_I   Division = "I"
	DIV_II  Division = "II"
	DIV_III Division = "III"
	DIV_IV  Division = "IV"
	DIV_V   Division = "V"
)

func (d Division) Equals(compareD Division) bool {
	return strings.EqualFold(string(d), string(compareD))
}

func (d Division) Ord() int {
	switch d {
	case DIV_V:
		return 1
	case DIV_IV:
		return 2
	case DIV_III:
		return 3
	case DIV_II:
		return 4
	case DIV_I:
		return 5
	default:
		return 0
	}
}

// To regenerate:
// curl http://ddragon.leagueoflegends.com/cdn/<PATCH_NUM>/data/en_US/summoner.json \
//   pj | grep -E "\"key\"|\"name\""
var SpellNamesByID = map[int64]string{
	1:  "cleanse",
	11: "smite",
	12: "teleport",
	13: "clarity",
	14: "ignite",
	21: "barrier",
	30: "totheking",
	31: "porotoss",
	32: "mark",
	3:  "exhaust",
	4:  "flash",
	6:  "ghost",
	7:  "heal",
}
