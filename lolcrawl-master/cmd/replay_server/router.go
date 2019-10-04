package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	vshttp "github.com/VantageSports/common/http"
	"github.com/VantageSports/riot/api"

	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

func Attach(router *mux.Router, svc *FakeRiotService) {
	router.Methods("OPTIONS").Handler(vshttp.ServeOptions())
	router.Path("/health").Handler(vshttp.ServeOK())
	router.Path("/version").Handler(vshttp.ServeFile("./version"))

	sr := router.Methods("GET").Subrouter()
	sr.Path("/match/{matchID}").Handler(getMatch(svc))
	sr.Path("/matchlist/by-summoner/{summonerID}").Handler(getMatchList())
	sr.Path("/observer-mode/rest/consumer/version").HandlerFunc(svc.getVersion)
	sr.Path("/observer-mode/rest/consumer/getSpectatorGameInfo/{platformID}/{summonerID}").Handler(getCurrentGame())
	sr.Path("/observer-mode/rest/consumer/getLastChunkInfo/{platformID}/{gameID}/{noonce}/token").Handler(getLastChunkInfo(svc))
	sr.Path("/observer-mode/rest/consumer/getGameMetaData/{platformID}/{gameID}/{noonce}/token").Handler(getGameMetaData(svc))
	sr.Path("/observer-mode/rest/consumer/getLastChunkInfo/{platformID}/{gameID}/{noonce}/null").Handler(getEndOfGameStats())
	sr.Path("/observer-mode/rest/consumer/getGameDataChunk/{platformID}/{gameID}/{chunkID}/token").HandlerFunc(svc.getGameDataChunk)
	sr.Path("/observer-mode/rest/consumer/getKeyFrame/{platformID}/{gameID}/{keyFrameID}/token").HandlerFunc(svc.getKeyFrame)

	sr.Path("/encryption-key/{platformID}/{gameID}").Handler(getEncryptionKey(svc))
}

func wrap(h vshttp.Handler) http.Handler {
	logging := vshttp.ResponseLogWrapper("")
	json := vshttp.JSONWriter{}
	cors := vshttp.SetAllowedOrigin("*")
	all := vshttp.WrapAll(h, logging, json, cors)
	return vshttp.BaseHandler(context.Background(), true, all)
}

func getMatch(svc *FakeRiotService) http.Handler {
	return wrap(func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		matchID := mux.Vars(r)["matchID"]
		if matchID == "" {
			return nil, errors.New("matchID is required")
		}
		matchDetail := &api.MatchDetail{}
		return matchDetail, nil
	})
}

func getMatchList() http.Handler {
	return wrap(func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		summonerID := mux.Vars(r)["summonerID"]
		if summonerID == "" {
			return nil, errors.New("summonerID is required")
		}
		matchListItem := &api.MatchListItem{}
		return matchListItem, nil
	})
}

func getCurrentGame() http.Handler {
	return wrap(func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		platformID := mux.Vars(r)["platformID"]
		summonerID := mux.Vars(r)["summonerID"]
		if platformID == "" || summonerID == "" {
			return nil, errors.New("platformID and summonerID are required")
		}
		currentGameInfo := &api.CurrentGameInfo{}
		return currentGameInfo, nil
	})
}

func getEncryptionKey(svc *FakeRiotService) http.Handler {
	return wrap(func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		platformID := mux.Vars(r)["platformID"]
		gameID := mux.Vars(r)["gameID"]
		if platformID == "" || gameID == "" {
			return nil, errors.New("platformID and gameID are required")
		}
		preventCaching(w)
		currentGameInfo := &api.CurrentGameInfo{}
		_, err := svc.serveJSON(&currentGameInfo, fmt.Sprintf("%s-%s", gameID, strings.ToLower(platformID)), "current_game.json")
		return currentGameInfo.Observers.EncryptionKey, err
	})
}

func getLastChunkInfo(svc *FakeRiotService) http.Handler {
	return wrap(func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		platformID := mux.Vars(r)["platformID"]
		gameID := mux.Vars(r)["gameID"]
		if platformID == "" || gameID == "" {
			return nil, errors.New("platformID and gameID are required")
		}
		preventCaching(w)
		chunkInfo := &api.ChunkInfo{}
		return svc.serveJSON(&chunkInfo, fmt.Sprintf("%s-%s", gameID, strings.ToLower(platformID)), "last_chunk.json")
	})
}

func getGameMetaData(svc *FakeRiotService) http.Handler {
	return wrap(func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		platformID := mux.Vars(r)["platformID"]
		gameID := mux.Vars(r)["gameID"]
		if platformID == "" || gameID == "" {
			return nil, errors.New("platformID and gameID are required")
		}
		preventCaching(w)
		gameMetaData := &api.GameMetaData{}
		return svc.serveJSON(&gameMetaData, fmt.Sprintf("%s-%s", gameID, strings.ToLower(platformID)), "meta.json")
	})
}

func getEndOfGameStats() http.Handler {
	return wrap(func(ctx context.Context, w http.ResponseWriter, r *http.Request) (interface{}, error) {
		platformID := mux.Vars(r)["platformID"]
		gameID := mux.Vars(r)["gameID"]
		if platformID == "" || gameID == "" {
			return nil, errors.New("platformID and gameID are required")
		}
		// TODO: what to return here
		return "OK", nil
	})
}

func preventCaching(w http.ResponseWriter) {
	w.Header().Add("Cache-Control", "no-cache")
	w.Header().Add("Cache-Control", "no-store")
	w.Header().Add("Cache-Control", "must-revalidate")
	w.Header().Add("Pragma", "no-cache")
	w.Header().Add("Expires", "0")
}
