package models

import (
	"fmt"
)

type RegionDescriptor struct {
	Name            string
	LaunchKey       string
	SecurityGroupID string
}

// Request to /video_request
type VideoRequest struct {
	ReplayRequest *ReplayRequest `json:"replay_request"`
	ChampFocus    int            `json:"champ_focus"` // Same as match details participantId
}

// Request to /datagen_request
type DatagenRequest struct {
	ReplayRequest          *ReplayRequest `json:"replay_request"`
	IncreasedPlaybackSpeed bool           `json:"increased_playback_speed"`
}

type ReplayRequest struct {
	GameID            int64  `json:"game_id"`
	PlatformID        string `json:"platform_id"`
	GameKey           string `json:"key"`
	GameLengthSeconds int    `json:"game_length_seconds"`
	SpectatorServer   string `json:"spectator_server"`
	MatchVersion      string `json:"match_version"`
	WorkerInstance    string `json:"worker_instance"`
	NeedsBootstrap    bool   `json:"needs_bootstrap"`
	Region            string `json:"region"`
}

// Request to /datagen_bootstrap_request
type DatagenBootstrapRequest struct {
	GameID          int64  `json:"game_id"`
	PlatformID      string `json:"platform_id"`
	GameKey         string `json:"key"`
	SpectatorServer string `json:"spectator_server"`
	WorkerInstance  string `json:"worker_instance"`
	Region          string `json:"region"`
}

// Request to /terminate_instance
type TerminateInstanceRequest struct {
	WorkerInstance string `json:"worker_instance"`
	Region         string `json:"region"`
}

func (rq *VideoRequest) Valid(validRegions map[string]*RegionDescriptor) error {
	if rq.ReplayRequest == nil {
		return fmt.Errorf("replay_request is required")
	}
	if err := rq.ReplayRequest.Valid(validRegions); err != nil {
		return fmt.Errorf("replay_request invalid: %v", err)
	}

	return nil
}

func (rq *DatagenRequest) Valid(validRegions map[string]*RegionDescriptor) error {
	if rq.ReplayRequest == nil {
		return fmt.Errorf("replay_request is required")
	}
	if err := rq.ReplayRequest.Valid(validRegions); err != nil {
		return fmt.Errorf("replay_request invalid: %v", err)
	}

	return nil
}

func (rq *ReplayRequest) Valid(validRegions map[string]*RegionDescriptor) error {
	if rq.WorkerInstance == "" || rq.Region == "" {
		return fmt.Errorf("worker_instance and region are required")
	}

	if _, ok := validRegions[rq.Region]; !ok {
		return fmt.Errorf("Invalid region: %s", rq.Region)
	}

	return nil
}

func (rq *TerminateInstanceRequest) Valid() error {
	if rq.WorkerInstance == "" || rq.Region == "" {
		return fmt.Errorf("Missing worker_instance and region")
	}
	return nil
}

func (rq *DatagenBootstrapRequest) Valid(validRegions map[string]*RegionDescriptor) error {
	if rq.WorkerInstance == "" || rq.Region == "" {
		return fmt.Errorf("worker_instance and region are required")
	}

	if _, ok := validRegions[rq.Region]; !ok {
		return fmt.Errorf("Invalid region: %s", rq.Region)
	}

	return nil
}
