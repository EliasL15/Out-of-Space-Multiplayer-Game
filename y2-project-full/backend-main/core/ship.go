package core

import (
	"errors"
	"go.uber.org/zap"
	"math"
	"math/rand"
	"os"
	"slices"
	"strconv"
	"time"
)

// playerFlagReach is the maximum distance a player can be from a flag when they activate it.
const playerFlagReach float64 = 50

// fullGameDuration is the amount of time between the point when the players enter the ship and the
// point when they see the end screen (assuming nobody disconnects early).
const fullGameDuration = 10 * time.Minute

// shipTickInterval is the amount of time we leave between main core timer updates given to each
// client.
const shipTickInterval = 5 * time.Second

// cooldownTickInterval is the amount of time we leave between flag cooldown timer updates.
const cooldownTickInterval = 1 * time.Second

// An activation describes the activation state of a flag.
type activation struct {
	// startTime is the time at which the flag was activated.
	startTime time.Time

	// lockedPlayers is the set of players who are locked to the flag.
	lockedPlayers map[*Player]struct{}
}

// A flag is an object which a team can capture by winning a minigame.
type flag struct {
	// pos is the position of the flag on the map.
	pos Position

	// minigameProto is the minigame prototype for this flag. It should never be mutated.
	minigameProto MinigamePrototype

	// minigame is the ongoing minigame activity, or nil if no minigame is ongoing for this flag.
	minigame *MinigameContext

	// store is the object used by the flag's minigame to store data when not running.
	store ScoreStore

	// owner is the team which last captured this flag, or nil if no team has captured the flag.
	owner *Team

	// cooldown is a pointer to the function timer for this flag's cooldown, if there is one
	// ongoing.
	cooldown FunctionTimer

	// activation is the flag's activation state.
	// This will be nil when the flag has not been activated.
	activation *activation
}

// isActivated returns true if and only if this flag is activated.
func (f *flag) isActivated() bool {
	return f.activation != nil
}

// needsPlayer returns true if and only if there is space for p in the minigame.
func (f *flag) needsPlayer(p *Player) bool {
	if !f.isActivated() {
		Logger.Panic(
			"needsPlayer makes no sense for a flag that has not been activated",
			zap.Any("flag", f),
		)
	}

	reqN := f.minigameProto.TeamSize()

	// Count how many players there are on the same team as p.
	realN := 0

	for pl := range f.activation.lockedPlayers {
		if pl.Team == p.Team {
			realN++
		}
	}

	// If the actual number of players we have on the team is fewer than the number required,
	// then we need the player.
	return realN < reqN
}

// Returns true if and only if the flag has enough players locked to it that the minigame can
// start with them.
func (f *flag) hasAllPlayers() bool {
	return len(f.activation.lockedPlayers) == f.minigameProto.PlayerCount
}

// The flagManager stores all of the flag information for the ship.
type flagManager struct {
	// flags maps flag IDs to flag pointers.
	flags map[string]*flag
}

// addMinigameFlag creates a new flag with the given ID at the given position for the given
// minigame prototype.
func (fm *flagManager) addMinigameFlag(id string, proto MinigamePrototype, pos Position) {
	if _, exists := fm.flags[id]; exists {
		Logger.Panic("flag ID is taken", zap.String("id", id))
	}

	var store ScoreStore

	// If the prototype provides a function for creating a store object, use it to give the flag
	// a data store. Otherwise, we just use nil.
	if proto.StoreCtor != nil {
		store = proto.StoreCtor()
	}

	fm.flags[id] = &flag{
		pos:           pos,
		minigameProto: proto,
		minigame:      nil,
		owner:         nil,
		cooldown:      ExpiredTimer(),
		store:         store,
		activation:    nil,
	}
}

// flagForPlayer returns a pointer to the flag that this player is locked to,
// or nil if the player is not locked to any flag.
func (fm *flagManager) flagForPlayer(p *Player) *flag {
	for _, f := range fm.flags {
		if !f.isActivated() {
			// Can't check locked players on a flag that has not been activated.
			continue
		}

		// Check if the locked players set contains this player pointer.
		if _, found := f.activation.lockedPlayers[p]; found {
			return f
		}
	}

	return nil
}

// oldestFlagNeedingPlayer returns a pointer to the earliest-activated flag that needs the given
// player, or nil if there are no flags which need the player.
func (fm *flagManager) oldestFlagNeedingPlayer(p *Player) *flag {
	var oldest *flag = nil

	for _, f := range fm.flags {
		if !f.isActivated() {
			continue
		}

		if oldest == nil || oldest.activation.startTime.After(f.activation.startTime) {
			// If the flag needs the player, we've found a new oldest.
			if f.needsPlayer(p) {
				oldest = f
			}
		}
	}

	return oldest
}

// idForFlag returns the ID of the given flag. It panics if the flag is not found in the flag map.
func (fm *flagManager) idForFlag(f *flag) string {
	for id, flag := range fm.flags {
		if f == flag {
			return id
		}
	}

	Logger.Panic("no such flag", zap.Any("flag", f))

	// Unreachable
	panic(nil)
}

