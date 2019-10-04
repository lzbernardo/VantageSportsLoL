// The downloading of riot replays is an incremental process. Chunks, keyframes,
// and other metadata files are each available for a short period of time (~ few
// minutes). If the server we're downloading from doesn't have chunk X, but does
// have chunk X+N (for some value of N) then we consider the download to have
// failed.

package lolobserver

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/VantageSports/lolusers"
	"github.com/VantageSports/riot/api"
)

func init() {
	// the riot api uses the http.DefaultClient, and we should never wait longer
	// than 30 seconds for a request to return (especially useful on 3rd party
	// sites that don't appear to time requests out, like replay.gg)
	// TODO(Cameron): Make the riot API create its own client with a
	// configurable timeout.
	http.DefaultClient.Timeout = time.Second * 30
}

// ReplaySaveState encapsulates the status of a match replay download. Replay
type ReplaySaveState struct {
	currentGame   api.CurrentGameInfo
	lolusers      []lolusers.LolUser
	lastChunkInfo api.ChunkInfo

	// The following fields indicate whether the replay files exist in the
	// destination directory.
	remoteGame bool
	chunks     map[int]bool
	keyframes  map[int]bool
	meta       bool
	version    bool
}

func NewReplaySaveState(m *MatchUsers, existingFiles []string) (*ReplaySaveState, error) {
	st := &ReplaySaveState{
		currentGame: m.CurrentGame,
		lolusers:    m.LolUsers,
		chunks:      map[int]bool{},
		keyframes:   map[int]bool{},
	}
	err := load(existingFiles, st)
	return st, err
}

// Save downloads (from the replay server) and persists (via the FileSaver) all
// the files necessary to replay a match. It will retry downloads several times
// (see retry login in save functions), but it will NOT skip files. E.g. if it
// is unable to download chunk 9, or keyframe 44, or the game meta, an error
// will be returned.
func (rs *ReplaySaveState) Save(server string, saver FileSaver) error {
	if err := saveCurrentGame(saver, rs); err != nil {
		return err
	}

	logPrefix := fmt.Sprintf("match: %d, failed to save", rs.currentGame.GameID)
	for {
		if err := saveChunkInfo(server, saver, rs); err != nil {
			return fmt.Errorf("%s chunk info: %v", logPrefix, err)
		}

		for num := 1; num <= rs.lastChunkInfo.ChunkID; num++ {
			if err := saveChunk(server, saver, rs, num); err != nil {
				return fmt.Errorf("%s chunk (num: %d): %v", logPrefix, num, err)
			}
		}

		for num := 1; num <= rs.lastChunkInfo.KeyFrameID; num++ {
			if err := saveKeyframe(server, saver, rs, num); err != nil {
				return fmt.Errorf("%s keyframe (num: %d): %v", logPrefix, num, err)
			}
		}

		finalChunkNum := rs.lastChunkInfo.EndGameChunkID
		if finalChunkNum > 0 && rs.chunks[finalChunkNum] {
			break
		}

		time.Sleep(rs.sleepDur())
	}

	// Step 3 - meta and version
	if err := saveMeta(server, saver, rs); err != nil {
		return fmt.Errorf("%s meta: %v", logPrefix, err)
	}
	return saveVersion(server, saver, rs)
}

func (rs *ReplaySaveState) sleepDur() time.Duration {
	dur := time.Duration(rs.lastChunkInfo.NextAvailableChunk) * time.Millisecond
	if dur < time.Second {
		// Just in case we have a zero-value chunkInfo
		dur = time.Second
	}
	return dur + time.Second
}

// load examines the list of file paths passed to it, and adds those recognized
// to the specified ReplaySaveState object, so that replays can "resume" from
// where they left off.
func load(paths []string, r *ReplaySaveState) error {
	for _, path := range paths {
		base := filepath.Base(path)
		switch base {
		case "current_game.json":
			r.remoteGame = true
		case "meta.json":
			r.meta = true
		case "version":
			r.version = true
		default:
			if err := parseChunkKeyframe(base, r); err != nil {
				return err
			}
		}
	}
	return nil
}

// parseChunkKeyframe determines the specific chunk or keyframe file that a
// given basename represents. This method is used by load to determine which
// chunks and keyframes still need to be downloaded.
func parseChunkKeyframe(basename string, r *ReplaySaveState) error {
	parts := strings.Split(basename, "_")
	if len(parts) != 2 || (parts[0] != "chunk" && parts[0] != "keyframe") {
		return nil
	}
	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("cannot parse chunk/keyframe num from filename %s. err: %v", basename, err)
	}
	if parts[0] == "chunk" {
		r.chunks[num] = true
	} else {
		r.keyframes[num] = true
	}
	return nil
}

