package main

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"google.golang.org/api/cloudmonitoring/v2beta2"

	vsaws "github.com/VantageSports/common/credentials/aws"
	"github.com/VantageSports/common/log"
	"github.com/VantageSports/lolvideo/coordinator"
)

func reportQueueSize(tsClient *cloudmonitoring.TimeseriesService, projID, queueID string, duration time.Duration) {
	// Only report cloudwatch to regions with persistent spot fleets
	regions := []string{"us-east-1", "us-west-2", "eu-west-1", "ap-northeast-1"}
	awsClients := []*cloudwatch.CloudWatch{}
	for _, region := range regions {
		session := session.New(&aws.Config{Credentials: vsaws.MustEnvCreds()})
		awsClients = append(awsClients, cloudwatch.New(session, aws.NewConfig().WithRegion(region)))
	}

	for {
		// Get the total number of queue adds in the last minute
		queueAdds := coordinator.GetQueueAdds(tsClient, projID, queueID, 1)
		metricValue := float64(queueAdds)

		// Publish the queue size to all aws regions
		for _, client := range awsClients {
			_, err := client.PutMetricData(&cloudwatch.PutMetricDataInput{
				Namespace: &coordinator.AutoscalePolicy.AlarmNamespace,
				MetricData: []*cloudwatch.MetricDatum{
					&cloudwatch.MetricDatum{
						MetricName: &coordinator.AutoscalePolicy.AlarmMetricName,
						Value:      &metricValue,
						Unit:       &coordinator.AutoscalePolicy.AlarmUnit,
					},
				},
			})

			if err != nil {
				log.Error(err)
			}
		}
		time.Sleep(duration)
	}
}