// flagForMinigame returns a pointer to the flag to which an ongoing minigame is attached.
// As we should never have a minigame that is not linked to a flag,
// this method will panic if no flag is found.
func (fm *flagManager) flagForMinigame(ctx *MinigameContext) *flag {
	for _, f := range fm.flags {
		if f.minigame == ctx {
			return f
		}
	}

	Logger.Panic("no flag for given minigame", zap.Any("minigame", ctx))

	// Unreachable
	panic(nil)
}

// clearForEndgame deactivates all flags and cancels all ongoing cooldown tickers.
func (fm *flagManager) clearForEndgame() {
	Logger.Info("clearing flags for endgame state")

	for _, flag := range fm.flags {
		flag.activation = nil
		flag.cooldown.Stop()
	}
}

// areAnyMinigamesOngoing returns true if and only if there is at least one flag for which there is
// an ongoing minigame.
func (fm *flagManager) areAnyMinigamesOngoing() bool {
	for _, flag := range fm.flags {
		if flag.minigame != nil {
			return true
		}
	}

	return false
}

// teamScores counts up the tokens captured by teams 0 and 1 and returns them in that order.
func (fm *flagManager) teamScores() (s0, s1 int) {
	for _, f := range fm.flags {
		if f.owner == nil {
			// Flag has not been captured.
			continue
		}

		worth := f.minigameProto.Worth

		if f.owner.Index() == 0 {
			s0 += worth
		} else {
			s1 += worth
		}
	}

	return
}

// The Ship is the main core environment.
type Ship struct {
	// lobby is the lobby from which this ship originated.
	lobby *Lobby

	// fm is the flag manager for this ship.
	fm *flagManager

	// pm manages the positions of the players in the ship.
	pm PositionManager

	// individualScores maps player pointers to the estimated number of tokens that player is
	// responsible for capturing.
	//
	// For example, if two players on the same team win a minigame that's worth four tokens, each
	// of the two players will have two tokens added to their individual score.
	individualScores map[*Player]float64

	// Scheduler is the scheduler which this ship and the minigames can use to trigger events.
	Scheduler Scheduler

	// timer is a pointer to the timer for the main core stage. It should be set to nil once it
	// expires.
	timer FunctionTimer

	// isEndgame is true if and only if the ship is in the endgame state.
	isEndgame bool
	// Recorder is the pointer to the Recorder, nil if flag --record is not used.
	Recorder *Recorder
}

// NewShip returns a pointer to a new ship created from the given lobby and using the given
// scheduler.
func NewShip(lobby *Lobby, scheduler Scheduler) *Ship {
	return &Ship{
		Scheduler:        scheduler,
		lobby:            lobby,
		fm:               &flagManager{flags: make(map[string]*flag)},
		pm:               NewPositionManager("ship_mov_"),
		individualScores: make(map[*Player]float64),
		timer:            ExpiredTimer(),
		isEndgame:        false,
	}
}

func (ship *Ship) logger() *zap.Logger {
	return Logger.With(zap.String("ship_lobby", ship.lobby.ID))
}

// createInitialScores adds a zero entry for every player to the individual score map.
func (ship *Ship) createInitialScores() {
	_ = ship.lobby.ForAllPlayers(func(player *Player) error {
		ship.individualScores[player] = 0

		return nil
	})
}

// createFlags places flags in the ship.
func (ship *Ship) createFlags() {
	// todo: Lay out flags randomly.
	// For now we just place a few demo flags for testing.

	ship.logger().Info("placing flags")

	protos := ship.lobby.manager.minigames

	ship.fm.addMinigameFlag("flag0", protos["shooter_3v3"], Position{
		X: 0,
		Y: 0,
	})

	ship.fm.addMinigameFlag("flag1", protos["race_2v2"], Position{
		X: -192,
		Y: 208,
	})

	ship.fm.addMinigameFlag("flag2", protos["card_match_sp"], Position{
		X: 192,
		Y: 208,
	})

	ship.fm.addMinigameFlag("shush", protos["fb_sp"], Position{
		X: 192,
		Y: 0,
	})

	ship.fm.addMinigameFlag("wam", protos["whack_a_mole"], Position{
		X: -192,
		Y: 0,
	})

	ship.fm.addMinigameFlag("blah", protos["rps_1v1"], Position{
		X: 0,
		Y: 256,
	})

	ship.fm.addMinigameFlag("dmspt", protos["shooter_1v1"], Position{
		X: 0,
		Y: -256,
	})

	ship.fm.addMinigameFlag("idfk", protos["cps_race_sp"], Position{
		X: 192,
		Y: -208,
	})

	ship.fm.addMinigameFlag("idfk_", protos["cps_race_1v1"], Position{
		X: -192,
		Y: -208,
	})
}

