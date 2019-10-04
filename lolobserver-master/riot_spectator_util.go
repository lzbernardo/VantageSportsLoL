package lolobserver

import (
	"fmt"
	"strconv"
	"time"

	"github.com/VantageSports/common/log"
	"github.com/VantageSports/riot/api"
)

// spectatorFunc is the abstraction of a riot spectator api call. This makes it
// easier to apply generic retry logic (see: riotWithRetry()) to several
// different riot api calls.
type spectatorFunc func() (interface{}, error, string)

func chunkInfo(server, platformID, gameID string) spectatorFunc {
	return func() (interface{}, error, string) {
		out, err := api.LastChunkInfo(server, platformID, gameID)
		return out, err, desc("LastChunkInfo", platformID, gameID)
	}
}

func gameDataChunk(server, platformID, gameID string, chunkID int) spectatorFunc {
	return func() (interface{}, error, string) {
		out, err := api.GameDataChunk(server, platformID, gameID, chunkID)
		return out, err, desc("GameDataChunk", platformID, gameID, chunkID)
	}
}

func keyframe(server, platformID, gameID string, keyframeID int) spectatorFunc {
	return func() (interface{}, error, string) {
		out, err := api.KeyFrame(server, platformID, gameID, keyframeID)
		return out, err, desc("KeyFrame", platformID, gameID, keyframeID)
	}
}

func meta(server, platformID, gameID string) spectatorFunc {
	return func() (interface{}, error, string) {
		matchID, err := strconv.ParseInt(gameID, 10, 64)
		if err != nil {
			return nil, err, gameID
		}
		out, err := api.SpectatorGameMetaData(server, platformID, matchID)
		return out, err, desc("GameMetaData", platformID, gameID, gameID)
	}
}

func version(server, platformID string) spectatorFunc {
	return func() (interface{}, error, string) {
		out, err := api.Version(server, platformID)
		return out, err, desc("Version", platformID)
	}
}

func riotWithRetry(fn spectatorFunc, numAttempts int) (out interface{}, err error) {
	var str string
	for i := 1; i <= numAttempts; i++ {
		out, err, str = fn()
		if err == nil {
			return
		}
		log.Debug(fmt.Sprintf("riot error for %s: %v", str, err))
		time.Sleep(time.Second * 4)
	}
	return
}

func desc(v ...interface{}) string {
	return fmt.Sprintf("%v", v)
}
