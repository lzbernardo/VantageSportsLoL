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

// HandleDatagenBootstrapRequest : Entrypoint to bootstrap an instance
func (h *LolVideoHandler) HandleDatagenBootstrapRequest() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {

		bootstrapRequest := &models.DatagenBootstrapRequest{}
		if err := vsjson.DecodeLimit(r.Body, 2<<20, bootstrapRequest); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := bootstrapRequest.Valid(coordinator.AwsRegionMap); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		messageJson, err := json.Marshal(bootstrapRequest)
		if err != nil {
			log.Println("Unable to marshal bootstrapRequest", err)
			messageJson = []byte{}
		}
		log.Println("Executing valid bootstrap request", string(messageJson))

		displayStr := InitConnection(bootstrapRequest.WorkerInstance, string(messageJson))
		ec2Client := coordinator.GetEc2Client(aws.MustEnvCreds(), bootstrapRequest.Region)

		commands := []*coordinator.VNCCommand{}

		pw, err := coordinator.GetWindowsPassword(ec2Client, bootstrapRequest.WorkerInstance, coordinator.AwsRegionMap[bootstrapRequest.Region].LaunchKey)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Generate the bootstrap  commands
		commands = append(coordinator.CommandFirstTimeLogin(pw), coordinator.CommandBootstrapEloBuddy()...)
		commands = append(commands, coordinator.CommandBootstrapClient(bootstrapRequest)...)

		// Run all the vnc commands. This can take several minutes
		// Run it in a thread to prevent timeout
		go func() {
			coordinator.RunTaskOnWorker(displayStr, bootstrapRequest.WorkerInstance, coordinator.VncPasswordFile, ec2Client, commands)
			Cleanup(bootstrapRequest.WorkerInstance, displayStr)
		}()
		w.WriteHeader(http.StatusOK)
	}
}
