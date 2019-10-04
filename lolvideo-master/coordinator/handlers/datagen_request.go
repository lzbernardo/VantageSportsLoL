package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/VantageSports/common/credentials/aws"
	vsjson "github.com/VantageSports/common/json"
	"github.com/VantageSports/lolvideo/coordinator"
	"github.com/VantageSports/lolvideo/models"
)

// HandleDatagenRequest : Entrypoint to generate data
func (h *LolVideoHandler) HandleDatagenRequest() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		datagenRequest := &models.DatagenRequest{}
		if err := vsjson.DecodeLimit(r.Body, 2<<20, datagenRequest); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := datagenRequest.Valid(coordinator.AwsRegionMap); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		messageJson, err := json.Marshal(datagenRequest)
		if err != nil {
			log.Println("Unable to marshal datagenRequest", err)
			messageJson = []byte{}
		}
		log.Println("Executing valid datagen request", string(messageJson))

		replayRequest := datagenRequest.ReplayRequest
		displayStr := InitConnection(replayRequest.WorkerInstance, string(messageJson))
		ec2Client := coordinator.GetEc2Client(aws.MustEnvCreds(), replayRequest.Region)

		commands := []*coordinator.VNCCommand{}

		commands = append(commands, coordinator.CommandStartEloBuddy()...)
		// Datagen requests should terminate when the game ends.
		// This time represents a timeout in case the client hangs and never ends.
		// Extend it out a bit more generously
		replayRequest.GameLengthSeconds += 300
		commands = append(commands, coordinator.CommandRunReplay(replayRequest, 0, false)...)
		if datagenRequest.IncreasedPlaybackSpeed {
			commands = append(commands, coordinator.CommandSpeedUpReplay()...)
		}

		// Run all the vnc commands. This can take several minutes
		// Run it in a thread to prevent timeout
		go func() {
			coordinator.RunTaskOnWorker(displayStr, replayRequest.WorkerInstance, coordinator.VncPasswordFile, ec2Client, commands)
			Cleanup(replayRequest.WorkerInstance, displayStr)
		}()
		w.WriteHeader(http.StatusOK)
	}
}