// spawnTeam randomly places the members of the team with the given index in the given positions.
//
// If there is only one member, they will be placed in the middle. If there are two, they will be
// placed on the left and right. If there are three, all three positions will be filled.
func (ship *Ship) spawnTeam(index int, left Position, middle Position, right Position) {
	// Get the members in a random order.
	members := ship.lobby.Teams[index].randomisedMembers()

	if len(members) == 1 {
		// Only one member, so put them in the middle.
		ship.pm.Map[members[0]] = middle

		return
	}

	// At least two members, so put the first two on the left and right.
	ship.pm.Map[members[0]] = left
	ship.pm.Map[members[1]] = right

	// If there's another member, put them in the middle.
	if len(members) == 3 {
		ship.pm.Map[members[2]] = middle
	}
}

// setInitialPositions creates position entries for all players in the ship at the locations they
// should be when they first join the ship.
func (ship *Ship) setInitialPositions() {
	ship.logger().Info("spawning players")

	// Spawn team zero at the bottom.
	ship.spawnTeam(
		0,

		Position{
			X: -32,
			Y: 416,
		},

		Position{
			X: 0,
			Y: 416,
		},

		Position{
			X: 32,
			Y: 416,
		},
	)

	// Spawn team one at the top.
	ship.spawnTeam(
		1,

		Position{
			X: -32,
			Y: -400,
		},

		Position{
			X: 0,
			Y: -400,
		},

		Position{
			X: 32,
			Y: -400,
		},
	)
}

// welcomeAll moves all players into the ship activity and sends them a welcome (i.e.
// initialisation) message.
func (ship *Ship) welcomeAll() error {
	ship.logger().Info("moving players into ship")

	// Switch all players to ship activity so we can use ForAllShipPeers/ForAllActivityPeers in
	// the loop after.
	_ = ship.lobby.ForAllPlayers(func(p *Player) error {
		p.Activity = ship
		return nil
	})

	return ship.lobby.ForAllPlayers(func(p *Player) error {
		return ship.welcomePlayer(p)
	})
}

// tick sends a message with the remaining main core time to all players in the ship (but not those
// in minigames).
func (ship *Ship) tick() error {
	if ship.timer.HasEnded() {
		// Nothing to do.
		return nil
	}

	secsLeft := ship.timer.TimeLeft().Seconds()

	msg := NewMessage("ship_tick").Add("seconds_left", secsLeft)

	return ship.ForAllShipPlayers(func(p *Player) error {
		return p.Client.Send(msg)
	})
}

// cachedGameDuration is used by gameDuration to store the game duration once it has been determined
// for the first time.
var cachedGameDuration time.Duration = 0

// gameDuration returns the duration that should be used for the ship stage. It first checks the
// command-line arguments for a duration override, but returns fullGameDuration if there is no
// override present.
func gameDuration() time.Duration {
	if cachedGameDuration != time.Duration(0) {
		return cachedGameDuration
	}

	args := os.Args[1:]

	flagNameIndex := slices.Index(args, "--ship-duration-secs")

	if flagNameIndex == -1 {
		// Fall back to the default.
		cachedGameDuration = fullGameDuration

		return fullGameDuration
	}

	if flagNameIndex >= len(args) {
		Logger.Fatal("--ship-duration-secs must be followed by an integer number of seconds")
		panic("unreachable")
	}

	// The number of seconds should come immediately after the flag name.
	secsStr := args[flagNameIndex+1]

	// Try to parse the number of seconds as a 32-bit decimal integer.
	secs, err := strconv.ParseInt(secsStr, 10, 32)

	if err != nil || secs == 0 {
		Logger.Fatal(
			"--ship-duration-secs was given an invalid number of seconds",
			zap.String("given", secsStr),
		)

		panic("unreachable")
	}

	// Game duration is the given number of seconds.
	cachedGameDuration = time.Duration(secs * int64(time.Second))

	return cachedGameDuration
}

// startTimer begins the ship timer. Once this method has been called, all clients will receive
// regular time updates whenever they are in the ship. If the timer is allowed to run to completion,
// the ship will be put into the endgame state.
func (ship *Ship) startTimer() {
	ship.logger().Info("starting core timer")

	ship.timer = TickingTimer(
		ship.Scheduler,

		time.Now().Add(gameDuration()),
		shipTickInterval,

		// Call tick on every interval.
		ship.tick,

		// If the timer runs to completion...
		func() error {
			return ship.triggerEndgame(nil)
		},
	)
}

// Start initialises the ship environment and reports the configuration to the clients.
func (ship *Ship) Start() error {
	ship.logger().Info("entering ship stage")

	ship.createInitialScores()
	ship.createFlags()
	ship.setInitialPositions()

	// Sending all the welcome messages could take a long time,
	// so we do it before starting the core timer.
	welcomeErr := ship.welcomeAll()

	ship.startTimer()
	record = slices.Contains(os.Args, "--record")
	if record {
		ship.logger().Info("recording ship data")
		ship.Recorder = NewRecorder(ship)
	}
	return welcomeErr
}

