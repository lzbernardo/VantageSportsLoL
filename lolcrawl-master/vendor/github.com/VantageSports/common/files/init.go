// I'm not sure this is the best direction to go, since it sort of hides the
// environment variables from the main() functions that consume this API, but
// it DOES cut down on a lot of boiler plate for creating a new client, so we're
// going to try it out for while.

package files

import (
	"os"

	"github.com/VantageSports/common/credentials/aws"
	"github.com/VantageSports/common/credentials/google"

	awsc "github.com/aws/aws-sdk-go/aws/credentials"
)

// Option is a functional argument for a client.
type Option func(*Client) error

// AutoRegisterS3 is a (super-)convenience function for parsing aws credentials
// and registering them with the
func AutoRegisterS3(region string) Option {
	return RegisterS3("s3://", region, aws.MustEnvCreds())
}

// RegisterS3 creates and registers an s3 remote manager in the specified region
// with the specified credentials
func RegisterS3(prefix, region string, creds *awsc.Credentials) Option {
	return func(c *Client) error {
		s3, err := NewS3Provider(creds, region)
		if err != nil {
			return err
		}
		return c.Register(prefix, s3, s3)
	}
}

// AutoRegisterGCS is a (super-)convenience function for parsing google
// credentials registering a new gcs remote manager.
// NOTE: Registers with oauth scope ScopeFullControl
func AutoRegisterGCS(projectId string, scopes ...string) Option {
	creds := google.MustEnvCreds(projectId, scopes...)
	return RegisterGCS("gs://", creds)
}

// RegisterGCS creates and registers a gcs remote manager with the specified
// credentials.
func RegisterGCS(prefix string, creds *google.Creds) Option {
	return func(c *Client) error {
		gcs, err := NewGCSProvider(creds)
		if err != nil {
			return err
		}
		return c.Register(prefix, gcs, gcs)
	}
}

func InitClient(options ...Option) (*Client, error) {
	c, err := NewClient(os.TempDir())
	if err != nil {
		return nil, err
	}
	for _, option := range options {
		if err = option(c); err != nil {
			return nil, err
		}
	}
	return c, nil
}
