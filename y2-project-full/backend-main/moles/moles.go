package moles

import (
	"go.uber.org/zap"
	"math/rand"
	"server/core"
	"time"
)

// gameDuration is the duration of the whack-a-mole minigame.
const gameDuration = time.Second * 30

// refreshInterval is the time after which we refresh the moles if the player has not clicked
// a mole yet.
const refreshInterval = time.Millisecond * 2500

// locationCountTotal is the number of different locations there are where moles can pop up.
// We don't care about the true positions here; it's up to the frontend to map integers to
// precise locations.
const locationCountTotal int = 33

// locationCountPerGame is the number of locations we use per core.
const locationCountPerGame int = 10

// A location is a value that identifies a unique position where a mole can appear.
type location = int

// locationSelection is a pair of random locations and a pool from which new locations can be
// picked.
type locationSelection struct {
	// locations is the slice of locations which are in use for this core.
	//
	// The first two locations in this slice are the currently selected ones.
	locations []location
}

// newLocationSelection returns a new locationSelection with a random location pool.
func newLocationSelection() locationSelection {
	// Take `locationCountPerGame` locations from `0..locationCountTotal`, ordered randomly. This
	// means that the struct returned by this function already has a random location selection
	// (since the selected locations are just [0] and [1]).
	locations := rand.Perm(locationCountTotal)[0:locationCountPerGame]

	return locationSelection{locations}
}

// refresh selects two new locations. The new locations are guaranteed to be different from the
// last pair.
func (sel *locationSelection) refresh() {
	locs := sel.locations
	n := len(locs)

	// The first two locations are the ones that are currently selected. We don't want to select
	// these again, so we move them to the end of the slice. We'll then exclude these from the
	// shuffle so that they remain at the end. As the current last two locations are the ones that
	// were selected the time before last, this swap allows them to be selected again.
	{
		// Swap the first and second-last locations.
		locs[0], locs[n-2] = locs[n-2], locs[0]

		// Swap the second and last locations.
		locs[1], locs[n-1] = locs[n-1], locs[1]
	}

	// Only shuffle the first `n - 2` elements so that the last two cannot end up selected.
	rand.Shuffle(n-2, func(i, j int) {
		// Swap.
		locs[i], locs[j] = locs[j], locs[i]
	})
}

// selected returns the two currently-selected locations.
func (sel *locationSelection) selected() [2]location {
	return [2]location(sel.locations[0:2])
}

// isSelected returns true if and only if the given location is one of the two selected locations.
func (sel *locationSelection) isSelected(loc location) bool {
	return sel.locations[0] == loc || sel.locations[1] == loc
}

// moleStore is the structure that we use to keep track of the threshold score for a flag for the
// mole minigame.
type moleStore struct {
	// threshold is the score that must be obtained in order to capture the minigame flag.
	threshold uint
}

// Prototype is the prototype for a single-player whack-a-mole minigame.
var Prototype = core.MinigamePrototype{
	Name:        "whack_a_mole",
	PlayerCount: 1,
	Worth:       1,
	Cooldown:    5 * time.Second,

	StoreCtor: func() core.ScoreStore {
		// Initial threshold is zero - any non-zero score will count as a win.
		return &moleStore{
			threshold: 0,
		}
	},

	Constructor: func(proto *core.MinigamePrototype, store core.ScoreStore, ship *core.Ship) *core.MinigameContext {
		return core.NewMinigameContext(proto, store, ship, newState())
	},
}

// state holds the state for the single-player whack-a-mole minigame.
type state struct {
	// locations is the location selection.
	locations locationSelection

	// refreshTimer is the timer we use to refresh the mole locations if the player is too slow.
	refreshTimer core.FunctionTimer

	// gameTimer is the timer we use for the duration of the whole core.
	gameTimer core.FunctionTimer

	// score is the player's current score (i.e. the number of moles they have hit).
	score uint
}

// newState returns a pointer to a new whack-a-mole minigame.
func newState() *state {
	return &state{
		locations:    newLocationSelection(),
		refreshTimer: core.ExpiredTimer(),
		gameTimer:    core.ExpiredTimer(),
		score:        0,
	}
}

func (game *state) Start(ctx *core.MinigameContext) error {
	// We're only expecting one player.
	player := ctx.ExactlyOnePlayer()

	msg := core.NewMessage("mole_welcome")
	_ = msg.Add("duration_seconds", gameDuration.Seconds())
	_ = msg.Add("interval_seconds", refreshInterval.Seconds())
	_ = msg.Add("initial_moles", game.locations.selected())
	_ = msg.Add("score_to_beat", ctx.Store.(*moleStore).threshold)

	// Send the message before starting the timers.
	err := player.Client.Send(msg)

	game.startTimers(ctx)

	return err
}

