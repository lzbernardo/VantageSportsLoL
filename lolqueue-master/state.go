package lolqueue

import (
	"time"
)

func NewStateManager() *StateManager {
	return &StateManager{bySessionID: map[int64]*Client{}}
}

type StateManager struct {
	bySessionID map[int64]*Client
}

func (s *StateManager) AddClient(c *Client) {
	sessionID := time.Now().UnixNano()

	s.bySessionID[sessionID] = c
}
