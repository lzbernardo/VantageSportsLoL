package handlers

import (
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/VantageSports/common/credentials/aws"
	"github.com/VantageSports/lolvideo/coordinator"
)

var MinutesToWaitForImageCreation = 5

// HandleUpgradeAmiCallbackRequest : Request from worker to initiate an upgrade
func (h *LolVideoHandler) HandleUpgradeAmiCallbackRequest() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		err := r.ParseForm()
		if err != nil {
			log.Println("Error parsing parameters")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		log.Println("Received Upgrade AMI Request: ", r.Form)
		params := r.Form

		if params["instanceId"] == nil || params["imageName"] == nil ||
			params["region"] == nil || params["waitMinutes"] == nil {
			log.Println("Missing params: instanceId, imageName, region, waitMinutes")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		waitMinutes, err := strconv.ParseInt(r.Form.Get("waitMinutes"), 10, 64)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		instance := r.Form.Get("instanceId")
		displayStr := InitConnection(instance, "Upgrade AMI Request to "+instance)
		ec2Client := coordinator.GetEc2Client(aws.MustEnvCreds(), r.Form.Get("region"))

		pw, err := coordinator.GetWindowsPassword(ec2Client, instance, coordinator.AwsRegionMap[r.Form.Get("region")].LaunchKey)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		commands := []*coordinator.VNCCommand{}
		commands = append(coordinator.CommandFirstTimeLogin(pw),
			coordinator.CommandPatchClient()...)

		// Run all the vnc commands. This should start the patch process.
		coordinator.RunTaskOnWorker(displayStr, instance, coordinator.VncPasswordFile, ec2Client, commands)
		Cleanup(instance, displayStr)

		// Wait a while for the patch to complete.
		log.Println("Patching. Waiting for", waitMinutes, "minutes")
		time.Sleep(time.Duration(waitMinutes) * time.Minute)

		// Package up the image as a new AMI
		// The image can take up to 30 minutes to be available
		newImage, err := coordinator.CreateAmi(ec2Client, instance, params.Get("imageName"), params.Get("imageDescription"))
		if err != nil {
			log.Println("Error creating new image", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println("New image requested:", *newImage)
		// We need to wait a few minutes for the snapshot to be created.
		// Don't terminate the instance during this time, or the creation will fail
		time.Sleep(time.Duration(MinutesToWaitForImageCreation) * time.Minute)

		// Tag the image. First, get the image id on the instance
		instanceObj, err := coordinator.GetInstance(ec2Client, instance)
		if err != nil {
			log.Println("Error looking up instance", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Then, use the imageId to get the tags on the old image
		imageObj, err := coordinator.GetImage(ec2Client, *instanceObj.ImageId)
		if err != nil {
			log.Println("Error looking up image", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		// Tag the new image with the tags from the old image
		_, err = ec2Client.CreateTags(&ec2.CreateTagsInput{
			Resources: []*string{newImage},
			Tags:      imageObj.Tags,
		})
		if err != nil {
			log.Println("Error adding tags to image", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Terminate the instance.
		err = coordinator.TerminateInstance(ec2Client, instance)
		if err != nil {
			log.Println("Error terminating instance", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		log.Println("Instance terminated")

		w.WriteHeader(http.StatusOK)
	}
}
