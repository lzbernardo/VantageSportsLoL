package server

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"github.com/hashicorp/golang-lru"
	"golang.org/x/net/context"

	"github.com/VantageSports/common/constants/privileges"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/service"
	"github.com/VantageSports/tasks"
	"github.com/VantageSports/users"
	"github.com/VantageSports/users/client"
)

type LolUserService struct {
	RiotProxy           service.RiotClient
	GcdClient           *datastore.Client
	AuthChecker         users.AuthCheckClient
	Users               users.UsersClient
	Emailer             tasks.EmailClient
	CrawlSummonersCache *lru.Cache
	PubTopic            *pubsub.Topic
	InternalKey         string
	LolActiveGroupID    string
}

// CrawlSummoners takes a list of summoner ids and submits Summoner Crawl
// messages to pubsub assuming one hasn't been submitted for the summoner
// id in the last 20 minutes
func (s *LolUserService) CrawlSummoners(ctx context.Context, req *lolusers.CrawlSummonersRequest) (*lolusers.SimpleResponse, error) {
	claims, err := client.ValidateCtxClaims(ctx, s.AuthChecker)
	if err != nil {
		return nil, err
	}
	crawlsInitiated := 0
	var returnErr error
	for _, summonerID := range req.SummonerIds {
		platform := riot.PlatformFromString(req.Platform)
		region := riot.RegionFromPlatform(platform)
		lolusers, err := fetchLolUsersBySummonerIds(ctx, s.GcdClient, region.String(), []int64{summonerID})
		if err != nil {
			break
		}
		userOwnsSummoner := false
		for _, l := range lolusers {
			if l.UserID == claims.Sub {
				userOwnsSummoner = true
			}
		}
		if userOwnsSummoner {
			key := fmt.Sprintf("%d - %s", summonerID, req.Platform)
			timeV, found := s.CrawlSummonersCache.Get(key)
			if found {
				if time.Now().Sub(timeV.(time.Time)).Minutes() > 20 {
					found = false
				}
			}
			if !found {
				if err := s.publishCrawlMessage(summonerID, req.Platform); err == nil {
					s.CrawlSummonersCache.Add(key, time.Now())
					crawlsInitiated++
				} else {
					log.Error(err)
					returnErr = err
				}
			}
		} else {
			log.Error("user does not own summoner")
		}
	}
	log.Info(fmt.Sprintf("Crawls Initiated: %d", crawlsInitiated))
	return &lolusers.SimpleResponse{}, returnErr
}

// Create registers a summoner id mapped to a user id, all subsequent changes
// to the LolUser will be done with the Update func
func (s *LolUserService) Create(ctx context.Context, req *lolusers.LolUserRequest) (*lolusers.LolUserResponse, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}
	claims, err := client.ValidateCtxClaims(ctx, s.AuthChecker)
	if err != nil {
		return nil, err
	}
	requested := req.LolUser
	if claims.Sub != requested.UserId && !claims.HasPrivilege(privileges.LOLUsersAdmin) {
		return nil, errors.New("not authorized")
	}
	sID, err := strconv.ParseInt(requested.SummonerId, 10, 64)
	if err != nil {
		return nil, err
	}
	_, err = s.RiotProxy.SummonersById(ctx, &service.SummonerIDRequest{Ids: []int64{sID}, Region: requested.Region})
	if err != nil && !claims.HasPrivilege(privileges.LOLUsersAdmin) {
		return nil, logRiotErr(err)
	}
	existing, err := fetchLolUserByUserId(ctx, s.GcdClient, requested.UserId)
	if err != nil {
		return nil, err
	}
	if len(existing) > 0 {
		return nil, errors.New("user id already exists")
	}
	lolUser := &LolUser{Confirmed: false, VantagePointBalance: 0}
	err = remarshalAs(requested, &lolUser)
	if err != nil {
		return nil, err
	}
	err = putLolUser(ctx, s.GcdClient, lolUser, 0, false)
	if err != nil {
		return nil, err
	}
	// add them to the LolActive group so they have lolstats_read and
	// sendtoken privileges
	associationRequest := &users.AssociateRequest{UserId: requested.UserId, GroupId: s.LolActiveGroupID}
	ctx = client.SetCtxToken(ctx, s.InternalKey)
	if _, err = s.Users.CreateAssociation(ctx, associationRequest); err != nil {
		// just log if there is an error here we've already created
		// a user and loluser, same with below
		log.Error(err)
	}
	// send confirm email to new Basic user encouraging them to upgrade to elite
	if err = s.sendConfirmEmail(ctx, requested.UserId, lolUser.Region); err != nil {
		log.Error(err)
	}
	out := &lolusers.LolUserResponse{LolUser: &lolusers.LolUser{}}
	err = remarshalAs(lolUser, &out.LolUser)
	return out, err
}

