package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/aws/aws-sdk-go/service/ec2"

	"github.com/VantageSports/common/credentials/aws"
	"github.com/VantageSports/lolvideo/coordinator"
	"github.com/VantageSports/lolvideo/models"
)

type statusContext struct {
	QueueSize             int64
	DatagenQueueSize      int64
	DisplayStatuses       []*DisplayStatus
	WorkerMachines        []*ec2.Instance
	WorkerMachineRequests []*ec2.SpotInstanceRequest
	GoogSubscriptionID    string
	ElogenQueueID         string
	GoogProjectID         string
	AwsRegions            map[string]*models.RegionDescriptor
	CurrentRegion         string
}

func (h *LolVideoHandler) HandleStatusRequest() func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if RequireAuth(w, r) != nil {
			return
		}

		err := r.ParseForm()
		awsRegion := coordinator.AwsDefaultRegion
		if r.Form.Get("region") != "" {
			awsRegion = r.Form.Get("region")
		}

		awsClient := coordinator.GetEc2Client(
			aws.MustEnvCreds(),
			awsRegion)

		workerMachines := coordinator.GetWorkerInstances(awsClient, []string{coordinator.Ec2WorkerInstanceTypeVideogen, coordinator.Ec2WorkerInstanceTypeElogen})
		// Look up all the spot instance requests for all the worker instances.
		// The custom tags created are on the requests, not the instances
		workerMachineRequests := coordinator.GetSpotInstanceRequests(awsClient, workerMachines)

		context := &statusContext{
			QueueSize: -1,
			DatagenQueueSize: coordinator.GetQueueSize(
				h.TimeseriesClient,
				coordinator.GoogProjectID,
				coordinator.PubsubInputElo,
			),
			DisplayStatuses:       GetDisplayStatuses(),
			WorkerMachines:        workerMachines,
			WorkerMachineRequests: workerMachineRequests,
			GoogSubscriptionID:    coordinator.PubsubInputVideo,
			ElogenQueueID:         coordinator.PubsubInputElo,
			GoogProjectID:         coordinator.GoogProjectID,
			AwsRegions:            coordinator.AwsRegionMap,
			CurrentRegion:         awsRegion,
		}

		// We need to compare *string objects because we're working with ec2 structs.
		// The standard template doesn't allow "eq" with string pointers, so write a deref func
		t, err := template.New("status.html").Funcs(template.FuncMap{"Deref": derefStr}).ParseFiles(coordinator.CoordinatorTemplatesDir + "/status.html")
		if err != nil {
			log.Println(err)
			return
		}

		err = t.Execute(w, context)
		if err != nil {
			log.Println(err)
		}
	}
}

// derefStr is a utility function for use in templates that deal with ec2 string values
// (which are mostly string pointers) for easy comparison.
func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
