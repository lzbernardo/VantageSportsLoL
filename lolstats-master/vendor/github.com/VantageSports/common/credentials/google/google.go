package google

import (
	"encoding/base64"
	"io/ioutil"
	"os"

	"github.com/VantageSports/common/log"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"golang.org/x/oauth2/jwt"
)

// GoogleCreds is an encapsulated authentication configuration for the Google
// Cloud Platform.
type Creds struct {
	Conf      *jwt.Config
	ProjectID string
}

func (gc *Creds) TokenSource(ctx context.Context) oauth2.TokenSource {
	if gc.Conf == nil {
		return oauth2.StaticTokenSource(&oauth2.Token{
			AccessToken: "faked",
			TokenType:   "Bearer",
		})
	}
	return gc.Conf.TokenSource(ctx)
}

// File expects a path to a json account credentials file, and
// attempts to generate a configuration with the requested cloud auth scopes.
func File(path, projectID string, scopes ...string) (*Creds, error) {
	if path == "_fake_" { // Development hook.
		log.Warning("using local dev credentials")
		return &Creds{ProjectID: projectID}, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return creds(data, projectID, scopes...)
}

// Base64String is like GoogleString, but assumes the string is base64
// encoded. This is useful sometimes as it allows us to at least obfuscate
// credentials, and even re-use our kubernetes secrets values as params in
// non-kubernetes configs (like container-optimized vms, which don't support
// secrets)
func Base64String(str, projectID string, scopes ...string) (*Creds, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, err
	}
	return creds(data, projectID, scopes...)
}

// MustEnvGoogleCreds attempts to parse Google credentials from one of the
// conventional environment locations (path, base64 string), exiting if unable
// to do so.
func MustEnvCreds(projectID string, scopes ...string) *Creds {
	var creds *Creds
	var err error

	if path := os.Getenv("GOOG_CREDS_JSON"); path != "" {
		creds, err = File(path, projectID, scopes...)
	} else if str := os.Getenv("GOOG_CREDS_B64"); str != "" {
		creds, err = Base64String(str, projectID, scopes...)
	}
	if err != nil {
		log.Fatal(err)
	}

	if creds == nil {
		log.Fatal("no google credentials found")
	}

	return creds
}

func creds(data []byte, projectID string, scopes ...string) (*Creds, error) {
	conf, err := google.JWTConfigFromJSON(data, scopes...)
	if err != nil {
		return nil, err
	}
	return &Creds{Conf: conf, ProjectID: projectID}, nil
}
