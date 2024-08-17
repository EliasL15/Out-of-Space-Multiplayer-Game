package core

import (
	"go.uber.org/zap"
)

// A LobbyManager is responsible for multiple lobby activities.
type LobbyManager struct {
	// scheduler is the scheduler which will be used by children of this lobby manager.
	scheduler Scheduler

	// activities maps lobby IDs to lobby activities.
	activities map[string]*LobbyActivity

	// minigames contains the minigames that can be played by lobbies created by this manager. The
	// keys are minigame IDs.
	minigames map[string]MinigamePrototype
}

// NewLobbyManager returns a new lobby manager with no lobbies.
func NewLobbyManager(scheduler Scheduler, minigames map[string]MinigamePrototype) *LobbyManager {
	return &LobbyManager{
		scheduler:  scheduler,
		activities: make(map[string]*LobbyActivity),
		minigames:  minigames,
	}
}

// generateLobbyID returns a lobby ID that is guaranteed to be unique within this manager.
func (mgr *LobbyManager) generateLobbyID() string {
	return randomLobbyCode()
}

// createLobby creates a new empty lobby under this manager and returns a pointer to the activity
// for it.
func (mgr *LobbyManager) createLobby() *LobbyActivity {
	lobby := &Lobby{
		manager: mgr,

		Teams: [2]*Team{
			{
				Lobby:   nil,
				Players: map[*Player]struct{}{},
			},

			{
				Lobby:   nil,
				Players: map[*Player]struct{}{},
			},
		},

		ID: mgr.generateLobbyID(),
	}

	lobby.Teams[0].Lobby = lobby
	lobby.Teams[1].Lobby = lobby

	act := NewLobbyActivity(lobby, mgr.scheduler)

	mgr.activities[lobby.ID] = act

	act.logger().Info("created new lobby")

	return act
}

// forget deletes the activity for the given lobby, stopping any new players joining.
func (mgr *LobbyManager) forget(lobby *Lobby) {
	Logger.Info("deleting lobby", zap.String("id", lobby.ID))

	delete(mgr.activities, lobby.ID)
}

// HandleLobbyCreate handles a lobby creation message.
func (mgr *LobbyManager) HandleLobbyCreate(client *Client) error {
	return mgr.createLobby().HandleJoinRequest(client)
}

// GetActivity returns a pointer to the lobby activity associated with the given ID,
// or nil if no such activity exists.
func (mgr *LobbyManager) GetActivity(id string) *LobbyActivity {
	if act, ok := mgr.activities[id]; ok {
		return act
	}

	return nil
}
