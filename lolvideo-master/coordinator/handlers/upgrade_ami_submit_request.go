package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/VantageSports/common/credentials/aws"
	"github.com/VantageSports/lolvideo/coordinator"
)

func (h *LolVideoHandler) HandleUpgradeAmiSubmitRequest() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if RequireAuth(w, r) != nil {
			return
		}

		err := r.ParseForm()
		params := r.Form
		if params["amiId"] == nil || params["newName"] == nil ||
			params["maxPrice"] == nil || params["region"] == nil ||
			params["waitMinutes"] == nil {
			log.Println("Missing params amiId, newName, maxPrice, region, waitMinutes")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		awsClient := coordinator.GetEc2Client(
			aws.MustEnvCreds(),
			params.Get("region"))

		// Always launch 1 instance to upgrade the AMI
		numInstances := int64(1)

		// Create ec2UserData
		base64UserData := coordinator.BuildUpgradeAmiUserDataBase64(
			params.Get("newName"), params.Get("newDescription"),
			coordinator.CoordinatorUrlBase, params.Get("waitMinutes"))

		req, err := coordinator.AddWorkers(
			awsClient,
			params.Get("amiId"),
			numInstances,
			params.Get("maxPrice"),
			base64UserData,
			GetLaunchKeyName(coordinator.AwsRegionMap[params.Get("region")].LaunchKey),
			coordinator.AwsRegionMap[params.Get("region")].SecurityGroupID,
			coordinator.Ec2WorkerInstanceTypeElogen,
			false,
		)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		context := &addWorkerSubmitContext{SpotFleetRequestId: req}

		t, err := template.ParseFiles(coordinator.CoordinatorTemplatesDir + "/addWorkerSubmit.html")
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		t.Execute(w, context)
	}
}
