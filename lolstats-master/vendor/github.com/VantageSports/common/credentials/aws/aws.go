package aws

import (
	"encoding/base64"
	"encoding/json"
	"os"

	"github.com/aws/aws-sdk-go/aws/credentials"

	vsjson "github.com/VantageSports/common/json"
	"github.com/VantageSports/common/log"
)

// TODO(Cameron): At some point we should rename the env keys _JSON to _PATH to
// more clearly distinguish keys that are expected to point to a json file from
// those that are expected to be json strings.

// File parses aws credentials from a json file.
func File(path string) (*credentials.Credentials, error) {
	c := &credentials.Value{}
	if err := vsjson.DecodeFile(path, c); err != nil {
		return nil, err
	}

	return creds(c.AccessKeyID, c.SecretAccessKey, c.SessionToken)
}

// Base64String works like AWSFile, but accepts a base64 encoded json object.
// This is useful sometimes as it allows us to at least obfuscate credentials,
// and even re-use our kubernetes secrets values as params in non-kubernetes
// configs (like container-optimized vms, which don't support secrets)
func Base64String(str string) (*credentials.Credentials, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}

	c := &credentials.Value{}
	if err = json.Unmarshal(data, c); err != nil {
		return nil, err
	}

	return creds(c.AccessKeyID, c.SecretAccessKey, c.SessionToken)
}

// String creates aws credentials from the supplied strings.
func String(accessKeyID, secretAccessKey, sessionToken string) (*credentials.Credentials, error) {
	return creds(accessKeyID, secretAccessKey, sessionToken)
}

func creds(keyID, secretKey, token string) (*credentials.Credentials, error) {
	creds := credentials.NewStaticCredentials(keyID, secretKey, token)
	_, err := creds.Get()
	return creds, err
}

// MustEnvAWSCreds attempts to parse AWS credentials from either the environment
// or a file path and exits (log.fatal) if unable to do so.
func MustEnvCreds() *credentials.Credentials {
	var creds *credentials.Credentials
	var err error
	if path := os.Getenv("AWS_CREDS_JSON"); path != "" {
		creds, err = File(path)
	} else if str := os.Getenv("AWS_CREDS_B64"); str != "" {
		creds, err = Base64String(str)
	} else {
		creds, err = String(
			os.Getenv("AWS_ACCESS_KEY_ID"),
			os.Getenv("AWS_SECRET_ACCESS_KEY"),
			os.Getenv("AWS_SESSION_TOKEN"))
	}
	if err != nil {
		log.Fatal(err)
	}
	if creds == nil {
		log.Fatal("no credentials found")
	}
	return creds
}
