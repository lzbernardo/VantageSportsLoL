package lolusers

import (
	"errors"
	"math"
)

func (r *ListLolUsersRequest) Valid() error {
	if len(r.SummonerIds) > 10 || (len(r.SummonerIds) > 0 && r.Region == "") {
		return errors.New("must include region with summoner ids (no more than 10)")
	}
	return nil
}

func (l *LolUserRequest) Valid() error {
	if l.LolUser == nil || l.LolUser.UserId == "" || l.LolUser.SummonerId == "" || l.LolUser.Region == "" {
		return errors.New("request requires loluser, user_id, summoner_id, and region")
	}
	return nil
}

func (l *VantagePointsRequest) Valid() error {
	if l.UserId == "" || math.IsNaN(float64(l.Amount)) {
		return errors.New("user_id and amount are required")
	}
	return nil
}
