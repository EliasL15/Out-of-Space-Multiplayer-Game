package race

import (
	"fmt"
	"server/core"
	"time"
)

var spWin = 0

// timeout is the maximum time that the game will run for. The game ends when this timeout is
// reached even if not all players have completed it.
const timeout = 150 * time.Second

// totalLaps is the number of laps a player must complete to finish the race.
const totalLaps = 3

// raceCtor is the constructor used for all race minigames, regardless of player count.
func raceCtor(
	proto *core.MinigamePrototype,
	store core.ScoreStore,
	ship *core.Ship,
) *core.MinigameContext {
	return core.NewMinigameContext(proto, store, ship, newState())
}

// ProtoSp is the prototype for the race minigame which uses only one player.
var ProtoSp = core.MinigamePrototype{
	Name:        "race_sp",
	PlayerCount: 1,
	Worth:       1,
	Cooldown:    15 * time.Second,

	// The singleplayer minigame is won or lost by comparison to the best time achieved on the
	// flag for the minigame, so we need to keep state.
	StoreCtor: func() core.ScoreStore {
		// The most sensible "time to beat" when nobody's actually played the minigame yet is just
		// the maximum possible time you could take.
		return &raceStore{bestTime: timeout}
	},

	Constructor: raceCtor,
}

// Proto1v1 is the prototype for the race minigame which uses two players.
var Proto1v1 = core.MinigamePrototype{
	Name:        "race_1v1",
	PlayerCount: 2,
	Worth:       2,
	Cooldown:    10 * time.Second,
	StoreCtor:   nil,
	Constructor: raceCtor,
}

// Proto2v2 is the prototype for the race minigame which uses four players.
var Proto2v2 = core.MinigamePrototype{
	Name:        "race_2v2",
	PlayerCount: 4,
	Worth:       4,
	Cooldown:    5 * time.Second,
	StoreCtor:   nil,
	Constructor: raceCtor,
}

// Proto3v3 is the prototype for the race minigame which uses six players.
var Proto3v3 = core.MinigamePrototype{
	Name:        "race_3v3",
	PlayerCount: 6,
	Worth:       6,
	Cooldown:    3 * time.Second,
	StoreCtor:   nil,
	Constructor: raceCtor,
}

// raceStore is the structure that we use to keep track of the best finishing time achieved at a
// singleplayer racing minigame flag.
type raceStore struct {
	// bestTime is the time that the player must beat in order to capture the flag.
	bestTime time.Duration
}

// playerState holds information about a single player.
type playerState struct {
	// pos is the player's position.
	pos core.Position

	// lapCount is the number of full laps the player has completed.
	lapCount int
}

// finishInfo holds information about how a player finished the race.
type finishInfo struct {
	// timeTaken is the amount of time it took the player to complete the race.
	timeTaken time.Duration

	// points is the number of points earned by this player for their finishing position.
	points int
}

// A state object represents the state of an ongoing racing minigame.
type state struct {
	// startTime is the time at which the race began.
	startTime time.Time

	// timer counts down to the race timeout.
	timer core.FunctionTimer

	// unfinishedPlayers maps players' names to their state while they are racing. Entries only
	// exist for players who have not yet finished.
	unfinishedPlayers map[string]playerState

	// finishedPlayers maps players' names to their finish information. Entries only exist for
	// players who have actually finished.
	finishedPlayers map[string]finishInfo
}

// newState returns a pointer to a new empty race minigame state.
func newState() *state {
	return &state{
		startTime:         time.Time{},
		timer:             core.ExpiredTimer(),
		unfinishedPlayers: make(map[string]playerState),
		finishedPlayers:   make(map[string]finishInfo),
	}
}

