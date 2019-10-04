// The convert_handler parses elo text files and produces match baseview json
// files, which we can derive advanced stats from (in a later process).

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/VantageSports/common/files"
	fileutil "github.com/VantageSports/common/files/util"
	vsjson "github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/lolelo/event"
	"github.com/VantageSports/lolelo/parse"
	"github.com/VantageSports/lolelo/validate"
	"github.com/VantageSports/lolstats/baseview"
	"github.com/VantageSports/riot"
	"github.com/VantageSports/riot/api"

	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"
)

type LolOCRExtractor struct {
	OutputDir     string
	Files         *files.Client
	PubTopic      *pubsub.Topic
	ReplayDataDir string
}

func (e *LolOCRExtractor) Handle(ctx context.Context, m *pubsub.Message) error {
	msg := messages.LolEloDataProcess{}
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		log.Error(err)
		return err
	}
	log.Debug(fmt.Sprintf("message: %s", m.Data))
	if err := msg.Valid(); err != nil {
		return err
	}

	workDir, err := ioutil.TempDir("", "lol_elodata_tmp")
	if err != nil {
		log.Error(err)
		return err
	}
	defer os.RemoveAll(workDir)

	platform := riot.PlatformFromString(msg.PlatformId)

	remoteBaseviewPath := fmt.Sprintf("%s/%d-%s.baseview.json", e.OutputDir, msg.MatchId, platform)
	exists, err := e.Files.Exists(remoteBaseviewPath)
	if err != nil {
		log.Error(err)
		return err
	}

	if exists[0] {
		log.Info(fmt.Sprintf("baseview already exists for %d-%s, override=%t", msg.MatchId, platform, msg.Override))
		if !msg.Override {
			// Download and parse the baseview
			baseviewLocal, err := downloadTo(e.Files, remoteBaseviewPath, workDir)
			if err != nil {
				return err
			}
			bv := &baseview.Baseview{}
			if err = vsjson.DecodeFile(baseviewLocal, bv); err != nil {
				return err
			}

			return publishMessage(ctx, e.PubTopic, bv, remoteBaseviewPath, &msg)
		}
	}

	// Download and parse elo data
	eloEvents, err := downloadEloEvents(e.Files, msg.EloDataPath, workDir)
	if err != nil {
		log.Error(err)
		return err
	}

	// Download and parse the api match details
	matchDetail, err := downloadMatchDetails(e.Files, msg.MatchDetailsPath, workDir)
	if err != nil {
		log.Error(err)
		return err
	}

	baseview, err := processEloText(ctx, e.Files, eloEvents, matchDetail, remoteBaseviewPath, workDir)
	if err != nil {
		log.Error(err)
		return err
	}

	// Make sure our task lease hasn't expired.
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// Publish a message.
	return publishMessage(ctx, e.PubTopic, baseview, remoteBaseviewPath, &msg)
}

func downloadEloEvents(fc *files.Client, eloDataPath, workDir string) ([]event.EloEvent, error) {
	// Download the raw events
	rawTextLocal, err := downloadTo(fc, eloDataPath, workDir)
	if err != nil {
		return nil, err
	}

	// Generate events from the elo text format.
	events, err := parse.LogFile(rawTextLocal)
	if err != nil {
		return nil, err
	}
	return events, nil
}

func downloadMatchDetails(fc *files.Client, matchDetailsPath, workDir string) (*api.MatchDetail, error) {
	// Download and parse the match details.
	matchDetailsLocal, err := downloadTo(fc, matchDetailsPath, workDir)
	if err != nil {
		return nil, err
	}
	matchDetail := &api.MatchDetail{}
	if err = vsjson.DecodeFile(matchDetailsLocal, matchDetail); err != nil {
		return nil, err
	}
	return matchDetail, nil
}

func processEloText(ctx context.Context, fc *files.Client, events []event.EloEvent, matchDetail *api.MatchDetail, remoteBaseviewPath, workDir string) (*baseview.Baseview, error) {
	// Align the raw events with the match detail events, and check to verify
	// that the data file is not truncated.
	var err error
	if err = validate.AlignAPI(events, matchDetail); err != nil {
		return nil, err
	}
	if err = validate.CheckTruncation(events, matchDetail.MatchDuration); err != nil {
		return nil, err
	}

	// Use the match details to build a participant list. (NOTE: we assume that
	// the match details have been augmented with other sources of data by the
	// match downloader and ALWAYS contain participant info.)
	participants, err := validate.ParticipantMapMD(events, matchDetail)
	if err != nil {
		return nil, fmt.Errorf("unable to determine participants from match: %v", err)
	}

	baseview, err := event.EloToBaseview(events, participants)
	if err != nil {
		return nil, fmt.Errorf("error extracting baseview: %v", err)
	}

	outputDir, localBaseview := filepath.Split(remoteBaseviewPath)
	if err = vsjson.Write(localBaseview, baseview, 0664); err != nil {
		return nil, err
	}
	defer os.Remove(localBaseview)

	remotePath, err := uploadCompressedJSON(fc, localBaseview, outputDir)
	if err != nil {
		return nil, err
	}
	if remotePath != remoteBaseviewPath {
		return nil, fmt.Errorf("remotePath (%s) does not equal remoteBaseviewPath (%s)", remotePath, remoteBaseviewPath)
	}

	// return the context error (which will be nil if the task lease has not expired)
	return baseview, ctx.Err()
}

func downloadTo(fc *files.Client, remotePath, localDir string) (string, error) {
	localPath := filepath.Join(localDir, filepath.Base(remotePath))
	log.Debug(fmt.Sprintf("downloading %s to %s", remotePath, localPath))

	err := fc.Copy(remotePath, localPath)

	return localPath, err
}

func uploadCompressedJSON(fc *files.Client, localPath, remoteDir string) (string, error) {
	remotePath := strings.TrimSuffix(remoteDir, "/") + "/" + filepath.Base(localPath)

	compressedPath := localPath + ".zip"
	log.Debug(fmt.Sprintf("compressing %s to %s", localPath, compressedPath))
	if err := fileutil.GzipFile(localPath, compressedPath, 0664); err != nil {
		return "", err
	}

	err := fc.Copy(compressedPath, remotePath,
		files.ContentType("application/json"), files.ContentEncoding("gzip"))
	defer os.Remove(compressedPath)

	return remotePath, err
}

func publishMessage(ctx context.Context, topic *pubsub.Topic, bv *baseview.Baseview, baseviewRemote string, msg *messages.LolEloDataProcess) error {
	// Loop over all the summonerIds to generate advanced stats for everyone, instead of just the people who paid for it
	for _, participant := range bv.Participants {
		ingestMsg := messages.LolAdvancedStatsIngest{
			BaseviewPath: baseviewRemote,
			BaseviewType: "elo",
			MatchId:      msg.MatchId,
			PlatformId:   msg.PlatformId,
			SummonerId:   participant.SummonerID,
		}

		data, err := json.Marshal(ingestMsg)
		if err != nil {
			return err
		}

		tctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
		_, err = topic.Publish(tctx, &pubsub.Message{Data: data})
		if err != nil {
			return err
		}
	}
	return ctx.Err()
}
