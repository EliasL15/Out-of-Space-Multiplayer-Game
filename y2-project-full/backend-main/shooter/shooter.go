package shooter

import (
	"errors"
	"fmt"
	"go.uber.org/zap"
	"server/core"
	"time"
)

// gameDuration is the maximum game duration. The game will end if it has lasted this long without
// one team wiping out the other.
const gameDuration = 1 * time.Minute

// initialHealth is the number of hitpoints each player starts with.
const initialHealth uint8 = 5

// shooterCtor is the constructor used for all shooter minigames, regardless of player count.
func shooterCtor(
	proto *core.MinigamePrototype,
	store core.ScoreStore,
	ship *core.Ship,
) *core.MinigameContext {
	return core.NewMinigameContext(proto, store, ship, newState())
}

// Proto1v1 is the prototype for the shooter minigame which uses two players.
var Proto1v1 = core.MinigamePrototype{
	Name:        "shooter_1v1",
	PlayerCount: 2,
	Worth:       2,
	Cooldown:    10 * time.Second,
	StoreCtor:   nil,
	Constructor: shooterCtor,
}

// Proto2v2 is the prototype for the shooter minigame which uses four players.
var Proto2v2 = core.MinigamePrototype{
	Name:        "shooter_2v2",
	PlayerCount: 4,
	Worth:       4,
	Cooldown:    5 * time.Second,
	StoreCtor:   nil,
	Constructor: shooterCtor,
}

// Proto3v3 is the prototype for the shooter minigame which uses six players.
var Proto3v3 = core.MinigamePrototype{
	Name:        "shooter_3v3",
	PlayerCount: 6,
	Worth:       6,
	Cooldown:    3 * time.Second,
	StoreCtor:   nil,
	Constructor: shooterCtor,
}

// playerState describes the state of an individual player.
type playerState struct {
	// health is the number of hitpoints the player has remaining. The player dies when this reaches
	// zero.
	health uint8

	// pos is the position of the player within the level.
	pos core.Position

	// armRotation is the angle of the player's arm.
	armRotation float64

	// bullets is a slice containing the positions of the bullets belonging to this player.
	bullets []core.Position
}

// parseBulletArray attempts to turn the given JSON-derived object into a slice of positions. If the
// conversion fails, a nil slice is returned.
func parseBulletArray(obj interface{}) []core.Position {
	arr, ok := obj.([]interface{})

	// We expect an array.
	if !ok {
		return nil
	}

	bullets := make([]core.Position, 0, len(arr))

	for _, obj := range arr {
		pos := core.PositionFromObj(obj)

		// If any single bullet position is invalid, the whole array is invalid.
		if pos == nil {
			return nil
		}

		bullets = append(bullets, *pos)
	}

	return bullets
}

type state struct {
	// timer counts down to the end of the game.
	timer core.FunctionTimer

	// alivePlayers maps player pointers to state objects for all of the currently-alive players.
	alivePlayers map[*core.Player]playerState

	// deadPlayerBullets maps player pointers to slices of bullet positions for dead players.
	deadPlayerBullets map[*core.Player][]core.Position
}

// newState returns a new empty shooter game state.
func newState() *state {
	return &state{
		timer:             core.ExpiredTimer(),
		alivePlayers:      make(map[*core.Player]playerState),
		deadPlayerBullets: make(map[*core.Player][]core.Position),
	}
}

// applyPhysicsReport parses a physics report message from the given player and makes the
// necessary changes to that player's state.
func (s *state) applyPhysicsReport(
	p *core.Player,
	m *core.Message,
) error {
	var newPos *core.Position

	// Parse the position object if present.
	if posInterface := m.TryGet("position"); posInterface != nil {
		newPos = core.PositionFromObj(*posInterface)

		if newPos == nil {
			return p.Client.Send(core.NewMessage("shooter_physics_report_position_invalid_error"))
		}
	}

	var newArm *float64

	// Parse the arm angle value if present.
	if m.TryGet("arm") != nil {
		arm, err := m.GetNumber("arm")

		if err != nil {
			return p.Client.Send(core.NewMessage("shooter_physics_report_invalid_arm_error"))
		}

		newArm = &arm
	}

	var newBullets []core.Position

	// Parse the bullet position array if present.
	if bulletsInterface := m.TryGet("bullets"); bulletsInterface != nil {
		newBullets = parseBulletArray(*bulletsInterface)

		if newBullets == nil {
			return p.Client.Send(core.NewMessage("shooter_physics_report_invalid_bullets_error"))
		}
	}

	if _, isDead := s.deadPlayerBullets[p]; isDead {
		if newPos != nil || newArm != nil || newBullets == nil {
			// Position and arm data should not be given for dead players, and bullet data must be
			// given.
			return p.Client.Send(core.NewMessage("shooter_physics_report_dead_data_error"))
		}

		s.deadPlayerBullets[p] = newBullets

		return nil
	}

	// If not dead, the player must be alive.
	aliveState := s.alivePlayers[p]

	// Only change fields that we've been given new values for.
	if newPos != nil {
		aliveState.pos = *newPos
	}

	if newArm != nil {
		aliveState.armRotation = *newArm
	}

	if newBullets != nil {
		aliveState.bullets = newBullets
	}

	s.alivePlayers[p] = aliveState

	return nil
}

