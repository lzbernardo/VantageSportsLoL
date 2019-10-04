package coordinator

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/applicationautoscaling"
	"github.com/aws/aws-sdk-go/service/cloudwatch"
	"github.com/aws/aws-sdk-go/service/ec2"
	"golang.org/x/net/context"
	"google.golang.org/api/cloudmonitoring/v2beta2"

	"github.com/VantageSports/common/credentials/google"
)

// VncPasswordDecryptor : This is the utility to get the plaintext password
const VncPasswordDecryptor = "vncpwd"

var vncPassword = ""

// RunTaskOnWorker : Entrypoint to execute a video request
func RunTaskOnWorker(virtualDisplay, workerInstance, vncPasswordFile string,
	ec2Client *ec2.EC2, commands []*VNCCommand) error {
	instance, err := GetInstance(ec2Client, workerInstance)
	if err != nil {
		return err
	}
	host := *instance.PublicDnsName
	vnc, err := openVnc(virtualDisplay, host, vncPasswordFile)
	if err != nil {
		return err
	}
	defer func() {
		log.Println("Disconnecting vnc session to", host)
		vnc.Process.Kill()
	}()

	return vncCmdList(host, vncPasswordFile, commands)
}

func openVnc(virtualDisplay, host, vncPasswordFile string) (c *exec.Cmd, error error) {
	os.Setenv("DISPLAY", virtualDisplay)

	log.Println("Running xvnc4viewer -geometry 1280x1024 -passwd", vncPasswordFile, host)
	cmd := exec.Command("xvnc4viewer", "-geometry", "1280x1024", "-passwd", vncPasswordFile, host)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	err := cmd.Start()

	// Wait a few seconds, then look for any errors in the output
	time.Sleep(3 * time.Second)

	if bytes.Contains(errBuf.Bytes(), []byte("unable to connect to host:")) {
		log.Println("VNC Failed", errBuf.String())
		err = errors.New("VNC Failed")
	} else if bytes.Contains(errBuf.Bytes(), []byte("Authentication failed")) {
		log.Println("VNC Failed", errBuf.String())
		err = errors.New("VNC Failed")
	}
	return cmd, err
}

func vncCmd(host, vncPasswordFile string, command *VNCCommand) error {
	if command.Name == "sleep" {
		log.Println("Sleeping for", command.Argument, "milliseconds")
		milliSeconds, err := strconv.Atoi(command.Argument)
		if err != nil {
			log.Println(err)
			return err
		}
		time.Sleep(time.Duration(milliSeconds) * time.Millisecond)
	} else {
		log.Println("Running cmd: vncdo -s", host, "-p", "****", command.Name, command.Argument)

		cmd := exec.Command("vncdo", "-s", host, "-p", getVncPassword(vncPasswordFile), command.Name, command.Argument)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Println("Error with vncCmd:", string(output))
		}
		return err
	}
	return nil
}

func vncCmdList(host, vncPasswordFile string, commands []*VNCCommand) error {
	for i := range commands {
		err := vncCmd(host, vncPasswordFile, commands[i])
		if err != nil {
			return err
		}
	}
	return nil
}

func getVncPassword(vncPasswordFile string) string {
	if vncPassword == "" {
		cmd := exec.Command(VncPasswordDecryptor, vncPasswordFile)

		out, err := cmd.CombinedOutput()
		if err != nil {
			log.Fatalln("Unable to decrypt vnc password", err.Error())
		}
		// The tool outputs a line with the prefix "Password: ", which is 10 characters
		vncPassword = strings.TrimSpace(string(out))[10:]
	}
	return vncPassword
}

func GetEc2Client(creds *awsCredentials.Credentials, region string) *ec2.EC2 {
	session := session.New(&aws.Config{Credentials: creds})
	return ec2.New(session, aws.NewConfig().WithRegion(region))
}

func GetInstance(ec2Client *ec2.EC2, instance string) (*ec2.Instance, error) {
	response, err := ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{&instance},
	})

	if err != nil {
		log.Println(err)
		return nil, err
	}
	if len(response.Reservations) == 0 ||
		len(response.Reservations[0].Instances) == 0 ||
		*response.Reservations[0].Instances[0].PublicDnsName == "" {
		log.Println("Invalid instance:", instance)
		return nil, errors.New("Invalid instance:" + instance)
	}
	ret := response.Reservations[0].Instances[0]
	return ret, nil
}