// welcomePlayer sends a player an initialisation message informing their client of the ship
// layout and content.
func (ship *Ship) welcomePlayer(p *Player) error {
	ship.logger().Info("welcoming player", zap.String("name", p.Name))

	msg := NewMessage("ship_welcome")

	_ = msg.Add("game_duration", gameDuration().Seconds())

	_ = msg.Add("your_spawn", map[string]float64{
		"x": ship.pm.Map[p].X,
		"y": ship.pm.Map[p].Y,
	})

	peerSpawns := make(map[string]map[string]float64)

	_ = p.ForAllShipPeers(func(peer *Player) error {
		peerSpawns[peer.Name] = map[string]float64{
			"x": ship.pm.Map[peer].X,
			"y": ship.pm.Map[peer].Y,
		}

		return nil
	})

	_ = msg.Add("peer_spawns", peerSpawns)

	flagInfo := make(map[string]map[string]interface{})

	for id, f := range ship.fm.flags {
		flagInfo[id] = map[string]interface{}{
			"pos": map[string]float64{
				"x": f.pos.X,
				"y": f.pos.Y,
			},
			"minigame":     f.minigameProto.Name,
			"worth":        f.minigameProto.Worth,
			"player_count": f.minigameProto.PlayerCount,
			"cooldown":     f.minigameProto.Cooldown.Seconds(),
		}
	}

	_ = msg.Add("flags", flagInfo)

	return p.Client.Send(msg)
}

// welcomePlayerBack sends a player a message bringing them up-to-date on the current ship
// environment. This should be sent to players when they come back to the ship after finishing a
// minigame because while in a minigame players do not receive updates about the ship.
func (ship *Ship) welcomePlayerBack(p *Player) error {
	ship.logger().Info("welcoming player back", zap.String("name", p.Name))

	msg := NewMessage("ship_welcome_back")

	// Players don't receive ship time updates while not in the ship, so we need to tell them how
	// long is left on the ship timer.
	secsLeft := ship.timer.TimeLeft().Seconds()

	_ = msg.Add("seconds_left", secsLeft)

	spawnObject := map[string]float64{
		"x": ship.pm.Map[p].X,
		"y": ship.pm.Map[p].Y,
	}

	_ = msg.Add("your_spawn", spawnObject)

	peerPositions := make(map[string]map[string]float64)

	_ = p.ForAllShipPeers(func(peer *Player) error {
		peerPositions[peer.Name] = map[string]float64{
			"x": ship.pm.Map[peer].X,
			"y": ship.pm.Map[peer].Y,
		}

		return nil
	})

	_ = msg.Add("peer_positions", peerPositions)

	flagStates := make(map[string]map[string]interface{})

	for id, f := range ship.fm.flags {
		info := make(map[string]interface{})

		if f.owner != nil {
			info["capture_team"] = f.owner.Index()
		}

		if !f.cooldown.HasEnded() {
			info["cooldown_left"] = f.cooldown.TimeLeft().Seconds()
		}

		if f.isActivated() {
			lockedNames := make([]string, 0)

			for locked := range f.activation.lockedPlayers {
				lockedNames = append(lockedNames, locked.Name)
			}

			info["locked_players"] = lockedNames
		}

		if f.minigame != nil {
			participantNames := make([]string, 0)

			_ = f.minigame.ForAllPlayers(func(participant *Player) error {
				participantNames = append(participantNames, participant.Name)
				return nil
			})

			info["ongoing_players"] = participantNames
		}

		if len(info) == 0 {
			// No interesting information, so don't include this flag.
			continue
		}

		flagStates[id] = info
	}

	_ = msg.Add("flag_states", flagStates)

	selfErr := p.Client.Send(msg)

	otherMsg := NewMessage("ship_welcome_back_peer")
	_ = otherMsg.Add("their_name", p.Name)
	_ = otherMsg.Add("spawn", spawnObject)

	// Notify players who are in the ship.
	peerErr := p.ForAllShipPeers(func(peer *Player) error {
		return peer.Client.Send(otherMsg)
	})

	return errors.Join(selfErr, peerErr)
}

// addPlayerToFlag locks a player to a flag. It panics if the flag is not active.
// The player and their peers will receive a notification of the new lock.
func (ship *Ship) addPlayerToFlag(f *flag, p *Player) error {
	if !f.isActivated() {
		Logger.Panic(
			"cannot add player to inactive flag",
			zap.Any("flag", f),
			zap.Any("player", p),
		)
	}

	// Add to the set.
	f.activation.lockedPlayers[p] = struct{}{}

	id := ship.fm.idForFlag(f)

	ship.logger().Info(
		"locked player to flag",
		zap.String("flag", id),
		zap.String("player", p.Name),
	)

	selfMsg := NewMessage("ship_player_lock_set")
	_ = selfMsg.Add("flag_id", id)

	selfErr := p.Client.Send(selfMsg)

	peerMsg := NewMessage("ship_peer_lock_set")
	_ = peerMsg.Add("their_name", p.Name)
	_ = peerMsg.Add("flag_id", id)

	peerErr := p.ForAllShipPeers(func(peer *Player) error {
		return peer.Client.Send(peerMsg)
	})

	return errors.Join(selfErr, peerErr)
}

