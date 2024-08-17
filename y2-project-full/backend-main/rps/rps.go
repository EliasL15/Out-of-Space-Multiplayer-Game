package rps

import (
	"errors"
	"go.uber.org/zap"
	"server/core"
	"time"
)

// selectionTimeout is how long the players have to make their selections before the result of the
// round is decided.
const selectionTimeout = 3 * time.Second

// postRoundWait is the time gap left after each round. After this period is over, the game will
// either end (if the end condition is satisfied) or the next round will begin.
const postRoundWait = 3 * time.Second

// targetWinCount is the number of rounds that a player must win in order to win the whole game.
const targetWinCount uint = 3

// Prototype is the prototype for the 1v1 rock-paper-scissors minigame.
var Prototype = core.MinigamePrototype{
	Name:        "rps_1v1",
	PlayerCount: 2,
	Worth:       2,
	Cooldown:    5 * time.Second,
	StoreCtor:   nil,

	Constructor: func(
		proto *core.MinigamePrototype,
		store core.ScoreStore,
		ship *core.Ship,
	) *core.MinigameContext {
		return core.NewMinigameContext(proto, store, ship, newState())
	},
}

// An element is an object that a user can select.
type element uint8

const (
	rock     element = iota
	paper    element = iota
	scissors element = iota
)

// elementFromString returns a pointer to an element matching that specified in the given string.
// If the string is not recognised, nil is returned.
func elementFromString(s string) *element {
	var e element

	switch s {
	case "rock":
		e = rock
	case "paper":
		e = paper
	case "scissors":
		e = scissors
	default:
		return nil
	}

	return &e
}

// beats returns true if and only if selection `a` wins against selection `b`.
func (a element) beats(b element) bool {
	// Rock blunts scissors.
	if a == rock && b == scissors {
		return true
	}

	// Scissors cut paper.
	if a == scissors && b == paper {
		return true
	}

	// Paper wraps rock.
	if a == paper && b == rock {
		return true
	}

	return false
}

// String returns a string representation of the given element. It takes a pointer receiver so that
// it may return the string "none" for a nil element.
func (a *element) String() string {
	if a == nil {
		return "none"
	}

	if *a == rock {
		return "rock"
	}

	if *a == paper {
		return "paper"
	}

	if *a == scissors {
		return "scissors"
	}

	core.Logger.Panic("invalid rps element", zap.Uint8("elem", uint8(*a)))

	panic("unreachable")
}

// state is the game state for a 1v1 rock-paper-scissors game.
type state struct {
	// players is an array containing the two players taking part in the game.
	players [2]*core.Player

	// elements is an array containing a pointer to the element chosen by each of the two players.
	// If a player has not made a selection yet, their element pointer will be nil.
	elements [2]*element

	// wins is an array containing the number of rounds each player has won.
	wins [2]uint

	// selectionTimer counts the time the players have for making their selections.
	selectionTimer core.FunctionTimer

	// postRoundTimer begins at the end of a round and counts down to either the next round or the
	// end of the game, depending on the current game state.
	postRoundTimer core.FunctionTimer
	// totalGameTimer is the timer from game start to game end, used for the Recorder.
	totalGameTimer core.FunctionTimer
}

// newState returns a pointer to a new empty rock-paper-scissors game state object.
func newState() *state {
	return &state{
		// The players are added when the minigame starts.
		players: [2]*core.Player{nil, nil},

		// Start with no selection for either player.
		elements: [2]*element{nil, nil},

		// Zero wins each.
		wins: [2]uint{0, 0},

		selectionTimer: core.ExpiredTimer(),
		postRoundTimer: core.ExpiredTimer(),
		totalGameTimer: core.ExpiredTimer(),
	}
}

// playerIndex returns the index of the player in the minigame state. This will only ever be zero
// or one. If the given player is not involved in the game, this method will panic.
func (s *state) playerIndex(p *core.Player) int {
	if s.players[0] == p {
		return 0
	}

	if s.players[1] == p {
		return 1
	}

	core.Logger.Panic(
		"rps minigame does not have the given player",
		zap.Any("player", p),
	)

	panic("unreachable")
}

