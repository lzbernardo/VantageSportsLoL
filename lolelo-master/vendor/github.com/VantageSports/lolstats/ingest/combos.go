package ingest

import (
	"container/list"
	"fmt"
	"sort"

	"github.com/VantageSports/lolstats/baseview"
)

var comboTimeout = 2.0
var damageTimeout = 2.0

type comboItem struct {
	Name string
	Time float64
}
type comboItemList list.List // Of comboItems

func newComboItemList() *comboItemList {
	return (*comboItemList)(list.New())
}

func (c *comboItemList) First() *comboItem {
	l := (*list.List)(c)
	if l.Front() != nil {
		firstComboItem, _ := l.Front().Value.(*comboItem)
		return firstComboItem
	}
	return nil
}

func (c *comboItemList) Last() *comboItem {
	l := (*list.List)(c)
	if l.Back() != nil {
		lastComboItem, _ := l.Back().Value.(*comboItem)
		return lastComboItem
	}
	return nil
}

func (c *comboItemList) String() string {
	s := ""
	l := (*list.List)(c)
	for e := l.Front(); e != nil; e = e.Next() {
		if s != "" {
			s += ","
		}
		val, _ := e.Value.(*comboItem)
		s += val.Name
	}
	return s
}

type combo struct {
	ItemList         *comboItemList
	DamageTimes      *list.List
	TotalDamageDealt float64
}

func newCombo() *combo {
	return &combo{
		ItemList:    newComboItemList(),
		DamageTimes: list.New(),
	}
}

func (c *combo) Trim(lastDamageTime float64) {
	l := (*list.List)(c.ItemList)
	for e := l.Back(); e != nil; e = e.Prev() {
		val, _ := e.Value.(*comboItem)
		if val.Time > lastDamageTime {
			l.Remove(e)
		}
	}
	numLeadingAas := 0
	for e := l.Front(); e != nil; e = e.Next() {
		val, _ := e.Value.(*comboItem)
		if val.Name == "aa" {
			numLeadingAas++
		} else {
			break
		}
	}
	for i := 0; i < numLeadingAas-1; i++ {
		l.Remove(l.Front())
	}

	numTrailingAas := 0
	for e := l.Back(); e != nil; e = e.Prev() {
		val, _ := e.Value.(*comboItem)
		if val.Name == "aa" {
			numTrailingAas++
		} else {
			break
		}
	}

	for i := 0; i < numTrailingAas-1; i++ {
		l.Remove(l.Back())
	}
}

func (c *combo) NumAbilities() int {
	numAbilities := 0
	l := (*list.List)(c.ItemList)
	for e := l.Front(); e != nil; e = e.Next() {
		val, _ := e.Value.(*comboItem)
		switch val.Name {
		case "Q", "W", "E", "R":
			numAbilities++
		}
	}
	return numAbilities
}

func (c *combo) String() string {
	mins := int(c.ItemList.First().Time / 60)
	secs := int(c.ItemList.First().Time) - (int(c.ItemList.First().Time)/60)*60
	return fmt.Sprintf("Combo %s (damage=%v) used at %v:%v (%v -> %v)", c.ItemList.String(), c.TotalDamageDealt, mins, secs, c.ItemList.First().Time, c.ItemList.Last().Time)
}

type ByComboDamage []*combo

func (c ByComboDamage) Len() int {
	return len(c)
}
func (c ByComboDamage) Swap(i, j int) {
	c[i], c[j] = c[j], c[i]
}
func (c ByComboDamage) Less(i, j int) bool {
	return c[i].TotalDamageDealt > c[j].TotalDamageDealt
}

type comboAnalysis struct {
	Combos       []*combo
	CurrentCombo *combo
	// We need this to get the summoner spell names
	Participant *baseview.Participant
}

func NewComboAnalysis(p *baseview.Participant) *comboAnalysis {
	return &comboAnalysis{
		Combos:       []*combo{},
		CurrentCombo: newCombo(),
		Participant:  p,
	}
}

