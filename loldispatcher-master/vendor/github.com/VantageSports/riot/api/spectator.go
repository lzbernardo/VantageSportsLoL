package api

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

type GameMetaData struct {
	ChunkTimeInterval            int64             `json:"chunkTimeInterval"`
	ClientAddedLag               int64             `json:"clientAddedLag"`
	ClientBackFetchingEnabled    bool              `json:"clientBackFetchingEnabled"`
	ClientBackFetchingFreq       int64             `json:"clientBackFetchingFreq"`
	CreateTime                   string            `json:"createTime"`
	DecodeEncryptionKey          string            `json:"decodeEncryptionKey"`
	DelayTime                    int64             `json:"delayTime"`
	EncryptionKey                string            `json:"encryptionKey"`
	EndGameChunkID               int               `json:"endGameChunkId"`
	EndGameKeyFrameID            int               `json:"endGameKeyFrameId"`
	EndStartupChunkID            int               `json:"endStartupChunkId"`
	FeaturedGame                 bool              `json:"featuredGame"`
	GameEnded                    bool              `json:"gameEnded"`
	GameKey                      GameKey           `json:"gameKey"`
	GameLength                   int64             `json:"gameLength"`
	GameServerAddress            string            `json:"gameServerAddress"`
	InterestScore                int64             `json:"interestScore"`
	KeyFrameTimeInterval         int64             `json:"keyFrameTimeInterval"`
	LastChunkID                  int               `json:"lastChunkId"`
	LastKeyFrameID               int               `json:"lastKeyFrameId"`
	PendingAvailableChunkInfo    []PendingChunk    `json:"pendingAvailableChunkInfo"`
	PendingAvailableKeyFrameInfo []PendingKeyFrame `json:"pendingAvailableKeyFrameInfo"`
	Port                         int               `json:"port"`
	StartGameChunkID             int               `json:"startGameChunkId"`
	StartTime                    string            `json:"startTime"`
}

type GameKey struct {
	PlatformId string `json:"platformId"`
	GameId     int64  `json:"gameId"`
}

type Blob interface {
	GetID() int
}
type PendingKeyFrame struct {
	ID           int    `json:"id"`
	ReceivedTime string `json:"receivedTime"`
	NextChunkID  int    `json:"nextChunkId"`
}

func (c PendingKeyFrame) GetID() int {
	return c.ID
}

type PendingChunk struct {
	ID           int    `json:"id"`
	Duration     int64  `json:"duration"`
	ReceivedTime string `json:"receivedTime"`
}

func (c PendingChunk) GetID() int {
	return c.ID
}

func SpectatorGameMetaData(server, platformID string, gameID int64) (gameMeta GameMetaData, err error) {
	if server == "" {
		server = spectatorBaseURL(platformID)
	}
	url := fmt.Sprintf("%s/observer-mode/rest/consumer/getGameMetaData/%v/%v/30000/token",
		server, platformID, gameID)

	err = getJSON(url, &gameMeta)
	return gameMeta, err
}

type ChunkInfo struct {
	EndStartupChunkID  int   `json:"endStartupChunkId"`
	Duration           int64 `json:"duration"`
	ChunkID            int   `json:"chunkId"`
	AvailableSince     int64 `json:"availableSince"`
	NextAvailableChunk int   `json:"nextAvailableChunk"`
	NextChunkID        int   `json:"nextChunkId"`
	KeyFrameID         int   `json:"keyFrameId"`
	StartGameChunkID   int   `json:"startGameChunkId"`
	EndGameChunkID     int   `json:"endGameChunkId"`
}

func LastChunkInfo(server, platformID, gameID string) (chunkInfo ChunkInfo, err error) {
	if server == "" {
		server = spectatorBaseURL(platformID)
	}
	url := fmt.Sprintf("%s/observer-mode/rest/consumer/getLastChunkInfo/%v/%v/30000/token",
		server, platformID, gameID)

	err = getJSON(url, &chunkInfo)
	return chunkInfo, err
}

func Version(server, platformID string) ([]byte, error) {
	if server == "" {
		server = spectatorBaseURL(platformID)
	}
	url := fmt.Sprintf("%s/observer-mode/rest/consumer/version", server)
	return getBytes(url)
}

func EndOfGameStats(server, platformID, gameID string, chunkID int) ([]byte, error) {
	if server == "" {
		server = spectatorBaseURL(platformID)
	}
	url := fmt.Sprintf("%s/observer-mode/rest/consumer/getLastChunkInfo/%v/%v/null",
		server, platformID, gameID)

	return getBytes(url)
}

func GameDataChunk(server, platformID, gameID string, chunkID int) (chunkData []byte, err error) {
	if server == "" {
		server = spectatorBaseURL(platformID)
	}
	url := fmt.Sprintf("%s/observer-mode/rest/consumer/getGameDataChunk/%v/%v/%v/token",
		server, platformID, gameID, chunkID)
	return getBytes(url)
}

func KeyFrame(server, platformID, gameID string, keyFrameID int) (keyFrameData []byte, err error) {
	if server == "" {
		server = spectatorBaseURL(platformID)
	}
	url := fmt.Sprintf("%s/observer-mode/rest/consumer/getKeyFrame/%v/%v/%v/token",
		server, platformID, gameID, keyFrameID)
	return getBytes(url)
}

func getBytes(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, NewAPIError(resp.StatusCode, "", "")
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if err := resp.Body.Close(); err != nil {
		return nil, err
	}
	return data, err
}
