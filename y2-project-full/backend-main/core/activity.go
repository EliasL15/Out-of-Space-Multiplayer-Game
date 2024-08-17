package core

import (
	"go.uber.org/zap"
	"time"
)

// An Activity is an environment which a player can be in.
type Activity interface {
	// HandleMessage handles a message from the given player.
	HandleMessage(player *Player, message *Message) error

	// Start starts the activity.
	Start() error
}

// A MinigameResult is the result of a finished minigame.
type MinigameResult struct {
	// winningTeam is the team which won the minigame.
	//
	// This will be nil if the result represents a loss (or disconnection) in a single-player core.
	winningTeam *Team

	// disconnected is a pointer to the player whose disconnection caused the minigame to end,
	// if any.
	//
	// If the minigame did not end due to a disconnection, this will be nil.
	disconnected *Player
}

// SinglePlayerDisconnection returns the MinigameResult that should be used when a single-player
// minigame finishes early because the given player disconnected.
func SinglePlayerDisconnection(player *Player) MinigameResult {
	return MinigameResult{
		winningTeam:  nil,
		disconnected: player,
	}
}

// MultiplayerDisconnection returns the MinigameResult that should be used when a multiplayer
// minigame finishes early because the given player disconnected.
func MultiplayerDisconnection(player *Player) MinigameResult {
	return MinigameResult{
		// The opposing team wins automatically when a player disconnects.
		winningTeam:  player.Team.OpposingTeam(),
		disconnected: player,
	}
}

// SinglePlayerWin returns a MinigameResult which represents a win for the given player in a
// single-player core.
func SinglePlayerWin(player *Player) MinigameResult {
	// TODO: Will add the power-ups here.
	return MinigameResult{winningTeam: player.Team, disconnected: nil}
}

// SinglePlayerLoss returns a MinigameResult which represents a loss in a single-player core.
func SinglePlayerLoss() MinigameResult {
	return MinigameResult{winningTeam: nil, disconnected: nil}
}

// MultiplayerResult returns a MinigameResult which corresponds to a multiplayer win for the
// given team.
func MultiplayerResult(winner *Team) MinigameResult {
	return MinigameResult{winningTeam: winner, disconnected: nil}
}

// A MinigamePrototype describes a minigame.
type MinigamePrototype struct {
	// Name is the name of the minigame.
	Name string

	// PlayerCount is the total number of players (across all teams) required to play this minigame.
	PlayerCount int

	// Worth is the number of tokens this minigame is worth.
	Worth int

	// Cooldown is the amount of time for which a flag for this minigame will be unavailable after
	// the minigame finishes.
	Cooldown time.Duration

	// StoreCtor is a function which returns a new store object for a flag for this minigame.
	// If it is nil, the ship does not store any state.
	StoreCtor func() ScoreStore

	// Constructor is a function which creates
	// (but does not start) the minigame that this prototype is for.
	// It should return a pointer to a minigame context object.
	Constructor func(proto *MinigamePrototype, store ScoreStore, ship *Ship) *MinigameContext
}

// TeamSize returns the number of players on a single team.
// This will return zero for single-player minigame prototypes.
func (p *MinigamePrototype) TeamSize() int {
	return p.PlayerCount / 2
}

// IndividualWorth returns the fraction of the minigame's token worth that should be added to each
// winning player's individual score.
//
// Single-player games award the full token worth to the winning player; multiplayer games split the
// token worth between the number of participants on the winning team.
func (p *MinigamePrototype) IndividualWorth() float64 {
	if p.PlayerCount == 1 {
		return float64(p.Worth)
	}

	return float64(p.Worth) / float64(p.TeamSize())
}

// A ScoreStore is an object stored in a flag that a minigame can use to keep track of state
// between core instances. For example, if the win condition is "getting a better score than the
// player who captured the flag previously", a minigame might decide to create a ScoreStore that
// simply stores the score achieved by the last player.
type ScoreStore interface{}

// A MinigameContext wraps a minigame implementation to provide useful information.
type MinigameContext struct {
	// Ship is the main core activity.
	Ship *Ship

	// Store is the score store from the flag that started this minigame.
	Store ScoreStore

	// proto is a pointer to the minigame prototype that this minigame context was created for.
	proto *MinigamePrototype

	// impl is the object which provides the minigame implementation.
	impl MinigameImpl
}