// startTimer begins the core duration and mole refresh timers.
func (game *state) startTimers(ctx *core.MinigameContext) {
	game.refreshTimer = core.SingleTimer(
		ctx.Ship.Scheduler,
		time.Now().Add(refreshInterval),

		func() error {
			return game.refresh(ctx)
		},
	)

	game.gameTimer = core.SingleTimer(
		ctx.Ship.Scheduler,
		time.Now().Add(gameDuration),

		func() error {
			return game.endNaturally(ctx)
		},
	)
}

// refresh refreshes the mole selection and notifies the player.
func (game *state) refresh(ctx *core.MinigameContext) error {
	// Pick new locations.
	game.locations.refresh()

	// Report the new locations to the player.
	msg := core.NewMessage("mole_timeout")
	_ = msg.Add("locations", game.locations.selected())

	return ctx.ExactlyOnePlayer().Client.Send(msg)
}

// resetRefreshTimer cancels and restarts the mole refresh timer.
func (game *state) resetRefreshTimer(ctx *core.MinigameContext) {
	game.refreshTimer.Stop()

	game.refreshTimer = core.SingleTimer(
		ctx.Ship.Scheduler,

		time.Now().Add(refreshInterval),

		func() error {
			if !game.refreshTimer.HasEnded() {
				core.Logger.Warn("refresh timer has already ended")

				// Function was scheduled before the timer ended, but the timer has been reset
				// since.
				return nil
			}

			return game.refresh(ctx)
		},
	)
}

// registerHit updates the core state in response to a mole hit.
func (game *state) registerHit(ctx *core.MinigameContext) {
	game.score += 1
	game.resetRefreshTimer(ctx)

	// Refresh but without notifying the player.
	game.locations.refresh()
}

// handleHitMessage processes a message from the client reporting that the player successfully
// hit a mole.
func (game *state) handleHitMessage(
	ctx *core.MinigameContext,
	player *core.Player,
	message *core.Message,
) error {
	loc, err := message.GetInt("location")

	if err != nil || loc < 0 || loc >= locationCountTotal {
		core.Logger.Warn(
			"mole minigame frontend sent invalid location value",
			zap.Int("loc", loc),
		)

		return player.Client.Send(core.NewMessage("mole_invalid_location_error"))
	}

	if !game.locations.isSelected(loc) {
		core.Logger.Warn("mole minigame frontend reported hit on hidden mole")

		// This error would only likely come as a result of a timing issue: if the server has
		// changed which moles are selected but the player clicks before the client has changed the
		// local mole selection, the client would send a message saying that the player clicked a
		// mole that, according to the server, is no longer selected.

		// fixme: Above.

		return player.Client.Send(core.NewMessage("mole_location_not_selected_error"))
	}

	game.registerHit(ctx)

	// Tell the player their score and the new mole positions.
	msg := core.NewMessage("mole_hit_valid").
		Add("score", game.score).
		Add("new_moles", game.locations.selected())

	return player.Client.Send(msg)
}

// endNaturally determines the core result and calls core.end().
func (game *state) endNaturally(ctx *core.MinigameContext) error {
	// The store should be a pointer to a mole store object.
	store := ctx.Store.(*moleStore)

	result := core.SinglePlayerLoss()

	// If the player did better than the threshold, they've won.
	if game.score > store.threshold {
		if ctx.Ship.Recorder != nil {
			ctx.Ship.Recorder.RecordSP(ctx.ExactlyOnePlayer(), float64(game.score),
				1, 30, "whack_a_mole")
		}
		result = core.SinglePlayerWin(ctx.ExactlyOnePlayer())

		// Update the store so that any future captures will need to do better than this.
		store.threshold = game.score
	} else {
		if ctx.Ship.Recorder != nil {
			ctx.Ship.Recorder.RecordSP(ctx.ExactlyOnePlayer(), float64(game.score),
				0, 30, "whack_a_mole")
		}
	}
	return game.end(ctx, result)
}

// end finishes the core with the given result.
func (game *state) end(ctx *core.MinigameContext, result core.MinigameResult) error {
	game.gameTimer.Stop()
	game.refreshTimer.Stop()

	return ctx.End(result)
}

func (game *state) Handle(ctx *core.MinigameContext, player *core.Player, message *core.Message) error {
	if ctx.ExactlyOnePlayer() != player {
		// If we get here, something went VERY wrong. This should NEVER occur.
		panic("invalid minigame state")
	}

	if message.Type == "mole_hit" {
		return game.handleHitMessage(ctx, player, message)
	}

	// We only understand a single message type; anything else is an error.
	msg := core.NewMessage("moles_unknown_message_type_error")
	_ = msg.Add("bad_type", message.Type)

	core.Logger.Warn(
		"mole minigame received unknown message",
		zap.Any("msg", message),
		zap.String("from", player.Name),
	)

	return player.Client.Send(msg)
}

func (game *state) HandleDisconnection(ctx *core.MinigameContext, player *core.Player) error {
	if ctx.ExactlyOnePlayer() != player {
		panic("invalid minigame state")
	}

	return game.end(ctx, core.SinglePlayerDisconnection(player))
}
