package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/datastore"
	"cloud.google.com/go/pubsub"
	"golang.org/x/net/context"

	"github.com/VantageSports/common/files"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/common/queue/messages"
	"github.com/VantageSports/lolvideo/coordinator"
	"github.com/VantageSports/lolvideo/models"
	"github.com/VantageSports/lolvideo/worker/lock"
)

// VideogenWorker : A struct to encapsulate the queue message logic
type VideogenWorker struct {
	InstanceID      string
	Region          string
	CoordinatorURL  string
	ClientVersion   string
	VideoOutputPath string
	Files           *files.Client
	DatastoreClient *datastore.Client
	DevServerMode   bool
}

const VideogenLockOwner = "LolVideogen"

// Handle : Take a Pub/Sub queue message and initiate a request to the coordinator web server.
//   The webserver should connect back to us with commands to record the video.
//   We also need to wait until the video is done recording before acknowledging the queue message
func (f *VideogenWorker) Handle(ctx context.Context, m *pubsub.Message) error {
	// Deserialize the message
	msg := messages.LolVideo{}
	if err := json.Unmarshal(m.Data, &msg); err != nil {
		return err
	}
	log.Notice(fmt.Sprintf("%+v", msg))

	if err := msg.Valid(); err != nil {
		return err
	}

	outputFile := expectedOutputFile(f.VideoOutputPath, &msg)

	// The client version (the description in the AMI) has to be a substring
	// of the version in the game message for it to work.
	// Example:
	//   AMI has clientVersion of "6.5."
	//   Message with MatchVersion of "6.5.1.255" passes
	//   Message with MatchVersion of "6.6.1.105" fails
	if msg.MatchVersion != "" && strings.Index(msg.MatchVersion, f.ClientVersion) != 0 {
		return errors.New("Client version mismatch")
	}

	// Construct our request to the coordinator
	vidRequest := models.VideoRequest{
		ReplayRequest: &models.ReplayRequest{
			GameID:            msg.MatchId,
			PlatformID:        msg.PlatformId,
			GameKey:           msg.Key,
			GameLengthSeconds: int(msg.GameLengthSeconds),
			SpectatorServer:   msg.SpectatorServer,
			MatchVersion:      msg.MatchVersion,
			WorkerInstance:    f.InstanceID,
			NeedsBootstrap:    false,
			Region:            f.Region,
		},
		ChampFocus: int(msg.ChampFocus),
	}

	// Check to see if this machine needs bootstrapping.
	// This file is created in the userdata template, and deleted in the replay.bat script
	if _, err := os.Stat(`c:\needs_bootstrap.txt`); err == nil {
		vidRequest.ReplayRequest.NeedsBootstrap = true
	}

	exists, err := f.Files.Exists(outputFile)
	if err != nil {
		return err
	}
	if exists[0] {
		log.Info("acking because file already exists: " + outputFile)
		return nil
	}

	if err := lock.Acquire(f.DatastoreClient, VideogenLockOwner, outputFile, msg.GameLengthSeconds); err != nil {
		return err
	}

	startTime := time.Now()

	if f.DevServerMode {
		// When running the coordinator locally, the ec2 machine can't access your local dev box.
		// So wait for the request to be triggered manually
		datajson, err := json.Marshal(vidRequest)
		if err != nil {
			_ = lock.DeleteLock(f.DatastoreClient, VideogenLockOwner, outputFile)
			return err
		}
		log.Debug("Please issue this request in the next 2 minutes: " +
			"curl -d '" + string(datajson) + "' " + f.CoordinatorURL + "/video_request")
		time.Sleep(2 * time.Minute)
	} else {
		// Hit the coordinator url to trigger the replay + video to start
		if err = postJSON(f.CoordinatorURL, "video_request", vidRequest); err != nil {
			// Delete the lock we created above
			_ = lock.DeleteLock(f.DatastoreClient, VideogenLockOwner, outputFile)
			return err
		}
	}

	// During the first startup, the bootstrap process starts and stops lots of League processes.
	// Wait for multiple no-process-found signals in a row before declaring the process done
	consecutiveNoProcessFound := 0
	// Wait a minute before checking. We need the coordinator to
	// do the vnc connection and the client to start up.
	// This also allows for the bootstrap process to fire up
	time.Sleep(5 * time.Minute)
	for i := 0; ; i++ {
		if !processIsRunningOnWindows() {
			consecutiveNoProcessFound++
		} else {
			consecutiveNoProcessFound = 0
		}

		if consecutiveNoProcessFound > 2 {
			// We're done. Delete the lock we created above
			err := lock.DeleteLock(f.DatastoreClient, VideogenLockOwner, outputFile)
			if err != nil {
				log.Error(fmt.Sprintf("Error deleting lock: %v", err))
			}
			endTime := time.Now()

			// If the process ended before the end of the game, then something went wrong, like
			// the client crashed or something.
			if endTime.Before(startTime.Add(time.Duration(msg.GameLengthSeconds) * time.Second)) {
				duration := endTime.Sub(startTime)
				return fmt.Errorf("Process ended before game length. Actual: %v Expected: %v", duration.Seconds(), msg.GameLengthSeconds)
			}

			// Check to see if the file exists in z:\
			localOutputFile := expectedLocalOutputFile(&msg)
			fileInfo, err := os.Stat(localOutputFile)
			if os.IsNotExist(err) {
				return errors.New("Video file not found")
			}

			// Check to see if the video is a reasonable file size
			// We record with an average bitrate of 1000kbps. If the video errors out and it's just a static image for a long time, the bitrate will go way down.
			// Lets set a cutoff of 900kbps. If it's less than that, then there's something wrong.
			// The video is 1 minute longer than the actual game length
			minimumFileSize := int64(coordinator.MinVideoBitrateFileCheck/8) * int64(msg.GameLengthSeconds+coordinator.VideoExtraDurationSeconds)
			if fileInfo.Size() < minimumFileSize {
				return fmt.Errorf("Video file too small. Actual: %d Expected to be at least %d", fileInfo.Size(), minimumFileSize)
			}

			// Upload it to gcs
			err = f.Files.Copy(localOutputFile, outputFile)
			if err != nil {
				return errors.New("Error uploading to gcs")
			}

			// Delete the local file
			err = os.Remove(localOutputFile)
			if err != nil {
				log.Error(fmt.Sprintf("Error deleting video file: %v", err))
			}

			// Success
			log.Notice("Successfully completed task")
			return nil
		}

		// debug log every 5 minutes
		if i%30 == 0 {
			log.Debug(fmt.Sprintf("waiting for videogen match %d to finish", msg.MatchId))
		}
		time.Sleep(10 * time.Second)
	}
	return nil
}

func expectedOutputFile(basePath string, msg *messages.LolVideo) string {
	basePath = strings.TrimRight(basePath, "/")
	return fmt.Sprintf("%s/%v-%s-%v.mp4", basePath, msg.MatchId, msg.PlatformId, msg.ChampFocus)
}

func expectedLocalOutputFile(msg *messages.LolVideo) string {
	return fmt.Sprintf(`z:\%v-%s-%v.mp4`, msg.MatchId, msg.PlatformId, msg.ChampFocus)
}
