// To update:
//
// 1) curl --request GET 'https://global.api.pvp.net/api/lol/static-data/na/v1.2/champion?champData=all&api_key=<KEY>' > /tmp/riot.json
// 2) Download the V1 champ abilities (abilities) sheet as a csv to /tmp/abilities.csv
//
// 3) $ go build && ./champ_metadata -abilities /tmp/abilities.csv -riot /tmp/riot.json -out /tmp/out.json
//    $ cp /tmp/out.json $GOPATH/src/github.com/VantageSports/webapps/lolanalytics/static/dist/json/champ_meta.json

package main

import (
	"bufio"
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/VantageSports/common/json"
)

var riotChampDataPath = flag.String("riot", "/tmp/riot.json", "Riot static champData")
var abilitiesCSVPath = flag.String("abilities", "/tmp/abilities.csv", "Vantage Abilities CSV")
var outPath = flag.String("out", "/tmp/out.json", "output path")

var champNameToID map[string]int64
var champIDToName map[int64]string

func init() {
	log.SetFlags(log.Lshortfile)

	champNameToID = map[string]int64{}
	champIDToName = map[int64]string{}
}

func main() {
	flag.Parse()

	rMeta := new(riotMeta)
	err := json.DecodeFile(*riotChampDataPath, rMeta)
	exitIf(err)

	buildChampMaps(rMeta)

	abilities := readAbilities(*abilitiesCSVPath)

	out := buildAbilityMap(rMeta, abilities)
	err = json.Write(*outPath, &out, 0664)
	exitIf(err)
	log.Println("success for", len(out), "champions")
}

func buildChampMaps(r *riotMeta) {
	for _, info := range r.Data {
		name := strings.ToLower(info.Name)
		champIDToName[info.ID] = name
		champNameToID[name] = info.ID
	}
}

func readAbilities(path string) []abilityRow {
	f, err := os.Open(path)
	exitIf(err)
	defer f.Close()

	res := []abilityRow{}

	r := csv.NewReader(bufio.NewReader(f))
	for rowNum := 0; ; rowNum++ {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if rowNum == 0 {
			log.Println("skipping header")
			continue
		}
		row := newAbilityRow(record)
		if row == nil {
			log.Println("skipping row num:", rowNum)
			continue
		}
		res = append(res, *row)
	}

	return res
}

func buildAbilityMap(rMeta *riotMeta, vsAbilities []abilityRow) map[string]vsChampAbilities {
	// we would like to use int64->abilities, but json only allows us to use
	// string keys.
	res := map[string]vsChampAbilities{}
	for _, row := range vsAbilities {
		champID, found := champNameToID[row.ChampName]
		if !found {
			log.Fatalln("champ name not found", row.ChampName)
		}
		riotChamp, found := rMeta.ByID(champID)
		if !found {
			log.Fatalln("no riot champ found with name:", row.ChampName)
		}
		spell := riotChamp.Spell(row.AbilityName)
		if spell == nil {
			log.Println("no spell found for name:", row.AbilityName)
			continue
		}
		champKey := fmt.Sprintf("%d", champID)
		if res[champKey] == nil {
			res[champKey] = vsChampAbilities{}
		}
		res[champKey][row.StatID()] = vsAbility{
			Feedback: row.Feedback,
			Display:  row.Display,
		}
	}
	return res
}

// riotMeta describes a subject of the json object returned by riot's static
// data champData endpoint.
// Docs: https://developer.riotgames.com/api/methods#!/1055/3633
type riotMeta struct {
	Data map[string]riotChampInfo
	byID map[int64]riotChampInfo `json:"-"` // constructed lazily
}

func (rm *riotMeta) ByID(id int64) (riotChampInfo, bool) {
	if rm.byID == nil {
		rm.byID = map[int64]riotChampInfo{}
		for _, info := range rm.Data {
			rm.byID[info.ID] = info
		}
	}
	info, found := rm.byID[id]
	return info, found
}

type riotChampInfo struct {
	ID     int64
	Name   string
	Blurb  string
	Spells []riotSpellInfo
}

func (rci *riotChampInfo) Spell(name string) *riotSpellInfo {
	for i := range rci.Spells {
		info := rci.Spells[i]
		if strings.ToLower(info.Name) == strings.ToLower(name) {
			return &info
		}
	}
	return nil
}

type riotSpellInfo struct {
	Key         string
	Name        string
	Cooldown    []float64
	Cost        []float64
	Description string
}

// vsChampAbilityMap is a map from stat id to information about the ability
// described by that stat. This structure is used by the frontend to provide
// textual feedback depending on the user's percentile in a given stat.
type vsChampAbilities map[string]vsAbility

type vsAbility struct {
	Description string   `json:"description"` // Not using in product right now.
	Feedback    feedback `json:"feedback"`
	Display     bool     `json:"display"`
}

type feedback struct {
	Great string `json:"great"`
	OK    string `json:"ok"`
	Poor  string `json:"poor"`
}

// vsAbilityRow describes one row in the abilities spreadsheet, which contains
// the champion, the ability, and the various feedback texts.
type abilityRow struct {
	ChampName   string
	Key         string
	Stat        string
	AbilityName string
	Display     bool
	Feedback    feedback
}

func newAbilityRow(fields []string) *abilityRow {
	if len(fields) < 8 {
		return nil
	}
	row := &abilityRow{
		ChampName:   strings.ToLower(fields[0]),
		Key:         strings.ToLower(fields[1]),
		Stat:        strings.ToLower(fields[2]),
		AbilityName: strings.ToLower(fields[3]),
		Display:     strings.ToLower(fields[4]) == "y",
		Feedback: feedback{
			Great: fields[6],
			OK:    fields[7],
			Poor:  fields[8],
		},
	}
	f := row.Feedback
	if f.Poor == "" && f.OK == "" {
		return nil
	}
	if row.Stat == "ignore" || row.Stat == "#N/A" {
		return nil
	}
	return row
}

func (a *abilityRow) StatID() string {
	switch a.Key {
	case "q", "w", "e", "r":
	default:
		log.Fatalln("unknown key: ", a.Key)
	}

	if strings.Contains(a.Stat, "volume") {
		return a.Key + "PerMinute"
	} else if strings.Contains(a.Stat, "accuracy") {
		return a.Key + "LandPercentZeroToTen"
	}
	return ""
}

func exitIf(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
