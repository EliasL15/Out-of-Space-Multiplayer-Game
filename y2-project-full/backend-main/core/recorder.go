package core

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"go.uber.org/zap"
	"strconv"
	"time"
)

// recordInterval determines interval of time between recording data.
const recordInterval = 200 * time.Millisecond

// A Recorder is responsible for processing and storing a variety of information about a currently running game.
type Recorder struct {
	Data       RecordedData
	ShipTarget *Ship
	Timer      FunctionTimer
}

// The RecordedData structure is used to determine the types of data to export to the database after logging stops.
type RecordedData struct {
	Heatmap   []PlayerHeatmap
	Minigames []MinigameSession
	GameEnd   GameEnd
}

// A MinigameSession is one minigame which has be played by a player/multiple players.
type MinigameSession struct {
	Name             string // Name of minigame.
	Duration         float64
	SessionTimestamp time.Time
	PlayerResults    []PlayerResult
}
type PlayerResult struct {
	PlayerName string
	Team       uint8
	Score      float64
	Won        uint8
}

// PlayerHeatmap is a heatmap for a specific player on a team.
type PlayerHeatmap struct {
	Username string
	Team     uint8
	Heatmap  Heatmap
}

// Heatmap contains the (x,y) co-ordinates at a specific time, so it can be visualised.
type Heatmap struct {
	X, Y float64
	T    float64
}

// GameEnd stores the scores of players and MVPs after a game has ended.
type GameEnd struct {
	TimeTaken float64
	Scores    map[*Player]int
}

// NewRecorder returns a new Recorder obj and starts the timer.
func NewRecorder(shipTarget *Ship) *Recorder {

	return &Recorder{
		Data:       RecordedData{},
		ShipTarget: shipTarget,
		Timer: TickingTimer(shipTarget.Scheduler,
			time.Now().Add(fullGameDuration),
			recordInterval, func() error { return Tick(shipTarget) },
			func() error { return Tick(shipTarget) }),
	}
}

// Tick gets the positions of all players at regular intervals.
func Tick(ship *Ship) error {
	if ship == nil { // Don't record if ship doesn't exist anymore.
		return nil
	}
	for p, data := range ship.pm.Map {
		if p.InShipActivity() {
			// Only record if the player is in the ship.
			ship.Recorder.HeatMapRecord(ship, data, p)
		}
	}
	return nil
}

// HeatMapRecord takes in the positional data of a Player on a Ship and records the time, username and team.
func (r *Recorder) HeatMapRecord(s *Ship, data Position, p *Player) {
	//fmt.Println(r.Data)
	ph := PlayerHeatmap{
		p.Name,
		p.Team.Index(),
		Heatmap{data.X, data.Y,
			fullGameDuration.Seconds() - s.timer.TimeLeft().Seconds()},
	}
	r.Data.Heatmap = append(r.Data.Heatmap, ph)
}