func GetWorkerInstances(ec2Client *ec2.EC2, instanceTypes []string) []*ec2.Instance {
	name1 := "instance-type"
	values1 := make([]*string, len(instanceTypes))
	for i, t := range instanceTypes {
		sCopy := t
		values1[i] = &sCopy
	}
	// instance-state-code 16 means it's running
	name2 := "instance-state-code"
	value2 := "16"

	response, err := ec2Client.DescribeInstances(&ec2.DescribeInstancesInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{
				Name:   &name1,
				Values: values1,
			},
			&ec2.Filter{
				Name:   &name2,
				Values: []*string{&value2},
			},
		},
	})

	if err != nil {
		log.Println(err)
		return nil
	}

	var instances []*ec2.Instance
	// Loop over all reservations and all instances, and flatten the result
	for _, res := range response.Reservations {
		for _, ins := range res.Instances {
			instances = append(instances, ins)
		}
	}
	return instances
}

func GetSpotInstanceRequests(ec2Client *ec2.EC2, instances []*ec2.Instance) []*ec2.SpotInstanceRequest {
	if len(instances) == 0 {
		return nil
	}

	requestIds := []*string{}
	for _, instance := range instances {
		if instance.SpotInstanceRequestId != nil {
			requestIds = append(requestIds, instance.SpotInstanceRequestId)
		}
	}

	response, err := ec2Client.DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
		SpotInstanceRequestIds: requestIds,
	})

	if err != nil {
		log.Println(err)
		return nil
	}

	return response.SpotInstanceRequests
}

func GetWindowsPassword(ec2Client *ec2.EC2, instance, launchKey string) (s string, err error) {
	maxTries := 5
	for i := 0; i < maxTries; i++ {
		response, err := ec2Client.GetPasswordData(&ec2.GetPasswordDataInput{
			InstanceId: &instance,
		})

		if err != nil {
			log.Fatalln(err)
		}

		if response.PasswordData == nil || *response.PasswordData == "" {
			log.Println("Unable to retrieve password:", "Waiting 1 minute")
			time.Sleep(time.Duration(1) * time.Minute)
		} else {
			// This gives us the encrypted password. We have to decrypt it with our launch key
			// Borrowed from https://github.com/tomrittervg/decrypt-windows-ec2-passwd/blob/master/decrypt-windows-ec2-passwd.go
			decoded, _ := base64.StdEncoding.DecodeString(strings.TrimSpace(*response.PasswordData))
			pemBytes, _ := ioutil.ReadFile(launchKey)
			block, _ := pem.Decode(pemBytes)
			key, _ := x509.ParsePKCS1PrivateKey(block.Bytes)
			out, _ := rsa.DecryptPKCS1v15(nil, key, decoded)

			return string(out), nil
		}
	}
	return "", errors.New("Exceeded max tries for getPassword")
}

func GetWorkerAmis(ec2Client *ec2.EC2, tagFilters map[string]string) ([]*ec2.Image, error) {
	tagFiltersList := []*ec2.Filter{{
		Name:   aws.String("platform"),
		Values: []*string{aws.String("windows")}},
		&ec2.Filter{
			Name:   aws.String("is-public"),
			Values: []*string{aws.String("false")}},
		&ec2.Filter{
			Name:   aws.String("state"),
			Values: []*string{aws.String("available")}},
	}

	for k, v := range tagFilters {
		str := "tag:" + k
		tagFiltersList = append(tagFiltersList, &ec2.Filter{
			Name:   &str,
			Values: []*string{&v},
		})
	}

	response, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{
		Filters: tagFiltersList,
	})

	if err != nil {
		log.Println(err)
		return nil, err
	}

	if len(response.Images) == 0 {
		log.Println("No images found!")
		return nil, err
	}
	return response.Images, nil
}

func GetImage(ec2Client *ec2.EC2, imageId string) (*ec2.Image, error) {
	response, err := ec2Client.DescribeImages(&ec2.DescribeImagesInput{
		ImageIds: []*string{&imageId},
	})

	if err != nil {
		log.Println(err)
		return nil, err
	}

	if len(response.Images) == 0 {
		log.Println("No images found!")
		return nil, err
	}
	return response.Images[0], nil
}

