// this file contains library functions for converting from the elo format to
// the (standard) baseview format.

package event

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats/baseview"
)

func EloToBaseview(events []EloEvent, participants []baseview.Participant) (*baseview.Baseview, error) {
	res := &baseview.Baseview{
		Participants: participants,
		Events:       []baseview.Event{},
		LastUpdated:  time.Now(),
	}

	// ward_placed events occur in two steps. first we notice a spell_cast,
	// then a ward_created.
	wardSpells := []*SpellCast{}
	wardCreations := []*OnCreate{}

	// iterate through all events initially to build up a mapping of network id
	// to elo object (participant ids and names).
	netIDs := NewNetworkIDMap()
	for _, e := range events {
		netIDs.AddEvent(e)
	}
	for _, p := range participants {
		netIDs.AddParticipant(p.ParticipantID, p.SummonerName)
	}

	// keep track of where each champ was last seen, so we can add positions to
	// events that don't have them already (like deaths)
	lastPosition := map[int64]baseview.Position{}

	for index, e := range events {
		var be baseview.Event
		switch t := e.(type) {

		case *ChampKill, *GameEnd, *NetworkIDMapping, *Kill, *OnDelete, *NexusDestroyed:
			//skip

		case *BasicAttack:
			attacker, err := netIDs.Get(t.NetworkID)
			if err != nil {
				return nil, err
			}
			target, err := netIDs.Get(t.TargetNetworkID)
			if err != nil {
				return nil, err
			}
			be = &baseview.Attack{
				CooldownExpires: 0.0,
				AttackerID:      attacker.ID,
				AttackerType:    attacker.Type,
				Slot:            "basic",
				TargetID:        target.ID,
				TargetType:      target.Type,
				TargetPosition:  &t.TargetPosition,
			}
			be.SetType("attack")

		case *ChampDie:
			champ, err := netIDs.Get(t.NetworkID)
			if err != nil {
				return nil, err
			}
			be = &baseview.Death{
				VictimID: champ.ID,
				Position: lastPosition[t.NetworkID],
			}
			be.SetType("death")

		case *Damage:
			attacker, err := netIDs.Get(t.NetworkID)
			if err != nil {
				return nil, err
			}
			victim, err := netIDs.Get(t.TargetNetworkID)
			if err != nil {
				return nil, err
			}
			senderMax := 1000.0 // default in case we can't get max health
			if p := nextPing(events[index:], t.TargetNetworkID); p != nil {
				senderMax = p.HealthMax
			}
			be = &baseview.Damage{
				AttackerID:   attacker.ID,
				AttackerType: attacker.Type,
				VictimID:     victim.ID,
				VictimType:   victim.Type,
				Total:        t.Damage,
				Percent:      t.Damage * 100.0 / senderMax,
			}
			be.SetType("damage")

		case *Die:
			victim, err := netIDs.Get(t.NetworkID)
			if err != nil {
				return nil, err
			}
			if victim.Type == baseview.ActorTurret || victim.Type == baseview.ActorInhibitor {
				be = &baseview.BuildingKill{
					BuildingType: victim.Type,
					BuildingID:   victim.ID,
				}
				be.SetType("building_kill")
			} else if isWardCreate(victim.Name) {
				be = &baseview.WardDeath{WardID: t.NetworkID}
				be.SetType("ward_death")
			}

		case *LevelUp:
			if t.Level > 0 {
				champ, err := netIDs.Get(t.NetworkID)
				if err != nil {
					return nil, err
				}
				be = &baseview.LevelUp{
					Level:         t.Level,
					ParticipantID: champ.ID,
				}
				be.SetType("level_up")
			}

		case *OnCreate:
			if isWardCreate(t.SenderName) {
				wardCreations = append(wardCreations, t)
			}

		case *Ping:
			person, err := netIDs.Get(t.NetworkID)
			if err != nil {
				return nil, err
			}
			lastPosition[t.NetworkID] = t.Position
			be = &baseview.StateUpdate{
				Gold:                 t.Gold,
				Health:               t.Health,
				HealthMax:            t.HealthMax,
				InGrass:              t.InGrass,
				Mana:                 t.Mana,
				ManaMax:              t.ManaMax,
				MinionsKilled:        t.MinionsKilled,
				NeutralMinionsKilled: t.NeutralMinionsKilled,
				ParticipantID:        person.ID,
				Position:             t.Position,
				UnderOwnTurret:       t.UnderTurret && !t.UnderEnemyTurret,
				UnderEnemyTurret:     t.UnderEnemyTurret,
			}
			be.SetType("state_update")

		case *SpellCast:
			if t.WardItemType() != nil {
				wardSpells = append(wardSpells, t)
			} else {
				attacker, err := netIDs.Get(t.NetworkID)
				if err != nil {
					return nil, err
				}
				target, err := netIDs.Get(t.TargetNetworkID)
				if err != nil {
					return nil, err
				}
				be = &baseview.Attack{
					CooldownExpires: getCooldown(events[index:], t.NetworkID, t.Slot),
					Start:           &t.PositionStart,
					End:             &t.PositionEnd,
					Slot:            t.Slot,
					AttackerID:      attacker.ID,
					AttackerType:    attacker.Type,
					TargetID:        target.ID,
					TargetType:      target.Type,
				}
				be.SetType("attack")
			}

		default:
			log.Warning(fmt.Sprintf("unhandled event: %s", e.Type()))
		}
		if be != nil {
			if be.Seconds() == 0 {
				be.SetSeconds(e.MatchSeconds())
			}
			res.Events = append(res.Events, be)
		}
	}

	wardsPlaced, err := placeWards(wardCreations, wardSpells, netIDs)
	if err != nil {
		return res, err
	}
	res.Events = append(res.Events, wardsPlaced...)

	// sort res.Events since we may have modified the relative ordering of
	// events by adjusting certain event offsets.
	sort.Stable(byTime(res.Events))

	return res, nil
}

