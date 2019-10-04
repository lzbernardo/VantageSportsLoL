package worker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"

	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/files/util"
	vsjson "github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/lolelo/event"
	"github.com/VantageSports/lolelo/parse"
	"github.com/VantageSports/lolelo/validate"
	"github.com/VantageSports/lolvideo/models"
	"github.com/VantageSports/riot/api"
)

// DatagenWorker : A struct to encapsulate the queue message logic
type DatagenWorker struct {
	InstanceID      string
	Region          string
	CoordinatorURL  string
	ClientVersion   string
	DataOutputPath  string
	Files           *files.Client
	DatastoreClient *datastore.Client
	OutTopic        *pubsub.Topic
	RetryTopic      *pubsub.Topic
	DevServerMode   bool
	MaxRetries      int64
	Constants       LolDatagenWorkerNumbers
}

// numDataExtractFailures tracks how many times the worker messed up.
// If we see too many of these in a row, it's likely the worker is in a bad state, and needs to be
// terminated / restarted
var numDataExtractFailures int = 0

type LolVideoBootstrapGame struct {
	GameID     int64  `json:"game_id" datastore:"game_id,noindex"`
	GameKey    string `json:"game_key" datastore:"game_key,noindex"`
	PlatformID string `json:"platform_id" datastore:"platform_id,noindex"`
}

type LolDatagenWorkerNumbers struct {
	BootstrapWait int64 `json:"bootstrap_wait" datastore:"bootstrap_wait,noindex"`
	GameStartWait int64 `json:"game_start_wait" datastore:"game_start_wait,noindex"`
}

