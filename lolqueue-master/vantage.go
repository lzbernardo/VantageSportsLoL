package lolqueue

// This file contains a collection of utility functions for interacting with
// internal vantage APIs to ascertain identity information.

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/net/context"

	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/payment"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/service"
	"github.com/VantageSports/users"
	"github.com/VantageSports/users/client"
)

var vantageUtil *VantageUtil

type VantageUtil struct {
	Users    users.AuthCheckClient
	LolUsers lolusers.LolUsersClient
	Riot     service.RiotClient
	Payment  payment.PaymentClient
	Fake     bool
}

func (vu *VantageUtil) UserID(token string) (string, error) {
	if vu.Fake {
		return "123456", nil
	}

	claimsRes, err := vu.Users.CheckToken(context.Background(), &users.TokenRequest{Token: token})
	if err != nil {
		return "", err
	}
	return claimsRes.Claims.Sub, nil
}

// HasAccess returns true iff the user is paying for elite.
func (vu *VantageUtil) HasAccess(token, userID string) (bool, error) {
	if vu.Fake {
		return true, nil
	}

	ctx := client.SetCtxToken(context.Background(), token)
	subRes, err := vu.Payment.ListSubscriptions(ctx, &payment.SubscriptionsListRequest{
		UserId: userID,
	})
	if err != nil {
		return false, err
	}

	for _, sub := range subRes.Subscriptions {
		if sub.Plan == "lol-paid-01" {
			return true, nil
		}
	}

	return false, fmt.Errorf("user not subscribed to lol-paid-01 plan")
}

var fakeIndex = 0
var fakeSummoners = []*Player{
	{SummonerName: "BentleyJ", SummonerID: 133177734, Rank: Rank{Tier: "Silver", Division: "V"}, Region: "na"},
	{SummonerName: "BrettMcD22", SummonerID: 1312377, Rank: Rank{Tier: "Bronze", Division: "IV"}, Region: "na"},
	{SummonerName: "Lumix3", SummonerID: 71117718, Rank: Rank{Tier: "Gold", Division: "III"}, Region: "na"},
	{SummonerName: "Savvee", SummonerID: 581133, Rank: Rank{Tier: "Bronze", Division: "II"}, Region: "na"},
	{SummonerName: "Scotty2Hotty10", SummonerID: 530774, Rank: Rank{Tier: "Bronze", Division: "V"}, Region: "na"},
}

// Vantage Util Functions
func (vu *VantageUtil) Player(token, userID string) (*Player, error) {
	if vu.Fake {
		fakeIndex = (fakeIndex + 1) % len(fakeSummoners)
		return fakeSummoners[fakeIndex], nil
	}

	summonerID, name, region, err := vu.summoner(token, userID)
	if err != nil {
		return nil, err
	}

	rank, err := vu.getBestRank(token, summonerID, region,
		riot.RANKED_FLEX_SR,
		riot.RANKED_SOLO_5x5,
		riot.TB_RANKED_SOLO,
		riot.TB_RANKED_SOLO_5x5)
	if err != nil {
		return nil, err
	}

	return &Player{
		Rank:         *rank,
		Region:       region,
		SummonerName: name,
		SummonerID:   summonerID,
	}, nil

}

func (vu *VantageUtil) summoner(token, userID string) (id int64, name string, region string, err error) {
	id = -1

	ctx := client.SetCtxToken(context.Background(), token)
	lu, err := vu.LolUsers.List(ctx, &lolusers.ListLolUsersRequest{
		UserId: userID,
	})
	if err == nil && len(lu.LolUsers) < 1 {
		err = fmt.Errorf("no lolusers found for user: %s", userID)
	}
	if err != nil {
		return id, name, region, err
	}

	id, err = strconv.ParseInt(lu.LolUsers[0].SummonerId, 10, 64)
	if err != nil {
		return id, name, region, err
	}

	summonerRes, err := vu.Riot.SummonersById(ctx, &service.SummonerIDRequest{
		Ids:    []int64{id},
		Region: lu.LolUsers[0].Region,
	})
	if err == nil && len(summonerRes.Summoners) != 1 {
		err = fmt.Errorf("expected 1 summoner with id: %d, found %d", id, len(summonerRes.Summoners))
	}
	if err != nil {
		return id, name, region, err
	}

	return id, summonerRes.Summoners[0].Name, lu.LolUsers[0].Region, nil
}

func (vu *VantageUtil) getBestRank(token string, summonerID int64, region string, queues ...riot.QueueType) (*Rank, error) {
	ctx := client.SetCtxToken(context.Background(), token)
	leagueEntryRes, err := vu.Riot.LeagueEntryBySummoner(ctx, &service.SummonerIDRequest{
		Ids:    []int64{summonerID},
		Region: region,
	})
	if err != nil && strings.Contains(err.Error(), "code: 404") {
		// This just means they're unranked. We'll pretend no leagues were
		// returned and they'll be assigned to WOOD V.
		err = nil
		leagueEntryRes = &service.LeaguesResponse{}
	}
	if err != nil {
		return nil, err
	}

	ranks := []Rank{{Tier: "WOOD", Division: "V"}}
	for _, league := range leagueEntryRes.NamedLeagues {
		for _, l := range league.Leagues {
			for _, q := range queues {
				if l.Queue == string(q) {
					for _, entry := range l.Entries {
						ranks = append(ranks, Rank{Tier: l.Tier, Division: entry.Division})
					}
				}
			}
		}
	}

	for i := range ranks {
		if ranks[i].Val() > ranks[0].Val() {
			ranks[0] = ranks[i]
		}
	}
	return &ranks[0], nil
}