func (ca *comboAnalysis) AddAttack(t *baseview.Attack) {
	comboName := ca.slotToComboName(t.Slot)
	if comboName == "" {
		return
	}

	firstComboItem := ca.CurrentCombo.ItemList.First()
	lastComboItem := ca.CurrentCombo.ItemList.Last()
	var lastNonAaComboItem *comboItem
	var lastDamageTime float64

	lst := (*list.List)(ca.CurrentCombo.ItemList)
	for e := lst.Back(); e != nil; e = e.Prev() {
		val, _ := e.Value.(*comboItem)
		if val.Name != "aa" {
			lastNonAaComboItem = val
			break
		}
	}

	if ca.CurrentCombo.DamageTimes.Back() != nil {
		lastDamageTime, _ = ca.CurrentCombo.DamageTimes.Back().Value.(float64)
	}

	// Attacks/Spells/Items used in close proximity should be considered as part of the combo
	if lastComboItem != nil &&
		// Close enough to the last attack
		(lastNonAaComboItem == nil || t.Seconds() <= lastNonAaComboItem.Time+comboTimeout) &&
		// Start of the combo, or probably did damage
		((lastDamageTime == 0 && t.Seconds() < firstComboItem.Time+damageTimeout) || t.Seconds() <= lastDamageTime+damageTimeout) {

		lst.PushBack(&comboItem{Name: comboName, Time: t.Seconds()})
	} else {
		// This attack is not part of a combo. First, check to see if there was a previous combo that we should end.
		// That means that the combo chain is > 1, and there's > 1 ability actually used (not just ability + aa)
		ca.CurrentCombo.Trim(lastDamageTime)
		if lst.Len() > 1 && ca.CurrentCombo.NumAbilities() > 1 {
			if ca.CurrentCombo.TotalDamageDealt > 0 {
				ca.Combos = append(ca.Combos, ca.CurrentCombo)
			}
		}
		// Then, start a new combo with this attack
		ca.CurrentCombo = newCombo()
		l2 := (*list.List)(ca.CurrentCombo.ItemList)
		l2.PushBack(&comboItem{Name: comboName, Time: t.Seconds()})
	}
}

func (ca *comboAnalysis) AddDamage(t *baseview.Damage) {
	if t.VictimType == baseview.ActorHero {
		ca.CurrentCombo.DamageTimes.PushBack(t.Seconds())
		ca.CurrentCombo.TotalDamageDealt += t.Total
	}
}

func (ca *comboAnalysis) slotToComboName(slot string) string {
	switch slot {
	case "basic":
		return "aa"
	case "Q", "W", "E", "R":
		// TODO: Figure out how to get the item names for these slot item usages
		// "Item1", "Item2", "Item3", "Item4", "Item5", "Item6", "Trinket":
		return slot
	case "Summoner1":
		return ca.Participant.Spell1
	case "Summoner2":
		return ca.Participant.Spell2
	default:
		return ""
	}
}

type ComboSummary struct {
	Name             string  `json:"name"`
	TotalDamageDealt float64 `json:"total_damage_dealt"`
	Begin            float64 `json:"begin"`
	End              float64 `json:"end"`
}

func (cs *ComboSummary) String() string {
	mins := int(cs.Begin / 60)
	secs := int(cs.Begin) - (int(cs.Begin)/60)*60
	return fmt.Sprintf("Combo %s (damage=%v) used at %v:%v (%v -> %v)", cs.Name, cs.TotalDamageDealt, mins, secs, cs.Begin, cs.End)
}

func (ca *comboAnalysis) ComboSummary() []*ComboSummary {
	summaries := make([]*ComboSummary, len(ca.Combos))
	sort.Sort(ByComboDamage(ca.Combos))
	for i, combo := range ca.Combos {
		summaries[i] = &ComboSummary{
			Name:             combo.ItemList.String(),
			TotalDamageDealt: combo.TotalDamageDealt,
			Begin:            combo.ItemList.First().Time,
			End:              combo.ItemList.Last().Time,
		}
	}
	return summaries
}

func (ca *comboAnalysis) ComboDamagePerMinute(matchDuration float64) float64 {
	sum := 0.0
	for _, combo := range ca.Combos {
		sum += combo.TotalDamageDealt
	}
	return sum / matchDuration * 60.0
}