// Update will update the LolUser in different ways depending on the
// contents of the request assuming the permissions are correct
// if confirmed is true in the requeset, we will check if a summoner
// changed a mastery page to the verification string
// we provided them, updates them to confirmed if they have returning the
// summoner or else it returns an error
// TODO(Scott): what else do we allow to be updated?
func (s *LolUserService) Update(ctx context.Context, req *lolusers.LolUserRequest) (*lolusers.LolUserResponse, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}
	claims, err := client.ValidateCtxClaims(ctx, s.AuthChecker)
	if err != nil {
		return nil, err
	}
	requested := req.LolUser
	if claims.Sub != requested.UserId && !claims.HasPrivilege(privileges.LOLUsersAdmin) {
		return nil, errors.New("not authorized")
	}
	lolUsers, err := fetchLolUserByUserId(ctx, s.GcdClient, requested.UserId)
	if err != nil {
		return nil, err
	}
	if len(lolUsers) == 0 {
		return nil, errors.New("user not found")
	}
	lolUser := lolUsers[0]
	if claims.HasPrivilege(privileges.LOLUsersAdmin) {
		err := remarshalAs(requested, &lolUser)
		if err != nil {
			return nil, err
		}
	} else if requested.Confirmed {
		// get the LolUser mastery pages and check to see if a page matches
		// the verification string
		sID, err := strconv.ParseInt(lolUser.SummonerID, 10, 64)
		if err != nil {
			return nil, err
		}
		masteryMap, err := s.RiotProxy.Masteries(ctx, &service.SummonerIDRequest{Ids: []int64{sID}, Region: riot.Region(lolUser.Region).String()})
		if err != nil {
			return nil, logRiotErr(err)
		}
		if len(masteryMap.NamedMasteries) == 0 {
			return nil, errors.New("expected to find at least one named mastery")
		}
		err = confirmMasteryNameChanged(masteryMap.NamedMasteries[0], lolUser.ID)
		if err != nil {
			return nil, err
		}
		// update this loluser as confirmed
		lolUser.Confirmed = true
	}
	err = putLolUser(ctx, s.GcdClient, lolUser, 0, false)
	if err != nil {
		return nil, err
	}
	out := &lolusers.LolUserResponse{LolUser: &lolusers.LolUser{}}
	err = remarshalAs(lolUser, &out.LolUser)
	return out, err
}

// List fetches lolusers given a user id or summoner id, and/or region
func (s *LolUserService) List(ctx context.Context, req *lolusers.ListLolUsersRequest) (*lolusers.LolUsersResponse, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}
	claims, err := client.ValidateCtxClaims(ctx, s.AuthChecker)
	if err != nil {
		return nil, err
	}
	if claims.Sub != req.UserId && !claims.HasPrivilege(privileges.LOLUsersAdmin) {
		return nil, errors.New("not authorized")
	}
	var lolUsers []*LolUser
	if req.UserId != "" {
		lolUsers, err = fetchLolUserByUserId(ctx, s.GcdClient, req.UserId)
	} else {
		lolUsers, err = fetchLolUsersBySummonerIds(ctx, s.GcdClient, req.Region, req.SummonerIds)
	}
	if err != nil {
		return nil, err
	}

	out := &lolusers.LolUsersResponse{LolUsers: []*lolusers.LolUser{}}
	err = remarshalAs(lolUsers, &out.LolUsers)
	return out, err
}

