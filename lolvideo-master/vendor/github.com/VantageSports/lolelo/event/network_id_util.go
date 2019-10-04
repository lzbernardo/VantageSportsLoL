package event

import (
	"fmt"
	"strings"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats/baseview"
)

type eloEntity struct {
	Name string
	Type baseview.ActorType
	// ID is type-dependent.
	// if hero, id is participant id.
	// if building/monster, id is the baseview id for that building/monster
	// else, id is the (original) network id.
	ID int64
}

// netIDMap maps from network id to eloEntity. eloEntity is a crude abstraction
// of various League entities (heroes, turrets, minions, etc.). Most entities
// will just have a name, but heroes also have a participant id, which we use
// to match up with objects from the riot API.
type networkIDMap struct {
	nameToPID      map[string]int64
	nIDToEloEntity map[int64]*eloEntity
}

func NewNetworkIDMap() *networkIDMap {
	return &networkIDMap{
		nameToPID:      map[string]int64{},
		nIDToEloEntity: map[int64]*eloEntity{},
	}
}

// Get returns the eloEntity if it has been added to the map, or an empty
// eloEntity otherwise.
func (n *networkIDMap) Get(nID int64) (*eloEntity, error) {
	// If there was no network id (e.g. a null target on a spell cast) we just
	// return an 'empty' eloevent.
	if nID == 0 {
		return &eloEntity{ID: 0, Name: "", Type: baseview.ActorType("")}, nil
	}

	entity := n.nIDToEloEntity[nID]
	if entity == nil {
		return nil, fmt.Errorf("id not found: %d", nID)
	}

	// If this network id represents a hero, but we haven't yet figured out that
	// hero's participant id, do that now.
	if entity.Type == baseview.ActorHero && entity.ID == 0 {
		entity.ID = n.nameToPID[entity.Name]
		if entity.ID == 0 {
			return nil, fmt.Errorf("unknown participant id for hero: %s", entity.Name)
		}
		n.nIDToEloEntity[nID] = entity
	}

	return entity, nil
}

func (n *networkIDMap) AddParticipant(pID int64, name string) {
	n.nameToPID[name] = pID
}

// AddEvent is how the netIDMap builds the mappings from network id to entity.
// Certain events contain both the name and network id, and when we see those
// events we add them to the map.
func (n *networkIDMap) AddEvent(e EloEvent) {
	var entity *eloEntity

	switch t := e.(type) {
	case *NetworkIDMapping:
		entity = &eloEntity{Name: t.SenderName}
		if t.EloType == "ID_HERO" {
			// We add participant ids to heros at Get time (after we have a name
			// to participant id mapping)
			entity.Type = baseview.ActorHero
		} else {
			entity.Type, entity.ID = lookupBaseviewID(t.SenderName, t.NetworkID)
		}
		n.nIDToEloEntity[t.NetworkID] = entity

	case *OnCreate:
		if _, found := n.nIDToEloEntity[t.NetworkID]; !found {
			a, id := lookupBaseviewID(t.SenderName, t.NetworkID)
			n.nIDToEloEntity[t.NetworkID] = &eloEntity{Name: t.SenderName, Type: a, ID: id}
		}

	case *OnDelete:
		// TODO: (Cameron) any value (other than memory pressure) to deleting
		// items from this map? E.g. useful to know WHICH minions were alive
		// at a given time?
	}
}

func lookupBaseviewID(name string, networkID int64) (baseview.ActorType, int64) {
	if isMinion(name) {
		return baseview.ActorMinion, networkID
	} else if isWard(name) {
		return baseview.ActorWard, networkID
	} else if id := turretID(name); id != 0 {
		return baseview.ActorTurret, id
	} else if id := monsterID(name); id != 0 {
		return baseview.ActorMonster, id
	} else if id := inhibitorID(name); id != 0 {
		return baseview.ActorInhibitor, id
	}
	return baseview.ActorType(""), networkID
}

func isMinion(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasPrefix(lower, "minion")
}

func isWard(name string) bool {
	switch strings.ToLower(name) {
	case "sightward", "visionward", "jammerdevice":
		return true
	}
	return false
}

func inhibitorID(name string) int64 {
	lower := strings.ToLower(name)
	if !strings.HasPrefix(lower, "barracks_") {
		return 0
	}

	switch strings.TrimPrefix(lower, "barracks_") {
	case "t1_l1":
		return baseview.InhibitorBlueTop
	case "t1_c1":
		return baseview.InhibitorBlueMid
	case "t1_r1":
		return baseview.InhibitorBlueBot
	case "t2_l1":
		return baseview.InhibitorRedTop
	case "t2_c1":
		return baseview.InhibitorRedMid
	case "t2_r1":
		return baseview.InhibitorRedBot
	default:
		log.Warning("no id found for name that seems like it should be a barracks: " + name)
		return 0
	}
}

