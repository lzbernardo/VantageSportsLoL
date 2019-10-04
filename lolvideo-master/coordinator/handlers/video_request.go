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

// HandleVideoRequest : Entrypoint to record a replay video
func (h *LolVideoHandler) HandleVideoRequest() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		vidRequest := &models.VideoRequest{}
		if err := vsjson.DecodeLimit(r.Body, 2<<20, vidRequest); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := vidRequest.Valid(coordinator.AwsRegionMap); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		messageJson, err := json.Marshal(vidRequest)
		if err != nil {
			log.Println("Unable to marshal vidRequest", err)
			messageJson = []byte{}
		}
		log.Println("Executing valid video request", string(messageJson))

		replayRequest := vidRequest.ReplayRequest
		displayStr := InitConnection(replayRequest.WorkerInstance, string(messageJson))
		ec2Client := coordinator.GetEc2Client(aws.MustEnvCreds(), replayRequest.Region)

		commands := []*coordinator.VNCCommand{}

		commands = append(commands, coordinator.CommandRunReplay(replayRequest, vidRequest.ChampFocus, true)...)
		commands = append(commands, coordinator.CommandFocusOnChamp(vidRequest.ChampFocus)...)

		// Run all the vnc commands. This can take several minutes
		// Run it in a thread to prevent timeout
		go func() {
			coordinator.RunTaskOnWorker(displayStr, replayRequest.WorkerInstance, coordinator.VncPasswordFile, ec2Client, commands)
			Cleanup(replayRequest.WorkerInstance, displayStr)
		}()
		w.WriteHeader(http.StatusOK)
	}
}
