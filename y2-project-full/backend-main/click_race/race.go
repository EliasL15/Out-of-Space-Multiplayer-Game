package click_race

import (
	"server/core"
	"time"
)

// duration is the time that a game lasts.
const duration = 5 * time.Second

// ProtoSp is the prototype for a single-player "cookie clicker" type game.
var ProtoSp = core.MinigamePrototype{
	Name:        "cps_race_sp",
	PlayerCount: 1,
	Worth:       1,
	Cooldown:    5 * time.Second,

	StoreCtor: func() core.ScoreStore {
		// Initial threshold score is zero. This means you must always click at least once in order
		// to win.
		s := 0
		return &s
	},

	Constructor: func(
		proto *core.MinigamePrototype,
		store core.ScoreStore,
		ship *core.Ship,
	) *core.MinigameContext {
		return core.NewMinigameContext(proto, store, ship, newState())
	},
}

// Proto1v1 is the prototype for a 1v1 "cookie clicker" type game.
var Proto1v1 = core.MinigamePrototype{
	Name:        "cps_race_1v1",
	PlayerCount: 2,
	Worth:       1,
	Cooldown:    10 * time.Second,
	StoreCtor:   nil,

	Constructor: func(
		proto *core.MinigamePrototype,
		store core.ScoreStore,
		ship *core.Ship,
	) *core.MinigameContext {
		return core.NewMinigameContext(proto, store, ship, newState())
	},
}

// state is a structure representing the game state for the CPS race game.
type state struct {
	// reportedScores maps player pointers to their respective scores, as reported via `cps_report`
	// messages. This map will be empty until the end of the game duration.
	reportedScores map[*core.Player]int

	// timer times the duration of the game.
	timer core.FunctionTimer
}

// newState returns a new CPS race game state. This state can be used with any number of players.
func newState() *state {
	return &state{
		reportedScores: make(map[*core.Player]int),
		timer:          core.ExpiredTimer(),
	}
}

// onTimeout is called when the main game timer expires. It informs all clients that the timer has
// expired. It is expected that clients will then send the score achieved during the time.
func (s *state) onTimeout(ctx *core.MinigameContext) error {
	msg := core.NewMessage("cps_timeout")

	return ctx.ForAllPlayers(func(p *core.Player) error {
		return p.Client.Send(msg)
	})
}

func (s *state) Start(ctx *core.MinigameContext) error {
	msg := core.NewMessage("cps_welcome")
	_ = msg.Add("duration", duration.Seconds())

	if ctx.PlayerCount() == 1 {
		_ = msg.Add("score_to_beat", *ctx.Store.(*int))
	}

	welcomeErr := ctx.ForAllPlayers(func(p *core.Player) error {
		return p.Client.Send(msg)
	})

	s.timer = core.SingleTimer(ctx.Ship.Scheduler, time.Now().Add(duration), func() error {
		return s.onTimeout(ctx)
	})

	return welcomeErr
}

// endSp determines the result of this singleplayer minigame and ends the minigame with it.
func (s *state) endSp(ctx *core.MinigameContext) error {
	player := ctx.ExactlyOnePlayer()

	// Get the score achieved by the current player.
	score := s.reportedScores[player]

	threshold := ctx.Store.(*int)
	if score > *threshold {
		// Update the threshold score.
		*threshold = score

		// Won.
		if ctx.Ship.Recorder != nil { // If recording is enabled...
			ctx.Ship.Recorder.RecordSP(player, float64(score), 1, 5, "cps_race_sp")
		}
		return ctx.End(core.SinglePlayerWin(player))
	}

	// Lost.
	if ctx.Ship.Recorder != nil {
		ctx.Ship.Recorder.RecordSP(player, float64(score), 0, 5, "cps_race_sp")
	}
	return ctx.End(core.SinglePlayerLoss())
}

