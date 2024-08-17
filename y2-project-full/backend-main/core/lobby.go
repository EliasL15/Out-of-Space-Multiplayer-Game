package core

import (
	"errors"
	"go.uber.org/zap"
)

// AllowSmallerLobbies disables the requirement of having six players to start a game.
const AllowSmallerLobbies = true

// A Lobby is a group of players who play together.
type Lobby struct {
	// manager is a pointer to the manager which is responsible for this lobby.
	manager *LobbyManager

	// Teams is an array of two team pointers. The order of these pointers will not change.
	Teams [2]*Team

	// ID is the lobby's unique identifier.
	ID string
}

// buildPlayerNameSet returns a set containing the name of every player in the lobby.
func (lobby *Lobby) buildPlayerNameSet() map[string]struct{} {
	names := make(map[string]struct{})

	_ = lobby.ForAllPlayers(func(p *Player) error {
		names[p.Name] = struct{}{}
		return nil
	})

	return names
}

// generatePlayerName returns a player name which is guaranteed to be unique within this lobby.
func (lobby *Lobby) generatePlayerName() string {
	taken := lobby.buildPlayerNameSet()

	for {
		name := randomUsername()

		if _, isTaken := taken[name]; !isTaken {
			return name
		}

		// Name was taken, so generate another.
	}
}

// AddClient adds the given client to this lobby.
// This will also give the client a player pointer.
// This method panics if the client already has a player pointer.
func (lobby *Lobby) AddClient(client *Client) {
	if client.Player != nil {
		Logger.Panic(
			"client already has a player",
			zap.Stringer("addr", client.conn.RemoteAddr()),
		)
	}

	client.Player = &Player{
		Team:     nil,
		Client:   client,
		Activity: nil,
		Name:     lobby.generatePlayerName(),
	}

	if len(lobby.Teams[0].Players) <= len(lobby.Teams[1].Players) {
		// Add to T0 if teams are balanced or T0 has fewer players.
		lobby.Teams[0].AddPlayer(client.Player)
	} else {
		// Add to T1 if T1 has fewer players.
		lobby.Teams[1].AddPlayer(client.Player)
	}
}

// RemovePlayer removes the given player from this lobby.
// It panics if the player is not in this lobby.
func (lobby *Lobby) RemovePlayer(player *Player) {
	if player.Lobby() != lobby {
		Logger.Panic("player was not in this lobby", zap.Any("player", player))
	}

	player.Team.RemovePlayer(player)

	// Disconnect the player and client so that they are no longer associated with one another.
	player.Client.Player = nil
	player.Client = nil

	if lobby.PlayerCount() == 0 {
		// Delete the lobby from the manager so that nobody else can join.
		lobby.manager.forget(lobby)
	}
}

// IsReady returns true if and only if both teams are even and there are enough players to start a
// game.
func (lobby *Lobby) IsReady() bool {
	n0 := len(lobby.Teams[0].Players)
	n1 := len(lobby.Teams[1].Players)

	if AllowSmallerLobbies {
		// There should never be zero players in the lobby, because empty lobbies are deleted.
		// We still check just in case.
		return n0 > 0 && n0 == n1
	}

	return n0 == 3 && n1 == 3
}

// ForAllPlayers calls fn for every player in the lobby.
func (lobby *Lobby) ForAllPlayers(fn func(*Player) error) error {
	return errors.Join(lobby.Teams[0].ForAllMembers(fn), lobby.Teams[1].ForAllMembers(fn))
}

// PlayerCount returns the number of players in the lobby.
func (lobby *Lobby) PlayerCount() int {
	return len(lobby.Teams[0].Players) + len(lobby.Teams[1].Players)
}
