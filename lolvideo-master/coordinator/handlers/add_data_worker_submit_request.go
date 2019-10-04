package handlers

import (
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/VantageSports/common/credentials/aws"
	"github.com/VantageSports/lolvideo/coordinator"
)

func (h *LolVideoHandler) HandleAddDataWorkerSubmitRequest() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if RequireAuth(w, r) != nil {
			return
		}

		err := r.ParseForm()
		params := r.Form
		if params["amiId"] == nil || params["numInstances"] == nil ||
			params["maxPrice"] == nil || params["region"] == nil {
			log.Println("Missing params amiId, numInstances, maxPrice, and region")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		awsClient := coordinator.GetEc2Client(
			aws.MustEnvCreds(),
			params.Get("region"))

		numInstances, err := strconv.ParseInt(params.Get("numInstances"), 10, 64)
		if err != nil || numInstances < 1 {
			log.Println("unable to parse numInstances:", params.Get("numInstances"))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		imageObj, err := coordinator.GetImage(awsClient, params.Get("amiId"))
		if err != nil {
			log.Println("Unable to retrieve ami")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Create ec2UserData
		base64UserData := coordinator.BuildDatagenReplayUserDataBase64(
			coordinator.GoogProjectID,
			coordinator.Ec2WorkerGoogCredsJSON,
			coordinator.CoordinatorUrlBase,
			coordinator.PubsubInputElo,
			*imageObj.Description,
			params.Get("region"))

		req, err := coordinator.AddWorkers(
			awsClient,
			params.Get("amiId"),
			numInstances,
			params.Get("maxPrice"),
			base64UserData,
			GetLaunchKeyName(coordinator.AwsRegionMap[params.Get("region")].LaunchKey),
			coordinator.AwsRegionMap[params.Get("region")].SecurityGroupID,
			coordinator.Ec2WorkerInstanceTypeElogen,
			params.Get("maintain") != "",
		)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if params.Get("autoscale") != "" {
			err = coordinator.AttachAutoscalePolicy(
				aws.MustEnvCreds(),
				params.Get("region"),
				*req)
			if err != nil {
				log.Println(err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
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