// spawnAll sets the initial state for each participating player. This method does not send any
// messages to clients.
func (s *state) spawnAll(ctx *core.MinigameContext) {
	availableSpawns := [][]core.Position{
		// Spawn positions for players from team 0.
		{
			{X: 374, Y: 789},
			{X: 403, Y: 820},
			{X: 421, Y: 840},
		},

		// Team 1.
		{
			{X: 400, Y: 789},
			{X: 432, Y: 820},
			{X: 445, Y: 840},
		},
	}

	if ctx.PlayerCount() == 1 {
		// In the singleplayer version of the game, the player's spawn position is not determined
		// by their team.
		s.unfinishedPlayers[ctx.ExactlyOnePlayer().Name] = playerState{
			pos:      availableSpawns[0][0],
			lapCount: 0,
		}

		return
	}

	_ = ctx.ForAllPlayers(func(p *core.Player) error {
		var spawn core.Position

		// Find the player's spawn position.
		{
			// Get a pointer to the slice of available spawns for this team so we can write back to
			// it.
			teamSpawns := &availableSpawns[p.Team.Index()]

			// Get the position and remove it so that the next player on this team gets a different
			// position.
			spawn, *teamSpawns = (*teamSpawns)[0], (*teamSpawns)[1:]
		}

		s.unfinishedPlayers[p.Name] = playerState{
			pos:      spawn,
			lapCount: 0,
		}

		return nil
	})
}

// welcomeMessageBase returns the core welcome message that we use, regardless of player count.
func (s *state) welcomeMessageBase(p *core.Player) *core.Message {
	msg := core.NewMessage("race_welcome")
	_ = msg.Add("your_spawn", s.unfinishedPlayers[p.Name].pos.ToMap())
	_ = msg.Add("laps", totalLaps)
	_ = msg.Add("timeout", timeout)

	return msg
}

// welcomeSp sends a welcome message to the only player of this singleplayer minigame.
func (s *state) welcomeSp(ctx *core.MinigameContext) error {
	p := ctx.ExactlyOnePlayer()
	store := ctx.Store.(*raceStore)

	msg := s.welcomeMessageBase(p)
	_ = msg.Add("to_beat", store.bestTime.Seconds())

	return p.Client.Send(msg)
}

// welcomePlayerMp sends a multiplayer welcome message to one player.
func (s *state) welcomePlayerMp(p *core.Player) error {
	msg := s.welcomeMessageBase(p)

	// Tell the player where the others are spawning.
	{
		peerSpawns := make(map[string]map[string]float64)

		_ = p.ForAllActivityPeers(func(peer *core.Player) error {
			peerSpawns[peer.Name] = s.unfinishedPlayers[peer.Name].pos.ToMap()

			return nil
		})

		_ = msg.Add("peer_spawns", peerSpawns)
	}

	return p.Client.Send(msg)
}

// welcomeAll sends each player a welcome message.
func (s *state) welcomeAll(ctx *core.MinigameContext) error {
	if ctx.PlayerCount() == 1 {
		return s.welcomeSp(ctx)
	}

	return ctx.ForAllPlayers(func(p *core.Player) error {
		return s.welcomePlayerMp(p)
	})
}

// end cleans up after the state and ends the game with the given result.
func (s *state) end(ctx *core.MinigameContext, result core.MinigameResult) error {
	if ctx.Ship.Recorder != nil {
		timeSpent := timeout.Seconds() - s.timer.TimeLeft().Seconds()
		t0, t1 := s.calculatePoints(ctx)
		playerCount := ctx.PlayerCount()
		if playerCount == 1 {
			if ctx.ExactlyOnePlayer().Team.Index() == 0 { // Record for race_sp
				ctx.Ship.Recorder.RecordSP(ctx.ExactlyOnePlayer(), float64(t0), uint8(spWin), timeSpent,
					"race_sp")
			} else {
				ctx.Ship.Recorder.RecordSP(ctx.ExactlyOnePlayer(), float64(t1), uint8(spWin), timeSpent,
					"race_sp")
			}
		} else {
			ps := make(map[*core.Player]float64)
			pw := make(map[*core.Player]uint8)
			var winningTeam *core.Team
			if t0 > t1 { // Get winning team.
				winningTeam = ctx.GetTeam(0)
			} else {
				winningTeam = ctx.GetTeam(1)
			}
			_ = ctx.ForAllPlayers(func(p *core.Player) error {
				if p.Team == winningTeam {
					pw[p] = 1
				} else {
					pw[p] = 0
				}
				var points float64 // Player gets 0 points for not finishing, else get the points from the store.
				finish, didFinish := s.finishedPlayers[p.Name]
				if !(didFinish) {
					points = 0
				} else {
					points = float64(finish.points)
				}
				ps[p] = points
				return nil
			})
			gameName := fmt.Sprintf("race_%[1]dv%[1]d", playerCount/2)
			// race_1v1,race2v2,race3v3
			ctx.Ship.Recorder.RecordMP(ps, pw, timeSpent, gameName)
		}
	}
	s.timer.Stop()
	return ctx.End(result)
}

