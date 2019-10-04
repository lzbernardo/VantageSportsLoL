package coordinator

import (
	"log"
	"strconv"
	"strings"

	"github.com/VantageSports/common/env"
	"github.com/VantageSports/lolvideo/models"
)

const (
	VideoExtraDurationSeconds = 60
	QueueMetricAdds           = "pubsub.googleapis.com/topic/send_message_operation_count"
	QueueMetricSize           = "pubsub.googleapis.com/subscription/num_undelivered_messages"
)

type AutoscaleStep struct {
	LowerBound        float64
	UpperBound        float64
	ScalingAdjustment int64
}

type AutoscalePolicyConfig struct {
	AlarmName               string
	AlarmComparisonOperator string
	AlarmEvaluationPeriods  int64

	PolicyName           string
	PolicyType           string
	PolicyAdjustmentType string
	PolicyCooldown       int64
	Steps                []AutoscaleStep
}

type AutoscaleConfig struct {
	MinCapacity       int64
	MaxCapacity       int64
	RoleARN           string
	ScalableDimension string
	ServiceNamespace  string

	AlarmNamespace         string
	AlarmMetricName        string
	AlarmThreshold         float64
	AlarmPeriod            int64
	AlarmEvaluationPeriods int64
	AlarmStatistic         string
	AlarmUnit              string

	ScaleOutPolicy *AutoscalePolicyConfig
	ScaleInPolicy  *AutoscalePolicyConfig
}

var GoogProjectID,
	GoogCredsJSON,
	AwsCredsJSON,
	AwsRegions,
	AwsDefaultRegion,
	Ec2LaunchKey,
	Ec2SecurityGroupID,
	Ec2WorkerInstanceTypeVideogen,
	Ec2WorkerInstanceTypeElogen,
	Ec2WorkerGoogCredsJSON,
	CoordinatorUrlBase,
	VncPasswordFile,
	ClientMode,
	ClientServer,
	CoordinatorTemplatesDir,
	ProjectSourceDir,
	VideoOutputPath,
	AuthHost,
	PubsubInputElo,
	PubsubInputVideo,
	PubsubOutputElo,
	PubsubOutputVideo,
	ElodataOutputPath,
	PluginDownloadPath,
	IamFleetRole string

var AwsRegionMap map[string]*models.RegionDescriptor
var MinVideoBitrateFileCheck int
var AutoscalePolicy AutoscaleConfig

// InitEnvironment : Validate that all environment variables are present
func InitEnvironment() {
	AuthHost = env.Must("AUTH_HOST")
	AwsCredsJSON = env.Must("AWS_CREDS_JSON")
	AwsDefaultRegion = env.Must("AWS_DEFAULT_REGION")
	AwsRegions = env.Must("AWS_REGIONS")
	ClientMode = env.Must("CLIENT_MODE")
	ClientServer = env.Must("CLIENT_SERVER")
	CoordinatorTemplatesDir = env.Must("GOPATH") + "/src/github.com/VantageSports/lolvideo/coordinator/handlers/templates"
	CoordinatorUrlBase = env.Must("COORDINATOR_URL_BASE")
	Ec2LaunchKey = env.Must("EC2_LAUNCH_KEY")
	Ec2SecurityGroupID = env.Must("EC2_SECURITY_GROUP_ID")
	Ec2WorkerGoogCredsJSON = env.Must("EC2_WORKER_GOOG_CREDS_JSON")
	Ec2WorkerInstanceTypeVideogen = env.Must("EC2_WORKER_INSTANCE_TYPE_VIDEOGEN")
	Ec2WorkerInstanceTypeElogen = env.Must("EC2_WORKER_INSTANCE_TYPE_ELOGEN")
	ElodataOutputPath = env.Must("ELODATA_OUTPUT_PATH")
	GoogCredsJSON = env.Must("GOOG_CREDS_JSON")
	GoogProjectID = env.Must("GOOG_PROJECT_ID")
	MinVideoBitrateFileCheck = env.MustInt("MIN_VIDEO_BITRATE_FILE_CHECK")
	ProjectSourceDir = env.Must("GOPATH") + "/src/github.com/VantageSports/lolvideo"
	PubsubInputElo = env.Must("PUBSUB_INPUT_ELO")
	PubsubInputVideo = env.Must("PUBSUB_INPUT_VIDEO")
	PubsubOutputElo = env.Must("PUBSUB_OUTPUT_ELO")
	VideoOutputPath = env.Must("VIDEO_OUTPUT_PATH")
	VncPasswordFile = env.Must("VNC_PASSWORD_FILE")
	PluginDownloadPath = env.Must("PLUGIN_DOWNLOAD_PATH")
	IamFleetRole = env.Must("IAM_FLEET_ROLE")
	InitAwsRegions()
	InitAutoscalePolicy(GoogProjectID, PubsubInputElo)
}