func (f *DatagenWorker) Bootstrap() error {
	// Check to see if this machine needs bootstrapping.
	// This file is created in the userdata template, and deleted in the replay.bat script

	if _, err := os.Stat(`c:\needs_bootstrap.txt`); err != nil {
		return nil
	}

	// Get the key+gameID+platformID for version
	gameInfo := LolVideoBootstrapGame{}
	dsKey := datastore.NameKey("LolVideoBootstrapGame", f.ClientVersion, nil)
	dsErr := f.DatastoreClient.Get(context.Background(), dsKey, &gameInfo)
	if dsErr == datastore.ErrNoSuchEntity {
		// If we don't find one, because it's a new version, then use a default
		// This will get set on the next message
		gameInfo = LolVideoBootstrapGame{
			GameID:     2514154995,
			GameKey:    "hABXSycP8GLXeJjzbhuXw2nC5qdV9eK0",
			PlatformID: "NA1",
		}
	} else if dsErr != nil {
		return dsErr
	}

	// Construct our request to the coordinator
	bootstrapReq := models.DatagenBootstrapRequest{
		GameID:          gameInfo.GameID,
		PlatformID:      gameInfo.PlatformID,
		GameKey:         gameInfo.GameKey,
		SpectatorServer: "lolstreamer.vantagesports.gg:8080",
		WorkerInstance:  f.InstanceID,
		Region:          f.Region,
	}

	if f.DevServerMode {
		// When running the coordinator locally, the ec2 machine can't access
		// your local dev box. So wait for the request to be triggered manually
		data, err := json.Marshal(bootstrapReq)
		if err != nil {
			return err
		}
		log.Debug("Please issue this request in the next 2 minutes: " +
			"curl -d '" + string(data) + "' " + f.CoordinatorURL + "/datagen_bootstrap_request")
		time.Sleep(2 * time.Minute)
	} else {
		// Hit the coordinator url to trigger the replay to start
		if err := postJSON(f.CoordinatorURL, "datagen_bootstrap_request", bootstrapReq); err != nil {
			return err
		}
	}

	sleepTime := time.Minute * time.Duration(f.Constants.BootstrapWait)
	log.Info("Starting bootstrap. Sleeping for " + sleepTime.String())
	time.Sleep(sleepTime)

	consecutiveNoProcessFound := 0
	for i := 0; ; i++ {
		if !processIsRunningOnWindows() {
			consecutiveNoProcessFound++
		} else {
			consecutiveNoProcessFound = 0
		}

		if consecutiveNoProcessFound > 2 {
			return nil
		}
		if i%60 == 0 {
			log.Debug(fmt.Sprintf("waiting for bootstrap to finish: %v minutes passed", f.Constants.BootstrapWait+int64(i/12)))
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

// Handle : Take a Pub/Sub queue message and initiate a request to the coordinator web server.
//   The webserver should connect back to us with commands to open up the spectator client.
//   We also need to wait until the game is done before acknowledging the queue message
func (f *DatagenWorker) Handle(ctx context.Context, m *pubsub.Message) error {
	// Deserialize the message
	msg := messages.LolReplayDataExtract{}
	log.Info(fmt.Sprintf("%s", m.Data))
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		return err
	}

	if err := msg.Valid(); err != nil {
		return err
	}

	if msg.RetryNum > f.MaxRetries {
		log.Warning(fmt.Sprintf("match %d-%s has exceeded max retries. skipping", msg.MatchId, msg.PlatformId))
		return nil
	}

	// The client version (the description in the AMI) has to be a substring
	// of the version in the game message for it to work.
	// Example:
	//   AMI has clientVersion of "6.5."
	//   Message with MatchVersion of "6.5.1.255" passes
	//   Message with MatchVersion of "6.6.1.105" fails
	if msg.MatchVersion != "" && strings.Index(msg.MatchVersion, f.ClientVersion) != 0 {
		return fmt.Errorf("client version mismatch")
	}

	// If there is no bootstrap game for this version, then set it with this match
	gameInfo := LolVideoBootstrapGame{}
	dsKey := datastore.NameKey("LolVideoBootstrapGame", f.ClientVersion, nil)
	dsErr := f.DatastoreClient.Get(context.Background(), dsKey, &gameInfo)
	if dsErr == datastore.ErrNoSuchEntity {
		gameInfo = LolVideoBootstrapGame{
			GameID:     msg.MatchId,
			GameKey:    msg.Key,
			PlatformID: msg.PlatformId,
		}
		_, err := f.DatastoreClient.Put(context.Background(), dsKey, &gameInfo)
		if err != nil {
			return err
		}
	}

	// Construct our request to the coordinator
	dataRequest := models.DatagenRequest{
		ReplayRequest: &models.ReplayRequest{
			GameID:            msg.MatchId,
			PlatformID:        msg.PlatformId,
			GameKey:           msg.Key,
			GameLengthSeconds: int(msg.GameLengthSeconds),
			SpectatorServer:   msg.SpectatorServer,
			MatchVersion:      msg.MatchVersion,
			WorkerInstance:    f.InstanceID,
			Region:            f.Region,
		},
		IncreasedPlaybackSpeed: msg.IncreasedPlaybackSpeed,
	}

	matchLogStr := fmt.Sprintf("%d-%s", msg.MatchId, msg.PlatformId)

	outputFile := expectedDataOutputFile(f.DataOutputPath, &msg)
	exists, err := f.Files.Exists(outputFile)
	if err != nil {
		return fmt.Errorf("error checking if output file exists: %v", err)
	}
	if exists[0] && !msg.Override {
		log.Info(fmt.Sprintf("output file already exists (and override not set): %s", outputFile))
		return f.sendCompletedMessage(&msg, outputFile)
	}

	started := time.Now()
	startReplay(f.DevServerMode, f.CoordinatorURL, dataRequest, f.Constants.GameStartWait)

	// During the first startup, the bootstrap process starts and stops lots of
	// League processes. Wait for multiple no-process-found signals in a row
	// before declaring the process done
	consecutiveNoProcessFound := 0

	for i := 0; ; i++ {
		if !processIsRunningOnWindows() {
			consecutiveNoProcessFound++
		} else {
			consecutiveNoProcessFound = 0
		}

		if consecutiveNoProcessFound > 2 {
			endEloBuddyProcess()

			// Check to see if the file exists in C:\Users\USER\AppData\Roaming\EloBuddy
			localOutputFile := expectedLocalDataOutputFile()
			fileInfo, err := os.Stat(localOutputFile)
			if os.IsNotExist(err) {
				numDataExtractFailures++
				if numDataExtractFailures >= 3 {
					log.Warning("terminating self " + f.InstanceID)
					req := models.TerminateInstanceRequest{
						WorkerInstance: f.InstanceID,
						Region:         f.Region,
					}
					if err := postJSON(f.CoordinatorURL, "terminate_instance", req); err != nil {
						log.Error(err)
					}
				}
				return fmt.Errorf("data file not found for %s", matchLogStr)
			}
			numDataExtractFailures = 0

			// Check log file size
			if fileInfo.Size() < 1 {
				return fmt.Errorf("log file size is 0 for %s", matchLogStr)
			}

			// Validate the number of deaths, or retry.
			localMatchDetails := filepath.Join(os.TempDir(), "match_details.json")
			if err := f.Files.Copy(msg.MatchDetailsPath, localMatchDetails); err != nil {
				return err
			}
			defer os.RemoveAll(localMatchDetails)
			if err := ValidateDataFile(localOutputFile, localMatchDetails); err != nil {
				log.Warning(fmt.Errorf("match failed validation %s - %v", matchLogStr, err.Error()))
				defer os.Remove(localOutputFile)
				// The only known error that we should skip is stalled games, which result in death count mismatch.
				// All other errors should be recoverable with retries
				if strings.HasPrefix(err.Error(), "api and elo differ in deaths") {
					msg.RetryNum++
				}
				return retryMessage(ctx, f.MaxRetries, f.RetryTopic, msg)
			}

			if err := upload(f.Files, localOutputFile, outputFile); err != nil {
				return err
			}

			err = f.sendCompletedMessage(&msg, outputFile)
			if err != nil {
				log.Error(fmt.Sprintf("error publishing completed message: %v", err))
				return err
			}

			// Success
			log.Notice(fmt.Sprintf("successfully completed task for %s in %.1f minutes", matchLogStr, time.Since(started).Minutes()))
			return nil
		}

		// debug log every 5 minutes
		if i%60 == 0 {
			log.Debug(fmt.Sprintf("waiting for datagen match %s to finish", matchLogStr))
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

// startReplay sends a message to the coordinator to kick off the replay process
// (unless devServerMode is true, in which case a command is simply logged)
func startReplay(devServerMode bool, url string, req models.DatagenRequest, gameStartWait int64) error {
	if devServerMode {
		// When running the coordinator locally, the ec2 machine can't access
		// your local dev box. So wait for the request to be triggered manually
		data, err := json.Marshal(req)
		if err != nil {
			return err
		}
		log.Debug("Please issue this request in the next 2 minutes: " +
			"curl -d '" + string(data) + "' " + url + "/datagen_request")
		time.Sleep(2 * time.Minute)
	} else {
		// Hit the coordinator url to trigger the replay to start
		if err := postJSON(url, "datagen_request", req); err != nil {
			return err
		}
	}

	sleepTime := time.Duration(gameStartWait) * time.Minute
	log.Info("sleeping for " + sleepTime.String())
	time.Sleep(sleepTime)

	return nil
}

func (f *DatagenWorker) sendCompletedMessage(msg *messages.LolReplayDataExtract, outputFile string) error {
	// Send message to elo extract
	eloMsg := messages.LolEloDataProcess{
		MatchId:          msg.MatchId,
		PlatformId:       msg.PlatformId,
		EloDataPath:      outputFile,
		MatchDetailsPath: msg.MatchDetailsPath,
		SummonerIds:      msg.SummonerIds,
	}

	log.Notice(fmt.Sprintf("Adding elo message: %v", eloMsg))
	eloBytes, err := json.Marshal(eloMsg)
	if err != nil {
		log.Error(fmt.Sprintf("Error marshalling elo msg: %v", err))
		return err
	}

	pubsubMsg := &pubsub.Message{Data: eloBytes}
	_, err = f.OutTopic.Publish(context.Background(), pubsubMsg)
	return err
}

func expectedDataOutputFile(basePath string, msg *messages.LolReplayDataExtract) string {
	basePath = strings.TrimRight(basePath, "/")
	return fmt.Sprintf("%s/%v-%s-data_extract.txt", basePath, msg.MatchId, msg.PlatformId)
}

func expectedLocalDataOutputFile() string {
	return `C:\Users\Administrator\AppData\Roaming\EloBuddy\raw-events.txt`
}

// ValidateDataFile is basically lolelo/cmd/parse.go
// TODO: We can basically get rid of the elo process runner since we're doing the
// same thing here. We can just upload the baseview and move on to the next step
func ValidateDataFile(eloPath, matchDetailPath string) error {
	events, err := parse.LogFile(eloPath)
	if err != nil {
		return err
	}

	matchDetail := &api.MatchDetail{}
	if err = vsjson.DecodeFile(matchDetailPath, matchDetail); err != nil {
		return err
	}

	if err = validate.AlignAPI(events, matchDetail); err != nil {
		return err
	}
	if err = validate.CheckTruncation(events, matchDetail.MatchDuration); err != nil {
		return err
	}
	participants, err := validate.ParticipantMapMD(events, matchDetail)
	if err != nil {
		return err
	}

	_, err = event.EloToBaseview(events, participants)
	return err
}

// retryMessage attempts to increment the retryNum field of the message and
// adds the modified message to the retry queue. If the incremented value is
// higher than maxRetries, then no message is added and nil is returned.
func retryMessage(ctx context.Context, maxRetries int64, topic *pubsub.Topic, msg messages.LolReplayDataExtract) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	log.Info(fmt.Sprintf("attempting to retry %d-%s", msg.MatchId, msg.PlatformId))
	_, err = topic.Publish(ctx, &pubsub.Message{Data: data})
	return err
}

// upload zips and moves the local path to the remote path.
func upload(fc *files.Client, localPath, remotePath string) error {
	// Compress the file
	err := util.GzipFile(localPath, localPath, 0666)
	if err != nil {
		return fmt.Errorf("error compressing file: %v", err)
	}

	// Upload it to gcs
	return fc.Move(localPath, remotePath, files.ContentType("text/plain"), files.ContentEncoding("gzip"))
}
