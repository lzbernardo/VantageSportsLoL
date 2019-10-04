package messages

import (
	"fmt"
	"time"
)

func (l *LolAdvancedStatsIngest) Valid() error {
	if l.MatchId == 0 || l.PlatformId == "" || l.SummonerId == 0 || l.BaseviewType == "" || l.BaseviewPath == "" {
		return fmt.Errorf("match_id, platform_id, summoner_id, baseview_type, baseview_path must be set")
	}
	return nil
}

func (l *LolBasicStatsIngest) Valid() error {
	if l.MatchId == 0 || l.PlatformId == "" || l.SummonerId == 0 || l.MatchDetailsPath == "" {
		return fmt.Errorf("match_id, platform_id, summoner_id, and match_details are required")
	}
	return nil
}

func (l *LolEloDataProcess) Valid() error {
	if l.MatchId == 0 || l.PlatformId == "" || l.MatchDetailsPath == "" || l.EloDataPath == "" || len(l.SummonerIds) == 0 {
		return fmt.Errorf("data_path, match_details, match_id, region, and at least one summoner_id must be set")
	}
	return nil
}

func (l *LolMatchDownload) Valid() error {
	if l.MatchId == 0 || l.PlatformId == "" {
		return fmt.Errorf("match_id and platform are required")
	}
	if len(l.ObservedSummonerIds) > 0 && l.Key == "" {
		return fmt.Errorf("encryption_key is required for observed matches")
	}
	return nil
}

func (rq *LolReplayDataExtract) Valid() error {
	if rq.MatchId == 0 ||
		rq.PlatformId == "" ||
		rq.Key == "" ||
		rq.GameLengthSeconds == 0 ||
		len(rq.SummonerIds) == 0 {
		return fmt.Errorf("game_id, platform_id, key, game_length_seconds, and at least one summoner_id are all required")
	}
	return nil
}

func (l *LolSummonerCrawl) Valid() error {
	if l.SummonerId == 0 || l.PlatformId == "" || l.HistoryType == "" {
		return fmt.Errorf("summoner_id, platform, and history_type are required")
	}
	if l.HistoryType != "ranked_history" && l.HistoryType != "recent" {
		return fmt.Errorf("history_type must be 'ranked_history' or 'recent'")
	}
	var t time.Time
	var err error
	if l.Since != "" {
		t, err = time.Parse(l.Since, time.RFC3339)
		if err != nil {
			return err
		}
	}
	if l.HistoryType == "ranked_history" && t.IsZero() {
		return fmt.Errorf("since is required for ranked_history type")
	}
	return nil
}

func (rq *LolVideo) Valid() error {
	if rq.MatchId == 0 ||
		rq.PlatformId == "" ||
		rq.Key == "" ||
		rq.GameLengthSeconds == 0 {
		return fmt.Errorf("game_id, platform_id, key, champ_focus, and game_length_seconds are all required")
	}
	if rq.ChampFocus < 1 || rq.ChampFocus > 10 {
		return fmt.Errorf("champ_focus must be between 1-10")
	}
	return nil
}