// findEligiblePlayersForFlags returns a map where the keys are flags and the values are slices
// containing pointers to all of the players in the ship which are eligible to be locked to that
// flag. When a player is eligible for multiple flags,
// they will be placed under the entry for the earliest-activated flag.
func (ship *Ship) findEligiblePlayersForFlags() map[*flag][]*Player {
	flagsAndEligiblePlayers := make(map[*flag][]*Player)

	_ = ship.ForAllShipPlayers(func(p *Player) error {
		if ship.fm.flagForPlayer(p) != nil {
			// Player is already locked to a flag.
			return nil
		}

		// Player is not locked to a flag. Find the one for which they are needed.
		f := ship.fm.oldestFlagNeedingPlayer(p)

		if f == nil {
			// Player is not needed for any flags.
			return nil
		}

		// Add this player to the slice of players eligible for the flag.
		if _, ok := flagsAndEligiblePlayers[f]; ok {
			flagsAndEligiblePlayers[f] = append(flagsAndEligiblePlayers[f], p)
		} else {
			flagsAndEligiblePlayers[f] = []*Player{p}
		}

		return nil
	})

	return flagsAndEligiblePlayers
}

// addPlayersToFlags locks as many players as possible to flags.
// It then starts minigames for flags that have become ready.
func (ship *Ship) addPlayersToFlags() error {
	if ship.isEndgame {
		// No new minigames allowed.
		return nil
	}

	ship.logger().Info("adding eligible players to flags")

	flagsAndEligiblePlayers := ship.findEligiblePlayersForFlags()

	errs := make([]error, 0)

	for f, players := range flagsAndEligiblePlayers {
		// Keep adding players until no more are needed.
		for len(players) > 0 {
			// Select a random player to add.
			p := players[rand.Int()%len(players)]

			// Add the player to the flag.
			errs = append(errs, ship.addPlayerToFlag(f, p))

			// Remove the player we just added to the flag along with any other players that we
			// do not need anymore.
			players = slices.DeleteFunc(players, func(pl *Player) bool {
				return pl == p || !f.needsPlayer(pl)
			})
		}

		// Now that we've added as many players as we can,
		// check if the flag has enough players to start the minigame.
		if f.hasAllPlayers() {
			ship.logger().Info("flag now has all of its players")

			// Start the minigame.
			errs = append(errs, ship.startMinigameForFlag(f))
		}
	}

	return errors.Join(errs...)
}

// notifyMinigameStart tells players that the minigame for the given flag is starting.
func (ship *Ship) notifyMinigameStart(f *flag) error {
	flagID := ship.fm.idForFlag(f)

	ship.logger().Info("notifying players of minigame start", zap.String("flag", flagID))

	joinErrs := make([]error, 0)

	// Notify the players that will be taking part in the minigame that they are joining it.
	for p := range f.activation.lockedPlayers {
		peerNames := make([]string, 0)

		// Build a list of peer names for the minigame.
		for peer := range f.activation.lockedPlayers {
			if peer == p {
				continue
			}

			peerNames = append(peerNames, peer.Name)
		}

		msgForPlayer := NewMessage("ship_minigame_join")
		_ = msgForPlayer.Add("flag_id", flagID)
		_ = msgForPlayer.Add("peers", peerNames)

		joinErrs = append(joinErrs, p.Client.Send(msgForPlayer))
	}

	// Notify the players that will be remaining in the ship.
	shipErr := ship.ForAllShipPlayers(func(shipPlayer *Player) error {
		if _, inMinigame := f.activation.lockedPlayers[shipPlayer]; inMinigame {
			// This player is going into the minigame.
			return nil
		}

		errs := make([]error, 0)

		// Notify the ship player of every player who is going into the minigame.
		for minigamePlayer := range f.activation.lockedPlayers {
			msg := NewMessage("ship_peer_minigame_join")
			_ = msg.Add("flag_id", flagID)
			_ = msg.Add("their_name", minigamePlayer.Name)

			errs = append(errs, shipPlayer.Client.Send(msg))
		}

		return errors.Join(errs...)
	})

	return errors.Join(errors.Join(joinErrs...), shipErr)
}

// startMinigameForFlag initialises and begins the minigame for the given flag.
func (ship *Ship) startMinigameForFlag(f *flag) error {
	// Notify everyone that the minigame is starting.
	notifyErr := ship.notifyMinigameStart(f)

	// Initialise the minigame.
	f.minigame = f.minigameProto.Constructor(&f.minigameProto, f.store, ship)

	for p := range f.activation.lockedPlayers {
		p.Activity = f.minigame
	}

	// Clear the flag's activation state.
	f.activation = nil

	// Start the minigame.
	startErr := f.minigame.Start()

	return errors.Join(notifyErr, startErr)
}

// sendCooldownTick sends a cooldown tick message for the flag with the given ID to the given
// player.
func (ship *Ship) sendCooldownTick(p *Player, id string, left float64) error {
	msg := NewMessage("ship_flag_cooldown_tick")
	_ = msg.Add("flag_id", id)
	_ = msg.Add("time_left", left)

	return p.Client.Send(msg)
}

// notifyCooldownTick reports the remaining cooldown time for the given flag to all players in
// the ship.
func (ship *Ship) notifyCooldownTick(flag *flag) error {
	return ship.ForAllShipPlayers(func(p *Player) error {
		// We recalculate the remaining time for each message we have to send in case it takes
		// a long time to send any of the messages.
		secsLeft := flag.cooldown.TimeLeft().Seconds()

		return ship.sendCooldownTick(p, ship.fm.idForFlag(flag), secsLeft)
	})
}

