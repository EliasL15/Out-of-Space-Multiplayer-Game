package match

import (
	"errors"
	"fmt"
	"math/rand"
	"server/core"
	"time"
)

// A pattern is a number that represents a particular type of card. Two cards with the same pattern
// are considered a pair.
type pattern = uint8

var won = 0

// patternCount is the number of different card patterns we have.
const patternCount = 8

// gameDuration is the amount of time that the core runs for.
const gameDuration = 1 * time.Minute

// tickInterval is the amount of time we leave between timer updates given to the client.
const tickInterval = 500 * time.Millisecond

const (
	// gridWidth is the number of columns of cards we have on the table.
	gridWidth = 8

	// gridHeight is the number of rows of cards we have on the table.
	gridHeight = 4

	// gridSize is the total number of cards we have on the table.
	gridSize = gridWidth * gridHeight

	// repetitionCount is the number of cards we have with each pattern type.
	repetitionCount = gridSize / patternCount
)

// Prototype is the prototype for a single-player card matching memory core.
var Prototype = core.MinigamePrototype{
	Name:        "card_match_sp",
	PlayerCount: 1,
	Worth:       1,
	Cooldown:    5 * time.Second,
	StoreCtor:   nil,
	Constructor: func(proto *core.MinigamePrototype, store core.ScoreStore, ship *core.Ship) *core.MinigameContext {
		return core.NewMinigameContext(proto, store, ship, newState())
	},
}

// A card is a virtual playing card.
type card struct {
	// pattern is the pattern on the face of this card.
	pattern pattern

	// matched is true if and only if the card has been matched with another card already.
	matched bool
}

// The table is the virtual surface on which the cards are placed.
type table struct {
	// grid is the 4x8 card layout.
	grid [gridSize]card
}

// newTable returns a new *unshuffled* table.
func newTable() table {
	var grid [gridSize]card

	// For every pattern...
	for pattern := pattern(0); pattern < patternCount; pattern++ {
		patIdx := int(pattern * repetitionCount)

		// ...for every repetition...
		for r := 0; r < repetitionCount; r++ {
			repIdx := patIdx + r

			// ...place a single card with this pattern.
			grid[repIdx] = card{
				pattern: pattern,
				matched: false,
			}
		}
	}

	return table{grid}
}

// shuffle randomises the order of the cards in-place.
func (tbl *table) shuffle() {
	rand.Shuffle(gridSize, func(a, b int) {
		// Swap [a] and [b].
		tbl.grid[a], tbl.grid[b] = tbl.grid[b], tbl.grid[a]
	})
}

// card returns a pointer to the card at (x, y). It panics if no such card exists.
func (tbl *table) card(x, y uint) *card {
	idx := y*gridWidth + x

	if idx >= gridSize {
		panic(fmt.Sprintf("no such card (%v, %v) in grid of w=%v, h=%v", x, y, gridWidth, gridHeight))
	}

	return &tbl.grid[idx]
}

// state is the state of the single-player card matching minigame.
type state struct {
	// table is the table of cards.
	table table

	// flipped is a pointer to the first card that was turned face-up.
	// This will be nil if there is no face-up card.
	flipped *card

	// timer is the timer that counts down to the core end.
	timer *core.FunctionTimer
}

// newState returns a pointer to a new card matching core.
func newState() *state {
	table := newTable()
	table.shuffle()

	return &state{
		table:   table,
		flipped: nil,

		// timer is set in Start.
	}
}

func (game *state) startTimer(ctx *core.MinigameContext) {
	timer := core.TickingTimer(
		ctx.Ship.Scheduler,

		time.Now().Add(gameDuration),
		tickInterval,

		// Notify the player on every tick.
		func() error {
			return game.tick(ctx)
		},

		// Make the player lose the core if the timer is allowed to expire.
		func() error {
			return ctx.End(core.SinglePlayerLoss())
		},
	)

	game.timer = &timer
}

func (game *state) Start(ctx *core.MinigameContext) error {
	// Get the player. There should only be one because this is a single-player core.
	player := ctx.ExactlyOnePlayer()

	// We keep the table state hidden from the player, so the only thing we have to tell the client
	// is how long the core will go on for.
	msg := core.NewMessage("match_welcome")
	_ = msg.Add("duration_seconds", gameDuration.Seconds())

	// Send the welcome message before we start the core timer so that we don't waste core time
	// sending the message.
	err := player.Client.Send(msg)

	// Start the core timer.
	game.startTimer(ctx)

	return err
}