// NewMinigameContext returns a pointer to a new minigame context created from the given
// prototype and implementation in the given ship.
func NewMinigameContext(
	proto *MinigamePrototype,
	store ScoreStore,
	ship *Ship,
	impl MinigameImpl,
) *MinigameContext {
	return &MinigameContext{
		Ship:  ship,
		Store: store,
		proto: proto,
		impl:  impl,
	}
}

// Start begins the minigame.
func (ctx *MinigameContext) Start() error {
	ctx.ensureValid()

	return ctx.impl.Start(ctx)
}

// HandleMessage passes a message through to the minigame implementation for processing.
func (ctx *MinigameContext) HandleMessage(player *Player, message *Message) error {
	ctx.ensureValid()

	if message.Type == "lobby_bye" {
		return ctx.impl.HandleDisconnection(ctx, player)
	}

	if message.Type == "ship_flag_activate" || message.Type == "ship_mov_position_update" {
		Logger.Warn(
			"minigame received delayed ship message; ignoring",
			zap.String("minigame", ctx.proto.Name),
			zap.String("player", player.Name),
		)

		return nil
	}

	err := ctx.impl.Handle(ctx, player, message)

	if err != nil {
		Logger.Warn(
			"minigame returned error when handling message",
			zap.String("minigame",
				ctx.proto.Name),
			zap.Any("msg",
				message),
			zap.String("player", player.Name),
			zap.Error(err),
		)
	}

	return err
}

// End finishes the minigame and reports the result back to the main core.
//
// After calling this method, ctx will be invalid and should not be used.
func (ctx *MinigameContext) End(result MinigameResult) error {
	ctx.ensureValid()
	return ctx.Ship.endMinigame(ctx, result)
}

// PlayerCount returns the number of players in the minigame.
func (ctx *MinigameContext) PlayerCount() int {
	return ctx.proto.PlayerCount
}

// ForAllPlayers calls fn for every player in the minigame.
func (ctx *MinigameContext) ForAllPlayers(fn func(*Player) error) error {
	ctx.ensureValid()

	return ctx.Ship.lobby.ForAllPlayers(func(p *Player) error {
		if p.Activity != ctx {
			// Not in this minigame.
			return nil
		}

		return fn(p)
	})
}

// ExactlyOnePlayer returns a pointer to one player who is playing this minigame.
// This method will panic if no players are found or if multiple players are found.
// It is intended for use by single-player games which should never have multiple players.
func (ctx *MinigameContext) ExactlyOnePlayer() *Player {
	ctx.ensureValid()

	var found *Player

	_ = ctx.ForAllPlayers(func(p *Player) error {
		if found == nil {
			// This is our player.
			found = p
			return nil
		}

		Logger.Panic("found a second player")

		// Unreachable
		return nil
	})

	if found == nil {
		Logger.Panic("did not find a player")
	}

	return found
}

// GetTeam returns a pointer to the team with the given index.
func (ctx *MinigameContext) GetTeam(index int) *Team {
	return ctx.Ship.lobby.Teams[index]
}

// invalidate clears the minigame context.
func (ctx *MinigameContext) invalidate() {
	ctx.proto = nil
	ctx.impl = nil
	ctx.Store = nil
	ctx.Ship = nil
}

// ensureValid panics if ctx appears to have been invalidated.
// This allows us to provide a very clear error message; the alternative is that the server
// panics later when the minigame tries to dereference some now-invalid pointer in ctx.
//
// Internal server code should never cause this method to panic.
// It is intended for minigames which may have scheduled events and do not cancel them if the
// minigame ends early, for example.
func (ctx *MinigameContext) ensureValid() {
	// We could do && here (see MinigameContext.invalidate),
	// but realistically if any of these are nil then something's gone wrong.
	if ctx.proto == nil || ctx.impl == nil || ctx.Ship == nil {
		Logger.Panic("minigame context method called on invalid context - did you forget to" +
			" cancel a" +
			" scheduled event when the minigame ended?")
	}
}

// A MinigameImpl provides an implementation for a minigame.
type MinigameImpl interface {
	// Start begins the minigame. It will only be called once all the players have been moved
	// into the minigame.
	//
	// Minigames which need to schedule events should do so in this method.
	Start(ctx *MinigameContext) error

	// Handle handles a message from a player.
	Handle(ctx *MinigameContext, player *Player, message *Message) error

	// HandleDisconnection is called when a minigame participant disconnects from the server.
	// This method should end the minigame.
	HandleDisconnection(ctx *MinigameContext, player *Player) error
}
