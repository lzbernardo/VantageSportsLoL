// The UserWatcher reads users from some source (the lolUsersClient, a flat
// file, etc) and consults riot's CurrentGame API to determine if any users are
// in a match. If so, it emits the match description via a channel for
// any subscribers.

// Optimization ideas (when needed):
// * Run each platform's API in its own goroutine, so that all platforms run
//   simultaneously.

package lolobserver

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/golang-lru"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/api"
)

// MatchUsers describes a match and the vantage subscribers that are playing
// in that match.
type MatchUsers struct {
	CurrentGame api.CurrentGameInfo
	LolUsers    []lolusers.LolUser
	Observed    bool
}

type UserWatcher struct {
	LolUsers *LolUsersBySummoners
	Api      api.APIs
	Done     bool
}

// Start begins the user_watcher in its own goroutine, returning a channel of
// all "discovered" matches, along with the list of all the lolusers that are
// participating in that match.
func (uw *UserWatcher) Start() <-chan *MatchUsers {
	out := make(chan *MatchUsers, 20)
	seen, err := lru.New(10000)

	go func() {
		if err != nil {
			log.Critical(err)
		}

		for !uw.Done {
			started := time.Now()

			userMap := uw.LolUsers.BySummonerID()
			for summonerID, users := range userMap {
				region := riot.RegionFromString(users[0].Region)
				platform := riot.PlatformFromRegion(region)

				sID, err := strconv.ParseInt(summonerID, 10, 64)
				if err != nil {
					log.Error(fmt.Sprintf("error converting summonerId %s: %v", summonerID, err))
					continue
				}

				api, err := uw.Api.Region(string(region))
				if err != nil {
					log.Error(fmt.Sprintf("error getting api for region %s: %v", users[0].Region, err))
					continue
				}

				cg, err := api.CurrentGame(string(platform), sID)
				if err != nil {
					handleRiotErr(err, summonerID)
					continue
				}

				cacheKey := fmt.Sprintf("%d-%s", cg.GameID, platform)
				if _, found := seen.Get(cacheKey); found || cg.GameID <= 0 {
					continue
				}
				out <- &MatchUsers{
					CurrentGame: cg,
					LolUsers:    lolUsersInGame(cg.Participants, userMap),
				}
				seen.Add(cacheKey, true)
			}

			handleElapsed(started)
		}
		close(out)
	}()

	return out
}

// lolUsersInGame examines the participants in the current match and determines
// which of them are actual customers (lolusers). Returns a list of all the
// lolusers (which may outnumber the participants, since we allow many lolusers
// per summoner) participating in the game.
func lolUsersInGame(participants []api.CurrentGameParticipant, users map[string][]*lolusers.LolUser) []lolusers.LolUser {
	res := []lolusers.LolUser{}
	for _, p := range participants {
		summonerID := fmt.Sprintf("%d", p.SummonerID)
		for _, s := range users[summonerID] {
			s := s
			res = append(res, *s)
		}
	}
	return res
}

// handleRiotErr determines whether to log or sleep based on the status code
// returned from riot.
func handleRiotErr(err error, summonerID string) {
	msg := fmt.Sprintf("riot error. skipping summoner %s: %v", summonerID, err)
	if apiErr, ok := err.(api.APIError); ok {
		switch apiErr.Code() {
		case 400, 401, 415:
			log.Warning(msg)
		case 429:
			log.Warning(msg)
			time.Sleep(time.Second * 5)
		case 404:
			return
		case 500, 503:
			log.Debug(msg)
			return
		default:
			log.Warning(msg)
		}
	}
	log.Warning(fmt.Sprintf("non-riot api error for summoner: %s: %v", summonerID, err))
}

// handleElapsed either sleeps or logs an error depending on how long it has
// been since start.
func handleElapsed(start time.Time) {
	elapsed := time.Since(start)

	if elapsed > time.Minute*3 {
		log.Critical(fmt.Sprintf("took %v, we may be missing games", elapsed))
	} else if elapsed < time.Minute {
		time.Sleep(time.Minute)
	}
}
