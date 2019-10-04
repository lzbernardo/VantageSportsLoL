// This package contains the contents and utilities for running TLS-protected
// servers and clients in development. These certs aren't set to expire until
// 2026, but to regenerate new ones for *.vantagesports.com:
//
// $ openssl genrsa -out server.key 2048
// $ openssl req -new -x509 -key server.key -out server.pem -days 3650

package certs

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"os"
	"path"

	"github.com/VantageSports/common/log"
)

var defaultClientTLSOptions []tlsOption

func init() {
	defaultClientTLSOptions = []tlsOption{ServerName("ssl.vantagesports.com")}
}

// DevServerCert is a development certificate.
const DevServerPem = `-----BEGIN CERTIFICATE-----
MIIEJzCCAw+gAwIBAgIJAKyC3mOnzW/TMA0GCSqGSIb3DQEBCwUAMIGpMQswCQYD
VQQGEwJVUzETMBEGA1UECAwKV2FzaGluZ3RvbjEQMA4GA1UEBwwHU2VhdHRsZTEX
MBUGA1UECgwOVmFudGFnZSBTcG9ydHMxFDASBgNVBAsMC0RldmVsb3BtZW50MRww
GgYDVQQDDBMqLnZhbnRhZ2VzcG9ydHMuY29tMSYwJAYJKoZIhvcNAQkBFhdhZG1p
bkB2YW50YWdlc3BvcnRzLmNvbTAeFw0xNjA4MjkyMjQ4MDBaFw0yNjA4MjcyMjQ4
MDBaMIGpMQswCQYDVQQGEwJVUzETMBEGA1UECAwKV2FzaGluZ3RvbjEQMA4GA1UE
BwwHU2VhdHRsZTEXMBUGA1UECgwOVmFudGFnZSBTcG9ydHMxFDASBgNVBAsMC0Rl
dmVsb3BtZW50MRwwGgYDVQQDDBMqLnZhbnRhZ2VzcG9ydHMuY29tMSYwJAYJKoZI
hvcNAQkBFhdhZG1pbkB2YW50YWdlc3BvcnRzLmNvbTCCASIwDQYJKoZIhvcNAQEB
BQADggEPADCCAQoCggEBAM5I3jOA1SNJ0OXLFpS+4HC7HddnjCIOOqGPqV09Oag5
Y8BUr0lDhdWhjhsHPqowY8aVcvEyiq+fV6zdlquh2y7i33DZAn5gyRPqFE2/NRn1
3xmqGK2e9v2kqhhjcH4wa04Muz5uQJ0DH6Vjb13PmDjEog2UVjpvk8Dcla+J2R52
CqK7DELT4jOlE6PFSP396qA6EuAxeZljGJsLy6jvKSezkoK67OEKnocSEv9Rqmp+
nD1eHCjPS72bJYyzOWV+De9xvPd4tkiGGVHMocB/TTR0NlQA9ERjy827edQE7NAj
fpsucJsqbvW159kMLdHBQCQeIv6kJnvdvS0XmZ9GKJcCAwEAAaNQME4wHQYDVR0O
BBYEFEF+sEKTnsGMb/4Lh4YF4XNYP2HhMB8GA1UdIwQYMBaAFEF+sEKTnsGMb/4L
h4YF4XNYP2HhMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBABcfstUi
+YSIuaW3KWetOESiViEDoygAH12+bOE8TMfJHohmjbxnMBqrmDxG4vUQv9AOXDW5
7Ucu79FmSx6llFd/GH2vazQMZrDWPBG8ex/gyo+nbU+yULCt/3AJep04O/+m7iYY
c0gNyOjl1bu1bVOeie7BVmtpm2tKKUUQc9LRCdHYYIV19wpOYRNdDlYEkM8ncqpF
f8JQ1ds4Alk0FwE705FRI9gxZ+BLJEhjkvC9zgM5kLeUPz6Thin2ulWJiiTS9SZj
Q7mMw9iUPyIqZgsFQs1RtobGFF0MwlQljzbTldjuGJkQq0hzF7jonrwdNyjl4sJH
yvvMNfsrMAWMTmQ=
-----END CERTIFICATE-----`

