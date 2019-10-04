package lolobserver

import (
	"strings"
	"testing"
	"time"

	"gopkg.in/tylerb/is.v1"

	"github.com/VantageSports/lolusers"
)

func TestSummary(t *testing.T) {
	lolUsers := []lolusers.LolUser{
		{SummonerId: "123", Id: "1234"},
		{SummonerId: "789", Id: "7890"},
		{SummonerId: "456", Id: "4560"},
		{SummonerId: "123", Id: "1235"},
		{SummonerId: "456", Id: "4561"},
	}

	summaryStr := Summary(lolUsers...)
	// map iteration order is undefined, so just check the prefix and then
	// search for each id individually.
	expectedPrefix := "5 lolusers (3 summoners: ["
	if !strings.HasPrefix(summaryStr, expectedPrefix) {
		t.Errorf("incorrect summary string: %s", summaryStr)
	}
	if !strings.Contains(summaryStr, "123") ||
		!strings.Contains(summaryStr, "456") ||
		!strings.Contains(summaryStr, "789") {
		t.Error("missing 1+ summoner ids in summary:", summaryStr)
	}
}

func TestLolUsersBySummoners(t *testing.T) {
	is := is.New(t)

	l := &testLister{
		users: []*lolusers.LolUser{
			{SummonerId: "123", Region: "na", Id: "2345"},
			{SummonerId: "123", Region: "na", Id: "5678"},
			{SummonerId: "123", Region: "na", Id: "7890"},
			{SummonerId: "234", Region: "euw", Id: "6543"},
			{SummonerId: "234", Region: "euw", Id: "5432"},
			{SummonerId: "45", Region: "foobar", Id: "9012"},
		},
	}

	bySummoners := &LolUsersBySummoners{
		Lister:   l,
		CacheDur: time.Minute,
	}
	m := bySummoners.BySummonerID()
	is.Equal(3, len(m))
	is.Equal(3, len(m["123"]))
	is.Equal(2, len(m["234"]))
	is.Equal("2345", m["123"][0].Id)
	is.Equal("7890", m["123"][2].Id)
}

// Test utilities

type testLister struct {
	users []*lolusers.LolUser
	err   error
}

func (t *testLister) List() ([]*lolusers.LolUser, error) {
	return t.users, t.err
}