// WriteToDB gets the Data field from Recorder and writes that data to the MySQL database.
func (r *Recorder) WriteToDB() {
	Logger.Info("writing lobby data to database...")
	cfg := mysql.Config{
		User:                 "david",
		Passwd:               "TUvBShyqkz.q9eDs",
		Net:                  "tcp",
		Addr:                 "127.0.0.1:3306",
		DBName:               "main-db",
		AllowNativePasswords: true, // Allows blank db password.
	}
	// Get a database handle and error handle.
	db, err := sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		panic(err)
	}
	timestamp := time.Now()
	// END OF GAME DATA
	// Adding up the scores.
	if r.ShipTarget == nil {
		Logger.Error("Pointer to ship doesn't exist. Data will not be inserted into the database.")
		return
	}
	teamScores := [2]float64{0, 0}
	type MVP struct {
		Player *Player
		Score  float64
	}
	mvp := MVP{nil, 0}
	// Determine pointer and score of the MVP.
	_ = r.ShipTarget.lobby.ForAllPlayers(func(p *Player) error {
		score := r.ShipTarget.individualScores[p]
		if score > mvp.Score {
			mvp.Player = p
			mvp.Score = score
		}
		teamScores[p.Team.Index()] += score
		return nil
	})
	if mvp.Player == nil {
		Logger.Error("Pointer to MVP doesn't exist. Data will not be inserted into the database.")
		return // mvp pointer doesn't exist, cannot get the MVP name.
	}
	glQuery := "INSERT INTO `gameLobbies` VALUES (default, ?, ?, ?, ?, ?, ?);"
	res, err := db.Exec(glQuery, timestamp, r.ShipTarget.lobby.ID,
		teamScores[0], teamScores[1], mvp.Score, mvp.Player.Name)
	if err != nil {
		Logger.Panic("Failed to insert into gameLobbies", zap.Error(err))
	}
	glPK, err := res.LastInsertId() // Get gameLobbies pk of the last inserted record as it a foreign key of heatMaps
	if err != nil {
		Logger.Panic("Failed to insert to get last insert id", zap.Error(err))
	}
	hmQuery := "INSERT INTO `heatMaps` VALUES (?, ?, ?, ?)"
	_, err = db.ExecContext(context.Background(), hmQuery, glPK, timestamp, r.ShipTarget.lobby.ID,
		r.Data.PlayerHeatmapToCSV())
	if err != nil {
		Logger.Panic("Failed to insert into heatmaps", zap.Error(err))
	}
	// INSERTING INDIVIDUAL MINIGAME DATA
	for _, mSession := range r.Data.Minigames {
		// Insert the data related to a single session.
		msQuery := "INSERT INTO `minigameSessions` VALUES (default,?, ?, ?, ?)"
		res, err = db.Exec(msQuery, mSession.Name, glPK, mSession.Duration,
			mSession.SessionTimestamp)
		if err != nil {
			Logger.Panic("Failed to insert into minigameSessions", zap.Error(err))
		}
		msPK, err := res.LastInsertId() // Get minigameSessions primary key, used as a foreign key of minigamePlayers.
		if err != nil {
			Logger.Panic("Failed to get last insert id", zap.Error(err))
		}
		// Now for all players in that session insert the score data of specific player(s)
		for _, p := range mSession.PlayerResults {
			mpQuery := "INSERT INTO `minigamePlayers` VALUES (? ,?, ?, ?, ?, ?)"
			_, err = db.ExecContext(context.Background(), mpQuery, msPK,
				p.PlayerName, mSession.SessionTimestamp, p.Team, p.Score, p.Won)
			if err != nil {
				Logger.Panic("Failed to insert into minigamePlayers", zap.Error(err))
			}
		}
	}
}

// ResultsToPlayerResults converts a map of player scores/player wins and a gameName and adds it to PlayerResult
func ResultsToPlayerResults(ps map[*Player]float64, pw map[*Player]uint8, gameName string) []PlayerResult {
	var pr []PlayerResult
	for p := range ps {
		pr = append(pr, PlayerResult{p.Name, p.Team.Index(), ps[p],
			pw[p]})
		Logger.Info(fmt.Sprintf("%s on team %d %s %s with score %.0f",
			p.Name, p.Team.Index(),
			func() string {
				if pw[p] == 1 {
					return "won"
				} else {
					return "lost"
				}
			}(), gameName, ps[p]))
	}
	return pr
}

