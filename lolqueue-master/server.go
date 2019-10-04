package lolqueue

import (
	"container/list"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/websocket"

	"github.com/VantageSports/common/log"
)

type state struct {
	activePlayers      *list.List
	clientsByPlayer    map[int64][]*Client // by summonerID
	pendingInvitations map[int64]bool      // by summonerID
	pendingMatches     map[int64]*Match    // by matchID

	// housekeeping
	lock            *sync.RWMutex
	matchID         int64
	lastMatchSearch time.Time
}

func (s *state) clientsForPlayer(player *Player) []*Client {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.clientsByPlayer[player.SummonerID]
}

func (s *state) add(client *Client) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.clientsByPlayer[client.player.SummonerID] = append(s.clientsByPlayer[client.player.SummonerID], client)
}

// removeClient deletes the client from the session map and may remove the
// player from the active queue if this is the last session referecing that
// player
func (s *state) remove(client *Client) {
	log.Info(fmt.Sprintf("removing client for %s", client.player.SummonerName))
	s.lock.Lock()
	defer s.lock.Unlock()

	clients := s.clientsByPlayer[client.player.SummonerID]
	if len(clients) == 0 {
		return
	}

	for i, c := range clients {
		if c == client {
			clients = append(clients[0:i], clients[i+1:]...)
			break
		}
	}
	s.clientsByPlayer[client.player.SummonerID] = clients

	// if other clients reference this player, keep it active.
	if len(clients) > 0 {
		return
	}

	delete(s.clientsByPlayer, client.player.SummonerID)
	for e := s.activePlayers.Front(); e != nil; e = e.Next() {
		if player, ok := e.Value.(*Player); ok {
			if player.SummonerID == client.player.SummonerID {
				s.activePlayers.Remove(e)
				return
			}
		}
	}
}

func (s *state) pendingMatchFor(p *Player) *Match {
	s.lock.RLock()
	defer s.lock.RUnlock()

	for _, match := range s.pendingMatches {
		for _, invitee := range match.Invited {
			if invitee == p {
				return match
			}
		}
	}
	return nil
}

func (s *state) rsvp(p *Player, accept bool) {
	match := s.pendingMatchFor(p)
	if match == nil {
		log.Warning(fmt.Sprintf("no match found for player:", p.SummonerName))
		return
	}
	defer s.checkIfMatchMade(match.ID)

	s.lock.Lock()
	defer s.lock.Unlock()

	log.Info(fmt.Sprintf("rsvp received for match %d by %s: %t", match.ID, p.SummonerName, accept))

	match.Accepted[p.Key()] = accept
	for _, client := range s.clientsByPlayer[p.SummonerID] {
		if !accept {
			client.Disconnect(time.Second * 0)
		} else {
			// updates the client with the current invtation status (including their acceptance)
			client.InviteTo(match)
		}
	}
}

func (s *state) checkIfMatchMade(matchID int64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	match := s.pendingMatches[matchID]
	if match == nil {
		return
	}

	if len(match.Accepted) != len(match.Invited) {
		// there are still pending invitations.
		return
	}
	delete(s.pendingMatches, matchID)

	participants := []*Player{}
	for _, p := range match.Invited {
		delete(s.pendingInvitations, p.SummonerID)
		if match.Accepted[p.Key()] {
			participants = append(participants, p)
		}
	}

	// even if some players declined/timed out, the match may still get made.
	// we don't need to worry about dealing with the people that did not
	// rsvp since they will have been disconnected automatically anyway.
	madeMatch := IsViableMatch(participants)
	for _, p := range participants {
		for _, client := range s.clientsByPlayer[p.SummonerID] {
			if madeMatch != nil {
				client.NotifyMadeMatch(madeMatch)
				client.Disconnect(time.Second * 5)
			} else {
				client.NotifyMadeFail()
				client.player.InvitationPending = false
			}
		}
	}
}

func (s *state) activate(p *Player) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for e := s.activePlayers.Front(); e != nil; e = e.Next() {
		if player, ok := e.Value.(*Player); ok {
			if player.SummonerID == p.SummonerID {
				return
			}
		}
	}

	s.activePlayers.PushBack(p)
	p.InvitationPending = false
}

// lookForMatches should be run in a goroutine. It attempts to find matches,
// sleeping after unsuccessful attempts (gives time for people to reconnect,
// etc.)
func lookForMatches(s *state) {
	for {
		if s.lookForMatch() {
			// we found a match, no need to sleep
			continue
		}
		time.Sleep(time.Second * 15)
	}
}