// roundWinner returns a pointer to the player that wins based on the current element selections.
// It takes into account the possibility of one or both players not making a selection. If the
// players make the same selection (or if neither makes a selection) then nil will be returned,
// indicating a draw.
func (s *state) roundWinner() *core.Player {
	if s.elements[0] == s.elements[1] {
		// Nobody wins when both players select the same.
		return nil
	}

	if s.elements[0] == nil {
		// Player 0 didn't pick anything, but player 1 did. P1 wins.
		return s.players[1]
	}

	if s.elements[1] == nil {
		// Player 1 didn't pick anything, but player 0 did. P0 wins.
		return s.players[0]
	}

	if *s.elements[0] == *s.elements[1] {
		return nil
	}

	if s.elements[0].beats(*s.elements[1]) {
		// Player 0's selection beats player 1's. P0 wins.
		return s.players[0]
	}

	// Not a draw, and P0's selection didn't beat P1's, so P1 must have won.
	return s.players[1]
}

// reportRoundResult informs the given player (p) that the round has ended and tells them the
// result, given a pointer to the player who won the round.
func (s *state) reportRoundResult(p *core.Player, winner *core.Player) error {
	var result string

	if winner == nil {
		result = "draw"
	} else if winner == p {
		result = "win"
	} else {
		result = "loss"
	}

	msg := core.NewMessage("rps_round_end").Add("result", result)

	// Tell the player what their opponent picked.
	if s.playerIndex(p) == 0 {
		msg.Add("opponent_selection", s.elements[1].String())
	} else {
		msg.Add("opponent_selection", s.elements[0].String())
	}

	return p.Client.Send(msg)
}

// endRound is called at the end of the round.
func (s *state) endRound(ctx *core.MinigameContext, winner *core.Player) error {
	if winner != nil {
		// Increment the win count for the winning player.
		s.wins[s.playerIndex(winner)] += 1
	}

	// Give both players the result.
	reportErr := ctx.ForAllPlayers(func(p *core.Player) error {
		return s.reportRoundResult(p, winner)
	})

	// Clear the selections so they don't carry over into the next round.
	s.elements = [2]*element{nil, nil}

	// Begin the post-round timer.
	s.postRoundTimer = core.SingleTimer(
		ctx.Ship.Scheduler,
		time.Now().Add(postRoundWait),

		func() error {
			return s.onPostRoundTimerEnd(ctx)
		},
	)

	return reportErr
}

// onRoundTimeout is called when the round timer finishes.
func (s *state) onRoundTimeout(ctx *core.MinigameContext) error {
	return s.endRound(ctx, s.roundWinner())
}

// endWithResult ends the game with the given result.
func (s *state) endWithResult(ctx *core.MinigameContext, result core.MinigameResult) error {
	s.postRoundTimer.Stop()
	s.selectionTimer.Stop()

	return ctx.End(result)
}

// endWithWinner ends the game, declaring the given player the winner.
func (s *state) endWithWinner(ctx *core.MinigameContext, winner *core.Player) error {
	if ctx.Ship.Recorder != nil {
		duration := (10*time.Minute - s.totalGameTimer.TimeLeft()).Seconds()
		s.totalGameTimer.Stop()
		ps := make(map[*core.Player]float64)
		pw := make(map[*core.Player]uint8)
		for _, v := range s.players {
			ps[v] = float64(s.wins[s.playerIndex(v)])
			if v == winner {
				pw[v] = 1
			} else {
				pw[v] = 0
			}
		}
		ctx.Ship.Recorder.RecordMP(ps, pw, duration, "rps_1v1")
	}
	return s.endWithResult(ctx, core.MultiplayerResult(winner.Team))
}

