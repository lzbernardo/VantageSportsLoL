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

func (h *LolVideoHandler) HandleTerminateInstanceRequest() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		terminateRequest := &models.TerminateInstanceRequest{}
		if err := vsjson.DecodeLimit(r.Body, 2<<20, terminateRequest); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := terminateRequest.Valid(); err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		messageJson, err := json.Marshal(terminateRequest)
		if err != nil {
			log.Println("Unable to marshal terminateRequest", err)
			messageJson = []byte{}
		}
		log.Println("Executing terminate request", string(messageJson))

		ec2Client := coordinator.GetEc2Client(aws.MustEnvCreds(), terminateRequest.Region)
		err = coordinator.TerminateInstance(ec2Client, terminateRequest.WorkerInstance)
		if err != nil {
			log.Println("Error terminating instance", err)
			w.Write([]byte("ERROR"))
		} else {
			w.Write([]byte("SUCCESS"))
		}
	}
}