// tick reports the remaining time to the client.
func (game *state) tick(ctx *core.MinigameContext) error {
	secondsLeft := game.timer.TimeLeft().Seconds()

	msg := core.NewMessage("match_tick")
	_ = msg.Add("seconds_left", secondsLeft)

	return ctx.ExactlyOnePlayer().Client.Send(msg)
}

// end cleans up and ends the minigame.
func (game *state) end(ctx *core.MinigameContext, result core.MinigameResult) error {
	// Prevent any further ticks.
	if ctx.Ship.Recorder != nil { // Record the result.
		duration := gameDuration.Seconds() - game.timer.TimeLeft().Seconds()
		ctx.Ship.Recorder.RecordSP(ctx.ExactlyOnePlayer(), 0, uint8(won), duration, "card_match_sp")

	}
	game.timer.Stop()
	return ctx.End(result)
}

// endIfClear ends the minigame if the player has matched all cards on the grid. Otherwise,
// it does nothing.
func (game *state) endIfClear(ctx *core.MinigameContext, p *core.Player) error {
	for _, card := range game.table.grid {
		// If any cards are not matched, the player hasn't cleared the grid.
		if !card.matched {
			return nil
		}
	}
	won = 1
	// Grid is clear, so the player has won.
	return game.end(ctx, core.SinglePlayerWin(p))
}

// flip turns over the card at (x, y). It reports an error to the client if the card has
// already been flipped or matched.
func (game *state) flip(ctx *core.MinigameContext, p *core.Player, card *card) error {
	if card == game.flipped {
		return p.Client.Send(core.NewMessage("match_flipped_twice_error"))
	}

	if card.matched {
		return p.Client.Send(core.NewMessage("match_already_matched_error"))
	}

	if game.flipped == nil {
		// We don't have a card flipped already, so set this as the first flipped.
		game.flipped = card

		// Tell the client what pattern is on the card.
		return p.Client.Send(core.NewMessage("match_first_flip").Add("pattern", card.pattern))
	}

	// We already have a flipped card, so check if the patterns match.
	if card.pattern == game.flipped.pattern {
		// Cards match.

		card.matched = true
		game.flipped.matched = true
	}

	game.flipped = nil

	msg := core.NewMessage("match_second_flip")
	_ = msg.Add("pattern", card.pattern)
	_ = msg.Add("match_found", card.matched)

	// Tell the client what pattern is on the second card,
	// and whether the two cards match. (Technically the latter is redundant.)
	flipErr := p.Client.Send(msg)

	if card.matched {
		// If the pair matched, we should check if the player has cleared the whole grid.
		// If they have, the core is over.
		endErr := game.endIfClear(ctx, p)

		return errors.Join(flipErr, endErr)
	}

	return flipErr
}

// handleFlipMessage parses and processes a "match_flip" message.
func (game *state) handleFlipMessage(
	ctx *core.MinigameContext,
	player *core.Player,
	message *core.Message,
) error {
	x, err := message.GetInt("card_x")

	if err != nil || x < 0 || x >= gridWidth {
		return player.Client.Send(core.NewMessage("match_flip_invalid_x_error"))
	}

	y, err := message.GetInt("card_y")

	if err != nil || y < 0 || y >= gridHeight {
		return player.Client.Send(core.NewMessage("match_flip_invalid_y_error"))
	}

	// Conversion to uint is safe because we've checked that neither value is negative.
	card := game.table.card(uint(x), uint(y))

	// Flip the card.
	return game.flip(ctx, player, card)
}

func (game *state) Handle(ctx *core.MinigameContext, player *core.Player, message *core.Message) error {
	if ctx.ExactlyOnePlayer() != player {
		// If we get here, something went VERY wrong. This should NEVER occur.
		panic("invalid minigame state")
	}

	if message.Type == "match_flip" {
		return game.handleFlipMessage(ctx, player, message)
	}

	// We only understand a single message type; anything else is an error.
	msg := core.NewMessage("match_unknown_message_type_error")
	_ = msg.Add("bad_type", message.Type)

	return player.Client.Send(msg)
}

func (game *state) HandleDisconnection(ctx *core.MinigameContext, player *core.Player) error {
	if ctx.ExactlyOnePlayer() != player {
		panic("invalid minigame state")
	}

	return game.end(ctx, core.SinglePlayerDisconnection(player))
}
