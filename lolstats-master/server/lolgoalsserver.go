package server

import (
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"
	"google.golang.org/api/iterator"

	"github.com/VantageSports/common/bigquery"
	"github.com/VantageSports/common/constants/privileges"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolstats"
	"github.com/VantageSports/lolstats/gcd"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/users"
	"github.com/VantageSports/users/client"
)

type GoalsServer struct {
	AuthClient     users.AuthCheckClient
	DsClient       *datastore.Client
	BqClient       *bigquery.Client
	BasicTable     string
	AdvancedTable  string
	LolUsersClient lolusers.LolUsersClient
	ProjectID      string
}

func (s *GoalsServer) CreateStats(ctx context.Context, in *lolstats.GoalCreateStatsRequest) (out *lolstats.CountResponse, err error) {
	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	if err := in.Valid(); err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, in.SummonerId, in.Platform); err != nil {
		return nil, err
	}

	percentileResults, err := s.percentileResults(ctx, in.SummonerId, in.Platform, in.RolePosition)
	if err != nil {
		return nil, err
	}

	// Look back a fixed number of games to figure out their goals
	gameWindow := int64(5)
	goalsForSummoner, err := s.findGoalTargets(ctx, in.SummonerId, in.Platform, in.RolePosition, gameWindow, in.TargetAchievementCount, percentileResults)
	if err != nil {
		return nil, err
	}

	// We want "NumGoals" goals for each category specified
	goalsAdded := map[lolstats.GoalCategory]int64{}
	for _, cat := range in.Categories {
		if cat != lolstats.GoalCategory_CATEGORY_NONE {
			goalsAdded[cat] = in.NumGoals
		}
	}
	totalGoalsAdded := int64(0)

	for _, goal := range goalsForSummoner {
		// Skip if we have enough goals for this category
		goalCategory := lolstats.GoalCategory(lolstats.GoalCategory_value[goal.Category])
		if goalsAdded[goalCategory] <= 0 {
			continue
		}

		// Make sure we're not overwriting existing goals
		keyToLookup := datastore.NameKey(gcd.KindLolGoal, goal.GenerateId(), nil)
		query := datastore.NewQuery(gcd.KindLolGoal).
			Filter("__key__ =", keyToLookup).
			KeysOnly()

		it := s.DsClient.Run(ctx, query)
		row := gcd.LolGoal{}
		_, err = it.Next(&row)
		if err == nil {
			// We found something. Skip to the next goal so we don't
			// overwrite the existing one
			continue
		} else if err != iterator.Done {
			return nil, err
		}

		err = s.createGoalInGcd(ctx, goal.GenerateId(), goal)
		if err != nil {
			return nil, err
		}
		totalGoalsAdded++
		goalsAdded[goalCategory]--

		// Check to see if all categories are satisfied. If so, we can break
		isDone := true
		for _, v := range goalsAdded {
			if v > 0 {
				isDone = false
				break
			}
		}
		if isDone {
			break
		}
	}
	return &lolstats.CountResponse{Count: totalGoalsAdded}, nil
}

func (s *GoalsServer) CreateCustom(ctx context.Context, in *lolstats.GoalCreateCustomRequest) (out *lolstats.SimpleResponse, err error) {
	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	if err := in.Valid(); err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, in.SummonerId, in.Platform); err != nil {
		return nil, err
	}

	goal := &gcd.LolGoal{
		Created:        time.Now(),
		SummonerID:     in.SummonerId,
		Platform:       in.Platform,
		UnderlyingStat: in.UnderlyingStat,
		Status:         lolstats.GoalStatus_name[int32(lolstats.GoalStatus_NEW)],
		RolePosition:   in.RolePosition,
	}

	// Make sure we're not overwriting existing goals
	keyToLookup := datastore.NameKey(gcd.KindLolGoal, goal.GenerateId(), nil)
	query := datastore.NewQuery(gcd.KindLolGoal).
		Filter("__key__ =", keyToLookup).
		KeysOnly()

	it := s.DsClient.Run(ctx, query)
	row := gcd.LolGoal{}
	_, err = it.Next(&row)
	if err == nil {
		// We found something. Skip to the next goal so we don't
		// overwrite the existing one
		return nil, fmt.Errorf("goal already exists")
	} else if err != iterator.Done {
		return nil, err
	}

	err = s.createGoalInGcd(ctx, goal.GenerateId(), goal)
	if err != nil {
		return nil, err
	}
	return &lolstats.SimpleResponse{}, nil
}

