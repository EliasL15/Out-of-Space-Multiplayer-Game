package core

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

// A Position is a 2D position value.
type Position struct {
	// X is the horizontal component of the position.
	X float64

	// Y is the vertical component of the position.
	Y float64
}

// DistSq returns the squared distance between p and q.
func (p Position) DistSq(q Position) float64 {
	dx := p.X - q.X
	dy := p.Y - q.Y

	return dx*dx + dy*dy
}

// ToMap returns a map with the keys "x" and "y" with the relevant values from p.
func (p Position) ToMap() map[string]float64 {
	return map[string]float64{"x": p.X, "y": p.Y}
}

// PositionFromObj looks for "x" and "y" fields with the float64 type in the given object. It
// returns nil if the object is not a map, either field is missing, or either field has the wrong
// type.
func PositionFromObj(obj interface{}) *Position {
	posMap, ok := obj.(map[string]interface{})

	if !ok {
		return nil
	}

	xInterface, ok := posMap["x"]

	if !ok {
		return nil
	}

	x, ok := xInterface.(float64)

	if !ok {
		return nil
	}

	yInterface, ok := posMap["y"]

	if !ok {
		return nil
	}

	y, ok := yInterface.(float64)

	if !ok {
		return nil
	}

	return &Position{
		X: x,
		Y: y,
	}
}

// Dist returns the distance between p and q.
//
// Square-root operations are expensive,
// so prefer using DistSq where possible to avoid square-rooting values. For example,
// the result of a distance comparison will be the same if squared distances are used instead of
// real distances.
func (p Position) Dist(q Position) float64 {
	return math.Sqrt(p.DistSq(q))
}

// RotateAboutOrigin returns the position that results from rotating p by angle radians
// around (0, 0).
func (p Position) RotateAboutOrigin(angle float64) (r Position) {
	s, c := math.Sincos(angle)

	r.X = p.X*c - p.Y*s
	r.Y = p.X*s + p.Y*c

	return
}

// A PositionManager holds and updates positions for players.
type PositionManager struct {
	// Map maps player pointers to position values.
	Map map[*Player]Position

	// msgPrefix is the prefix that will be taken off message type strings before checking them.
	// For example, the ship activity uses "ship_mov_" here.
	msgPrefix string
}

// NewPositionManager returns an empty position manager that uses the given message type prefix.
func NewPositionManager(prefix string) PositionManager {
	return PositionManager{
		Map:       make(map[*Player]Position),
		msgPrefix: prefix,
	}
}

// SpawnPlayer places the player at pos and notifies them and their activity peers.
func (pm *PositionManager) SpawnPlayer(player *Player, pos Position) error {
	pm.Map[player] = pos

	msg := NewMessage(pm.msgPrefix + "spawn")
	_ = msg.Add("x", pos.X)
	_ = msg.Add("y", pos.Y)

	msgErr := player.Client.Send(msg)

	peerMsg := NewMessage(pm.msgPrefix + "peer_spawn")
	_ = peerMsg.Add("their_name", player.Name)
	_ = peerMsg.Add("x", pos.X)
	_ = peerMsg.Add("y", pos.Y)

	peerErr := player.ForAllActivityPeers(func(peer *Player) error {
		return peer.Client.Send(peerMsg)
	})

	return errors.Join(msgErr, peerErr)
}

// notifyNewPosition notifies other players of a position change for the given player.
// Only players in the same activity are notified.
func (pm *PositionManager) notifyNewPosition(player *Player) error {
	pos := pm.Map[player]

	msg := NewMessage(pm.msgPrefix + "peer_position_update")
	_ = msg.Add("their_name", player.Name)
	_ = msg.Add("x", pos.X)
	_ = msg.Add("y", pos.Y)

	// Send the update to all other players in the activity.
	return player.ForAllActivityPeers(func(peer *Player) error {
		return peer.Client.Send(msg)
	})
}

// doSetPosition updates the position for the given player and notifies their peers.
func (pm *PositionManager) doSetPosition(player *Player, pos Position) error {
	// todo: Validate position change instead of blindly accepting it.
	pm.Map[player] = pos

	return pm.notifyNewPosition(player)
}

// doPositionUpdate handles a position update message.
func (pm *PositionManager) doPositionUpdate(player *Player, message *Message) error {
	var newPos Position

	if x, err := message.GetNumber("x"); err == nil {
		newPos.X = x
	} else {
		return player.Client.Send(NewMessage(pm.msgPrefix + "position_update_bad_x_error"))
	}

	if y, err := message.GetNumber("y"); err == nil {
		newPos.Y = y
	} else {
		return player.Client.Send(NewMessage(pm.msgPrefix + "position_update_bad_y_error"))
	}

	return pm.doSetPosition(player, newPos)
}

// HandleMessage attempts to handle a message for the position manager.
// The first return value is true if and only if the
func (pm *PositionManager) HandleMessage(player *Player, message *Message) (bool, error) {
	typ := strings.TrimPrefix(message.Type, pm.msgPrefix)

	if typ == message.Type {
		// Not an error; this just isn't a position manager message.
		return false, nil
	}

	if typ != "position_update" {
		// The message type has the position manager prefix,
		// so presumably this is meant to be handled by the position manager.
		// We just don't know how to handle it.
		return true, fmt.Errorf("position manager does not recognise '%v'", typ)
	}

	return true, pm.doPositionUpdate(player, message)
}
