package demo

import (
	"errors"
	"fmt"
	"server/core"
	"time"
)

// ProtoSp is the prototype for the demo minigame which only uses a single player.
var ProtoSp = core.MinigamePrototype{
	Name:        "demo_minigame_sp",
	PlayerCount: 1,
	Worth:       1,
	Cooldown:    5 * time.Second,
	StoreCtor:   nil,
	Constructor: func(proto *core.MinigamePrototype, store core.ScoreStore, ship *core.Ship) *core.MinigameContext {
		return core.NewMinigameContext(proto, store, ship, &state{
			pm: core.NewPositionManager("demo_mov_"),
		})
	},
}

// Proto1v1 is the prototype for the demo minigame which uses two players.
var Proto1v1 = core.MinigamePrototype{
	Name:        "demo_minigame_1v1",
	PlayerCount: 2,
	Worth:       2,
	Cooldown:    10 * time.Second,
	StoreCtor:   nil,
	Constructor: func(proto *core.MinigamePrototype, store core.ScoreStore, ship *core.Ship) *core.MinigameContext {
		return core.NewMinigameContext(proto, store, ship, &state{
			pm: core.NewPositionManager("demo_mov_"),
		})
	},
}

// Proto2v2 is the prototype for the demo minigame which uses four players.
var Proto2v2 = core.MinigamePrototype{
	Name:        "demo_minigame_2v2",
	PlayerCount: 4,
	Worth:       4,
	Cooldown:    5 * time.Second,
	StoreCtor:   nil,
	Constructor: func(proto *core.MinigamePrototype, store core.ScoreStore, ship *core.Ship) *core.MinigameContext {
		return core.NewMinigameContext(proto, store, ship, &state{
			pm: core.NewPositionManager("demo_mov_"),
		})
	},
}

// state is our game state.
type state struct {
	// pm manages player movement within the minigame.
	pm core.PositionManager
}

func (demo *state) Start(ctx *core.MinigameContext) error {
	playerCount := 0

	_ = ctx.ForAllPlayers(func(p *core.Player) error {
		playerCount += 1
		return nil
	})

	if playerCount == 1 {
		return ctx.End(core.SinglePlayerWin(ctx.ExactlyOnePlayer()))
	}

	// Spawn all players.
	return ctx.ForAllPlayers(func(p *core.Player) error {
		return demo.pm.SpawnPlayer(p, core.Position{
			X: 0,
			Y: 0,
		})
	})
}

// doWinCheck checks if the given player has won the minigame and ends it if they have.
func (demo *state) doWinCheck(ctx *core.MinigameContext, player *core.Player) error {
	pos, ok := demo.pm.Map[player]

	if !ok {
		return fmt.Errorf("no position entry for player %v", player.Name)
	}

	if pos.X > 185 {
		// Player has won.
		return ctx.End(core.MultiplayerResult(player.Team))
	}

	// Player has not won.
	return nil
}

func (demo *state) Handle(ctx *core.MinigameContext, player *core.Player, message *core.Message) error {
	handled, err := demo.pm.HandleMessage(player, message)

	if handled {
		// After a movement message, check if the win condition has been met.
		winCheckErr := demo.doWinCheck(ctx, player)

		return errors.Join(err, winCheckErr)
	}

	return player.Client.Send(core.NewMessage("demo_unknown_message_type_error"))
}

func (demo *state) HandleDisconnection(ctx *core.MinigameContext, player *core.Player) error {
	return ctx.End(core.MultiplayerDisconnection(player))
}