// issuePhysicsReport sends a physics report describing the state of the given player to the
// player's peers.
func (s *state) issuePhysicsReport(p *core.Player) error {
	msg := core.NewMessage("shooter_peer_physics_report").
		Add("their_name", p.Name)

	if bullets, isDead := s.deadPlayerBullets[p]; isDead {
		_ = msg.Add("their_bullets", bullets)
	} else {
		aliveState := s.alivePlayers[p]

		// Add everything for alive players.
		_ = msg.Add("their_bullets", aliveState.bullets)
		_ = msg.Add("their_position", aliveState.pos.ToMap())
		_ = msg.Add("their_arm", aliveState.armRotation)
	}

	return p.ForAllActivityPeers(func(peer *core.Player) error {
		return peer.Client.Send(msg)
	})
}

// handlePhysicsReport processes a physics update message from the given player and passes the
// information to the player's peers.
func (s *state) handlePhysicsReport(
	p *core.Player,
	m *core.Message,
) error {
	// Update the server-side player state.
	applyErr := s.applyPhysicsReport(p, m)

	if applyErr != nil {
		return applyErr
	}

	// Tell the player's peers about the updated state.
	return s.issuePhysicsReport(p)
}

// livingPlayerByName returns a pointer to the alive player with the given name, or nil if no such
// alive player exists.
func (s *state) livingPlayerByName(name string) *core.Player {
	for p := range s.alivePlayers {
		if p.Name == name {
			return p
		}
	}

	return nil
}

// teamPlayerCounts returns the number of players there are alive on each team.
func (s *state) teamPlayerCounts() (count0, count1 uint8) {
	for p := range s.alivePlayers {
		if p.Team.Index() == 0 {
			count0 += 1
		} else {
			count1 += 1
		}
	}

	return
}

// end ends the game with the given result.
func (s *state) end(ctx *core.MinigameContext, result core.MinigameResult) error {
	// Stop the timer so it doesn't fire later.
	if ctx.Ship.Recorder != nil {
		c0, c1 := s.teamPlayerCounts()
		var winningTeam *core.Team
		if c0 > c1 {
			winningTeam = ctx.GetTeam(0)
		} else {
			winningTeam = ctx.GetTeam(1)
		}
		ps := make(map[*core.Player]float64)
		pw := make(map[*core.Player]uint8)
		players := 0 // Count the players so the correct game name can be added to db.
		_ = ctx.ForAllPlayers(func(p *core.Player) error {
			players++
			if p.Team == winningTeam {
				pw[p] = 1
			} else {
				pw[p] = 0
			}
			_, alive := s.alivePlayers[p] // Score is 1 if alive, 0 if dead.
			if alive {
				ps[p] = 1
			} else {
				ps[p] = 0
			}
			return nil
		})
		duration := gameDuration.Seconds() - s.timer.TimeLeft().Seconds()
		gameName := fmt.Sprintf("shooter_%[1]dv%[1]d", players/2)
		// shooter1v1, shooter2v2, shooter3v3
		ctx.Ship.Recorder.RecordMP(ps, pw, duration, gameName)

	}
	s.timer.Stop()

	return ctx.End(result)
}

// tryEnd ends the game if one team has been wiped out.
func (s *state) tryEnd(ctx *core.MinigameContext) error {
	c0, c1 := s.teamPlayerCounts()

	if c0+c1 == 0 {
		core.Logger.Panic(
			"shooter minigame has no players on either team",
			zap.Any("state", s),
		)

		panic("unreachable")
	}

	if c0 == 0 {
		// Team 1 has eliminated T0, so T1 wins.
		return s.end(ctx, core.MultiplayerResult(ctx.GetTeam(1)))
	}

	if c1 == 0 {
		// Team 0 has eliminated T1, so T0 wins.
		return s.end(ctx, core.MultiplayerResult(ctx.GetTeam(0)))
	}

	// There are players remaining on both teams, so the game is not over yet.
	return nil
}

// onTimeout is called when the main game timer expires.
func (s *state) onTimeout(ctx *core.MinigameContext) error {
	c0, c1 := s.teamPlayerCounts()

	// Draw by default.
	var winningTeam *core.Team = nil

	if c0 > c1 {
		// Team 0 has more players alive than T1, so T0 wins.
		winningTeam = ctx.GetTeam(0)
	} else if c1 > c0 {
		// Team 1 has more players alive than T0, so T1 wins.
		winningTeam = ctx.GetTeam(1)
	}

	// If both teams have equal player counts, the result is a draw (so we don't need to do
	// anything).

	// End the game.
	return s.end(ctx, core.MultiplayerResult(winningTeam))
}