// DevServerKey is a development signing key.
const DevServerKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEoAIBAAKCAQEAzkjeM4DVI0nQ5csWlL7gcLsd12eMIg46oY+pXT05qDljwFSv
SUOF1aGOGwc+qjBjxpVy8TKKr59XrN2Wq6HbLuLfcNkCfmDJE+oUTb81GfXfGaoY
rZ72/aSqGGNwfjBrTgy7Pm5AnQMfpWNvXc+YOMSiDZRWOm+TwNyVr4nZHnYKorsM
QtPiM6UTo8VI/f3qoDoS4DF5mWMYmwvLqO8pJ7OSgrrs4QqehxIS/1Gqan6cPV4c
KM9LvZsljLM5ZX4N73G893i2SIYZUcyhwH9NNHQ2VAD0RGPLzbt51ATs0CN+my5w
mypu9bXn2Qwt0cFAJB4i/qQme929LReZn0YolwIDAQABAoIBADjUfFrGCHIDWQil
rZ/nAiAsfKkmPsX3E5IE6Qzl3CeLP/7xlaTidFGJg4PQuCPc1vATpinq2ahOhrH6
DhNUsPS2zgXh0z0L3vmciGFu5uszAHaqlMfoPnVcUvwLESfCoR1Mhp1JdIWNIb/H
qBu5IF+rVFXXYvYQXS1J1/cXS/iG3+nLHT9cbp1FVerXRjIIjqvXRB/Zj2WvlfBh
X46gH4Up7LslBMMxvlcA+MOiV90mNeaza3IurGc2nSkwYZ7SybKK+tmOUHSAvTAV
+h06+Q5bDNKVgfu/ajPTDuyCH5Lpcm+AiYPw0zaeAd0HGbL++ifnaFjy6LfcpADC
sgOHhxkCgYEA7vi6vVz4Aw8z9chefuMmg5h1OdorKgyv/7ihpL6cuVoeghxyq/Io
ptiiM2nJa4Xbh80aAZEBKr80NnOWWucQG1XZlTRH2DahRHzfv5O/9Af5ozD2vayN
qUtqKIiQALrBFUE+HZi8hyCQ4wwSphl22gmkWDXmZBqGFQs8jTPDDMUCgYEA3Pve
qT00luBwybBH4QN6ZKDjQxDcnwwVQ1YnDvBWobHMrNk4Tb9ydQMjJ5cTgPW1FhlG
IR707aAOMnb/bBh6OB1K45GMWXxRY6/GNv1yBtiuFYueZy459Hd+1dwCijTDFA1C
ef6jg4n6qNmbz79QUs7AoFd2X457oWuBUD+mLasCgYBZoZdBEU7ArwIZLNcslnxK
nXNSiJ7p/apBqvmN9iSIbOHGgW8NSR2wPs7Po9z1ZFaL4ATiE4Yxh1fJGJ+LZuEm
r5OVn+9WjF7+diviCkPT/u3GJMZXLBJSLkQaAeapL2LeWwM2TKRhtnhBgLS58hEf
BMDMiSLp5vzlhbpnZ8O/1QKBgEZzznMqXj/T4mGK3/+u+jBm0s2o7I2Qbp7/Moii
uxvl8ZkRz8+Jv9XqYCTZF5C/xTH8cM1Sb0frBVu1cI1n6F9Hh36yAortBY3AlnMl
rlrcbfPUhXXj2PpFtpw7/ltNovwblhwKwNrI/K8kYXaESfl8jRERkV25sVSvT5Tk
FfjtAn8X4mNWRKBTYgdEbGEjWA2nS9XRF3N56yo0/srxi1S3uPoE0hJ5eU+CY5IL
AMODMp7YAS7o7eL8CBx6X4aUufS3YmGtnWCuiMHpeAnWAN1t+SMiIfZHNP8PEdIN
fTA2WHZiad1WnEpgcM3+6asKlesdoAfPo+MId/zMANlEoUaP
-----END RSA PRIVATE KEY-----
`

// MustWriteDevCerts writes the above cert and key to the temp directory, so
// that they can be loaded by utilities such as http.ServiceAndListenTLS and
// grpc.credentials.New{Client,Server}TLSFromFile
func MustWriteDevCerts() (string, string) {
	log.Info("Writing development certificates")
	pemPath := write("tmp.crt", []byte(DevServerPem))
	keyPath := write("tmp.key", []byte(DevServerKey))
	return pemPath, keyPath
}

func write(file string, data []byte) string {
	filename := path.Join(os.TempDir(), file)
	if err := ioutil.WriteFile(filename, data, 0660); err != nil {
		log.Fatal(err)
	}
	return filename
}

// ClientTLS returns a tlsConfig that trusts either the provided certificate
// or the dev certificate (if empty string is passed). This is useful for
// establishing grpc client credentials that trust our self-signed certificates.
func ClientTLS(certPath string, opts ...tlsOption) (c *tls.Config, err error) {
	var pemData []byte
	if certPath != "" {
		pemData, err = ioutil.ReadFile(certPath)
		if err != nil {
			return nil, err
		}
	} else {
		log.Warning("No certificate path specified. Using development ssl certificates")
		pemData = []byte(DevServerPem)
	}

	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(pemData)
	config := &tls.Config{RootCAs: pool}

	opts = append(defaultClientTLSOptions, opts...)
	for _, opt := range opts {
		opt(config)
	}
	return config, nil
}

type tlsOption func(c *tls.Config)

func ServerName(path string) tlsOption {
	return func(c *tls.Config) { c.ServerName = path }
}

func Insecure(b bool) tlsOption {
	return func(c *tls.Config) { c.InsecureSkipVerify = b }
}
