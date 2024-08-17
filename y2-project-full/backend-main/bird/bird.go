package bird

import (
	"server/core"
	"time"
)

// duration is the length of the minigame.
const duration = 60 * time.Second

var won = 0

// ProtoSp is the prototype for a single-player "Flappy Bird" type game.
var ProtoSp = core.MinigamePrototype{
	Name:        "fb_sp",
	PlayerCount: 1,
	Worth:       1,
	Cooldown:    5 * time.Second,

	StoreCtor: func() core.ScoreStore {
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

// state is the state of the Flappy Bird minigame.
type state struct {
	// timer counts down to the end of the game.
	timer core.FunctionTimer
}

func newState() *state {
	return &state{
		timer: core.ExpiredTimer(),
	}
}

func (s *state) Start(ctx *core.MinigameContext) error {
	msg := core.NewMessage("bird_welcome")
	_ = msg.Add("duration", duration.Seconds())
	_ = msg.Add("to_beat", *ctx.Store.(*int))

	s.timer = core.SingleTimer(ctx.Ship.Scheduler, time.Now().Add(duration), func() error {
		return ctx.ExactlyOnePlayer().Client.Send(core.NewMessage("bird_timeout"))
	})

	return ctx.ExactlyOnePlayer().Client.Send(msg)
}

// end cleans up and ends the minigame.
func (s *state) end(ctx *core.MinigameContext, result core.MinigameResult) error {
	if ctx.Ship.Recorder != nil {
		timeSpent := duration.Seconds() - s.timer.TimeLeft().Seconds()
		ctx.Ship.Recorder.RecordSP(ctx.ExactlyOnePlayer(), float64(*(ctx.Store.(*int))), uint8(won), timeSpent,
			"fb_sp")
	}
	s.timer.Stop()

	return ctx.End(result)
}

func (s *state) Handle(ctx *core.MinigameContext, player *core.Player, message *core.Message) error {
	if message.Type != "bird_end" {
		return player.Client.Send(core.NewMessage("bird_unexpected_message_error"))
	}

	score, err := message.GetInt("score")

	if err != nil {
		return player.Client.Send(core.NewMessage("bird_end_score_missing_error"))
	}

	store := ctx.Store.(*int)

	if score > *store {
		*store = score
		won = 1
		return s.end(ctx, core.SinglePlayerWin(player))
	}

	return s.end(ctx, core.SinglePlayerLoss())
}

func (s *state) HandleDisconnection(ctx *core.MinigameContext, player *core.Player) error {
	return s.end(ctx, core.SinglePlayerDisconnection(player))
}