// onPostRoundTimerEnd is called when the post-round timer finishes.
func (s *state) onPostRoundTimerEnd(ctx *core.MinigameContext) error {
	if s.wins[0] == targetWinCount {
		return s.endWithWinner(ctx, s.players[0])
	} else if s.wins[1] == targetWinCount {
		return s.endWithWinner(ctx, s.players[1])
	}

	// Nobody has won yet, so the game goes on to the next round.
	return s.startRound(ctx)
}

// startRound begins the selection process for a new round.
func (s *state) startRound(ctx *core.MinigameContext) error {
	msg := core.NewMessage("rps_selection_start")

	// Tell all of the clients to start the selection process.
	err := ctx.ForAllPlayers(func(p *core.Player) error {
		return p.Client.Send(msg)
	})

	// Start the selection timer.
	s.selectionTimer = core.SingleTimer(
		ctx.Ship.Scheduler,
		time.Now().Add(selectionTimeout),

		func() error {
			return s.onRoundTimeout(ctx)
		},
	)

	return err
}

// handleSelectionMessage processes the given `rps_selection` message.
func (s *state) handleSelectionMessage(p *core.Player, m *core.Message) error {
	if s.selectionTimer.HasEnded() {
		// This is an error because the client should not allow the player to make a selection
		// once the selection period has ended.

		// fixme: Client/server latency means that the client will likely misrepresent the true
		//  time remaining, allowing this "error" to occur even when the player made a selection
		//  that appeared to be within the period.

		return p.Client.Send(core.NewMessage("rps_selection_too_late_error"))
	}

	if s.elements[s.playerIndex(p)] != nil {
		// The client should not allow the player to change their selection once one has been made.

		return p.Client.Send(core.NewMessage("rps_selection_already_made_error"))
	}

	elemStr, err := m.GetString("element")

	if err != nil {
		core.Logger.Warn(
			"rps minigame received non-string or no value for element",
			zap.Any("msg", m),
		)

		return p.Client.Send(core.NewMessage("rps_selection_invalid_string_error"))
	}

	elem := elementFromString(elemStr)

	if elem == nil {
		core.Logger.Warn(
			"rps minigame received invalid element name",
			zap.Any("msg", m),
		)

		return p.Client.Send(core.NewMessage("rps_invalid_selection_error"))
	}

	s.elements[s.playerIndex(p)] = elem

	return nil
}

func (s *state) Start(ctx *core.MinigameContext) error {
	welcome := core.NewMessage("rps_welcome").
		Add("selection_secs", selectionTimeout.Seconds()).
		Add("post_round_secs", postRoundWait.Seconds()).
		Add("target_win_count", targetWinCount)

	// Add the two participants to our array and welcome them both.
	welcomeErr := ctx.ForAllPlayers(func(p *core.Player) error {
		if s.players[0] == nil {
			// First player we've found.
			s.players[0] = p
		} else if s.players[1] == nil {
			// Second player.
			s.players[1] = p
		} else {
			core.Logger.Panic("rps minigame started with >2 players", zap.Any("ctx", ctx))

			// Unreachable.
		}

		return p.Client.Send(welcome)
	})
	if ctx.Ship.Recorder != nil { // If recording is enabled, start the timer to record the total game time.
		s.totalGameTimer = core.SingleTimer(
			ctx.Ship.Scheduler, // The max game time is not specified so make the timer expire after 10 mins.
			time.Now().Add(10*time.Minute),
			func() error {
				return nil
			},
		)
	}
	// Start the first round.
	startErr := s.startRound(ctx)

	return errors.Join(welcomeErr, startErr)
}

func (s *state) Handle(ctx *core.MinigameContext, player *core.Player, message *core.Message) error {
	if message.Type != "rps_selection" {
		core.Logger.Warn(
			"rps minigame expects only rps_selection messages",
			zap.Any("msg", message),
		)

		return player.Client.Send(core.NewMessage("rps_unexpected_message_type_error"))
	}

	return s.handleSelectionMessage(player, message)
}

func (s *state) HandleDisconnection(ctx *core.MinigameContext, player *core.Player) error {
	return s.endWithResult(ctx, core.MultiplayerDisconnection(player))
}