func (s *state) lookForMatch() bool {
	start := time.Now()
	defer func() {
		s.lastMatchSearch = time.Now()
		took := time.Since(start)
		if took.Seconds() > 10 {
			log.Info(fmt.Sprintf("finished looking for matches in %v", took))
		}
	}()

	s.lock.Lock()
	defer s.lock.Unlock()

	active := make([]*Player, 0, s.activePlayers.Len())
	for e := s.activePlayers.Front(); e != nil; e = e.Next() {
		if p, ok := e.Value.(*Player); ok {
			if _, found := s.pendingInvitations[p.SummonerID]; !found {
				active = append(active, p)
			}
		}
	}

	match := Make(active...)
	if match == nil {
		return false
	}

	s.matchID = (s.matchID % 10000000) + 1
	match.ID = s.matchID
	s.pendingMatches[match.ID] = match

	log.Info(fmt.Sprintf("inviting: %d to match %d", len(match.Invited), match.ID))
	// invite people
	for _, p := range match.Invited {
		s.pendingInvitations[p.SummonerID] = true
		for _, client := range s.clientsByPlayer[p.SummonerID] {
			client.InviteTo(match)
		}
	}

	go handleMatchTimeout(s, match)
	return true
}

func handleMatchTimeout(s *state, match *Match) {
	time.Sleep(time.Second * 15)

	for _, p := range match.Invited {
		key := p.Key()
		// RSVP "no" for everyone that hasn't yet responded.
		if _, found := match.Accepted[key]; !found {
			s.rsvp(p, false)
		}
	}
}

//
// Server
//

type Server struct {
	state       *state
	vantageUtil *VantageUtil
}

func NewServer(vutil *VantageUtil) *Server {
	s := &Server{
		state: &state{
			activePlayers:      list.New(),            // fifo ordering
			clientsByPlayer:    map[int64][]*Client{}, // by summonerID
			pendingInvitations: map[int64]bool{},      // by summonerID
			pendingMatches:     map[int64]*Match{},    // by matchID
			lock:               &sync.RWMutex{},
		},
		vantageUtil: vutil,
	}
	go lookForMatches(s.state)
	return s
}

func (s *Server) Debug(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `Players In Queue: %d
All Players: %d
Pending Matches: %d
Last Searched For Matches: %v
Connected Players:`,
		s.state.activePlayers.Len(),
		len(s.state.clientsByPlayer),
		len(s.state.pendingMatches),
		s.state.lastMatchSearch)
	for e := s.state.activePlayers.Front(); e != nil; e = e.Next() {
		fmt.Fprintf(w, "%s ", e.Value.(*Player).SummonerName)
	}

}

func (s *Server) OnConnect(ws *websocket.Conn) {
	client := NewClient(ws, s)
	go client.startWriter()

	err := s.addPlayer(client)
	if err != nil {
		client.NotifyError("error: You do not have access to Vantage Queue")
		log.Warning(fmt.Sprintf("error initializing client: %v", err))
		time.Sleep(time.Second)
		return // closes connection
	}

	s.state.add(client)
	defer func() {
		s.state.remove(client)
		client.Disconnect(0)
	}()

	client.NotifyActive(false, client.player, s.state.activePlayers.Len())
	client.startReader() // blocks
}

func (s *Server) addPlayer(client *Client) error {
	token, err := readToken(client.conn)
	if err != nil {
		return err
	}

	userID, err := s.vantageUtil.UserID(token)
	if err != nil {
		return err
	}

	access, err := s.vantageUtil.HasAccess(token, userID)
	if err != nil {
		return err
	}
	if !access {
		return fmt.Errorf("user cannot access vantage queue")
	}

	player, err := s.vantageUtil.Player(token, userID)
	client.player = player
	return err
}

// TODO(Cameron): MOVE TO CLIENT?
// readToken waits 10 seconds for the client to send a token before returning
// an error.
func readToken(ws *websocket.Conn) (string, error) {
	deadline := time.Now().Add(time.Second * 10)
	ws.SetDeadline(deadline)
	tokenMsg := Token{}
	if err := websocket.JSON.Receive(ws, &tokenMsg); err != nil {
		return "", err
	}

	// reset to no deadline
	ws.SetDeadline(time.Time{})
	return tokenMsg.Token, nil
}

func (s *Server) ActivatePlayer(player *Player, criteria *Criteria) {
	clients := s.state.clientsForPlayer(player)

	if criteria != nil {
		player.Criteria = *criteria
	}

	s.state.activate(player)
	log.Info(fmt.Sprintf("activating %s. will notify %d clients", player.SummonerName, len(clients)))
	for _, client := range clients {
		client.NotifyActive(true, player, s.state.activePlayers.Len())
	}
}

func (s *Server) PlayerRSVP(player *Player, accept bool) {
	s.state.rsvp(player, accept)
}
