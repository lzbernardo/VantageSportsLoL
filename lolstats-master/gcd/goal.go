package gcd

import (
	"fmt"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/golang/protobuf/ptypes"

	"github.com/VantageSports/lolstats"
)

const (
	KindLolGoal = "LolGoal"
)

type LolGoal struct {
	// Administrative columns
	Created     time.Time `json:"created" datastore:"created"`
	LastUpdated time.Time `json:"last_updated" datastore:"last_updated"`

	// Indexed Fields
	Status string `json:"status" datastore:"status"`

	// Required fields
	SummonerID             int64   `json:"summoner_id" datastore:"summoner_id,noindex"`
	Platform               string  `json:"platform" datastore:"platform,noindex"`
	UnderlyingStat         string  `json:"underlying_stat" datastore:"underlying_stat,noindex"`
	TargetValue            float64 `json:"target_value" datastore:"target_value,noindex"`
	Comparator             string  `json:"comparator" datastore:"comparator,noindex"`
	AchievementCount       int64   `json:"achievement_count" datastore:"achievement_count,noindex"`
	TargetAchievementCount int64   `json:"target_achievement_count" datastore:"target_achievement_count,noindex"`
	ImportanceWeight       float64 `json:"importance_weight" datastore:"importance_weight,noindex"`

	RolePosition string  `json:"role_position" datastore:"role_position,noindex"`
	ChampionID   int64   `json:"champion_id" datastore:"champion_id,noindex"`
	LastValue    float64 `json:"last_value" datastore:"last_value,noindex"`
	Category     string  `json:"category" datastore:"category,noindex"`
	LastMatchID  int64   `json:"last_match_id" datastore:"last_match_id,noindex"`
}

func (g *LolGoal) Save() ([]datastore.Property, error) {
	now := time.Now()
	if g.Created.IsZero() {
		g.Created = now
	}
	g.LastUpdated = now
	return datastore.SaveStruct(g)
}

func (g *LolGoal) Load(p []datastore.Property) error {
	return datastore.LoadStruct(g, p)
}

type LolGoalArray []*LolGoal

func (a LolGoalArray) Len() int           { return len(a) }
func (a LolGoalArray) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a LolGoalArray) Less(i, j int) bool { return a[i].ImportanceWeight > a[j].ImportanceWeight }

func (g *LolGoal) GenerateId() string {
	return fmt.Sprintf("%v-%v-%v-%v-%v-%v", g.SummonerID, g.Platform, g.RolePosition, g.ChampionID, g.UnderlyingStat, g.Created.Unix())
}

func (r *LolGoal) ToProtoGoal() (*lolstats.Goal, error) {
	createdTs, err := ptypes.TimestampProto(r.Created)
	if err != nil {
		return nil, err
	}

	lastUpdatedTs, err := ptypes.TimestampProto(r.LastUpdated)
	if err != nil {
		return nil, err
	}

	g := &lolstats.Goal{
		Id:                     r.GenerateId(),
		Created:                createdTs,
		LastUpdated:            lastUpdatedTs,
		Status:                 lolstats.GoalStatus(lolstats.GoalStatus_value[r.Status]),
		SummonerId:             r.SummonerID,
		Platform:               r.Platform,
		UnderlyingStat:         r.UnderlyingStat,
		TargetValue:            r.TargetValue,
		Comparator:             lolstats.GoalComparator(lolstats.GoalComparator_value[r.Comparator]),
		AchievementCount:       r.AchievementCount,
		TargetAchievementCount: r.TargetAchievementCount,
		ImportanceWeight:       r.ImportanceWeight,
		RolePosition:           r.RolePosition,
		ChampionId:             r.ChampionID,
		LastValue:              r.LastValue,
		Category:               lolstats.GoalCategory(lolstats.GoalCategory_value[r.Category]),
	}

	return g, nil
}