// AdjustVantagePoints adds to the vantage point balance of
// the loluser, only be can be done by someone with LolAdmin rights
func (s *LolUserService) AdjustVantagePoints(ctx context.Context, req *lolusers.VantagePointsRequest) (*lolusers.SimpleResponse, error) {
	if err := req.Valid(); err != nil {
		return nil, err
	}
	if _, err := client.ValidateCtxClaims(ctx, s.AuthChecker, privileges.LOLUsersAdmin); err != nil {
		return nil, errors.New("not authorized")
	}
	lolUsers, err := fetchLolUserByUserId(ctx, s.GcdClient, req.UserId)
	if err != nil {
		return nil, err
	}
	if len(lolUsers) == 0 {
		return nil, fmt.Errorf("no user found to adjust %d vantage points, user_id: %s", req.Amount, req.UserId)
	}
	err = putLolUser(ctx, s.GcdClient, lolUsers[0], req.Amount, req.Absolute)
	return &lolusers.SimpleResponse{}, err
}

// sendConfirmEmail sends a confirmation email using an email template to
// the user owning the passed userId
func (s *LolUserService) sendConfirmEmail(ctx context.Context, userId, region string) error {
	listUsersRequest := &users.ListUsersRequest{Ids: []string{userId}}
	usersList, err := s.Users.ListUsers(ctx, listUsersRequest)
	if err != nil {
		return err
	}
	if len(usersList.Users) != 1 {
		return fmt.Errorf("Expected 1 user for user id %s, found %d", userId, len(usersList.Users))
	}
	user := usersList.Users[0]
	emailTemplate := "elite_welcome.html"
	if region == "kr" {
		emailTemplate = "korea_elite_welcome.html"
	}

	var data interface{}
	htmlBody, err := parseTemplate(fmt.Sprintf("email_templates/%s", emailTemplate), data)
	if err != nil {
		return err
	}

	emailReq := &tasks.EmailRequest{
		Email: &messages.Email{
			Emails:   []string{user.Email},
			FromName: "Vantage Sports",
			FromAddr: "service@vantagesports.com",
			Subject:  "Your Vantage Basic Membership",
			HtmlBody: htmlBody,
		},
	}
	_, err = s.Emailer.Send(ctx, emailReq)
	return err
}

func (s *LolUserService) publishCrawlMessage(summonerId int64, platform string) error {
	msg := messages.LolSummonerCrawl{
		SummonerId:  summonerId,
		PlatformId:  platform,
		HistoryType: "recent",
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	_, err = s.PubTopic.Publish(context.Background(), &pubsub.Message{
		Data: data,
	})
	return err
}

// parseTemplate creates an html string for sending an email given a template
func parseTemplate(templateFileName string, data interface{}) (string, error) {
	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		return "", err
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		return "", err
	}
	return buf.String(), err
}

// checks if the summoner changed a mastery page to the given name
func confirmMasteryNameChanged(masteryPages *service.NamedMasteries, verification string) error {
	confirmed := false
	for _, page := range masteryPages.MasteryPages {
		if page.Name == verification {
			confirmed = true
			break
		}
	}
	if !confirmed {
		return fmt.Errorf("Did not find a master page named %s", verification)
	}
	return nil
}

// logRiotErr logs the error and then just returns an error
// saying the riot api returned an error
func logRiotErr(err error) error {
	log.Info(err)
	return errors.New("riot error encountered")
}
