package core

import (
	"errors"
	"go.uber.org/zap"
)

var record bool

// A LobbyActivity is the activity which the players use to organise themselves before starting
// the core.
type LobbyActivity struct {
	// scheduler is the scheduler which can be used by this lobby activity.
	scheduler Scheduler

	// lobby is a pointer to the lobby which this activity is building.
	lobby *Lobby

	// readyPlayers is the set of players who have asserted that they are ready for the core to
	// begin.
	readyPlayers map[*Player]struct{}
}

// NewLobbyActivity returns a pointer to a new lobby activity for the given lobby and scheduler.
func NewLobbyActivity(lobby *Lobby, scheduler Scheduler) *LobbyActivity {
	return &LobbyActivity{lobby: lobby, readyPlayers: make(map[*Player]struct{}), scheduler: scheduler}
}

func (act *LobbyActivity) logger() *zap.Logger {
	return Logger.With(zap.String("lobby", act.lobby.ID))
}

func (act *LobbyActivity) playerLogger(p *Player) *zap.Logger {
	return act.logger().With(zap.String("player", p.Name))
}

// notifyTeamChange sends a message to the peers of the given player notifying them that the
// player has changed to the team with the given index.
func (act *LobbyActivity) notifyTeamChange(player *Player, teamInt int) error {
	msg := NewMessage("lobby_peer_team_change")
	_ = msg.Add("their_name", player.Name)
	_ = msg.Add("team", teamInt)

	return player.ForAllLobbyPeers(func(peer *Player) error {
		return peer.Client.Send(msg)
	})
}

// doTeamChange handles a team change message.
func (act *LobbyActivity) doTeamChange(player *Player, message *Message) error {
	teamInt, err := message.GetInt("team")

	if err != nil || (teamInt != 0 && teamInt != 1) {
		return player.Client.Send(NewMessage("lobby_team_change_format_error"))
	}

	act.playerLogger(player).Info("changing player team", zap.Int("team", teamInt))

	player.SwitchTeam(act.lobby.Teams[teamInt])

	// A change to the teams forces all players to become unready.
	clear(act.readyPlayers)

	// Notify all other players of the team change.
	return act.notifyTeamChange(player, teamInt)
}

// notifyBye sends a message to all players in the lobby reporting that the given player has just
// left.
func (act *LobbyActivity) notifyBye(player *Player) error {
	msg := NewMessage("lobby_peer_left")
	_ = msg.Add("their_name", player.Name)

	return act.lobby.ForAllPlayers(func(p *Player) error {
		return p.Client.Send(msg)
	})
}

// doBye handles a message sent by a player reporting that they are leaving the lobby.
func (act *LobbyActivity) doBye(player *Player) error {
	act.playerLogger(player).Info("removing player")

	act.lobby.RemovePlayer(player)

	// Unready the player.
	delete(act.readyPlayers, player)

	// Notify remaining players.
	return act.notifyBye(player)
}

// notifyReadyChange sends a message to all peers of the given player reporting that the player's
// readiness has changed.
func (act *LobbyActivity) notifyReadyChange(player *Player, ready bool) error {
	msg := NewMessage("lobby_peer_ready_change")
	_ = msg.Add("their_name", player.Name)
	_ = msg.Add("ready", ready)

	return player.ForAllLobbyPeers(func(peer *Player) error {
		return peer.Client.Send(msg)
	})
}

// doStartGame moves the players into the main core.
func (act *LobbyActivity) doStartGame() error {
	act.logger().Info("starting core")

	// Unready all of the players so that this lobby activity can be reused without players being
	// automatically ready.
	clear(act.readyPlayers)

	ship := NewShip(act.lobby, act.scheduler)
	return ship.Start()
}

// doReadyUp handles a message indicating that the given player should be marked as ready.
func (act *LobbyActivity) doReadyUp(player *Player) error {
	// Readying up requires the teams to be ready.
	// A well-formed client will not attempt to modify the ready state if this is not the case.
	if !act.lobby.IsReady() {
		return player.Client.Send(NewMessage("lobby_ready_change_teams_not_ready_error"))
	}

	act.playerLogger(player).Info("setting player to ready")

	// Add to the ready set.
	act.readyPlayers[player] = struct{}{}

	// Notify peers.
	peerErr := act.notifyReadyChange(player, true)

	readyCount := len(act.readyPlayers)

	if readyCount == 6 || (AllowSmallerLobbies && readyCount == act.lobby.PlayerCount()) {
		// All players ready.
		startErr := act.doStartGame()

		return errors.Join(peerErr, startErr)
	}

	return peerErr
}

