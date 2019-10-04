package lolqueue

import (
	"strings"
	"time"
)

type position string

const (
	Top     position = "top"
	Mid     position = "mid"
	Jungle  position = "jungle"
	Adc     position = "adc"
	Support position = "support"

	any position = "any"
)

// Rank describes a player's ranking as a combination of their tier and division.
type Rank struct {
	Tier     string `json:"tier"`     // bronze, gold, diamond...
	Division string `json:"division"` // i, ii, iii, iv, v
}

var orderedTiers = []string{"bronze", "silver", "gold", "platinum", "diamond", "master", "challenger"}
var orderedDivisions = []string{"v", "iv", "iii", "ii", "i"}

// Val returns a numeric indication of rank that can be relatively compared to
// another rank.
func (r *Rank) Val() int64 {
	res := 0

	lowerTier := strings.ToLower(r.Tier)
	for i, tier := range orderedTiers {
		if lowerTier == tier {
			res = i * 10
			break
		}
	}

	lowerDiv := strings.ToLower(r.Division)
	for i, div := range orderedDivisions {
		if lowerDiv == div {
			res += i
			break
		}
	}

	return int64(res)
}

// Criteria describes a single player's match-making preferences.
type Criteria struct {
	Region      string     `json:"region"`
	MyPositions []position `json:"my_positions"`
	MinPlayers  int        `json:"min_players"`
	MinRank     Rank       `json:"mix_rank"`
	MaxRank     Rank       `json:"max_rank"`
}

func (c *Criteria) Satisfied(m *Match) bool {
	for _, p := range m.Invited {
		if c.Region != p.Region {
			return false
		}
		rankVal := p.Rank.Val()
		if rankVal < c.MinRank.Val() || rankVal > c.MaxRank.Val() {
			return false
		}
	}
	if c.MinPlayers > len(m.Invited) {
		return false
	}
	return true
}

type Player struct {
	SummonerName      string   `json:"summoner_name,omitempty"`
	SummonerID        int64    `json:"summoner_id,omitempty"`
	Region            string   `json:"region,omitempty"`
	Rank              Rank     `json:"rank"`
	Criteria          Criteria `json:"criteria"`
	InvitationPending bool     `json:"invite_pending"`
}

func (p *Player) Key() string {
	return playerKey(p.SummonerName, p.Region)
}

func playerKey(name, region string) string {
	return strings.ToLower(name + "-" + region)
}

type Match struct {
	Started   time.Time       `json:"started"`
	ID        int64           `json:"id"`
	Invited   []*Player       `json:"invited"`
	Accepted  map[string]bool `json:"accepted"`
	Positions []position      `json:"positions"`
}

//
// MATCHMAKING
//
// Each non-idle player sits in the queue awaiting a match. The player at the
// front of the queue is the player that has waited the longest to be part of
// a 'made' match (though he may have been invited to many matches that never
// ended up getting 'made').
//
// The "MatchMaker" is a visitor that visits each player in the queue from front
// to back. For each player visited, if that player has no invitations pending,
// it attempts to form a match with as many subsequent enqueued players as
// possible, such that each match participant also has no invitations pending,
// and the match meets the criteria of all involved players.
//
// MATCHMAKING CLIENTS (consumers of MatchMaker API)
//
// Clients are expected to receive a match and immediately notify all match
// invitees of their invitation, in addition to marking each player as having a
// pending invitation.
//
// Clients are further expected to remove any player that fails to accept an
// invitation and place them into the idle queue.
//
// Finally, once all invitations have been either accepted or ignored, the match
// should be re-evaluated to see if it still meets the criteria for each
// accepted player. If so, this becomes a "made" match. All involved players
// should be pulled out of the queue, marked as idle, and are sent a
// notification that their match has been made. If the match is not made, each
// accepting player is marked as having no invitation pending and they remain in
// their queue order.
//

// Make considers all possible combinations of available players (with no
// invitation pending) to try to make a match. It returns the first one it can
// make.
func Make(active ...*Player) *Match {
	for i, p := range active {
		if match := bestMatchFor([]*Player{p}, active[i+1:]); match != nil {
			return match
		}
	}

	return nil
}

// bestMatchFor returns the largest possible match containing the "in" players,
// choosing from a list of "candidate" players such that the resulting match
// contains the most number of players for whom that match fills all criteria.
// NOTE: it's possible for a match to be returned that contains no candidate
// players, but impossible for one with none of the "in" players.
func bestMatchFor(in, candidates []*Player) *Match {
	if len(in) > 10 {
		return nil
	}

	var best *Match

	for i, c := range candidates {
		consider := bestMatchFor(append(in, c), candidates[i+1:])
		if consider == nil {
			continue
		}
		if best == nil || len(best.Invited) < len(consider.Invited) {
			best = consider
		}
	}

	if best == nil {
		return IsViableMatch(in)
	}
	return IsViableMatch(best.Invited)
}

// IsViableMatch tries all position permutations for the list of supplied
// players, returning the first match which satisfies the criteria of all
// players.
func IsViableMatch(players []*Player) *Match {
	if len(players) > 5 || len(players) < 2 {
		return nil
	}

	desiredPositions := [][]position{}
	for _, p := range players {
		for _, pos := range p.Criteria.MyPositions {
			if pos == any && len(p.Criteria.MyPositions) > 1 {
				p.Criteria.MyPositions = []position{any}
				break
			}
		}
		desiredPositions = append(desiredPositions, p.Criteria.MyPositions)
	}

	positions := resolvePositions([]position{}, desiredPositions)
	if len(positions) == 0 {
		return nil
	}

	match := &Match{
		ID:        -1, // will be assigned by the server if this is a good match.
		Started:   time.Now(),
		Accepted:  map[string]bool{},
		Invited:   players,
		Positions: positions,
	}

	for _, p := range players {
		if !p.Criteria.Satisfied(match) {
			return nil
		}
	}

	return match
}

// resolvePositions tests all permutations of the fixed/desired positions to try
// to find a set of non-overlapping positions (excluding "any", which is
// obviously ignored) that includes ALL "fixed" positions, and at least one from
// each of the desired lists.
func resolvePositions(fixed []position, desired [][]position) []position {
	if len(desired) == 0 {
		return fixed
	}

	// create a position lookup table.
	fixedMap := map[position]bool{}
	for _, s := range fixed {
		fixedMap[s] = true
	}

	for i, d := range desired {
		for _, pos := range d {
			if pos != any && fixedMap[pos] {
				// this position is already taken, skip it.
				continue
			}
			tmpFixed := append(fixed, pos)
			if res := resolvePositions(tmpFixed, desired[i+1:]); len(res) == len(fixed)+len(desired) {
				return res
			}
		}
	}

	// no satisfactory configuration found.
	return nil
}