func (s *GoalsServer) createGoalInGcd(ctx context.Context, keyStr string, goal *gcd.LolGoal) error {
	key := datastore.NameKey(gcd.KindLolGoal, keyStr, nil)

	log.Info(fmt.Sprintf("Creating goal: %v\n", keyStr))
	_, err := s.DsClient.Put(ctx, key, goal)
	return err
}

func (s *GoalsServer) Get(ctx context.Context, in *lolstats.GoalGetRequest) (out *lolstats.GoalGetResponse, err error) {
	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	if err := in.Valid(); err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, in.SummonerId, in.Platform); err != nil {
		return nil, err
	}

	prefixStr := fmt.Sprintf("%v-%v", in.SummonerId, in.Platform)
	if in.RolePosition != "" {
		prefixStr = fmt.Sprintf("%v-%v", prefixStr, in.RolePosition)
		if in.ChampionId != 0 {
			prefixStr = fmt.Sprintf("%v-%v", prefixStr, in.ChampionId)
		}
	}
	keyPrefix := datastore.NameKey(gcd.KindLolGoal, prefixStr, nil)
	keyPrefixEnd := datastore.NameKey(gcd.KindLolGoal, fmt.Sprintf("%v0", prefixStr), nil)
	query := datastore.NewQuery(gcd.KindLolGoal).
		Filter("__key__ >", keyPrefix).
		Filter("__key__ <", keyPrefixEnd)

	if in.Status != lolstats.GoalStatus_STATUS_NONE {
		query = query.Filter("status =", lolstats.GoalStatus_name[int32(in.Status)])
	}
	it := s.DsClient.Run(ctx, query)

	var res = &lolstats.GoalGetResponse{Goals: []*lolstats.Goal{}}

	var row gcd.LolGoal
	for _, err = it.Next(&row); err == nil; _, err = it.Next(&row) {
		rowCopy, err := row.ToProtoGoal()
		if err != nil {
			return nil, err
		}
		res.Goals = append(res.Goals, rowCopy)
	}
	if err != iterator.Done {
		return nil, err
	}

	return res, nil
}

func (s *GoalsServer) Delete(ctx context.Context, in *lolstats.GoalDeleteRequest) (out *lolstats.SimpleResponse, err error) {
	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	if err := in.Valid(); err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, in.SummonerId, in.Platform); err != nil {
		return nil, err
	}

	key := datastore.NameKey(gcd.KindLolGoal, in.GoalId, nil)
	if err := s.DsClient.Delete(ctx, key); err != nil {
		return nil, err
	}

	return &lolstats.SimpleResponse{}, nil
}

func (s *GoalsServer) UpdateStatus(ctx context.Context, in *lolstats.GoalUpdateStatusRequest) (out *lolstats.SimpleResponse, err error) {
	claims, err := s.validateCtx(ctx, privileges.LOLstatsRead)
	if err != nil {
		return nil, err
	}

	if err := in.Valid(); err != nil {
		return nil, err
	}

	if err = s.validateSummonerAccess(ctx, claims, in.SummonerId, in.Platform); err != nil {
		return nil, err
	}

	key := datastore.NameKey(gcd.KindLolGoal, in.GoalId, nil)
	goal := gcd.LolGoal{}
	if err := s.DsClient.Get(ctx, key, &goal); err != nil {
		return nil, err
	}
	// Update the goal
	goal.Status = lolstats.GoalStatus_name[int32(in.Status)]

	if _, err := s.DsClient.Put(ctx, key, &goal); err != nil {
		return nil, err
	}

	return &lolstats.SimpleResponse{}, nil
}

func (s *GoalsServer) validateCtx(ctx context.Context, claims string) (*users.Claims, error) {
	// Don't validate tokens in dev
	if s.ProjectID == "vs-dev" {
		return nil, nil
	}
	return client.ValidateCtxClaims(ctx, s.AuthClient, claims)
}

// validateSummonerAccess verifies that the request came from that summoner
func (s *GoalsServer) validateSummonerAccess(ctx context.Context, claims *users.Claims, summonerID int64, platformID string) error {
	// Don't validate summoner access in dev
	if s.ProjectID == "vs-dev" {
		return nil
	}

	resp, err := s.LolUsersClient.List(ctx, &lolusers.ListLolUsersRequest{UserId: claims.Sub})
	if err != nil {
		return err
	}

	region := riot.RegionFromPlatform(riot.PlatformFromString(platformID))

	summonerStr := fmt.Sprintf("%d", summonerID)
	for _, u := range resp.LolUsers {
		if u.SummonerId == summonerStr && u.Region == region.String() {
			return nil
		}
	}

	return fmt.Errorf("user not allowed to request data for summoner %d", summonerID)
}
