// contains library and utilty methods for extracting elo events from the text
// format emitted by the elobuddy DataSpectator program.

package parse

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	//"github.com/VantageSports/common/json"
	"github.com/VantageSports/lolelo/event"

	"github.com/mitchellh/mapstructure"
)

var intRE = regexp.MustCompile(`^-?[0-9]+$`)
var floatRE = regexp.MustCompile(`^-?[0-9]*(\.[0-9]+)?$`)
var positionRE = regexp.MustCompile(`^X:(.+) Y:(.+) Z:(.+)$`)
var quotedStringRE = regexp.MustCompile(`^"([^"]*)"$`)

// LogFile parses an array of event.EloEvent from the text file at the specified
// path. It does so in two steps.
// 1) it iterates through all lines looking for mappings from network id -> name
// 2) it uses that mapping to parse events.
func LogFile(filepath string) ([]event.EloEvent, error) {
	events := []event.EloEvent{}

	it, err := NewFieldsIterator(filepath)
	defer it.Close()
	if err != nil {
		return nil, err
	}

	for {
		lineNum, m, err := it.Next()
		if err != nil {
			return nil, err
		}
		if lineNum < 0 {
			break // EOF
		}
		event, err := EloEvent(m)
		if err != nil {
			return nil, fmt.Errorf("error parsing event from line %d: %v", lineNum, err)
		}
		if event != nil {
			events = append(events, event)
		}
	}

	return events, nil
}

// fields returns a map containing the fields in a elogen line.
func fields(s string) (map[string]interface{}, error) {
	fields := strings.Split(strings.TrimSpace(s), "\t")
	if len(fields) < 1 {
		return nil, fmt.Errorf("must have at least 1 field")
	}

	if len(fields)%2 != 1 {
		fmt.Println(fields)
		return nil, fmt.Errorf("expected odd number of elements, found %d", len(fields))
	}

	m := map[string]interface{}{"event": fields[0]}

	for i := 1; i < len(fields); i += 2 {
		k := removeQuotes(fields[i])

		v, err := getVal(fields[i+1])
		if err != nil {
			return nil, err
		}

		m[k] = v
	}

	return m, nil
}

// EloEvent parses an event.EloEvent from a generic json map.
func EloEvent(m map[string]interface{}) (event.EloEvent, error) {
	eventType := m["event"]
	if eventType == nil {
		return nil, fmt.Errorf("no 'event' key for event\n%+v", m)
	}
	eventTypeStr, ok := eventType.(string)
	if !ok {
		return nil, fmt.Errorf("unexpected 'event' value for event: %v", eventType)
	}

	var e event.EloEvent
	switch eventTypeStr {
	case "BASIC_ATTACK":
		e = &event.BasicAttack{}
	case "CHAMP_DIE":
		e = &event.ChampDie{}
	case "CHAMP_KILL":
		e = &event.ChampKill{}
	case "DAMAGE":
		e = &event.Damage{}
	case "DIE":
		e = &event.Die{}
	case "GAME_END":
		e = &event.GameEnd{}
	case "ID_BARRACKS", "ID_HERO", "ID_TURRET":
		e = &event.NetworkIDMapping{EloType: eventTypeStr}
	case "KILL":
		e = &event.Kill{}
	case "LEVEL_UP":
		e = &event.LevelUp{}
	case "NEXUS_DESTROYED":
		e = &event.NexusDestroyed{}
	case "ON_CREATE":
		e = &event.OnCreate{}
	case "ON_DELETE":
		e = &event.OnDelete{}
	case "PING":
		e = &event.Ping{}
	case "SPELL_CAST":
		e = &event.SpellCast{}
	case "END_GAME", "GAME_STALL", "DAMPENER_RESPAWN", "DAMPENER_RESPAWN_SOON", "SURRENDER_AGREED":
		// Do nothing
		return nil, nil
	default:
		return nil, fmt.Errorf("unepected event type (%s) for event", eventTypeStr)
	}

	err := mapstructure.Decode(m, e)
	return e, err
}

func removeQuotes(s string) string {
	if strings.HasPrefix(s, `"`) && strings.HasSuffix(s, `"`) {
		return s[1 : len(s)-1]
	}
	return s
}

func getVal(s string) (interface{}, error) {
	if intRE.MatchString(s) {
		return strconv.ParseInt(s, 10, 64)
	}

	if floatRE.MatchString(s) {
		return strconv.ParseFloat(s, 64)
	}

	switch strings.ToLower(s) {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	if matches := positionRE.FindAllStringSubmatch(s, -1); len(matches) > 0 {
		return parseLocation(matches[0])
	}

	if quotedStringRE.MatchString(s) {
		return removeQuotes(s), nil
	}
	return removeQuotes(s), nil
}

func parseLocation(strs []string) (map[string]interface{}, error) {
	x, err := strconv.ParseFloat(strs[1], 64)
	if err != nil {
		return nil, err
	}
	y, err := strconv.ParseFloat(strs[2], 64)
	if err != nil {
		return nil, err
	}
	z, err := strconv.ParseFloat(strs[3], 64)
	if err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"x": x,
		"y": y,
		"z": z,
	}, nil
}

//
// i/o utility
//

// lineFieldIterator reads lines of elotext, composes generic json maps from
// the key/value pairs, and returns one (with the line number it came from)
// with each call to Next()
type lineFieldIterator struct {
	fileHandle *os.File
	scanner    *bufio.Scanner
	lineNum    int
}

func NewFieldsIterator(filepath string) (*lineFieldIterator, error) {
	f, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}

	return &lineFieldIterator{
		fileHandle: f,
		scanner:    bufio.NewScanner(f),
		lineNum:    0,
	}, nil
}

func (i *lineFieldIterator) Next() (int, map[string]interface{}, error) {
	if !i.scanner.Scan() {
		return -1, nil, i.scanner.Err()
	}
	i.lineNum++
	m, err := fields(i.scanner.Text())
	return i.lineNum, m, err
}

func (i *lineFieldIterator) Close() error {
	return i.fileHandle.Close()
}