// endSp decides the outcome of a singleplayer game and terminates the game.
func (s *state) endSp(ctx *core.MinigameContext) error {
	p := ctx.ExactlyOnePlayer()

	if finish, didFinish := s.finishedPlayers[p.Name]; didFinish {
		store := ctx.Store.(*raceStore)

		if finish.timeTaken < store.bestTime {
			store.bestTime = finish.timeTaken
			spWin = 1
			// Player beat the record time, so they win.
			return s.end(ctx, core.SinglePlayerWin(p))
		}
	}

	// The player either didn't finish, or they finished with a slower time than the time to beat.
	// In either case, they lose.
	return s.end(ctx, core.SinglePlayerLoss())
}

// calculatePoints returns the point totals for teams 0 and 1, in that order.
func (s *state) calculatePoints(ctx *core.MinigameContext) (t0, t1 int) {
	_ = ctx.ForAllPlayers(func(p *core.Player) error {
		finish, didFinish := s.finishedPlayers[p.Name]

		if !didFinish {
			// Players who have not finished do not earn any points.
			return nil
		}

		if p.Team.Index() == 0 {
			t0 += finish.points
		} else {
			t1 += finish.points
		}

		return nil
	})

	return
}

// endMp decides the outcome of a multiplayer game and terminates the game.
func (s *state) endMp(ctx *core.MinigameContext) error {
	t0, t1 := s.calculatePoints(ctx)

	var result core.MinigameResult

	// Default result is a draw, so we only modify it if the result is _not_ a draw.
	if t0 > t1 {
		result = core.MultiplayerResult(ctx.GetTeam(0))
	} else {
		result = core.MultiplayerResult(ctx.GetTeam(1))
	}

	return s.end(ctx, result)
}

// end decides the game outcome and terminates the game.
func (s *state) finishRace(ctx *core.MinigameContext) error {
	if ctx.PlayerCount() == 1 {
		return s.endSp(ctx)
	}

	return s.endMp(ctx)
}

// onTimeout is called when the game timer finishes, unless the game has already ended.
func (s *state) onTimeout(ctx *core.MinigameContext) error {
	return s.finishRace(ctx)
}

// startTimers starts the game timers.
func (s *state) startTimers(ctx *core.MinigameContext) {
	// Store a reference time so we can calculate how long each player takes to finish the race.
	s.startTime = time.Now()

	s.timer = core.SingleTimer(ctx.Ship.Scheduler, time.Now().Add(timeout), func() error {
		if s.timer.WasStopped() {
			return nil
		}

		return s.onTimeout(ctx)
	})
}

func (s *state) Start(ctx *core.MinigameContext) error {
	// Spawn all of the players.
	s.spawnAll(ctx)

	// Welcome them.
	welcomeErr := s.welcomeAll(ctx)

	// Start the game.
	s.startTimers(ctx)

	return welcomeErr
}

// notifyPlayerFinish sends a message to every player telling them that the given player has
// finished the race.
func (s *state) notifyPlayerFinish(p *core.Player, ctx *core.MinigameContext) error {
	fInfo := s.finishedPlayers[p.Name]

	selfMsg := core.NewMessage("race_you_finished")
	_ = selfMsg.Add("time_taken", fInfo.timeTaken.Seconds())
	_ = selfMsg.Add("points_earned", fInfo.points)

	peerMsg := core.NewMessage("race_peer_finished")
	_ = peerMsg.Add("their_name", p.Name)
	_ = peerMsg.Add("time_taken", fInfo.timeTaken.Seconds())
	_ = peerMsg.Add("points_earned", fInfo.points)

	return ctx.ForAllPlayers(func(player *core.Player) error {
		if player == p {
			return player.Client.Send(selfMsg)
		}

		return player.Client.Send(peerMsg)
	})
}