// notifyCooldownEnd tells all players in the ship that the remaining time for the given flag's
// cooldown is zero.
func (ship *Ship) notifyCooldownEnd(flag *flag) error {
	return ship.ForAllShipPlayers(func(p *Player) error {
		return ship.sendCooldownTick(p, ship.fm.idForFlag(flag), 0)
	})
}

// startCooldown begins the cooldown period for the given flag.
func (ship *Ship) startCooldown(flag *flag) {
	ship.logger().Info(
		"starting cooldown for flag",
		zap.String("flag", ship.fm.idForFlag(flag)),
	)

	flag.cooldown = TickingTimer(
		ship.Scheduler,

		time.Now().Add(flag.minigameProto.Cooldown),
		cooldownTickInterval,

		func() error {
			return ship.notifyCooldownTick(flag)
		},

		func() error {
			return ship.notifyCooldownEnd(flag)
		},
	)
}

// notifyMinigameResult reports the given result for the minigame attached to the flag with the
// given ID to all players in the ship.
func (ship *Ship) notifyMinigameResult(flagID string, result MinigameResult) error {
	msg := NewMessage("ship_minigame_finished")
	_ = msg.Add("flag_id", flagID)

	if result.winningTeam != nil {
		_ = msg.Add("winning_team", result.winningTeam.Index())
	}

	return ship.ForAllShipPlayers(func(p *Player) error {
		return p.Client.Send(msg)
	})
}

// notifyPeerLeft sends a message to all players in the ship reporting that the peer with the
// given name has disconnected.
func (ship *Ship) notifyPeerLeft(name string) error {
	msg := NewMessage("ship_peer_left").Add("their_name", name)

	return ship.lobby.ForAllPlayers(func(p *Player) error {
		return p.Client.Send(msg)
	})
}

// namedIndividualScores returns a map which maps players' names to their individual scores.
func (ship *Ship) namedIndividualScores() map[string]float64 {
	m := make(map[string]float64)

	for p, s := range ship.individualScores {
		m[p.Name] = s
	}

	return m
}

// end notifies all players that the core has ended and puts them all back into a lobby activity.
func (ship *Ship) end() error {
	ship.logger().Info("ending ship stage")
	if ship.Recorder != nil { // Is recorder a not a nil pointer?
		ship.Recorder.Timer.Stop() // Stop the timer, and write to db.
		ship.Recorder.WriteToDB()
	}
	if ship.lobby.PlayerCount() == 0 {
		// Nothing to do.
		return nil
	}

	// Create the message that we'll use to tell the players that the core has ended.
	endMsg := NewMessage("ship_game_end")

	// Add the individual scores so that the frontend can rank players by their individual
	// performances.
	_ = endMsg.Add("individual_scores", ship.namedIndividualScores())

	// Add the team scores so that the frontend can display those and the overall winner.
	{
		s0, s1 := ship.fm.teamScores()

		// Add as an array.
		_ = endMsg.Add("team_scores", [2]int{s0, s1})
	}

	// Get the activity for the lobby that our players are in.
	lobbyAct := ship.lobby.manager.GetActivity(ship.lobby.ID)

	if lobbyAct == nil {
		// We know there are players in the lobby, so why is there no activity?
		Logger.Panic(
			"missing activity for non-empty lobby",
			zap.String("lobby", ship.lobby.ID),
		)
	}

	return ship.lobby.ForAllPlayers(func(p *Player) error {
		// Move all the players back to the lobby activity. Any further messages will be handled by
		// the lobby activity, not this ship activity.
		p.Activity = lobbyAct

		return p.Client.Send(endMsg)
	})
}

// tryEnd ends the core if and only if there are no ongoing minigames. Otherwise, it does nothing.
func (ship *Ship) tryEnd() error {
	ship.logger().Info("attempting to end ship stage")

	if ship.fm.areAnyMinigamesOngoing() {
		ship.logger().Info("can't end ship stage; minigames are ongoing")

		// The core can't end until all minigames have finished.
		return nil
	}

	// Sanity check.
	_ = ship.lobby.ForAllPlayers(func(p *Player) error {
		if p.Activity != ship {
			Logger.Panic(
				"found a player outside of the ship, but there are no minigames ongoing",
				zap.Any("player", p),
			)
		}

		return nil
	})

	return ship.end()
}

// enterEndgame configures the ship for the endgame state, notifies the players of the new state,
// and then tries to end the core.
func (ship *Ship) enterEndgame() error {
	ship.logger().Info("entering endgame")

	ship.isEndgame = true

	// Make sure the timer is stopped so we don't deliver any more tick messages.
	ship.timer.Stop()

	// Clear player locks and stop cooldown tickers.
	ship.fm.clearForEndgame()

	// Notify all players. (Not just the players in the ship.)
	msg := NewMessage("ship_endgame")

	notifyErr := ship.lobby.ForAllPlayers(func(p *Player) error {
		return p.Client.Send(msg)
	})

	// Try to actually end the core.
	endErr := ship.tryEnd()

	return errors.Join(notifyErr, endErr)
}

