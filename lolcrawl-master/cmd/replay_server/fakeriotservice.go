package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/VantageSports/common/files"

	"cloud.google.com/go/storage"
	"github.com/gorilla/mux"
	"golang.org/x/net/context"
)

type FakeRiotService struct {
	gcsReplayPrefix string
	gcsClient       *storage.Client
}

func NewFakeRiotService(gcsReplayPrefix string, gcsClient *storage.Client) *FakeRiotService {
	return &FakeRiotService{gcsReplayPrefix: gcsReplayPrefix, gcsClient: gcsClient}
}

func (frs *FakeRiotService) getGameDataChunk(w http.ResponseWriter, r *http.Request) {
	platformID := mux.Vars(r)["platformID"]
	gameID := mux.Vars(r)["gameID"]
	chunkID := mux.Vars(r)["chunkID"]
	if platformID == "" || gameID == "" || chunkID == "" {
		http.Error(w, "platformID, gameID, and chunkID are required", http.StatusInternalServerError)
	}
	preventCaching(w)
	frs.serveFile(w, r, fmt.Sprintf("%s-%s/chunk_%s", gameID, strings.ToLower(platformID), chunkID))
}

func (frs *FakeRiotService) getKeyFrame(w http.ResponseWriter, r *http.Request) {
	platformID := mux.Vars(r)["platformID"]
	gameID := mux.Vars(r)["gameID"]
	keyFrameID := mux.Vars(r)["keyFrameID"]
	if platformID == "" || gameID == "" || keyFrameID == "" {
		http.Error(w, "platformID, gameID, and keyFrameID are required", http.StatusInternalServerError)
		return
	}
	preventCaching(w)
	frs.serveFile(w, r, fmt.Sprintf("%s-%s/keyframe_%s", gameID, strings.ToLower(platformID), keyFrameID))
}

func (frs *FakeRiotService) getVersion(w http.ResponseWriter, r *http.Request) {
	frs.serveFile(w, r, "version")
}

func (frs *FakeRiotService) openGCSFile(fp string) (*storage.Reader, error, int) {
	ctx := context.Background()
	b, k, err := files.BucketKey(fp)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	_, err = frs.gcsClient.Bucket(b).Object(k).Attrs(ctx)
	if err != nil && (err == storage.ErrObjectNotExist || err == storage.ErrBucketNotExist) {
		return nil, err, http.StatusNotFound
	}
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	f, err := frs.gcsClient.Bucket(b).Object(k).NewReader(ctx)
	if err != nil {
		return nil, err, http.StatusInternalServerError
	}
	return f, nil, http.StatusAccepted
}

func (frs *FakeRiotService) serveFile(w http.ResponseWriter, r *http.Request, file string) {
	fp := fmt.Sprintf("%s/%s", frs.gcsReplayPrefix, file)
	f, err, code := frs.openGCSFile(fp)
	if err != nil {
		log.Println(err, fp)
		http.Error(w, err.Error(), code)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Length", strconv.FormatInt(f.Size(), 10))
	_, err = io.Copy(w, f)
	if err != nil {
		log.Println(err)
	}
}

func (frs *FakeRiotService) serveJSON(data interface{}, matchID, fileName string) (interface{}, error) {
	fp := fmt.Sprintf("%s/%s/%s", frs.gcsReplayPrefix, matchID, fileName)
	f, err, _ := frs.openGCSFile(fp)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	slurp, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(slurp, data); err != nil {
		return nil, err
	}
	return data, nil
}
