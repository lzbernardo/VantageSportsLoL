package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/api/cloudmonitoring/v2beta2"

	"github.com/VantageSports/common/constants/privileges"
	"github.com/VantageSports/lolvideo/coordinator"
	"github.com/VantageSports/users"
)

var currentDisplay = ":0"

// We can probably bump up this number a lot in order to handle more simultaneous requests.
var maxDisplays = 500
var mu = &sync.Mutex{}

// DisplayStatus : The status of the virtual displays on the coordinator
type DisplayStatus struct {
	ID         string
	InUse      bool
	LastUsedBy string
	LastUsedAt string
	Message    string
	XorgCmd    *exec.Cmd
}

type LolVideoHandler struct {
	TimeseriesClient *cloudmonitoring.TimeseriesService
}

var displays = make(map[string]*DisplayStatus)

// ByTime implements sort.Interface for []*DisplayStatus based on
// the LastUsedAt field.
type ByTime []*DisplayStatus

func (a ByTime) Len() int           { return len(a) }
func (a ByTime) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByTime) Less(i, j int) bool { return a[i].LastUsedAt < a[j].LastUsedAt }

// GetDisplayStatuses : Return the displayStatuses sorted by lastUsedAt
func GetDisplayStatuses() []*DisplayStatus {
	d := []*DisplayStatus{}
	for _, val := range displays {
		d = append(d, val)
	}
	sort.Sort(ByTime(d))
	return d
}

// InitConnection : Call this to set up the display before starting
//   a vnc command chain on a remote machine
func InitConnection(instance string, message string) string {
	displayStr := getDisplayAndIncrement()

	// Initialize the maps
	if _, ok := displays[displayStr]; !ok {
		displays[displayStr] = &DisplayStatus{
			ID:         "",
			InUse:      false,
			LastUsedBy: "",
			LastUsedAt: "",
			Message:    "",
			XorgCmd:    startDisplay(displayStr),
		}
	}

	log.Printf("Display:%v\n", displayStr)
	// If the display is in use, then wait for it to free up
	for displays[displayStr].InUse {
		// Time out after 1 hour
		lastUsedTime, err := time.Parse(time.RFC3339, displays[displayStr].LastUsedAt)
		if err != nil {
			log.Printf("Unable to parse last used time", err)
			break
		}
		if time.Since(lastUsedTime) > time.Hour {
			log.Printf("Display is stale. Continuing")
			break
		}
		log.Printf("Display is in use. Waiting 1 minute")
		time.Sleep(time.Minute)
	}
	localLocation, _ := time.LoadLocation("America/Los_Angeles")
	displays[displayStr].ID = displayStr
	displays[displayStr].InUse = true
	displays[displayStr].LastUsedBy = instance
	displays[displayStr].LastUsedAt = time.Now().
		In(localLocation).
		Format(time.RFC3339)
	displays[displayStr].Message = message

	return displayStr
}

// Cleanup : Call this to clean up after you're done issuing commands
func Cleanup(instance string, displayStr string) {
	// When we're done, the replay is recording without us. Approximate when the worker will be free
	displays[displayStr].InUse = false
}

// GetLaunchKeyName: Converts the actual file name in the environment variable, like
//   dev/linux_g2_2.pem into the key "linux_g2_2" that AWS needs.
// Also, replace dashes with underscores. This is because the kubernetes secrets config
//   cannot have file names with underscores in them, so we use dashes, but we have to
//   convert them back
func GetLaunchKeyName(fullPath string) string {
	leftIndex := strings.LastIndex(fullPath, "/")
	if leftIndex == -1 {
		leftIndex = 0
	}

	rightIndex := strings.LastIndex(fullPath, ".pem")
	if rightIndex == -1 {
		rightIndex = len(fullPath)
	}

	return strings.Replace(fullPath[leftIndex+1:rightIndex], "-", "_", -1)
}

func RequireAuth(w http.ResponseWriter, r *http.Request) error {

	token, err := r.Cookie("vstoken")
	if err == nil {
		// Validate the token
		var jsonStr = []byte(`{"token":"` + token.Value + `"}`)
		validateRequest, err := http.NewRequest("POST", coordinator.AuthHost+"/users/v2/CheckToken", bytes.NewBuffer(jsonStr))
		validateRequest.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(validateRequest)
		if err != nil || resp.StatusCode != 200 {
			// Failed to validate token
			log.Println("Failed to validate token", err)
			return showSignin(w)
		}
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)

		// Parse the response into a ClaimsResponse
		claimsResponse := &users.ClaimsResponse{}
		err = json.Unmarshal(body, claimsResponse)
		if err != nil {
			log.Println("Error unmarshalling CheckToken response", err)
			return showSignin(w)
		}

		// Check that they have the right permission
		if val, ok := claimsResponse.Claims.Privileges[string(privileges.VideogenWrite)]; !ok || !val {
			log.Println("Token doesn't have videogen_write permission")
			return showSignin(w)
		}
		return nil
	}

	// If there's no token, then show signin
	return showSignin(w)
}

func showSignin(w http.ResponseWriter) error {
	t, err := template.ParseFiles(coordinator.CoordinatorTemplatesDir + "/sign_in.html")
	if err != nil {
		log.Println(err)
		return err
	}

	context := struct {
		AuthHost string
	}{
		AuthHost: coordinator.AuthHost,
	}
	t.Execute(w, context)

	return errors.New("Auth failed")
}

// If two requests come in at the same time, make sure they get different displays
func getDisplayAndIncrement() string {
	mu.Lock()

	prevDisplay := currentDisplay

	displayNum, _ := strconv.Atoi(prevDisplay[1:len(prevDisplay)])
	displayStr := ":" + strconv.Itoa((displayNum+1)%maxDisplays)
	currentDisplay = displayStr

	mu.Unlock()
	return prevDisplay
}

func startDisplay(displayStr string) *exec.Cmd {
	log.Println("Executing Xorg -noreset +extension GLX +extension RANDR +extension RENDER",
		"-logfile disp_"+displayStr+".log -config ./xorg.conf", displayStr)
	xcmd := exec.Command("Xorg", "-noreset", "+extension", "GLX",
		"+extension", "RANDR", "+extension", "RENDER",
		"-logfile", "disp_"+displayStr+".log",
		"-config", "./xorg.conf", displayStr)
	xcmd.Start()
	time.Sleep(2 * time.Second)

	var outBuf, errBuf bytes.Buffer
	xcmd.Stdout = &outBuf
	xcmd.Stderr = &errBuf

	// TODO: Detect failures in Xorg
	return xcmd
}
