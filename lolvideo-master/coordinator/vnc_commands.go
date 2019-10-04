package coordinator

import (
	"fmt"
	"strconv"

	"github.com/VantageSports/lolvideo/models"
)

// VNCCommand : One instruction to send over VNC
type VNCCommand struct {
	Name     string
	Argument string
}

// CommandFirstTimeLogin : Returns a list of commands to log into a windows worker machine for the first time
func CommandFirstTimeLogin(windowsPassword string) []*VNCCommand {
	return []*VNCCommand{
		{"key", "esc"},
		{"key", "ctrl-alt-del"},
		// Select all delete, just in case
		{"key", "ctrl-a"},
		{"key", "del"},
		{"type", windowsPassword},
		{"key", "enter"},
		// Windows takes a while to start up
		{"sleep", "120000"},
	}
}

func appendLaunchTask(cmdList *[]*VNCCommand, isInBasicMode bool) {
	*cmdList = append(*cmdList, []*VNCCommand{
		// Open task manager
		{"key", "ctrl-shift-esc"},
		{"sleep", "2000"},
	}...)

	// By default, the task manager is in basic mode.
	// This moves it to advanced mode so we can access the run command option
	if isInBasicMode {
		*cmdList = append(*cmdList, []*VNCCommand{
			{"key", "tab"},
			{"key", "space"},
		}...)
	}

	// Go to the menu and start a new task
	*cmdList = append(*cmdList, []*VNCCommand{
		{"key", "alt"},
		{"key", "enter"},
		{"key", "enter"},
		{"sleep", "1000"},
	}...)
}

func CommandBootstrapClient(req *models.DatagenBootstrapRequest) []*VNCCommand {
	cmdList := []*VNCCommand{}

	localClientServer := ClientServer
	if req.SpectatorServer != "" {
		localClientServer = req.SpectatorServer
	}
	// Warm up the spectator client
	// After it's bootstrapped, the replays load a lot faster
	// Open the command execute from task manager
	appendLaunchTask(&cmdList, false)

	// The "1" at the end signifies that the script shouldn't try to record anything
	cmdList = append(cmdList, []*VNCCommand{
		// Run the replay.bat for 325 + 20 + 10 seconds
		{"type", fmt.Sprintf("%s %d %s %s %s %d %s %d %d 1",
			"C:\\Users\\Administrator\\Desktop\\replay.bat",
			325,
			ClientMode,
			localClientServer,
			req.GameKey,
			req.GameID,
			req.PlatformID,
			20,
			0)},
		{"key", "enter"},

		// Wait for the replay.bat to finish
		{"sleep", "360000"},
	}...)
	return cmdList
}

// CommandRunReplay : Returns a list of commands to execute the entire replay on the worker machine
func CommandRunReplay(req *models.ReplayRequest, champFocus int, withVideoRecording bool) []*VNCCommand {
	cmdList := []*VNCCommand{}

	localClientServer := ClientServer
	if req.SpectatorServer != "" {
		localClientServer = req.SpectatorServer
	}
	appendLaunchTask(&cmdList, false)

	// How long to wait for the spectator client to load before we start recording
	delay := 20

	noVideo := 1
	if withVideoRecording {
		noVideo = 0
	}

	// Launch the replay batch file
	cmdList = append(cmdList, []*VNCCommand{
		// The 0 here represents the fact that we want to record video
		{"type", fmt.Sprintf("%s %d %s %s %s %d %s %d %d %d %s",
			"C:\\Users\\Administrator\\Desktop\\replay.bat",
			// Add a minute buffer to account for possible long load times
			req.GameLengthSeconds+VideoExtraDurationSeconds,
			ClientMode,
			localClientServer,
			req.GameKey,
			req.GameID,
			req.PlatformID,
			delay,
			champFocus,
			noVideo,
			VideoOutputPath)},
		{"key", "enter"},

		// Wait for the client to load
		{"sleep", strconv.Itoa(delay * 1000)},
	}...)

	return cmdList
}

// CommandStartEloBuddy : Returns a list of commands to start EloBuddy, used for datagen
func CommandStartEloBuddy() []*VNCCommand {
	cmdList := []*VNCCommand{}

	cmdList = append(cmdList, []*VNCCommand{
		{"key", "alt-f4"},
		{"key", "alt-f4"},
	}...)

	appendLaunchTask(&cmdList, false)

	cmdList = append(cmdList, []*VNCCommand{
		{"type", "C:\\Program Files (x86)\\EloBuddy\\EloBuddy.Loader.exe"},
		{"key", "enter"},

		// Wait for the program to update, log in, and start
		{"sleep", "10000"},
	}...)

	return cmdList
}

func CommandBootstrapEloBuddy() []*VNCCommand {
	return append(CommandStartEloBuddy(), []*VNCCommand{
		{"sleep", "180000"},
		{"key", "alt-f4"},
		{"key", "alt-f4"},
	}...)
}

func CommandFocusOnChamp(focus int) []*VNCCommand {
	cmdList := []*VNCCommand{}

	// Spend the first 60 seconds attempting to focus on the desired champion
	for i := 0; i < 12; i++ {
		cmdList = append(cmdList, []*VNCCommand{
			{"key", GetFocusOnChampKeyBind(focus)},
			{"sleep", "100"},
			{"key", GetFocusOnChampKeyBind(focus)},
			{"key", GetFogOfWarBind(focus)},
			{"sleep", "5000"},
		}...)
	}

	return cmdList
}

func CommandSpeedUpReplay() []*VNCCommand {
	cmdList := []*VNCCommand{}

	// Spend the first 20 seconds attempting to speed up the replay to 8x
	for i := 0; i < 4; i++ {
		cmdList = append(cmdList, []*VNCCommand{
			// When starting League with EloBuddy, it loses focus. We need to
			// regain focus by clicking on the game
			{"click", "1"},
			// Patches before 6.22 use shift-= to increase playback speed
			// {"key", "shift-="},
			// {"key", "shift-="},
			// {"key", "shift-="},
			// Patch 6.22 just uses = to increase playback speed
			{"key", "="},
			{"key", "="},
			{"key", "="},
			{"sleep", "5000"},
		}...)
	}

	return cmdList
}

// CommandPatchClient : Returns a list of commands to patch the Lol Client
func CommandPatchClient() []*VNCCommand {
	cmdList := []*VNCCommand{}

	appendLaunchTask(&cmdList, true)

	// Launch the main client. Give plenty of time for the patch to finish
	cmdList = append(cmdList, []*VNCCommand{
		{"type", "\"C:\\Users\\Public\\Desktop\\League of Legends.lnk\""},
		{"key", "enter"},
	}...)
	return cmdList
}