// tryEnterEndgame puts the ship into the endgame state if it is not already in it. Regardless of
// the initial state, this method will also try to end the core immediately.
func (ship *Ship) tryEnterEndgame() error {
	// If we're already in the endgame state, all we can do is keep trying to end the core ASAP.
	if ship.isEndgame {
		return ship.tryEnd()
	}

	return ship.enterEndgame()
}

// triggerEndgame should be called whenever a core-ending event occurs. A "core-ending event" can be
// either the disconnection of a player or the expiry of the main core timer.
//
// If the event was a player disconnecting, nameOfLeaver should be a pointer to the name of that
// player. Otherwise, nameOfLeaver should be nil.
func (ship *Ship) triggerEndgame(nameOfLeaver *string) error {
	var leaveErr error

	if nameOfLeaver != nil {
		// Tell the remaining player(s) that a peer has left.
		leaveErr = ship.notifyPeerLeft(*nameOfLeaver)
	}

	// Enter the endgame state if needed.
	endgameErr := ship.tryEnterEndgame()

	return errors.Join(leaveErr, endgameErr)
}

// removePlayer removes the given player from the ship and the lobby. Other players will be notified
// and the ship will enter the endgame state.
//
// This will panic if player.Activity is not the ship activity. This is intended to force the caller
// to consider any cleanup that may be required if the player is in some non-ship activity.
func (ship *Ship) removePlayer(player *Player) error {
	ship.logger().Info("removing player", zap.String("name", player.Name))

	if player.Activity != ship {
		Logger.Panic(
			"players can only be removed from the ship activity",
			zap.Any("player", player),
		)
	}

	// Delete the player's position and score.
	delete(ship.pm.Map, player)
	delete(ship.individualScores, player)

	// Whatever the player was doing before, they can't do it when they're not in the ship
	// anymore...
	player.Activity = nil

	// Remove from the lobby. Among other things, this removes the link between the Player and
	// Client objects, making the Player object practically useless.
	ship.lobby.RemovePlayer(player)

	// When a player disconnects, we should move into the endgame state.
	return ship.triggerEndgame(&player.Name)
}

// spreadPlayers places the given players evenly along an arc such that they are all `dist` away
// from `around`.
func (ship *Ship) spreadPlayers(around Position, dist float64, players []*Player) {
	if len(players) == 0 {
		// Nothing to do.
		return
	}

	// If we only have one player, just put them in the middle.
	if len(players) == 1 {
		ship.pm.Map[players[0]] = Position{
			X: around.X,
			Y: around.Y + dist,
		}

		return
	}

	startAngle := 0.55 * math.Pi
	endAngle := (2 * math.Pi) - startAngle

	offset := Position{
		X: 0,
		Y: -dist,
	}

	angleSpacing := (endAngle - startAngle) / float64(len(players)-1)

	for i, p := range players {
		angle := startAngle + float64(i)*angleSpacing
		rotOffset := offset.RotateAboutOrigin(angle)

		ship.pm.Map[p] = Position{
			X: around.X + rotOffset.X,
			Y: around.Y + rotOffset.Y,
		}
	}
}

// endMinigame reports the result for the given minigame activity, moves the participants back
// into the ship and starts the cooldown.
func (ship *Ship) endMinigame(ctx *MinigameContext, result MinigameResult) error {
	flag := ship.fm.flagForMinigame(ctx)
	flagID := ship.fm.idForFlag(flag)

	ship.logger().Info(
		"minigame ended",
		zap.String("flag", flagID),
		zap.String("core", flag.minigameProto.Name),
		zap.Any("result", result),
	)

	if !ship.isEndgame {
		// Start the cooldown immediately.
		ship.startCooldown(flag)
	}

	// Retain the winning team as the flag owner. If no winning team is given, the previous flag
	// owner stays.
	if result.winningTeam != nil {
		flag.owner = result.winningTeam
	}

	// Announce the result to the players who are in the ship.
	resultErr := ship.notifyMinigameResult(flagID, result)

	flag.minigame = nil

	var connectedParticipants []*Player

	// Find the fraction of the minigame's worth that gets added to each winning player's individual
	// score.
	individualWorth := flag.minigameProto.IndividualWorth()

	_ = ctx.ForAllPlayers(func(p *Player) error {
		if p == result.disconnected {
			// Player is no longer connected.
			return nil
		}

		connectedParticipants = append(connectedParticipants, p)

		if p.Team == result.winningTeam {
			// While we're here, add this player's share of the minigame's token worth to their
			// individual score.
			ship.individualScores[p] += individualWorth
		}

		return nil
	})

	// Shuffle so that positions are allocated randomly.
	rand.Shuffle(len(connectedParticipants), func(i, j int) {
		connectedParticipants[i],
			connectedParticipants[j] =
			connectedParticipants[j],
			connectedParticipants[i]
	})

	ship.spreadPlayers(flag.pos, 20, connectedParticipants)

	var wbErr error

	// Welcome all connected participants back to the ship. The welcome back message also informs
	// them of the result of the minigame.
	for _, p := range connectedParticipants {
		p.Activity = ship

		wbErr = errors.Join(wbErr, ship.welcomePlayerBack(p))
	}

	// Make sure the minigame context can no longer be used.
	ctx.invalidate()

	// Try to end the ship stage if we're past the end of the game.
	var endErr error

	if ship.timer.HasEnded() {
		endErr = ship.triggerEndgame(nil)
	}

	if result.disconnected != nil {
		// Move the player back into the ship so that removePlayer is satisfied that any appropriate
		// cleanup has been performed.
		result.disconnected.Activity = ship

		// The minigame ended because a player disconnected. Remove that player from the ship.
		return errors.Join(resultErr, wbErr, endErr, ship.removePlayer(result.disconnected))
	}

	// Try to start other waiting minigames now that we have more players free.
	gameErr := ship.addPlayersToFlags()

	return errors.Join(resultErr, wbErr, endErr, gameErr)
}

