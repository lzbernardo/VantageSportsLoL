package lolobserver

import (
	"testing"

	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot/api"
)

func TestLoad(t *testing.T) {
	files := []string{
		"gs://keyframe_3",
		"s3://chunk_2",
		"/tmp/keyframe_4",
		"gs://foo/bar/zee/chunk_11",
		"chunk_5",
	}

	ms := &MatchUsers{api.CurrentGameInfo{}, []lolusers.LolUser{}, false}
	r, err := NewReplaySaveState(ms, files)
	if err != nil {
		t.Error(err)
	}

	expectNums(t, r.chunks, 2, 11, 5)
	expectNums(t, r.keyframes, 3, 4)

	files[0] = "some_other_file_1"
	r, err = NewReplaySaveState(ms, files)
	if err != nil {
		t.Error(err)
	}

	files[0] = "/usr/local/not_chunk_file_19"
	r, err = NewReplaySaveState(ms, files)
	if err != nil {
		t.Error(err)
	}

	files[0] = "gs://chunk_nine"
	r, err = NewReplaySaveState(ms, files)
	if err == nil {
		t.Errorf("expected error for file chunk_nine, got none")
	}
}

func expectNums(t *testing.T, m map[int]bool, nums ...int) {
	for _, num := range nums {
		if !m[num] {
			t.Errorf("expected value for key: %d", num)
		}
	}
}

func TestShouldDownload(t *testing.T) {
	cg := api.CurrentGameInfo{
		GameMode:          "CLASSIC",
		MapID:             11,
		GameQueueConfigID: 0,
		Participants:      []api.CurrentGameParticipant{},
	}

	for i := 0; i < 9; i++ {
		cg.Participants = append(cg.Participants, api.CurrentGameParticipant{
			Bot:        false,
			SummonerID: int64(i + 10000),
		})
	}

	if ShouldDownload(cg) {
		t.Error("expected false for 9 non-bot players")
	}

	cg.Participants = append(cg.Participants, api.CurrentGameParticipant{
		Bot:        false,
		SummonerID: 1234,
	})
	if !ShouldDownload(cg) {
		t.Error("expected true for 10 non-bot players")
	}

	cg.Participants[6].Bot = true
	if ShouldDownload(cg) {
		t.Error("expected false for even 1 bot player")
	}
	cg.Participants[6].Bot = false

	cg.MapID = 10
	if ShouldDownload(cg) {
		t.Error("expected false when not on summoners rift")
	}
	cg.MapID = 11

	cg.GameMode = "ARAM"
	if ShouldDownload(cg) {
		t.Error("expected false when not in CLASSIC mode")
	}
	cg.GameMode = "CLASSIC"

	cg.GameQueueConfigID = 20
	if ShouldDownload(cg) {
		t.Error("expected false for game queue config id of 20")
	}
	cg.GameQueueConfigID = 410

	if !ShouldDownload(cg) {
		t.Error("expected true for valid game")
	}
}
