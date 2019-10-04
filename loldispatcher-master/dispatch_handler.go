package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/VantageSports/common/files"
	vsjson "github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/api"
	"github.com/VantageSports/users/client"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"
)

type DispatchHandler struct {
	fClient          *files.Client
	luClient         lolusers.LolUsersClient
	matchDir         string
	internalKey      string
	gcdClient        *datastore.Client
	eloTopic         *pubsub.Topic
	basicIngestTopic *pubsub.Topic
}

func (dh *DispatchHandler) Handle(ctx context.Context, m *pubsub.Message) error {
	msg := messages.LolMatchDownload{}
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Error(err)
		return nil
	}
	log.Debug(string(m.Data))

	if err := msg.Valid(); err != nil {
		return err
	}

	platform := riot.PlatformFromString(msg.PlatformId)
	region := riot.RegionFromPlatform(platform)

	remotePath := fmt.Sprintf("%s/%d-%s.json", dh.matchDir, msg.MatchId, region)
	md, err := dh.getMatch(remotePath)
	if err != nil {
		return err
	}

	if md.MatchDuration < 600 {
		log.Info(fmt.Sprintf("skipping match %d-%s because it was only %d seconds (< 10 min)", md.MatchID, md.Region, md.MatchDuration))
		return nil
	}
	if len(md.ParticipantIdentities) != 10 {
		log.Info(fmt.Sprintf("skipping match %d-%s, bacause %d participants found in match detail", md.MatchID, md.Region, len(md.ParticipantIdentities)))
		return nil
	}

	lolUsers, err := dh.getLolUsers(ctx, md, region)
	if err != nil {
		return err
	}

	if err = dh.sendBasicStats(ctx, md, msg.MatchId, string(platform), remotePath); err != nil {
		return err
	}
	if len(msg.ObservedSummonerIds) > 0 && msg.Key != "" {
		if err = dh.sendElogen(ctx, lolUsers, md, remotePath, msg.Key, msg.ReplayServer); err != nil {
			return err
		}
	}

	return ctx.Err()
}

func (dh *DispatchHandler) getMatch(path string) (*api.MatchDetail, error) {
	localPath := fmt.Sprintf("%s/match.json", os.TempDir())
	defer os.RemoveAll(localPath)

	log.Debug("downloading match from " + path)
	if err := dh.fClient.Copy(path, localPath); err != nil {
		return nil, err
	}
	md := &api.MatchDetail{}
	err := vsjson.DecodeFile(localPath, md)
	return md, err
}

func (dh *DispatchHandler) getLolUsers(ctx context.Context, md *api.MatchDetail, region riot.Region) ([]*lolusers.LolUser, error) {
	// get summoner ids
	summonerIDs := []int64{}
	for _, p := range md.ParticipantIdentities {
		summonerIDs = append(summonerIDs, p.Player.SummonerID)
	}
	log.Info(fmt.Sprintf("querying for summoners %v in region %v", summonerIDs, region))

	authCtx := client.SetCtxToken(ctx, dh.internalKey)
	res, err := dh.luClient.List(authCtx, &lolusers.ListLolUsersRequest{
		SummonerIds: summonerIDs,
		Region:      region.String(),
	})
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("found %d lolusers for match %d", len(res.LolUsers), md.MatchID))
	return res.LolUsers, nil
}

func (dh *DispatchHandler) sendBasicStats(ctx context.Context, md *api.MatchDetail, matchID int64, platformID, matchDetailPath string) error {
	summonerIDs := []int64{}
	for _, p := range md.ParticipantIdentities {
		summonerIDs = append(summonerIDs, p.Player.SummonerID)
	}

	// add a basic ingest message for each summoner
	for _, sID := range uniqueInts(summonerIDs...) {
		msg := messages.LolBasicStatsIngest{
			MatchDetailsPath: matchDetailPath,
			MatchId:          matchID,
			PlatformId:       platformID,
			SummonerId:       sID,
		}
		if err := msg.Valid(); err != nil {
			return err
		}

		log.Info(fmt.Sprintf("sending basic stats for summoner %d and match %d-%s", sID, matchID, platformID))
		if err := addTopicMsg(ctx, dh.basicIngestTopic, msg); err != nil {
			return err
		}
	}
	return nil
}