func AddWorkers(ec2Client *ec2.EC2, amiId string, numInstances int64,
	maxPrice, base64UserData, launchKey, securityGroup,
	workerInstanceType string, maintain bool) (*string, error) {

	typeString := "request"
	if maintain {
		typeString = "maintain"
	}
	// Create the launch spec
	launchSpec := ec2.SpotFleetLaunchSpecification{
		ImageId: &amiId,
		KeyName: &launchKey,
		SecurityGroups: []*ec2.GroupIdentifier{
			&ec2.GroupIdentifier{
				GroupId: &securityGroup,
			}},
		UserData:     &base64UserData,
		InstanceType: &workerInstanceType,
	}

	// Issue the request spot fleet request
	output, err := ec2Client.RequestSpotFleet(&ec2.RequestSpotFleetInput{
		SpotFleetRequestConfig: &ec2.SpotFleetRequestConfigData{
			IamFleetRole:         &IamFleetRole,
			SpotPrice:            &maxPrice,
			TargetCapacity:       &numInstances,
			Type:                 &typeString,
			LaunchSpecifications: []*ec2.SpotFleetLaunchSpecification{&launchSpec},
		},
	})

	if err != nil {
		log.Println(err)
		return nil, err
	}
	return output.SpotFleetRequestId, nil
}

func BuildRecordReplayUserDataBase64(projectID, workerGoogCreds, coordinatorUrlBase, inputQueueID, clientVersion string) string {
	context := struct {
		GoogProjectID   string
		GoogCreds       string
		UrlBase         string
		InputQueueID    string
		ClientVersion   string
		VideoOutputPath string
		OutputQueueID   string
	}{
		GoogProjectID:   projectID,
		GoogCreds:       workerGoogCreds,
		UrlBase:         coordinatorUrlBase,
		InputQueueID:    inputQueueID,
		ClientVersion:   clientVersion,
		VideoOutputPath: VideoOutputPath,
		OutputQueueID:   PubsubOutputVideo,
	}
	t, _ := template.ParseFiles(ProjectSourceDir + "/worker/ec2-userdata.template")
	var output bytes.Buffer
	t.Execute(&output, context)

	return base64.StdEncoding.EncodeToString(output.Bytes())
}

func BuildUpgradeAmiUserDataBase64(newName, newDescription, coordinatorUrlBase, waitMinutes string) string {
	context := struct {
		NewName        string
		NewDescription string
		UrlBase        string
		WaitMinutes    string
	}{
		NewName:        newName,
		NewDescription: newDescription,
		UrlBase:        coordinatorUrlBase,
		WaitMinutes:    waitMinutes,
	}
	t, _ := template.ParseFiles(ProjectSourceDir + "/worker/ec2-userdata-upgrade.template")
	var output bytes.Buffer
	t.Execute(&output, context)

	return base64.StdEncoding.EncodeToString(output.Bytes())
}

func BuildDatagenReplayUserDataBase64(projectID, workerGoogCreds, coordinatorUrlBase, inputQueueID, clientVersion, awsRegion string) string {
	context := struct {
		GoogProjectID      string
		GoogCreds          string
		UrlBase            string
		InputQueueID       string
		ClientVersion      string
		DataOutputPath     string
		OutputQueueID      string
		PluginDownloadPath string
		AwsRegion          string
	}{
		GoogProjectID:      projectID,
		GoogCreds:          workerGoogCreds,
		UrlBase:            coordinatorUrlBase,
		InputQueueID:       inputQueueID,
		ClientVersion:      clientVersion,
		DataOutputPath:     ElodataOutputPath,
		OutputQueueID:      PubsubOutputElo,
		PluginDownloadPath: PluginDownloadPath,
		AwsRegion:          awsRegion,
	}
	t, _ := template.ParseFiles(ProjectSourceDir + "/worker/ec2-userdata-datagen.template")
	var output bytes.Buffer
	t.Execute(&output, context)

	return base64.StdEncoding.EncodeToString(output.Bytes())
}

func CreateAmi(ec2Client *ec2.EC2, instance, name, description string) (*string, error) {
	output, err := ec2Client.CreateImage(&ec2.CreateImageInput{
		InstanceId:  &instance,
		Name:        &name,
		Description: &description,
	})

	if err != nil {
		return nil, err
	}

	return output.ImageId, nil
}

func TerminateInstance(ec2Client *ec2.EC2, instance string) error {
	_, err := ec2Client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{&instance}})
	return err
}