func turretID(name string) int64 {
	// NOTE below that the turret names for t1 and t2 are currently different.
	// T1 has 7 'c' turrets, and 2 'l' and 'r' turrets (02, and 03)
	// T2 has 5 'c' turrets, and 3 'l' and 'r' turrets (01, 02, and 03)

	lower := strings.ToLower(name)
	if !strings.HasPrefix(lower, "turret_") {
		return 0
	}
	switch strings.TrimPrefix(lower, "turret_") {
	case "t1_l_02_a":
		return baseview.TurretBlueTopInner
	case "t1_l_03_a":
		return baseview.TurretBlueTopOuter
	case "t1_c_01_a":
		return baseview.TurretBlueUpperNexus
	case "t1_c_02_a":
		return baseview.TurretBlueLowerNexus
	case "t1_c_03_a":
		return baseview.TurretBlueMidBase
	case "t1_c_04_a":
		return baseview.TurretBlueMidInner
	case "t1_c_05_a":
		return baseview.TurretBlueMidOuter
	// as of 6.22, only the first is expected, but leaving the 2nd in in case
	// these revert again.
	case "t1_c_06_a", "t1_l_01_a":
		return baseview.TurretBlueTopBase
	case "t1_c_07_a", "t1_r_01_a":
		return baseview.TurretBlueBotBase
	case "t1_r_02_a":
		return baseview.TurretBlueBotInner
	case "t1_r_03_a":
		return baseview.TurretBlueBotOuter

	// as of 6.22, only the first is expected, but leaving the 2nd in in case
	// these revert again.
	case "t2_l_01_a", "t2_c_06_a":
		return baseview.TurretRedTopBase
	case "t2_l_02_a":
		return baseview.TurretRedTopInner
	case "t2_l_03_a":
		return baseview.TurretRedTopOuter
	case "t2_c_01_a":
		return baseview.TurretRedLowerNexus
	case "t2_c_02_a":
		return baseview.TurretRedUpperNexus
	case "t2_c_03_a":
		return baseview.TurretRedMidBase
	case "t2_c_04_a":
		return baseview.TurretRedMidInner
	case "t2_c_05_a":
		return baseview.TurretRedMidOuter
	// as of 6.22, only the first is expected, but leaving the 2nd in in case
	// these revert again.
	case "t2_r_01_a", "t2_c_07_a":
		return baseview.TurretRedBotBase
	case "t2_r_02_a":
		return baseview.TurretRedBotInner
	case "t2_r_03_a":
		return baseview.TurretRedBotOuter
	case "chaosturretshrine_a":
		return baseview.TurretRedFountain
	case "orderturretshrine_a":
		return baseview.TurretBlueFountain
	default:
		// TODO (cameron): there are objects created without the "_a" suffix
		// that would trigger this warning for every game, so I'm leaving this
		// commented out until we better understand what the difference between
		// these similarly-named objects are.
		//log.Warning("no id found for name that seems like it should be a turret: " + name)
		return 0
	}
}

func monsterID(name string) int64 {
	// NOTE: Not all monsters have the "sru_" prefix. E.g. minikrug
	cleanName := strings.TrimPrefix(strings.ToLower(name), "sru_")
	switch {
	case strings.HasPrefix(cleanName, "riftherald"):
		return baseview.MonsterRiftHerald
	case strings.HasPrefix(cleanName, "baron") && !strings.HasPrefix(cleanName, "baronspawn"):
		return baseview.MonsterBaron
	case strings.HasPrefix(cleanName, "dragon"):
		if strings.HasPrefix(cleanName, "dragon_elder") {
			return baseview.MonsterDragonElder
		}
		return baseview.MonsterDragonElemental
	case strings.HasPrefix(cleanName, "bluemini"):
		return baseview.MonsterBlueSentinelMini
	case strings.HasPrefix(cleanName, "blue"):
		return baseview.MonsterBlueSentinel
	case strings.HasPrefix(cleanName, "gromp"):
		return baseview.MonsterGromp
	case strings.HasPrefix(cleanName, "krugmini") || strings.HasPrefix(cleanName, "minikrug"):
		return baseview.MonsterKrugMini
	case strings.HasPrefix(cleanName, "krug"):
		return baseview.MonsterKrug
	case strings.HasPrefix(cleanName, "murkwolfmini"):
		return baseview.MonsterMurkwolfMini
	case strings.HasPrefix(cleanName, "murkwolf"):
		return baseview.MonsterMurkwolf
	case strings.HasPrefix(cleanName, "razorbeakmini"):
		return baseview.MonsterRazorMini
	case strings.HasPrefix(cleanName, "razorbeak"):
		return baseview.MonsterRazor
	case strings.HasPrefix(cleanName, "redmini"):
		return baseview.MonsterRedBrambMini
	// NOTE: these are gone, but we're leaving in for backwards compatibility
	case strings.HasPrefix(cleanName, "red"):
		return baseview.MonsterRedBramb
	case strings.HasPrefix(cleanName, "crab"):
		return baseview.MonsterRedBramb
	default:
		return 0
	}
}
