package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"cloud.google.com/go/datastore"
	"golang.org/x/net/context"

	"github.com/VantageSports/common/log"
)

const (
	kindLolUser             = "LolUser"
	kindPurchaseMatchRecord = "PurchaseMatchRecord"
)

// LolUser is the "store" model
type LolUser struct {
	ID                  string `json:"id" datastore:"-"`
	UserID              string `json:"user_id" datastore:"user_id"`
	SummonerID          string `json:"summoner_id" datastore:"summoner_id"`
	Region              string `json:"region" datastore:"region"`
	Verification        string `json:"verification" datastore:"verification,noindex"`
	VantagePointBalance int64  `json:"vantage_point_balance" datastore:"vantage_point_balance,noindex"`
	Confirmed           bool   `json:"confirmed" datastore:"confirmed,noindex"`
}

// fetchLolUser fetches an loluser given a user id
func fetchLolUserByUserId(ctx context.Context, c *datastore.Client, id string) (res []*LolUser, err error) {
	if id == "" {
		return nil, errors.New("user_id required")
	}
	q := datastore.NewQuery(kindLolUser)
	q = q.Filter("user_id=", id)
	keys, err := c.GetAll(ctx, q, &res)
	if len(keys) == 0 {
		return res, nil
	}
	if len(keys) > 1 {
		return nil, fmt.Errorf("expected 0 or 1 user(s), got %d", len(keys))
	}
	res[0].ID = keys[0].Name
	return res, err
}

// putLolUser saves a row for the passed in LolUser in a transaction
// if this call creates the loluser, ther will be no existing loluser
// with the given user id so we automatically save this new loluser
// if the loluser exists we get the loluser from gcd and update either all of
// its fields if there vpAdjustment is 0 or just VP is vpAdjustment does not
// equal 0
func putLolUser(ctx context.Context, c *datastore.Client, l *LolUser, vpAdjustment int64, absolute bool) (err error) {
	var key *datastore.Key
	if l.ID == "" {
		if key, err = allocateKey(ctx, c, kindLolUser, nil); err != nil {
			return err
		}
	} else {
		key = datastore.NameKey(kindLolUser, l.ID, nil)
	}

	if vpAdjustment == 0 && !absolute {
		_, err = c.RunInTransaction(ctx, func(tx *datastore.Transaction) error { return putLolUserDetails(tx, key, l) })
	} else {
		_, err = c.RunInTransaction(ctx, func(tx *datastore.Transaction) error { return putLolUserPoints(tx, key, vpAdjustment, absolute) })
	}
	return err
}

func putLolUserDetails(tx *datastore.Transaction, key *datastore.Key, l *LolUser) error {
	existing := LolUser{}
	if err := tx.Get(key, &existing); err != nil && err != datastore.ErrNoSuchEntity {
		return err
	}
	l.VantagePointBalance = existing.VantagePointBalance
	// new user start, them off with 400 points (two free games)
	if l.ID == "" {
		l.VantagePointBalance = 400
	}
	_, err := tx.Put(key, l)
	return err
}

// putLolUserPoints updates the loluser's vantage points either by adding the
// adjustment amount to the current total, or setting the total to the ammount
// if absolute is true
func putLolUserPoints(tx *datastore.Transaction, key *datastore.Key, adjustment int64, absolute bool) error {
	existing := LolUser{}
	if err := tx.Get(key, &existing); err != nil {
		return err
	}
	if absolute {
		existing.VantagePointBalance = adjustment
	} else {
		existing.VantagePointBalance += adjustment
	}
	if existing.VantagePointBalance < 0 {
		return errors.New("point balance can not be less than 0")
	}
	_, err := tx.Put(key, &existing)
	return err
}

// FetchLolUsersBySummonerIds retrieves all lolusers stored in the cloud datastore.
func fetchLolUsersBySummonerIds(ctx context.Context, c *datastore.Client, region string, summonerIDs []int64) (res []*LolUser, err error) {
	keys := []*datastore.Key{}
	q := datastore.NewQuery(kindLolUser)
	if region != "" {
		q = q.Filter("region=", region)
	}
	for _, summonerID := range summonerIDs {
		tempLolUsers := []*LolUser{}
		tempQ := q.Filter("summoner_id=", fmt.Sprintf("%d", summonerID))
		tempKeys, err := c.GetAll(ctx, tempQ, &tempLolUsers)
		if err != nil {
			return nil, err
		}
		res = append(res, tempLolUsers...)
		keys = append(keys, tempKeys...)
	}
	if len(summonerIDs) == 0 {
		keys, err = c.GetAll(ctx, q, &res)
		if err != nil {
			return nil, err
		}
	}
	for i := range res {
		res[i].ID = keys[i].Name
	}
	return res, err
}

// a PurchaseMatchRecord is created for every match that is purchased

type PurchaseMatchRecord struct {
	ID            string    `json:"id" datastore:"-"`
	UserID        string    `json:"user_id" datastore:"user_id"`
	SummonerID    int64     `json:"summoner_id" datastore:"summoner_id"`
	Platform      string    `json:"platform" datastore:"platform,noindex"`
	MatchID       int64     `json:"match_id" datastore:"match_id,noindex"`
	timePurchased time.Time `json:"-" datastore: "time_purchased,noindex"`
}

// putPurchaseMatchRecord saves the details of the match that was just
// purchased
func putPurchaseMatchRecord(c *datastore.Client, r *PurchaseMatchRecord) (err error) {
	ctx := context.Background()
	key, err := getOrMakeKey(ctx, c, kindPurchaseMatchRecord, r.ID, nil)
	if err != nil {
		return err
	}
	r.ID = key.Name
	if _, err = c.Put(ctx, key, r); err != nil {
		log.Error(err)
	}
	return err
}

// remarshalAs json-marshals the 'from' object and then json-unmarshal's the
// bytes into the 'to' object. This is useful for translating between RPC
// objects and store objects.
func remarshalAs(from interface{}, to interface{}) error {
	data, err := json.Marshal(from)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, to)
	return err
}

// TODO (Scott): add to common??
func getOrMakeKey(ctx context.Context, c *datastore.Client, kind, name string, parent *datastore.Key) (key *datastore.Key, err error) {
	if name != "" {
		return datastore.NameKey(kind, name, nil), nil
	}
	if key, err = allocateKey(ctx, c, kind, nil); err != nil {
		return nil, err
	}
	return key, nil
}

func allocateKey(ctx context.Context, c *datastore.Client, kind string, parent *datastore.Key) (*datastore.Key, error) {
	incomplete := datastore.IncompleteKey(kind, nil)
	complete, err := c.AllocateIDs(ctx, []*datastore.Key{incomplete})
	if err != nil {
		return nil, err
	}
	name := strconv.FormatInt(complete[0].ID, 10)
	return datastore.NameKey(kind, name, parent), nil
}
