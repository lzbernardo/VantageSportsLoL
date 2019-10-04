package scrape

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/blang/semver"
)

const (
	LOL_BASEURL       = "https://ddragon.leagueoflegends.com"
	LOL_VERSIONS_URI  = "/api/versions.json"
	LOL_CHAMPIONS_URI = "/cdn/%s/data/en_US/champion.json"
)

type ChampionsMeta struct {
	Type    string                  `json:"type"`
	Format  string                  `json:"format"`
	Version string                  `json:"version"`
	Data    map[string]ChampionMeta `json:"data"`
}

type ChampionMeta struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Format  string        `json:"format"`
	Version string        `json:"version"`
	Key     string        `json:"key"`
	Name    string        `json:"name"`
	Title   string        `json:"title"`
	Blurb   string        `json:"blurb"`
	Info    ChampionInfo  `json:"info"`
	Image   ChampionImage `json:"image"`
	Tags    []string      `json:"tags"`
	ParType string        `json:"partype"`
	Stats   ChampionStats `json:"stats"`
}

type ChampionInfo struct {
	Attack     int `json:"attack"`
	Defense    int `json:"defense"`
	Magic      int `json:"magic"`
	Difficulty int `json:"difficulty"`
}

type ChampionImage struct {
	Full   string `json:"full"`
	Sprite string `json:"sprite"`
	Group  string `json:"group"`
	X      int    `json:"x"`
	Y      int    `json:"y"`
	W      int    `json:"w"`
	H      int    `json:"h"`
}

type ChampionStats struct {
	HP                   float64 `json:"hp"`
	HPPerlevel           float64 `json:"hpperlevel"`
	MP                   float64 `json:"mp"`
	MPPerlevel           float64 `json:"mpperlevel"`
	MoveSpeed            float64 `json:"movespeed"`
	Armor                float64 `json:"armor"`
	ArmorPerLevel        float64 `json:"armorperlevel"`
	SpellBlock           float64 `json:"spellblock"`
	SpellBlockPerLevel   float64 `json:"spellblockperlevel"`
	AttackRange          float64 `json:"attackrange"`
	HPRegen              float64 `json:"hpregen"`
	HPRegenPerLevel      float64 `json:"hpregenperlevel"`
	MPRegen              float64 `json:"mpregen"`
	MPRegenPerLevel      float64 `json:"mpregenperlevel"`
	Crit                 float64 `json:"crit"`
	CritPerLevel         float64 `json:"critperlevel"`
	AttackDamage         float64 `json:"attackdamage"`
	AttackDamagePerLevel float64 `json:"attackdamageperlevel"`
	AttackSpeedOffset    float64 `json:"attackspeedoffset"`
	AttackSpeedPerLevel  float64 `json:"attackspeedperlevel"`
}

func GetMostRecentChampionsMetaData() (*ChampionsMeta, error) {
	versions, err := GetLolVersions()
	if err != nil {
		return nil, err
	}

	v, err := GetRecentLolVersion(versions)
	if err != nil {
		return nil, err
	}

	champsMeta, err := GetChampionsMetaData(v)
	if err != nil {
		return nil, err
	}

	data := map[string]ChampionMeta{}
	for key, value := range champsMeta.Data {
		newKey := getChampNameMetaKey(key)
		data[newKey] = value
	}
	champsMeta.Data = data

	return champsMeta, nil
}

// getChampNameMetaKey modifies the name to remove spacing and punctuation.
// For example:
//   - Cho'Gath: Chogath
//   - Dr. Mundo: DrMundo
// Name keys are not consistent with preserving case when spacing and
// punctuation are removed. Therefore ToLower must be called to allow for
// predictability retrieving the metadata by key.
func getChampNameMetaKey(name string) string {
	result := strings.Replace(name, " ", "", -1)
	result = strings.Replace(result, ".", "", -1)
	return strings.ToLower(strings.Replace(result, "'", "", -1))
}

// GetChampionsMetaData retrieves metadata for all champions in league of
// legends for the specified version.
func GetChampionsMetaData(version string) (*ChampionsMeta, error) {
	if version == "" {
		err := fmt.Errorf("empty version string")
		return nil, err
	}

	champions := &ChampionsMeta{}
	url := LOL_BASEURL + fmt.Sprintf(LOL_CHAMPIONS_URI, version)
	err := getJSON(url, champions)

	return champions, err
}

// GetRecentLolVersion returns the most up to date version of league of
// legends.
// Assumes league of legends versioning keeps with the standard pattern of
// <major>.<minor>.<patch> versioning scheme.
func GetRecentLolVersion(versions []string) (string, error) {
	if len(versions) == 0 {
		err := fmt.Errorf("empty versions list")
		return "", err
	}
	maxVersionStr := versions[0]
	maxVersion, _ := semver.Make(maxVersionStr)
	for _, v := range versions {
		if strings.Contains(v, "lolpatch") {
			continue
		}

		nextVersion, err := semver.Make(v)
		if err != nil {
			continue
		}

		if nextVersion.GT(maxVersion) {
			maxVersionStr = v
			maxVersion = nextVersion
		}
	}

	return maxVersionStr, nil
}

// GetLolVersions retrieves all versions listed for league of legends.
func GetLolVersions() ([]string, error) {
	versions := []string{}
	url := LOL_BASEURL + LOL_VERSIONS_URI
	err := getJSON(url, &versions)

	return versions, err
}

func getJSON(url string, v interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected status code from url: %s: %d", url, resp.StatusCode)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, v)
}