// doUnready handles a message indicating that the given player should be marked as unready.
func (act *LobbyActivity) doUnready(player *Player) error {
	act.playerLogger(player).Info("setting player to unready")

	delete(act.readyPlayers, player)

	return act.notifyReadyChange(player, false)
}

// doReadySet handles a message for a readiness change.
func (act *LobbyActivity) doReadySet(player *Player, ready bool) error {
	if ready {
		return act.doReadyUp(player)
	}

	return act.doUnready(player)
}

// doReadyChange parses and handles a readiness change message.
func (act *LobbyActivity) doReadyChange(player *Player, message *Message) error {
	if ready := message.TryGet("ready"); ready != nil {
		ready, ok := (*ready).(bool)

		if !ok {
			return player.Client.Send(NewMessage("lobby_ready_change_format_error"))
		}

		return act.doReadySet(player, ready)
	}

	return player.Client.Send(NewMessage("lobby_ready_change_format_error"))
}

// notifyJoineePeers sends a message to all peers of the given player telling them that the
// player has joined the lobby.
func (act *LobbyActivity) notifyJoineePeers(player *Player) error {
	msg := NewMessage("lobby_peer_joined")
	_ = msg.Add("their_name", player.Name)
	_ = msg.Add("their_team", player.Team.Index())

	return player.ForAllLobbyPeers(func(peer *Player) error {
		return peer.Client.Send(msg)
	})
}

// notifyJoinee sends a message to the given player telling them that they have joined the lobby.
func (act *LobbyActivity) notifyJoinee(player *Player) error {
	msg := NewMessage("lobby_welcome")
	_ = msg.Add("your_name", player.Name)
	_ = msg.Add("your_team", player.Team.Index())
	_ = msg.Add("lobby_id", act.lobby.ID)

	peerTeamMap := make(map[string]uint8)

	_ = player.ForAllLobbyPeers(func(peer *Player) error {
		peerTeamMap[peer.Name] = peer.Team.Index()

		return nil
	})

	_ = msg.Add("peer_teams", peerTeamMap)

	return player.Client.Send(msg)
}

// notifyPlayerJoin notifies the joining player and their
// (new) peers that they have joined the lobby.
func (act *LobbyActivity) notifyPlayerJoin(player *Player) error {
	return errors.Join(act.notifyJoinee(player), act.notifyJoineePeers(player))
}

// HandleJoinRequest handles a join request from the given client.
func (act *LobbyActivity) HandleJoinRequest(client *Client) error {
	l := act.logger().With(zap.Stringer("addr", client.conn.RemoteAddr()))

	l.Info("handling join request")

	if client.Player != nil {
		// Warn here because this indicates a frontend bug.
		l.Warn("client is already in a lobby")

		return client.Send(NewMessage("client_already_in_lobby_error"))
	}

	if act.lobby.PlayerCount() > 5 {
		// Info here because this is a response to user behaviour.
		l.Info("lobby is full")

		return client.Send(NewMessage("lobby_full"))
	}

	act.lobby.AddClient(client)

	act.playerLogger(client.Player).Info(
		"added player",
		zap.Stringer("addr", client.conn.RemoteAddr()),
	)

	// Put the player into the lobby activity.
	client.Player.Activity = act

	return act.notifyPlayerJoin(client.Player)
}

func (act *LobbyActivity) HandleMessage(player *Player, message *Message) error {
	switch message.Type {
	case "lobby_team_change":
		return act.doTeamChange(player, message)

	case "lobby_ready_change":
		return act.doReadyChange(player, message)

	case "lobby_bye":
		return act.doBye(player)
	}

	return player.Client.Send(NewMessage("lobby_unrecognised_message_type").Add(
		"bad_type",
		message.Type,
	))
}

func (act *LobbyActivity) Start() error {
	// Do nothing.
	return nil
}
