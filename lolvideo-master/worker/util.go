package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
)

func GetInstanceAndRegion(devMode bool) (string, string, error) {
	if devMode {
		return "dev", "dev", nil
	}

	// Get the current instance id.
	workerName, err := httpGet("http://169.254.169.254/latest/meta-data/instance-id")
	if err != nil {
		return "", "", err
	}

	// Get the current region id
	regionID, err := httpGet("http://169.254.169.254/latest/meta-data/placement/availability-zone")
	if err != nil || regionID == "" {
		return "", "", err
	}
	// Trim off the last character. Availability zones are like us-east-1b
	return workerName, regionID[0 : len(regionID)-1], err
}

func httpGet(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func processIsRunningOnWindows() bool {
	cmd := exec.Command("tasklist.exe", "/fo", "csv", "/nh")
	out, err := cmd.Output()
	if err != nil {
		return false
	}

	// The replay client is "League of Legends.exe"
	// The launcher (started during bootstrapping) is "LoLLauncher.exe"
	// Also, make sure the video recorder is not running
	return bytes.Contains(out, []byte("League of Legends.exe")) ||
		bytes.Contains(out, []byte("LoLLauncher.exe")) ||
		bytes.Contains(out, []byte("ffmpeg.exe"))
}

func endEloBuddyProcess() {
	// After the Elobuddy process kills the League process, we also need
	// to clean up the EloBuddy launcher, and the replay.bat window
	exec.Command("taskkill", "/IM", "EloBuddy.Loader.exe", "/F", "/S", "localhost").Run()
	exec.Command("taskkill", "/IM", "cmd.exe", "/F").Run()
	exec.Command("taskkill", "/IM", "timeout.exe", "/F").Run()
}

func postJSON(urlBase, path string, v interface{}) error {
	urlStr := strings.TrimSuffix(urlBase, "/") + "/" + strings.TrimPrefix(path, "/")

	postData, err := json.Marshal(v)
	if err != nil {
		return err
	}
	body := bytes.NewReader(postData)

	res, err := http.Post(urlStr, "application/json", body)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("unsuccessful status code from POST to %s: %d", urlStr, res.StatusCode)
	}
	return nil
}
