package lolqueue

import (
	"encoding/json"
	"fmt"
)

type Token struct {
	MsgType string `json:"type"`
	Token   string `json:"token"`
}

// Sent from client to server to indicate that a player wants to go from
// active -> idle or idle -> active.
type ActivateRequest struct {
	MsgType  string    `json:"type"` // activate
	Active   bool      `json:"active"`
	Criteria *Criteria `json:"criteria"`
}

type PlayerStatusResponse struct {
	MsgType             string  `json:"type"` // player_resp
	Active              bool    `json:"active"`
	Player              *Player `json:"player"`
	NumPlayersConnected int     `json:"num_connected"`
}

func NewPlayerStatusResponse(player *Player, active bool, numConnected int) *PlayerStatusResponse {
	return &PlayerStatusResponse{
		MsgType:             "player_resp",
		Active:              active,
		Player:              player,
		NumPlayersConnected: numConnected,
	}
}

type MatchInvite struct {
	MsgType string `json:"type"` // invite
	Match   *Match `json:"match"`
}

func NewMatchInvite(match *Match) *MatchInvite {
	return &MatchInvite{MsgType: "invite", Match: match}
}

type MatchRSVP struct {
	MsgType string `json:"type"` // rsvp
	MatchID int64  `json:"match_id"`
	Accept  bool   `json:"accept"`
}

type MatchMade struct {
	MsgType string `json:"type"` // made
	Match   *Match `json:"match"`
}

func NewMatchMade(match *Match) *MatchMade {
	return &MatchMade{MsgType: "made", Match: match}
}

type MatchFail struct {
	MsgType string `json:"type"` // fail
}

func NewMatchFail() *MatchFail {
	return &MatchFail{MsgType: "fail"}
}

type Error struct {
	MsgType string `json:"type"` // error
	Text    string `json:"text"`
}

func NewErrorMessage(s string) *Error {
	return &Error{MsgType: "error", Text: s}
}

type ClientPing struct {
	MsgType string `json:"type"` // ping
}

type MessageParser struct {
	inner interface{}
}

func (m *MessageParser) V() interface{} {
	return m.inner
}

func (m *MessageParser) UnmarshalJSON(data []byte) error {
	mapVal := map[string]interface{}{}
	if err := json.Unmarshal(data, &mapVal); err != nil {
		return err
	}

	switch mapVal["type"] {
	case "token":
		m.inner = &Token{}
	case "activate":
		m.inner = &ActivateRequest{}
	case "player_resp":
		m.inner = &PlayerStatusResponse{}
	case "invite":
		m.inner = &MatchInvite{}
	case "rsvp":
		m.inner = &MatchRSVP{}
	case "made":
		m.inner = &MatchMade{}
	case "ping":
		m.inner = &ClientPing{}
	}
	if m.inner == nil {
		return fmt.Errorf("unknown message type received: %v", mapVal["type"])
	}

	// decode as the correct type now.
	return json.Unmarshal(data, m.inner)
}