// moveToFinishedState moves the given player to the finished state, recording their finishing time
// and notifying all players.
func (s *state) moveToFinishedState(p *core.Player, ctx *core.MinigameContext) error {
	fInfo := finishInfo{
		timeTaken: time.Now().Sub(s.startTime),

		// The number of points earned is equal to the number of players beaten plus one*. For
		// example, in a 3v3 game the player who finishes first earns six points, the player who
		// finishes second earns five, and so on until the player who finishes last, who earns a
		// single point. A score of zero is reserved for players who do not finish.
		//
		// *The "plus one" is implicit here because we haven't yet removed the finishing player from
		// unfinishedPlayers, which means they are counted as having "beaten themselves"; this is
		// one extra player, so replaces the explicit addition.
		points: len(s.unfinishedPlayers),
	}

	// This player has now finished, so remove their unfinished state...
	delete(s.unfinishedPlayers, p.Name)

	// ...and add their finished state.
	s.finishedPlayers[p.Name] = fInfo

	if len(s.unfinishedPlayers) == 0 {
		// No unfinished players left, so the game is over.
		return s.finishRace(ctx)
	}

	// Notify everyone about the state change.
	return s.notifyPlayerFinish(p, ctx)
}

// increaseLapCount handles a lap count increase message from the given player.
func (s *state) increaseLapCount(p *core.Player, ctx *core.MinigameContext) error {
	pState, ok := s.unfinishedPlayers[p.Name]

	// Can't register a lap count increase if the player has already finished the race.
	if !ok {
		return p.Client.Send(core.NewMessage("race_player_lap_increase_already_finished_error"))
	}

	pState.lapCount += 1

	// If the player has now met the target lap count, they have finished the race.
	if pState.lapCount == totalLaps {
		return s.moveToFinishedState(p, ctx)
	}

	// Write the modified state back.
	s.unfinishedPlayers[p.Name] = pState

	// Notify the peers of this player.
	msg := core.NewMessage("race_peer_completed_lap")
	_ = msg.Add("their_name", p.Name)
	_ = msg.Add("laps_completed", pState.lapCount)

	return p.ForAllActivityPeers(func(peer *core.Player) error {
		return peer.Client.Send(msg)
	})
}

// notifyPosChange notifies the peers of the given player that the player's position has changed.
func (s *state) notifyPosChange(p *core.Player) error {
	msg := core.NewMessage("race_peer_pos_changed")
	_ = msg.Add("their_name", p.Name)
	_ = msg.Add("their_pos", s.unfinishedPlayers[p.Name].pos)

	return p.ForAllActivityPeers(func(peer *core.Player) error {
		return peer.Client.Send(msg)
	})
}

// handlePosChange parses and handles a position change message from the given player.
func (s *state) handlePosChange(p *core.Player, m *core.Message) error {
	pState, isUnfinished := s.unfinishedPlayers[p.Name]

	if !isUnfinished {
		// note: The frontend might want to explicitly catch and ignore this message, because it's
		// quite possible that network delay might lead to the backend receiving a position update
		// after it has decided that a player has finished the race.
		return p.Client.Send(core.NewMessage("race_pos_change_for_finished_player_error"))
	}

	posObj := m.TryGet("pos")

	if posObj == nil {
		return p.Client.Send(core.NewMessage("race_pos_change_missing_pos_error"))
	}

	pos := core.PositionFromObj(*posObj)

	if pos == nil {
		return p.Client.Send(core.NewMessage("race_pos_change_invalid_pos_error"))
	}

	pState.pos = *pos

	// Write back the updated state.
	s.unfinishedPlayers[p.Name] = pState

	// Notify the other players.
	return s.notifyPosChange(p)
}

func (s *state) Handle(
	ctx *core.MinigameContext,
	player *core.Player,
	message *core.Message,
) error {
	switch message.Type {
	case "race_completed_lap":
		return s.increaseLapCount(player, ctx)

	case "race_pos_changed":
		return s.handlePosChange(player, message)
	}

	return player.Client.Send(core.NewMessage("race_unknown_message_type_error"))
}

func (s *state) HandleDisconnection(ctx *core.MinigameContext, player *core.Player) error {
	if ctx.PlayerCount() == 1 {
		return s.end(ctx, core.SinglePlayerDisconnection(player))
	}

	return s.end(ctx, core.MultiplayerDisconnection(player))
}
