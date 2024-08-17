package core

import (
	"errors"
	"go.uber.org/zap"
	"math/rand"
)

// A Team is three players who work together.
type Team struct {
	// Lobby is a pointer to the lobby that this team is in.
	Lobby *Lobby

	// Players is a set containing a pointer to each player in this team.
	Players map[*Player]struct{}
}

// AddPlayer adds the given player to this team.
// It panics if the player is already a member of a team.
func (team *Team) AddPlayer(player *Player) {
	if player.Team != nil {
		Logger.Panic("player is already in a team", zap.Any("player", player))
	}

	// Add the player to the member set.
	team.Players[player] = struct{}{}

	// Give the player a pointer back to the team.
	player.Team = team
}

// RemovePlayer removes the given player from this team.
// It panics if the player is not a member of this team.
func (team *Team) RemovePlayer(player *Player) {
	if player.Team != team {
		Logger.Panic(
			"cannot remove a player from a team they are not a member of",
			zap.Any("player", player),
		)
	}

	// Remove the player from the member set.
	delete(team.Players, player)

	// Remove the pointer from the player to the team.
	player.Team = nil
}

// OpposingTeam returns a pointer to the team that this team is against.
func (team *Team) OpposingTeam() *Team {
	if team.Lobby.Teams[0] == team {
		return team.Lobby.Teams[1]
	}

	return team.Lobby.Teams[0]
}

// Index returns the index of this team in its lobby.
// It panics if the lobby is neither index 0 nor 1.
func (team *Team) Index() uint8 {
	if team.Lobby.Teams[0] == team {
		return 0
	}

	if team.Lobby.Teams[1] == team {
		return 1
	}

	Logger.Panic("team is not in correct lobby")

	// Unreachable
	return 0
}

// ForAllMembers calls fn for every member of this team.
func (team *Team) ForAllMembers(fn func(*Player) error) error {
	errs := make([]error, 0)

	for member := range team.Players {
		errs = append(errs, fn(member))
	}

	return errors.Join(errs...)
}

// randomisedMembers returns a randomly-ordered slice of the team members.
func (team *Team) randomisedMembers() []*Player {
	var players []*Player

	for member := range team.Players {
		players = append(players, member)
	}

	rand.Shuffle(len(players), func(i, j int) {
		players[i], players[j] = players[j], players[i]
	})

	return players
}
