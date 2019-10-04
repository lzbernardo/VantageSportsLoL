package lolqueue

import (
	"fmt"
	"io"
	"strings"
	"time"

	"golang.org/x/net/websocket"

	"github.com/VantageSports/common/log"
)

type availability string

const (
	Ready  availability = "ready"
	InGame availability = "ingame"
	Idle   availability = "idle"
)

type Client struct {
	player  *Player
	server  *Server
	conn    *websocket.Conn
	outMsgs chan interface{}
	done    bool
}

func NewClient(ws *websocket.Conn, server *Server) *Client {
	if ws == nil || server == nil {
		log.Info("skipping client, player, socket and/or server are nil")
		return nil
	}

	return &Client{
		player:  &Player{},
		server:  server,
		outMsgs: make(chan interface{}, 50),
		conn:    ws,
	}
}

func (c *Client) startWriter() {
	for !c.done {
		select {
		case out := <-c.outMsgs:
			if err := websocket.JSON.Send(c.conn, out); err != nil {
				log.Warning("send error: " + err.Error())
				return
			}
		case <-time.After(time.Second * 5):
			// an opportunity to notice if done == true
		}
	}
}

func (c *Client) startReader() {
	start := time.Now()
	for !c.done {
		if time.Since(start).Hours() > 2 {
			c.Disconnect(time.Second)
		}

		mp := MessageParser{}
		if err := websocket.JSON.Receive(c.conn, &mp); err != nil {
			// its normal that when the socket is closed we get a read error. no need to log that.
			if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
				log.Warning(fmt.Sprintf("read error: %v", err))
			}
			return
		}

		switch t := mp.V().(type) {
		case *ActivateRequest:
			if t.Criteria != nil {
				if err := validateCriteria(t.Criteria); err != nil {
					log.Warning(fmt.Sprintf("error validating criteria: %v", err))
					c.NotifyError(err.Error())
					return
				}
				t.Criteria.Region = c.player.Region // we always override region to player region.
			}
			if t.Active {
				c.server.ActivatePlayer(c.player, t.Criteria)
			} else {
				c.Disconnect(time.Millisecond * 250)
			}

		case *MatchRSVP:
			c.server.PlayerRSVP(c.player, t.Accept)

		case *ClientPing:
			c.outMsgs <- map[string]interface{}{"type": "pong"}
		}
	}
}

func (c *Client) Disconnect(delay time.Duration) {
	go func() {
		if c.done {
			return
		}
		time.Sleep(delay)
		c.done = true
		if err := c.conn.Close(); err != nil {
			log.Warning(fmt.Sprintf("error closing connection: %v", err))
		}
	}()
}

func (c *Client) NotifyActive(active bool, player *Player, numConnected int) {
	c.outMsgs <- NewPlayerStatusResponse(player, active, numConnected)
}

func (c *Client) InviteTo(match *Match) {
	c.outMsgs <- NewMatchInvite(match)
}

func (c *Client) NotifyMadeMatch(match *Match) {
	c.outMsgs <- NewMatchMade(match)
}

func (c *Client) NotifyMadeFail() {
	c.outMsgs <- NewMatchFail()
}

func (c *Client) NotifyError(msg string) {
	c.outMsgs <- NewErrorMessage(msg)
}

func validateCriteria(c *Criteria) error {
	if c.Region == "" {
		return fmt.Errorf("region required")
	}
	if c.MinPlayers <= 1 {
		c.MinPlayers = 2
	}
	if c.MinRank.Division == "" || c.MinRank.Tier == "" {
		c.MinRank = Rank{Division: "Bronze", Tier: "V"}
	}
	if c.MaxRank.Division == "" || c.MaxRank.Tier == "" {
		c.MaxRank = Rank{Division: "Challenger", Tier: "I"}
	}
	if len(c.MyPositions) == 0 {
		c.MyPositions = []position{any}
	}

	return nil
}
