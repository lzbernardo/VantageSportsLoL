package lolobserver

import (
	"context"
	"fmt"
	"time"

	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/users/client"
)

// Lister is an abstraction that provides a list of LolUser objects. It can be
// implemented simply (flat-file-backed) or more complex(ly?) (via a wrapper
// around the LolUsers GRPC API).
type Lister interface {
	List() ([]*lolusers.LolUser, error)
}

// LolUsersVPLister is a Lister implementation that queries for all lolusers
// and then filters/returns only those that have a minimum number of vantage
// points.
type LolUsersVPLister struct {
	Client           lolusers.LolUsersClient
	MinVantagePoints int64
	InternalKey      string
}

func (lu *LolUsersVPLister) List() ([]*lolusers.LolUser, error) {
	ctx := client.SetCtxToken(context.Background(), lu.InternalKey)

	res, err := lu.Client.List(ctx, &lolusers.ListLolUsersRequest{})
	if err != nil {
		return nil, err
	}

	filtered := []*lolusers.LolUser{}
	for i := range res.LolUsers {
		u := res.LolUsers[i]
		if u.VantagePointBalance < lu.MinVantagePoints {
			continue
		}
		filtered = append(filtered, u)
	}

	unique := map[string]bool{}
	for _, u := range filtered {
		unique[u.SummonerId] = true
	}

	log.Debug(fmt.Sprintf("retrieved %d lolusers (%d unique summoners)", len(filtered), len(unique)))
	return filtered, nil
}

// FlatFileLister is a Lister implementation backed by a flat file.
type FlatFileLister struct {
	Files    *files.Client
	FilePath string
}

func (lis *FlatFileLister) List() ([]*lolusers.LolUser, error) {
	if err := lis.Files.Copy(lis.FilePath, "/tmp/users.json"); err != nil {
		return nil, err
	}

	lolusers := []*lolusers.LolUser{}
	if err := json.DecodeFile("/tmp/users.json", &lolusers); err != nil {
		return nil, err
	}

	// figure out the number of unique summoners we have (for prettier logging)
	unique := map[string]bool{}
	for _, u := range lolusers {
		unique[u.SummonerId] = true
	}

	log.Debug(fmt.Sprintf("retrieved %d lolusers (%d unique summoners)", len(lolusers), len(unique)))
	return lolusers, nil
}

// LolUsersBySummoners caches a Lister for a certain period of time.
type LolUsersBySummoners struct {
	Lister   Lister
	CacheDur time.Duration
	lastPoll time.Time
	cached   map[string][]*lolusers.LolUser
}

// BySummonerID returns a list of LolUsers mapped by summoner id.
func (l *LolUsersBySummoners) BySummonerID() map[string][]*lolusers.LolUser {
	if l.lastPoll.Add(l.CacheDur).After(time.Now()) {
		return l.cached
	}

	lolUsers, err := l.Lister.List()
	if err != nil {
		log.Error(fmt.Sprintf("error retrieving lol users: %v", err))
		return l.cached
	}

	bySummonerID := map[string][]*lolusers.LolUser{}
	for i := range lolUsers {
		lu := lolUsers[i]
		existing := bySummonerID[lu.SummonerId]
		if existing == nil {
			existing = []*lolusers.LolUser{}
		}
		existing = append(existing, lu)

		bySummonerID[lu.SummonerId] = existing
	}

	l.lastPoll = time.Now()
	l.cached = bySummonerID
	return l.cached
}

// Summary prints a string description of the lolusers specified, including
// total length, number of unique summoners, and each unique summoner id.
func Summary(users ...lolusers.LolUser) string {
	uniqueSummoners := map[string]bool{}
	summonerIDs := []string{}
	for _, lu := range users {
		sID := lu.SummonerId
		if !uniqueSummoners[sID] {
			summonerIDs = append(summonerIDs, sID)
			uniqueSummoners[sID] = true
		}
	}
	return fmt.Sprintf("%d lolusers (%d summoners: %v)", len(users), len(uniqueSummoners), summonerIDs)
}
