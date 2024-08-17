package core

import (
	"go.uber.org/zap"
)

// A Player is the in-core representation of a human user.
type Player struct {
	// Team is a pointer to the team that this player is on.
	Team *Team

	// Client is our connection to the user's instance of the frontend.
	Client *Client

	// Activity is the player's current activity.
	Activity Activity

	// Name is the unique server-generated name for the player.
	Name string
}

// Lobby returns a pointer to the lobby that the player is in.
func (player *Player) Lobby() *Lobby {
	return player.Team.Lobby
}

// InLobbyActivity returns true if and only if the player is in a lobby activity.
func (player *Player) InLobbyActivity() bool {
	_, ok := player.Activity.(*LobbyActivity)
	return ok
}

// InShipActivity returns true if and only if the player is in a ship activity.
func (player *Player) InShipActivity() bool {
	_, ok := player.Activity.(*Ship)
	return ok
}

// SwitchTeam removes the player from its current team and adds it to the given team.
// It panics if the player is not in the lobby activity.
func (player *Player) SwitchTeam(team *Team) {
	if !player.InLobbyActivity() {
		Logger.Panic(
			"can't change team when not in lobby activity",
			zap.Any("player", player),
		)
	}

	// Remove from the current team. This will crash if the player's current team is nil,
	// but that's what we want because a well-formed player always has a team.
	player.Team.RemovePlayer(player)

	// Add to the new team.
	team.AddPlayer(player)
}

// ForAllLobbyPeers calls fn for every other player in the lobby.
func (player *Player) ForAllLobbyPeers(fn func(*Player) error) error {
	return player.Lobby().ForAllPlayers(func(p *Player) error {
		if p == player {
			return nil
		}

		return fn(p)
	})
}

// ForAllActivityPeers calls fn for every other player in the same activity as this player.
func (player *Player) ForAllActivityPeers(fn func(*Player) error) error {
	return player.ForAllLobbyPeers(func(lobbyPeer *Player) error {
		if lobbyPeer.Activity != player.Activity {
			return nil
		}

		return fn(lobbyPeer)
	})
}

// ForAllShipPeers calls fn for every other player in the ship activity.
// It panics if the player themselves is not in the ship activity.
func (player *Player) ForAllShipPeers(fn func(*Player) error) error {
	if _, ok := player.Activity.(*Ship); !ok {
		Logger.Panic("player is not in ship", zap.Any("player", player))
	}

	return player.ForAllActivityPeers(fn)
}