func InitAwsRegions() {
	regionNames := strings.Split(AwsRegions, ",")
	regionLaunchKeys := strings.Split(Ec2LaunchKey, ",")
	regionSecurityGroups := strings.Split(Ec2SecurityGroupID, ",")

	if len(regionNames) != len(regionLaunchKeys) || len(regionLaunchKeys) != len(regionSecurityGroups) {
		log.Fatalln("AWS_REGIONS, EC2_LAUNCH_KEY, and EC2_SECURITY_GROUP_ID need to be the same length")
	}

	AwsRegionMap = make(map[string]*models.RegionDescriptor)
	for i, val := range regionNames {
		AwsRegionMap[val] = &models.RegionDescriptor{regionNames[i], regionLaunchKeys[i], regionSecurityGroups[i]}
	}
}

// GetFocusOnChampKeyBind : The League Client hotkey to focus the camera on a specific champion
func GetFocusOnChampKeyBind(champ int) string {
	switch {
	case champ >= 1 && champ <= 5:
		return strconv.Itoa(champ)
	case champ == 6:
		return "q"
	case champ == 7:
		return "w"
	case champ == 8:
		return "e"
	case champ == 9:
		return "r"
	case champ == 10:
		return "t"
	}
	return ""
}

// GetFogOfWarBind : The League Client hotkey to change the fog of war to that of a specific champion
func GetFogOfWarBind(champ int) string {
	switch {
	case champ >= 1 && champ <= 5:
		return "f1"
	case champ >= 5 && champ <= 10:
		return "f2"
	}
	return ""
}

func InitAutoscalePolicy(projID, elogenQueueID string) {
	// Approximate number of spot fleets we have running.
	numSpotFleets := 2.0
	// Approximate number of tasks a worker can complete per "add metric".
	// Should be the metric interval divided by the approximate time it takes to complete a task
	// Make this a little pessimistic to always have some spare capacity
	tasksPerMetric := 1.0 / 8.0

	AutoscalePolicy = AutoscaleConfig{
		MinCapacity:       1,
		MaxCapacity:       50,
		RoleARN:           "arn:aws:iam::316394917097:role/aws-ec2-spot-fleet-autoscale-role",
		ScalableDimension: "ec2:spot-fleet-request:TargetCapacity",
		ServiceNamespace:  "ec2",

		AlarmNamespace:         "lol-videogen-coordinator",
		AlarmMetricName:        projID + "_" + elogenQueueID + "_adds_1m",
		AlarmThreshold:         0,
		AlarmPeriod:            60,
		AlarmEvaluationPeriods: 3,
		AlarmStatistic:         "Average",
		AlarmUnit:              "None",

		ScaleOutPolicy: &AutoscalePolicyConfig{

			AlarmName:               projID + "_" + elogenQueueID + "_High",
			AlarmComparisonOperator: "GreaterThanOrEqualToThreshold",
			AlarmEvaluationPeriods:  60,

			PolicyName:           "DefaultElogenScaleOut",
			PolicyType:           "StepScaling",
			PolicyAdjustmentType: "ExactCapacity",
			PolicyCooldown:       300, // Ignored for StepScaling. See http://docs.aws.amazon.com/autoscaling/latest/userguide/Cooldown.html
			// The alarm is set to trigger at the AlarmThreshold, so these bounds are values relative to that
			Steps: []AutoscaleStep{
				AutoscaleStep{
					LowerBound:        0,
					UpperBound:        numSpotFleets * tasksPerMetric,
					ScalingAdjustment: 1,
				},
				// Queue add rate between what 1-2 instances can handle. Set to 2
				AutoscaleStep{
					LowerBound:        numSpotFleets * tasksPerMetric,
					UpperBound:        2.0 * numSpotFleets * tasksPerMetric,
					ScalingAdjustment: 2,
				},
				AutoscaleStep{
					LowerBound:        2.0 * numSpotFleets * tasksPerMetric,
					UpperBound:        5.0 * numSpotFleets * tasksPerMetric,
					ScalingAdjustment: 4,
				},
				AutoscaleStep{
					LowerBound:        5.0 * numSpotFleets * tasksPerMetric,
					UpperBound:        10.0 * numSpotFleets * tasksPerMetric,
					ScalingAdjustment: 8,
				},
				AutoscaleStep{
					LowerBound:        10.0 * numSpotFleets * tasksPerMetric,
					UpperBound:        18.0 * numSpotFleets * tasksPerMetric,
					ScalingAdjustment: 13,
				},
				AutoscaleStep{
					LowerBound:        18.0 * numSpotFleets * tasksPerMetric,
					ScalingAdjustment: 21,
				},
			},
		},

		ScaleInPolicy: &AutoscalePolicyConfig{

			AlarmName:               projID + "_" + elogenQueueID + "_Low",
			AlarmComparisonOperator: "LessThanOrEqualToThreshold",
			AlarmEvaluationPeriods:  60,

			PolicyName:           "DefaultElogenScaleIn",
			PolicyType:           "StepScaling",
			PolicyAdjustmentType: "ExactCapacity",
			PolicyCooldown:       300, // Ignored
			// The alarm is set to trigger at the AlarmThreshold, so these bounds are values relative to that
			Steps: []AutoscaleStep{
				// Unused. Put everything in the scaleOutPolicy so that they all share the same policyCooldown
				AutoscaleStep{
					UpperBound:        -1,
					ScalingAdjustment: 1,
				},
			},
		},
	}
}