func GetSpotPrices(ec2Client *ec2.EC2, instanceType string) ([]*ec2.SpotPrice, error) {
	value1 := instanceType
	value2 := "Windows"
	startTime := time.Now().Add(-5 * time.Minute)
	endTime := time.Now()
	response, err := ec2Client.DescribeSpotPriceHistory(&ec2.DescribeSpotPriceHistoryInput{
		InstanceTypes:       []*string{&value1},
		ProductDescriptions: []*string{&value2},
		StartTime:           &startTime,
		EndTime:             &endTime,
	})

	if err != nil {
		log.Println(err)
		return nil, err
	}

	return response.SpotPriceHistory, nil
}

func GetGoogleTimeseriesClient(creds *google.Creds) *cloudmonitoring.TimeseriesService {
	ctx := context.Background()
	cloudClient, err := cloudmonitoring.New(creds.Conf.Client(ctx))
	exitIf(err)

	return cloudmonitoring.NewTimeseriesService(cloudClient)
}

func GetQueueSize(client *cloudmonitoring.TimeseriesService, projectID, subscriptionID string) int64 {
	now := time.Now().Format(time.RFC3339)
	ltsr := cloudmonitoring.ListTimeseriesRequest{}

	resp, err := client.List(projectID, QueueMetricSize, now, &ltsr).
		Labels("pubsub.googleapis.com/resource_id==" + subscriptionID).
		Timespan("1h").
		Do()

	exitIf(err)
	if len(resp.Timeseries) == 0 {
		log.Println("No timeseries data found for subscription: " + subscriptionID)
		return -1
	}
	// Just get the first one. Assume it's the lol-videogen-tasks one because of the filter above
	ts := resp.Timeseries[0]
	// Grab the first data point, which is the most recent one
	return *ts.Points[0].Int64Value
}

// GetQueueAdds returns the sum of the number of queue inserts over the last "windowMinutes" minutes
func GetQueueAdds(client *cloudmonitoring.TimeseriesService, projectID, subscriptionID string, windowMinutes int) int64 {
	// Subtract 3 minutes because there's a delay when google populates its data
	now := time.Now().Add(-3 * time.Minute).Format(time.RFC3339)
	ltsr := cloudmonitoring.ListTimeseriesRequest{}
	windowStr := fmt.Sprintf("%vm", windowMinutes)

	resp, err := client.List(projectID, QueueMetricAdds, now, &ltsr).
		Labels("pubsub.googleapis.com/resource_id==" + subscriptionID).
		Aggregator("sum").
		Timespan(windowStr).
		Window(windowStr).
		Do()

	exitIf(err)
	if len(resp.Timeseries) == 0 {
		return 0
	}
	// Just get the first one. Assume it's the lol-videogen-tasks one because of the filter above
	ts := resp.Timeseries[0]
	// Grab the first data point, which is the most recent one
	return *ts.Points[0].Int64Value
}