// endMp determines the result of this multiplayer game and ends the minigame with it.
func (s *state) endMp(ctx *core.MinigameContext) error {
	if ctx.Store != nil {
		panic("multiplayer CPS game should have nil store")
	}

	teamScores := [2]int{0, 0}
	// Add each player's score to their team's total.
	_ = ctx.ForAllPlayers(func(p *core.Player) error {
		teamScores[p.Team.Index()] += s.reportedScores[p]

		return nil
	})
	if teamScores[0] == teamScores[1] {
		if ctx.Ship.Recorder != nil { // If recording is enabled...
			ps := make(map[*core.Player]float64)
			pw := make(map[*core.Player]uint8)
			for k := range ctx.GetTeam(0).Players { // Team 0 lost
				ps[k] = float64(teamScores[0])
				pw[k] = 0
			}
			for k := range ctx.GetTeam(1).Players { // Team 1 lost
				ps[k] = float64(teamScores[1])
				pw[k] = 0
			}
			ctx.Ship.Recorder.RecordMP(ps, pw, 5, "cps_race_1v1") // Record the results.
		}
		return ctx.End(core.MultiplayerResult(nil))
	}
	if teamScores[0] > teamScores[1] {
		if ctx.Ship.Recorder != nil {
			ps := make(map[*core.Player]float64)
			pw := make(map[*core.Player]uint8)
			for k := range ctx.GetTeam(0).Players { // Team 0 won
				ps[k] = float64(teamScores[0])
				pw[k] = 1
			}
			for k := range ctx.GetTeam(1).Players { // Team 1 lost
				ps[k] = float64(teamScores[1])
				pw[k] = 0
			}
			ctx.Ship.Recorder.RecordMP(ps, pw, 5, "cps_race_sp")
		}
		return ctx.End(core.MultiplayerResult(ctx.GetTeam(0)))
	}
	if ctx.Ship.Recorder != nil {
		ps := make(map[*core.Player]float64)
		pw := make(map[*core.Player]uint8)
		for k := range ctx.GetTeam(0).Players { // Team 0 lost
			ps[k] = float64(teamScores[0])
			pw[k] = 0
		}
		for k := range ctx.GetTeam(1).Players { // Team 1 won
			ps[k] = float64(teamScores[1])
			pw[k] = 1
		}
		ctx.Ship.Recorder.RecordMP(ps, pw, 5, "cps_race_sp")
	}
	return ctx.End(core.MultiplayerResult(ctx.GetTeam(1)))
}

// tryEnd checks to see if all participating clients have reported scores. If they have, a winner
// is determined and the game ends.
func (s *state) tryEnd(ctx *core.MinigameContext) error {
	if len(s.reportedScores) < ctx.PlayerCount() {
		// Not every player has reported a score yet, so the game can't end.
		return nil
	}

	// All players have reported scores, so the game can finish.

	if ctx.PlayerCount() == 1 {
		return s.endSp(ctx)
	}

	return s.endMp(ctx)
}

func (s *state) Handle(
	ctx *core.MinigameContext,
	player *core.Player,
	message *core.Message,
) error {
	if message.Type != "cps_report" {
		return player.Client.Send(core.NewMessage("cps_unknown_message_type_error"))
	}

	if !s.timer.HasEnded() {
		return player.Client.Send(core.NewMessage("cps_game_not_over_error"))
	}

	if _, hasScore := s.reportedScores[player]; hasScore {
		return player.Client.Send(core.NewMessage("cps_report_player_has_score_error"))
	}

	score, err := message.GetInt("clicks")

	if err != nil {
		return player.Client.Send(core.NewMessage("cps_report_bad_clicks_value_error"))
	}

	// Retain the score.
	s.reportedScores[player] = score

	return s.tryEnd(ctx)
}

func (s *state) HandleDisconnection(ctx *core.MinigameContext, player *core.Player) error {
	// Prevent the timer finishing if it hasn't finished already.
	s.timer.Stop()

	if ctx.PlayerCount() == 1 {
		return ctx.End(core.SinglePlayerDisconnection(player))
	}

	return ctx.End(core.MultiplayerDisconnection(player))
}