// placeWards iterates in order through each ward creation and greedily finds
// the best match among all the spell cast events (assumed to already be
// filtered to just those "ward-creating" casts), returning an error if there
// is no suitable pairing for all spells and creations.
// NOTES:
//  - this is not guaranteed to find a maximally-optimal pairing
//  - we TYPICALLY see spell_cast slightly before ward_created, but not always
func placeWards(creations []*OnCreate, spells []*SpellCast, netIDs *networkIDMap) ([]baseview.Event, error) {
	if len(spells) != len(creations) {
		return nil, fmt.Errorf("found %d ward spell cast events, but %d ward_created events", len(spells), len(creations))
	}
	res := []baseview.Event{}
	for len(creations) > 0 {
		create := creations[0]
		spellIndex, err := firstMatching(create, spells, netIDs)
		if err != nil {
			return nil, err
		}
		if spellIndex > 0 {
			// swap with the element at the front so that we can (cheaply)
			// slice the start of the list off.
			spells[0], spells[spellIndex] = spells[spellIndex], spells[0]
		}
		spell := spells[0]
		creations = creations[1:]
		spells = spells[1:]

		itemType := spell.WardItemType()
		champ, err := netIDs.Get(spell.NetworkID)
		if err != nil {
			return nil, err
		}
		ev := &baseview.WardPlaced{
			ParticipantID: champ.ID,
			Position:      create.Position,
			WardItemType:  *itemType,
			WardID:        create.NetworkID,
			TeamID:        create.TeamID,
			WardType:      itemType.Ward().Type(),
		}
		// sometimes a ward_placed is at the exact same second as a ward_death.
		// for stats purposes, we need the "placed" to be time-sorted before the
		// death, so we subtract a very small amount from the time.
		ev.SetSeconds(create.MatchSeconds() - 0.01)
		ev.SetType("ward_placed")
		res = append(res, ev)
	}
	return res, nil
}

// firstMatching attempts to find the lowest-cost match for the ward creation
// among the ward cast events. They must be within 30 seconds of one another,
// and the cost is simply the distance difference + the time difference, where
// each second of time difference counts equally as costly as 100 distance
// units.
func firstMatching(create *OnCreate, spells []*SpellCast, netIDs *networkIDMap) (int, error) {
	minIndex := -1
	bestCost := 100000.0

	for i := 0; i < len(spells); i++ {
		spell := spells[i]
		champ, err := netIDs.Get(spell.NetworkID)
		if err != nil {
			return -1, err
		}
		if baseview.TeamID(champ.ID) != create.TeamID {
			continue
		}

		secondsDiff := math.Abs(create.MatchSeconds() - spell.MatchSeconds())
		if secondsDiff > 30 {
			break
		}

		distDiff := spell.PositionEnd.DistanceXY(create.Position)
		cost := distDiff + (secondsDiff * 100)
		if cost < bestCost {
			bestCost = cost
			minIndex = i
		}
	}
	if minIndex < 0 {
		return -1, fmt.Errorf("no spell_cast event found for ward_created at %.1f at %v", create.MatchSeconds(), create.Position)
	}
	return minIndex, nil
}

// byTime is a sort implementation that permits sorting baseview events by
// match seconds.
type byTime []baseview.Event

func (t byTime) Len() int           { return len(t) }
func (t byTime) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t byTime) Less(i, j int) bool { return t[i].Seconds() < t[j].Seconds() }

func nextPing(events []EloEvent, networkID int64) *Ping {
	for _, e := range events {
		switch t := e.(type) {
		case *Ping:
			if t.NetworkID == networkID {
				return t
			}
		}
	}
	return nil
}

// getCooldown searches through the events for a ping from the specified
// participantID, until it finds a cooldown for the specified slot greater than
// the first event in the slice.
//
// The reason we do this is that we only capture ability/skill cooldowns in the
// ping events. Worse, it seems that sometimes the first ping after an ability
// use doesn't always reflect the fact that the ability was recently used.
// Therefore, we search for the first ping within reason (up to 5 seconds after)
// that has an increased cooldown expiration.
func getCooldown(events []EloEvent, networkID int64, slotName string) float64 {
	first := events[0]

	exp := -1.0
	for _, e := range events {
		// quit if we've gone too far from the original time
		if e.MatchSeconds()-first.MatchSeconds() > 5 {
			break
		}
		if p, ok := e.(*Ping); ok {
			if p.NetworkID != networkID {
				continue
			}
			switch slotName {
			case "Q":
				exp = p.SlotQCooldownExp
			case "W":
				exp = p.SlotWCooldownExp
			case "E":
				exp = p.SlotECooldownExp
			case "R":
				exp = p.SlotRCooldownExp
			case "Summoner1":
				exp = p.SlotS1CooldownExp
			case "Summoner2":
				exp = p.SlotS2CooldownExp
			}
			if exp > e.Time() {
				return exp + first.MatchSeconds() - first.Time()
			}
		}
	}
	// if we fail to find a bigger cooldown expiration, just assume there is no
	// cooldown (it expires at time of use)
	return first.MatchSeconds()
}

func isWardCreate(name string) bool {
	switch strings.ToLower(name) {
	case "sightward", "visionward", "jammerdevice":
		return true
	}
	return false
}