func AttachAutoscalePolicy(creds *awsCredentials.Credentials, region string, sfr string) error {
	session := session.New(&aws.Config{Credentials: creds})
	autoscaling := applicationautoscaling.New(session, aws.NewConfig().WithRegion(region))
	cloudwatchApi := cloudwatch.New(session, aws.NewConfig().WithRegion(region))

	policyResourceId := "spot-fleet-request/" + sfr

	// We have to register the spot fleet request id as a scalable target first
	_, err := autoscaling.RegisterScalableTarget(&applicationautoscaling.RegisterScalableTargetInput{
		MinCapacity:       &AutoscalePolicy.MinCapacity,
		MaxCapacity:       &AutoscalePolicy.MaxCapacity,
		ResourceId:        &policyResourceId,
		ScalableDimension: &AutoscalePolicy.ScalableDimension,
		ServiceNamespace:  &AutoscalePolicy.ServiceNamespace,
		RoleARN:           &AutoscalePolicy.RoleARN,
	})

	if err != nil {
		return err
	}

	// Attach the scale out policy
	outResp, err := AttachScalingPolicy(autoscaling, AutoscalePolicy.ScaleOutPolicy, true, policyResourceId)
	if err != nil {
		return err
	}

	// Then, we attach the scale in policy
	inResp, err := AttachScalingPolicy(autoscaling, AutoscalePolicy.ScaleInPolicy, false, policyResourceId)
	if err != nil {
		return err
	}

	// Finally, we need to set the alarm to trigger the policies
	// NOTE: This overwrites any previous alarm with the same name,
	// so this doesn't support multiple autoscaling fleets (every time we
	// launch a new autoscale fleet, the alarm will overwrite the previous alarm)
	// To enable this behavior, we need to read existing alarms, and append our
	// action on top of the existing actions.

	// Start with the scale out alarm
	actionsEnabled := true

	_, err = cloudwatchApi.PutMetricAlarm(&cloudwatch.PutMetricAlarmInput{
		ActionsEnabled:     &actionsEnabled,
		AlarmActions:       []*string{outResp.PolicyARN},
		AlarmName:          &AutoscalePolicy.ScaleOutPolicy.AlarmName,
		ComparisonOperator: &AutoscalePolicy.ScaleOutPolicy.AlarmComparisonOperator,
		EvaluationPeriods:  &AutoscalePolicy.ScaleOutPolicy.AlarmEvaluationPeriods,
		MetricName:         &AutoscalePolicy.AlarmMetricName,
		Namespace:          &AutoscalePolicy.AlarmNamespace,
		Period:             &AutoscalePolicy.AlarmPeriod,
		Statistic:          &AutoscalePolicy.AlarmStatistic,
		Threshold:          &AutoscalePolicy.AlarmThreshold,
		Unit:               &AutoscalePolicy.AlarmUnit,
	})

	if err != nil {
		return err
	}

	// Then do the scalein alarm
	_, err = cloudwatchApi.PutMetricAlarm(&cloudwatch.PutMetricAlarmInput{
		ActionsEnabled:     &actionsEnabled,
		AlarmActions:       []*string{inResp.PolicyARN},
		AlarmName:          &AutoscalePolicy.ScaleInPolicy.AlarmName,
		ComparisonOperator: &AutoscalePolicy.ScaleInPolicy.AlarmComparisonOperator,
		EvaluationPeriods:  &AutoscalePolicy.ScaleInPolicy.AlarmEvaluationPeriods,
		MetricName:         &AutoscalePolicy.AlarmMetricName,
		Namespace:          &AutoscalePolicy.AlarmNamespace,
		Period:             &AutoscalePolicy.AlarmPeriod,
		Statistic:          &AutoscalePolicy.AlarmStatistic,
		Threshold:          &AutoscalePolicy.AlarmThreshold,
		Unit:               &AutoscalePolicy.AlarmUnit,
	})

	return err
}

func AttachScalingPolicy(api *applicationautoscaling.ApplicationAutoScaling, policyConfig *AutoscalePolicyConfig, isScaleUp bool, policyResourceId string) (*applicationautoscaling.PutScalingPolicyOutput, error) {
	// Prepare the input
	policySteps := []*applicationautoscaling.StepAdjustment{}
	for _, step := range policyConfig.Steps {
		stepCopy := step
		policySteps = append(policySteps, &applicationautoscaling.StepAdjustment{
			MetricIntervalLowerBound: &stepCopy.LowerBound,
			MetricIntervalUpperBound: &stepCopy.UpperBound,
			ScalingAdjustment:        &stepCopy.ScalingAdjustment,
		})
	}
	if isScaleUp {
		// Set the last step's UpperBound to null (infinity)
		policySteps[len(policySteps)-1].MetricIntervalUpperBound = nil
	} else {
		// Set the last step's LowerBound to null (negative infinity)
		policySteps[len(policySteps)-1].MetricIntervalLowerBound = nil
	}

	stepPolicyConfig := applicationautoscaling.StepScalingPolicyConfiguration{
		AdjustmentType:  &policyConfig.PolicyAdjustmentType,
		Cooldown:        &policyConfig.PolicyCooldown,
		StepAdjustments: policySteps,
	}
	// Then, we attach the scale out policy
	return api.PutScalingPolicy(&applicationautoscaling.PutScalingPolicyInput{
		PolicyName:                     &policyConfig.PolicyName,
		PolicyType:                     &policyConfig.PolicyType,
		ResourceId:                     &policyResourceId,
		ScalableDimension:              &AutoscalePolicy.ScalableDimension,
		ServiceNamespace:               &AutoscalePolicy.ServiceNamespace,
		StepScalingPolicyConfiguration: &stepPolicyConfig,
	})
}

func exitIf(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
