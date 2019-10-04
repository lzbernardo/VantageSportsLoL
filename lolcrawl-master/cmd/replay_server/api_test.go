package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
	"gopkg.in/tylerb/is.v1"
)

var (
	server          *httptest.Server
	matchUrl        string
	matchListUrl    string
	currentGameUrl  string
	endOfGameUrl    string
	healthUrl       string
	gcsReplayPrefix string
)

func init() {
	router := mux.NewRouter().StrictSlash(true)
	gcsReplayPrefix = "gs://esports/lol/replay"
	ctx := context.Background()
	gcsClient, _ := storage.NewClient(ctx)
	svc := NewFakeRiotService(gcsReplayPrefix, gcsClient)
	// Register plain-old HTTP
	Attach(router, svc)
	server = httptest.NewServer(router)
	matchUrl = fmt.Sprintf("%s/match/123456", server.URL)
	matchListUrl = fmt.Sprintf("%s/matchlist/by-summoner/123456", server.URL)
	currentGameUrl = fmt.Sprintf("%s/observer-mode/rest/consumer/getSpectatorGameInfo/BR1/123456", server.URL)
	endOfGameUrl = fmt.Sprintf("%s/observer-mode/rest/consumer/getLastChunkInfo/NA1/123/22/null", server.URL)
	healthUrl = fmt.Sprintf("%s/health", server.URL)
}

func TestHealth(t *testing.T) {
	is := is.New(t)
	res, err := send("GET", healthUrl, nil)

	is.Nil(err)
	is.Equal(200, res.StatusCode)
}

func TestGetMatch(t *testing.T) {
	is := is.New(t)

	res, err := send("GET", matchUrl, nil)
	is.Nil(err)
	is.Equal(200, res.StatusCode)
}

func TestGetMatchList(t *testing.T) {
	is := is.New(t)

	res, err := send("GET", matchListUrl, nil)
	is.Nil(err)
	is.Equal(200, res.StatusCode)
}

func TestGetCurrentGame(t *testing.T) {
	is := is.New(t)

	res, err := send("GET", currentGameUrl, nil)

	is.Nil(err)
	is.Equal(200, res.StatusCode)
}

func TestGetEndOfGameStats(t *testing.T) {
	is := is.New(t)

	res, err := send("GET", endOfGameUrl, nil)
	is.Nil(err)
	is.Equal(200, res.StatusCode)
}

func send(method, url string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reqBody = bytes.NewReader(data)
	}

	request, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, err
	}
	return http.DefaultClient.Do(request)
}