/*
	func (r *Recorder) ClickRaceRecordMP(ps map[*Player]float64, pw map[*Player]uint8) {
		pr := ResultsToPlayerResults(ps, pw, "cps_race_1v1")
		ms := MinigameSession{"cps_race_1v1", 10, time.Now(), pr}
		r.Data.Minigames = append(r.Data.Minigames, ms)
	}

	func (r *Recorder) ClickRaceRecordSP(p *Player, score float64, won uint8) {
		pr := ResultsToPlayerResults(map[*Player]float64{p: score}, map[*Player]uint8{p: won}, "cps_race_sp")
		ms := MinigameSession{"cps_race_sp", 10, time.Now(), pr}
		r.Data.Minigames = append(r.Data.Minigames, ms)
	}

	func (r *Recorder) RPSRecordMP(ps map[*Player]float64, pw map[*Player]uint8, duration float64) {
		pr := ResultsToPlayerResults(ps, pw, "rps_1v1")
		ms := MinigameSession{"rps_1v1", duration, time.Now(), pr}
		r.Data.Minigames = append(r.Data.Minigames, ms)
	}

	func (r *Recorder) MolesRecordSP(p *Player, score float64, won uint8) {
		pr := ResultsToPlayerResults(map[*Player]float64{p: score}, map[*Player]uint8{p: won}, "whack_a_mole")
		ms := MinigameSession{"whack_a_mole", 30, time.Now(), pr}
		r.Data.Minigames = append(r.Data.Minigames, ms)
	}

	func (r *Recorder) ShooterRecord(ps map[*Player]float64, pw map[*Player]uint8, duration float64, players int) {
		pr := ResultsToPlayerResults(ps, pw, "shooter_1v1")
		ms := MinigameSession{"rps_1v1", duration, time.Now(), pr}
		r.Data.Minigames = append(r.Data.Minigames, ms)
	}

	func (r *Recorder) CardRecordSP(ps map[*Player]float64, pw map[*Player]uint8, duration float64) {
		pr := ResultsToPlayerResults(ps, pw, "card_match_sp")
		ms := MinigameSession{"rps_1v1", duration, time.Now(), pr}
		r.Data.Minigames = append(r.Data.Minigames, ms)
	}

	func (r *Recorder) FBRecordSP(p *Player, score float64, won uint8, duration float64) {
		pr := ResultsToPlayerResults(map[*Player]float64{p: score}, map[*Player]uint8{p: won}, "fb_sp")
		ms := MinigameSession{"fb_sp", duration, time.Now(), pr}
		r.Data.Minigames = append(r.Data.Minigames, ms)
	}

	func (r *Recorder) FBRecordSP(p *Player, score float64, won uint8, duration float64) {
		pr := ResultsToPlayerResults(map[*Player]float64{p: score}, map[*Player]uint8{p: won}, "fb_sp")
		ms := MinigameSession{"fb_sp", duration, time.Now(), pr}
		r.Data.Minigames = append(r.Data.Minigames, ms)
	}
*/
func (r *Recorder) RecordSP(p *Player, score float64, won uint8, duration float64, gameName string) {
	pr := ResultsToPlayerResults(map[*Player]float64{p: score}, map[*Player]uint8{p: won}, "fb_sp")
	ms := MinigameSession{gameName, duration, time.Now(), pr}
	r.Data.Minigames = append(r.Data.Minigames, ms)
}
func (r *Recorder) RecordMP(ps map[*Player]float64, pw map[*Player]uint8, duration float64, gameName string) {
	pr := ResultsToPlayerResults(ps, pw, gameName)
	ms := MinigameSession{gameName, duration, time.Now(), pr}
	r.Data.Minigames = append(r.Data.Minigames, ms)
}

// PlayerHeatmapToCSV converts the PlayerHeatmap structure to CSV to save space on the database.
func (r *RecordedData) PlayerHeatmapToCSV() string {
	// See https://stackoverflow.com/a/75740486, used similar method.
	csvOut := make([][]string, len(r.Heatmap)+1)
	csvOut[0] = []string{"username", "team", "x", "y", "t"}
	for i := range r.Heatmap {
		csvOut[i] = []string{r.Heatmap[i].Username, strconv.Itoa(int(r.Heatmap[i].Team)),
			strconv.FormatFloat(r.Heatmap[i].Heatmap.X, 'f', -1, 64),
			strconv.FormatFloat(r.Heatmap[i].Heatmap.Y, 'f', -1, 64),
			strconv.FormatFloat(r.Heatmap[i].Heatmap.T, 'f', -1, 64)}
	}
	b := new(bytes.Buffer)
	w := csv.NewWriter(b)
	err := w.WriteAll(csvOut)
	if err != nil {
		panic(err)
	}
	csvString := b.String()
	return csvString
}