// handleBulletHit processes a bullet hit message.
func (s *state) handleBulletHit(
	ctx *core.MinigameContext,
	p *core.Player,
	m *core.Message,
) error {
	victimName, err := m.GetString("victim")

	if err != nil {
		return p.Client.Send(core.NewMessage("shooter_bullet_hit_invalid_victim_name_error"))
	}

	victim := s.livingPlayerByName(victimName)

	if victim == nil {
		return p.Client.Send(core.NewMessage("shooter_bullet_hit_no_such_alive_player_error"))
	}

	vState := s.alivePlayers[victim]
	vState.health -= 1

	if vState.health == 0 {
		// The player is now dead.
		delete(s.alivePlayers, victim)
		s.deadPlayerBullets[victim] = vState.bullets
	} else {
		// The player is still alive.
		s.alivePlayers[victim] = vState
	}

	// Prepare a message to be sent to the victim.
	victimMsg := core.NewMessage("shooter_you_got_hit")
	_ = victimMsg.Add("shooter", p.Name)
	_ = victimMsg.Add("remaining_health", vState.health)

	// Prepare a message to be sent to the shooter. We shouldn't really need to give the victim's
	// name here, since the shooter has already provided us with this. However, we give it anyway
	// so that there is no possibility of confusion about which player the health value is for.
	shooterMsg := core.NewMessage("shooter_you_hit_someone")
	_ = shooterMsg.Add("victim", victim.Name)
	_ = shooterMsg.Add("victim_remaining_health", vState.health)

	// Prepare a message to be sent to all non-shooter and non-victim players. This message will
	// only be sent if there are more than two players in the whole game.
	thirdPartyMsg := core.NewMessage("shooter_someone_got_hit")
	_ = thirdPartyMsg.Add("shooter", p.Name)
	_ = thirdPartyMsg.Add("victim", victim.Name)
	_ = thirdPartyMsg.Add("victim_remaining_health", vState.health)

	// Send each player the relevant message.
	msgErrs := ctx.ForAllPlayers(func(player *core.Player) error {
		var toSend *core.Message

		if player == victim {
			toSend = victimMsg
		} else if player == p {
			toSend = shooterMsg
		} else {
			toSend = thirdPartyMsg
		}

		return player.Client.Send(toSend)
	})

	// Try to end the game.
	endErr := s.tryEnd(ctx)

	return errors.Join(msgErrs, endErr)
}

// spawnAll creates a record for each participating player without sending any messages.
func (s *state) spawnAll(ctx *core.MinigameContext) {
	_ = ctx.ForAllPlayers(func(p *core.Player) error {
		s.alivePlayers[p] = playerState{
			health: initialHealth,

			// todo: Generate proper spawn positions.
			pos: core.Position{
				X: 0,
				Y: 0,
			},

			armRotation: 0,
			bullets:     make([]core.Position, 0),
		}

		return nil
	})
}

// welcomeAll sends a welcome message to all players in the game.
func (s *state) welcomeAll(ctx *core.MinigameContext) error {
	return ctx.ForAllPlayers(func(p *core.Player) error {
		pState := s.alivePlayers[p]

		msg := core.NewMessage("shooter_welcome")
		_ = msg.Add("health_initial", pState.health)
		_ = msg.Add("your_spawn", pState.pos.ToMap())
		_ = msg.Add("duration", gameDuration.Seconds())

		// Add the map of peer spawns.
		{
			peerSpawns := map[string]map[string]float64{}

			_ = p.ForAllActivityPeers(func(peer *core.Player) error {
				peerSpawns[peer.Name] = s.alivePlayers[peer].pos.ToMap()
				return nil
			})

			_ = msg.Add("peer_spawns", peerSpawns)
		}

		return p.Client.Send(msg)
	})
}

// startTimer begins the game timer.
func (s *state) startTimer(ctx *core.MinigameContext) {
	s.timer = core.SingleTimer(ctx.Ship.Scheduler, time.Now().Add(gameDuration), func() error {
		if s.timer.WasStopped() {
			// Game finished between timer expiry and this function running.
			return nil
		}

		return s.onTimeout(ctx)
	})
}

func (s *state) Start(ctx *core.MinigameContext) error {
	// Add all of the players.
	s.spawnAll(ctx)

	// Welcome each player.
	welcomeErr := s.welcomeAll(ctx)

	// Begin the game.
	s.startTimer(ctx)

	return welcomeErr
}

func (s *state) Handle(ctx *core.MinigameContext, player *core.Player, message *core.Message) error {
	switch message.Type {
	case "shooter_physics_report":
		return s.handlePhysicsReport(player, message)

	case "shooter_bullet_player_hit":
		return s.handleBulletHit(ctx, player, message)
	}

	return player.Client.Send(core.NewMessage("shooter_unknown_message_error"))
}

func (s *state) HandleDisconnection(ctx *core.MinigameContext, player *core.Player) error {
	return s.end(ctx, core.MultiplayerDisconnection(player))
}