// ShouldDownload returns true if this game appears to be worth downloading.
// Currently the only hueristic applied is whether the game contains 10 non-bot
// players.
func ShouldDownload(g api.CurrentGameInfo) bool {
	nonBots := 0
	for _, p := range g.Participants {
		if !p.Bot {
			nonBots++
		}
	}
	if nonBots != 10 {
		return false
	}
	if g.MapID != 11 { // summoner's rift
		return false
	}
	if g.GameMode != "CLASSIC" {
		return false
	}
	switch g.GameQueueConfigID {
	// see https://developer.riotgames.com/docs/game-constants for defs
	case 0, // CUSTOM
		2,   // NORMAL_5x5_BLIND
		42,  // RANKED_TEAM_5x5
		400, // TEAM_BUILDER_DRAFT_UNRANKED_5x5
		410, // TEAM_BUILDER_DRAFT_RANKED_5x5
		420, // TEAM_BUILDER_RANKED_SOLO
		440: // RANKED_FLEX_SR
	default:
		// all other games queues should be skipped.
		return false
	}
	return true
}

//
// "Saver" functions. The following functions all attempt to download data from
// a replay server and save it (maybe remotely). Any returned error indicates
// failure worth aborting the download for.
//

func saveCurrentGame(saver FileSaver, rs *ReplaySaveState) (err error) {
	// No server request is necessary, since we already have the current game
	// info (which is required to start this process)
	if rs.remoteGame {
		return nil
	}
	return saver.SaveAs(rs.currentGame, "current_game.json", true, 3)
}

func saveChunkInfo(server string, saver FileSaver, rs *ReplaySaveState) (err error) {
	fn := chunkInfo(server, rs.currentGame.PlatformID, rs.currentGame.GameIDStr())
	out, err := riotWithRetry(fn, 15)
	if err == nil {
		rs.lastChunkInfo = out.(api.ChunkInfo)
		sanitizeLastChunk(&rs.lastChunkInfo)
		err = saver.SaveAs(rs.lastChunkInfo, "last_chunk.json", true, 3)
	}
	return err
}

func sanitizeLastChunk(lastChunk *api.ChunkInfo) {
	lastChunk.Duration = 30000
	lastChunk.NextAvailableChunk = 0
	if lastChunk.NextChunkID < lastChunk.ChunkID {
		lastChunk.NextChunkID = lastChunk.ChunkID
	}

}

func saveChunk(server string, saver FileSaver, rs *ReplaySaveState, chunkNum int) (err error) {
	if rs.chunks[chunkNum] {
		return nil
	}

	fn := gameDataChunk(server, rs.currentGame.PlatformID, rs.currentGame.GameIDStr(), chunkNum)
	chunk, err := riotWithRetry(fn, 15)
	if err == nil {
		rs.chunks[chunkNum] = true
		name := fmt.Sprintf("chunk_%d", chunkNum)
		err = saver.SaveAs(chunk, name, false, 3)
	}
	return err
}

func saveKeyframe(server string, saver FileSaver, rs *ReplaySaveState, keyframeNum int) (err error) {
	if rs.keyframes[keyframeNum] {
		return nil
	}

	fn := keyframe(server, rs.currentGame.PlatformID, rs.currentGame.GameIDStr(), keyframeNum)
	keyframe, err := riotWithRetry(fn, 15)
	if err == nil {
		rs.keyframes[keyframeNum] = true
		name := fmt.Sprintf("keyframe_%d", keyframeNum)
		err = saver.SaveAs(keyframe, name, false, 3)
	}
	return err
}

func saveMeta(server string, saver FileSaver, rs *ReplaySaveState) (err error) {
	if rs.meta {
		return nil
	}

	fn := meta(server, rs.currentGame.PlatformID, rs.currentGame.GameIDStr())
	meta, err := riotWithRetry(fn, 15)
	if err == nil {
		rs.meta = true
		err = saver.SaveAs(meta, "meta.json", true, 3)
	}
	return err
}

func saveVersion(server string, saver FileSaver, rs *ReplaySaveState) (err error) {
	if rs.version {
		return nil
	}

	fn := version(server, rs.currentGame.PlatformID)
	version, err := riotWithRetry(fn, 10)
	if err == nil {
		rs.version = true
		err = saver.SaveAs(version, "version", false, 3)
	}
	return err
}
