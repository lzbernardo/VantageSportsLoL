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

type upgradeAmiContext struct {
	Amis          []*ec2.Image
	AwsRegions    map[string]*models.RegionDescriptor
	CurrentRegion string
	SpotPrices    map[string]string
}

func (h *LolVideoHandler) HandleUpgradeAmiRequest() func(http.ResponseWriter, *http.Request) {
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

		images, err := coordinator.GetWorkerAmis(awsClient, nil)
		spotPrices, _ := coordinator.GetSpotPrices(awsClient, coordinator.Ec2WorkerInstanceTypeElogen)
		// Transform the []*ec2.SpotPrice into a map of availabilityZone -> price
		// The array is already sorted in reverse chronological order, so we just
		//   get the first price for each availability zone
		priceMap := make(map[string]string)
		for _, value := range spotPrices {
			if _, ok := priceMap[*value.AvailabilityZone]; !ok {
				priceMap[*value.AvailabilityZone] = *value.SpotPrice
			}
		}

		context := &addWorkerContext{
			Amis:          images,
			AwsRegions:    coordinator.AwsRegionMap,
			CurrentRegion: awsRegion,
			SpotPrices:    priceMap,
		}

		t, err := template.ParseFiles(coordinator.CoordinatorTemplatesDir + "/upgradeAmi.html")
		if err != nil {
			log.Fatalln(err)
		}
		t.Execute(w, context)
	}
}