func (dh *DispatchHandler) sendElogen(ctx context.Context, users []*lolusers.LolUser, md *api.MatchDetail, matchDetailsPath, encryptionKey, spectatorServer string) error {
	summonerIDs := []int64{}

	for _, u := range users {
		if u.VantagePointBalance < int64(matchCostVP) {
			log.Info(fmt.Sprintf("skipping loluser %s (user: %s) for insufficient VP balance: %d", u.Id, u.UserId, u.VantagePointBalance))
			continue
		}
		intSummoner, err := strconv.ParseInt(u.SummonerId, 10, 64)
		if err != nil {
			log.Error("error converting summoner id " + u.SummonerId + " " + err.Error())
			continue
		}

		exists, err := SaveMatchReceipt(ctx, dh.gcdClient, md.PlatformID, md.MatchID, intSummoner)
		if err != nil {
			log.Warning(fmt.Sprintf("failed to save match receipt (%d-%s) for summoner (%d): %v", md.MatchID, md.PlatformID, intSummoner, err))
			return err
		}

		if !exists {
			log.Info(fmt.Sprintf("charging user %s for match %d-%s", u.UserId, md.MatchID, md.PlatformID))
			authCtx := client.SetCtxToken(ctx, dh.internalKey)
			_, err = dh.luClient.AdjustVantagePoints(authCtx, &lolusers.VantagePointsRequest{
				UserId: u.UserId,
				Amount: int64(0 - matchCostVP),
			})
			if err != nil {
				// It's likely that the VP adjustment failed. Meaning that the user
				// DID have enough vantage points at the start of this loop block,
				// but now doesn't. We fail this request with an error so that it
				// is retried shortly (and the user will be skipped if they have
				// insufficient VP).
				log.Error(fmt.Sprintf("failed to charge loluser (user: %s loluserid: %s): %v", u.UserId, u.Id, err))
				return err
			}
		}

		summonerIDs = append(summonerIDs, intSummoner)
	}

	if len(summonerIDs) == 0 {
		return nil
	}

	log.Info(fmt.Sprintf("adding advanced stats for match %d-%s and summoners %v", md.MatchID, md.PlatformID, summonerIDs))
	msg := messages.LolReplayDataExtract{
		GameLengthSeconds:      md.MatchDuration,
		IncreasedPlaybackSpeed: true,
		Key:              encryptionKey,
		MatchDetailsPath: matchDetailsPath,
		MatchId:          md.MatchID,
		MatchVersion:     md.MatchVersion,
		PlatformId:       md.PlatformID,
		SpectatorServer:  spectatorServer,
		SummonerIds:      uniqueInts(summonerIDs...),
	}
	if err := msg.Valid(); err != nil {
		return err
	}

	return addTopicMsg(ctx, dh.eloTopic, msg)
}

func addTopicMsg(ctx context.Context, topic *pubsub.Topic, v interface{}) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	_, err = topic.Publish(ctx, &pubsub.Message{Data: data})
	return err
}

const KindMatchPurchaseReceipt = "MatchPurchaseReceipt"

type MatchReceipt struct {
	MatchID    int64  `datastore:"match_id"`
	PlatformID string `datastore:"platform_id"`
	SummonerID int64  `datastore:"summoner_id"`
}

func SaveMatchReceipt(ctx context.Context, gc *datastore.Client, platformID string, matchID, summonerID int64) (exists bool, err error) {
	keyName := fmt.Sprintf("%d-%s-%d", matchID, platformID, summonerID)
	key := datastore.NameKey(KindMatchPurchaseReceipt, keyName, nil)

	receipt := &MatchReceipt{}
	err = gc.Get(ctx, key, receipt)
	if err == nil {
		// This match has already been purchased by this summoner.
		return true, nil
	} else if err != datastore.ErrNoSuchEntity {
		return false, err
	}

	// Mark this match as purchased by this summoner.
	_, err = gc.Put(ctx, key, &MatchReceipt{
		MatchID:    matchID,
		PlatformID: platformID,
		SummonerID: summonerID,
	})
	return false, err
}

func uniqueInts(ints ...int64) []int64 {
	seen := map[int64]bool{}
	res := []int64{}
	for _, i := range ints {
		if seen[i] {
			continue
		}
		seen[i] = true
		res = append(res, i)
	}
	return res
}
