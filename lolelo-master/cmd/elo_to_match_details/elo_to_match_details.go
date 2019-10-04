package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/VantageSports/lolelo/event"
	"github.com/VantageSports/lolelo/parse"
	"github.com/VantageSports/lolelo/validate"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/lolstats/ingest"
	"github.com/VantageSports/riot/api"
)

var team1IdStart = flag.Int64("t1id", 0, "first id of team 1")
var team2IdStart = flag.Int64("t2id", 0, "first id of team 2")
var matchID = flag.Int64("id", 0, "matchID to assign")
var eloFile = flag.String("elo", "", "elo file")
var outFile = flag.String("o", "", "output file")

func main() {
	flag.Parse()

	if *matchID == 0 || *eloFile == "" || *team1IdStart == 0 || *team2IdStart == 0 {
		fmt.Println("Must pass in -id, -elo, -t1id, -t2id")
		return
	}

	md := api.MatchDetail{}
	md.PlatformID = "NA1"
	md.MatchID = *matchID
	md.MatchCreation = time.Now().Unix() * 1000
	md.MatchVersion = "7.1.165.3566"
	md.QueueType = "CUSTOM"
	md.Teams = []api.Team{
		api.Team{TeamID: 100},
		api.Team{TeamID: 200},
	}
	md.MapID = 11
	md.MatchMode = "CLASSIC"
	md.MatchType = "CUSTOM_GAME"

	md.ParticipantIdentities = []api.ParticipantIdentity{}
	md.Participants = []api.Participant{}

	events, err := parse.LogFile(*eloFile)
	if err != nil {
		log.Fatalln(err)
	}

	nameToIndex := map[string]int{}
	champNetIds := map[int64]int{}
	lastKiller := int64(0)
	lastKillTime := 0.0
	goldSpent := map[int]int64{}
	goldCurrent := map[int]int64{}

	index := 1
	for _, ev := range events {
		switch t := ev.(type) {
		case *event.NetworkIDMapping:
			if t.EloType == "ID_HERO" && index <= 10 {
				sumId := int64(0)
				if index <= 5 {
					sumId = *team1IdStart + int64(index-1)
				} else {
					sumId = *team2IdStart + int64(index-5-1)
				}

				md.ParticipantIdentities = append(md.ParticipantIdentities,
					api.ParticipantIdentity{
						ParticipantID: index,
						Player: api.Player{
							SummonerID:   sumId,
							SummonerName: t.SenderName,
						},
					})
				md.Participants = append(md.Participants,
					api.Participant{
						ParticipantID: index,
						TeamID:        int(baseview.TeamID(int64(index))),
						Timeline:      api.ParticipantTimeline{},
						Stats: api.ParticipantStats{
							ChampLevel: 1,
						},
					})
				nameToIndex[t.SenderName] = index
				champNetIds[t.NetworkID] = index
				index++
			}
		case *event.GameEnd:
			md.MatchDuration = int64(t.Time())
		case *event.SpellCast:
			if len(nameToIndex) != 10 {
				break
			}
			if t.Slot == "Q" && md.Participants[nameToIndex[t.SenderName]-1].ChampionID == 0 {
				md.Participants[nameToIndex[t.SenderName]-1].ChampionID = QToChampId(t.SpellName)
			}
			if t.Slot == "Summoner1" && md.Participants[nameToIndex[t.SenderName]-1].Spell1ID == 0 {
				md.Participants[nameToIndex[t.SenderName]-1].Spell1ID = SummonerSpellId(t.SpellName)
			}
			if t.Slot == "Summoner2" && md.Participants[nameToIndex[t.SenderName]-1].Spell2ID == 0 {
				md.Participants[nameToIndex[t.SenderName]-1].Spell2ID = SummonerSpellId(t.SpellName)
			}
		case *event.Die:
			if champId, ok := champNetIds[t.NetworkID]; ok {
				md.Participants[champId-1].Stats.Deaths++
				if lastKillTime == t.Time() {
					if killerId, ok := champNetIds[lastKiller]; ok {
						md.Participants[killerId-1].Stats.Kills++
					} else {
						fmt.Printf("Last killer of %v at %v was not a champion\n", champId, t.Time())
					}
				} else {
					fmt.Printf("Unable to find killer for death of %v at %v\n", champId, t.Time())
				}
			}
		case *event.Kill:
			lastKillTime = t.Time()
			lastKiller = t.NetworkID
		case *event.LevelUp:
			if t.Level > 0 {
				if champId, ok := champNetIds[t.NetworkID]; ok {
					md.Participants[champId-1].Stats.ChampLevel++
				}
			}
		case *event.Ping:
			if champId, ok := champNetIds[t.NetworkID]; ok {
				if t.Gold < goldCurrent[champId] {
					goldSpent[champId] += goldCurrent[champId] - t.Gold
				}
				goldCurrent[champId] = t.Gold

				md.Participants[champId-1].Stats.MinionsKilled = t.MinionsKilled
				md.Participants[champId-1].Stats.NeutralMinionsKilled = t.NeutralMinionsKilled
			}
		}
	}
	// Fill in gold amounts
	for champId, _ := range goldCurrent {
		md.Participants[champId-1].Stats.GoldSpent = goldSpent[champId]
		md.Participants[champId-1].Stats.GoldEarned = goldSpent[champId] + goldCurrent[champId]
	}

	// Have to go full circle. Use the partial match details to generate the baseview
	validate.UpdateMatchSeconds(events, 0.0)
	participants, err := validate.ParticipantMapMD(events, &md)
	if err != nil {
		log.Fatalln(err)
	}

	bv, err := event.EloToBaseview(events, participants)

	// Use the baseview to generate advanced stats.
	processedWards := false
	for i, startId := range []int64{*team1IdStart, *team2IdStart} {
		for index := int64(0); index < 5; index++ {
			fmt.Println("Computing advanced stats for", startId+index)
			advancedStats, err := ingest.ComputeAdvanced(*bv, startId+index, *matchID, "NA1")
			if err != nil {
				log.Fatalln(err)
			}

			// Use the advanced stats to fill in the match details
			switch advancedStats.RolePosition {
			case baseview.RoleTop:
				md.Participants[i*5+int(index)].Timeline.Lane = "TOP"
				md.Participants[i*5+int(index)].Timeline.Role = "SOLO"
			case baseview.RoleMid:
				md.Participants[i*5+int(index)].Timeline.Lane = "MIDDLE"
				md.Participants[i*5+int(index)].Timeline.Role = "SOLO"
			case baseview.RoleAdc:
				md.Participants[i*5+int(index)].Timeline.Lane = "BOTTOM"
				md.Participants[i*5+int(index)].Timeline.Role = "DUO_CARRY"
			case baseview.RoleJungle:
				md.Participants[i*5+int(index)].Timeline.Lane = "JUNGLE"
				md.Participants[i*5+int(index)].Timeline.Role = "NONE"
			case baseview.RoleSupport:
				md.Participants[i*5+int(index)].Timeline.Lane = "BOTTOM"
				md.Participants[i*5+int(index)].Timeline.Role = "DUO_SUPPORT"
			}

			// Add ward info. Only need to do this once
			if !processedWards {
				for pId, wardLives := range advancedStats.WardLives {
					for _, ward := range wardLives {
						md.Participants[pId-1].Stats.WardsPlaced++
						if ward.ClearedBy != 0 {
							md.Participants[ward.ClearedBy-1].Stats.WardsKilled++
						}
					}
				}
				processedWards = true
			}
		}
	}

	data, _ := json.MarshalIndent(md, "", "  ")

	if *outFile == "" {
		fmt.Println(string(data))
	} else {
		err := ioutil.WriteFile(*outFile, data, 0666)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func QToChampId(spell string) int {
	switch spell {
	case "AsheQ":
		return 22 // Ashe
	case "BardQ":
		return 432 // Bard
	case "CamilleQ":
		return 164 // Camille
	case "CassiopeiaQ":
		return 69 // Cassiopeia
	case "PhosphorusBomb":
		return 42 // Corki
	case "EzrealMysticShot":
		return 81 // Ezreal
	case "FioraQ":
		return 114 // Fiora
	case "JayceShockBlast":
		return 126 // Jayce
	case "JhinQ":
		return 202 // Jhin
	case "KalistaMysticShot":
		return 429 // Kalista
	case "BlindMonkQOne":
		return 64 // Lee Sin
	case "LuluQ":
		return 117 // Lulu
	case "MalzaharQ":
		return 90 // Malzahar
	case "MaokaiTrunkLine":
		return 57 // Maokai
	case "OlafAxeThrowCast":
		return 2 // Olaf
	case "RekSaiQ":
		return 421 // Reksai
	case "RengarQ":
		return 107 // Rengar
	case "RyzeQ":
		return 13 // Ryze
	case "PoisonTrail":
		return 27 // Singed
	case "SyndraQ":
		return 134 // Syndra
	case "VarusQ":
		return 110 // Varus
	case "ZacQ":
		return 154 // Zac
	case "ZileanQ":
		return 26 // Zilean
	case "ZyraQ":
		return 143 // Zyra
	default:
		log.Fatalln("Unknown spell:", spell)
		return 0
	}
}

func SummonerSpellId(spell string) int {
	switch spell {
	case "SummonerBoost":
		return 1 // Cleanse
	case "SummonerExhaust":
		return 3
	case "SummonerFlash":
		return 4
	case "SummonerHaste":
		return 6
	case "SummonerHeal":
		return 7
	case "SummonerSmite":
		return 11
	case "SummonerTeleport":
		return 12
	default:
		log.Fatalln("Unknown summoner spell:", spell)
		return 0
	}
}