// findNearestFlag returns a pointer to the flag closest to the given player.
// Only flags within reach are considered; this method will return nil if there are no flags in
// reach.
func (ship *Ship) findNearestFlag(player *Player) *flag {
	ship.logger().Info("finding flag nearest to player", zap.String("name", player.Name))

	var nearest *flag = nil
	var nearestDistSq float64

	// flagReachSq is the square of playerFlagReach.
	// We use this to avoid the need for square-rooting during distance comparisons.
	const flagReachSq = playerFlagReach * playerFlagReach

	for _, flag := range ship.fm.flags {
		distSq := ship.pm.Map[player].DistSq(flag.pos)

		if distSq > flagReachSq {
			// Can't reach this flag.
			continue
		}

		// If we haven't found a flag yet, this flag automatically becomes the closest so far.
		// Otherwise, it's only the closest if it's actually closer.
		if nearest == nil || distSq < nearestDistSq {
			nearestDistSq = distSq
			nearest = flag
		}
	}

	return nearest
}

// activateFlag activates the given flag on behalf of a player.
// If the flag is for a single-player minigame, the core will start immediately. Otherwise,
// the player will become locked to the flag and the core will only start once enough players
// become available.
func (ship *Ship) activateFlag(player *Player, flag *flag) error {
	ship.logger().Info(
		"activating flag",
		zap.String("flag", ship.fm.idForFlag(flag)),
		zap.String("player", player.Name),
	)

	if flag.minigameProto.PlayerCount == 1 {
		// Single-player core. Lock the player silently.
		flag.activation = &activation{
			startTime:     time.Now(),
			lockedPlayers: map[*Player]struct{}{player: {}},
		}

		return ship.startMinigameForFlag(flag)
	}

	// Multiplayer core.
	flag.activation = &activation{
		startTime:     time.Now(),
		lockedPlayers: make(map[*Player]struct{}),
	}

	// Lock and notify.
	lockErr := ship.addPlayerToFlag(flag, player)

	// Add as many other players as possible and maybe start the core.
	addErr := ship.addPlayersToFlags()

	return errors.Join(lockErr, addErr)
}

// tryActivateFlag handles a flag activation request for the given player.
// The request may be rejected.
func (ship *Ship) tryActivateFlag(player *Player) error {
	if ship.isEndgame {
		return player.Client.Send(NewMessage("ship_no_flags_in_endgame"))
	}

	flag := ship.findNearestFlag(player)

	if flag == nil {
		return player.Client.Send(NewMessage("ship_no_flags_in_reach"))
	}

	if ship.fm.flagForPlayer(player) != nil {
		return player.Client.Send(NewMessage("ship_player_locked"))
	}

	id := ship.fm.idForFlag(flag)

	if flag.owner == player.Team {
		return player.Client.Send(NewMessage("ship_flag_already_captured").Add("flag_id", id))
	}

	if flag.minigame != nil {
		return player.Client.Send(NewMessage("ship_flag_in_use").Add("flag_id", id))
	}

	if !flag.cooldown.HasEnded() {
		return player.Client.Send(NewMessage("ship_flag_cooling_down").Add("flag_id", id))
	}

	if flag.isActivated() {
		return player.Client.Send(NewMessage("ship_flag_already_activated").Add("flag_id", id))
	}

	return ship.activateFlag(player, flag)
}

func (ship *Ship) HandleMessage(player *Player, message *Message) error {
	if message.Type == "ship_flag_activate" {
		return ship.tryActivateFlag(player)
	}

	if message.Type == "lobby_bye" {
		return ship.removePlayer(player)
	}

	handled, err := ship.pm.HandleMessage(player, message)

	if handled {
		return err
	}

	return player.Client.Send(NewMessage("ship_unknown_message_type_error"))
}

// ForAllShipPlayers calls fn for all players who are currently in the ship environment.
func (ship *Ship) ForAllShipPlayers(fn func(*Player) error) error {
	return ship.lobby.ForAllPlayers(func(p *Player) error {
		if p.Activity != ship {
			return nil
		}

		return fn(p)
	})
}
