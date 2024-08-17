## This file implements the `MinigameBase` class, which should be extended by all minigame scripts.

extends Node2D
class_name MinigameBase

## == The following items are minigame-specific. ==

## Called when the server sends a message to the minigame. This method should only return `true` if
## the message was understood and processed.
##
## Minigames **must** override this.
func on_message(_msg: Dictionary) -> bool:
	assert(false, "`on_message` was not overridden")

	# Unreachable
	return false

## Called when the server ends the minigame. This method should be used to perform any necessary
## cleanup before the scene is changed back to the ship.
##
## Minigames **must** override this.
func on_minigame_end():
	assert(false, "`on_minigame_end` was not overridden")

## Called on when the ship stage enters the endgame state. The ship will wait for this minigame to
## finish, so this method should **not** assume that the minigame will end imminently. The primary
## purpose of this method is to allow minigames to notify the player that the ship is in the
## endgame state.
##
## Minigames **must** override this.
func on_ship_endgame():
	assert(false, "`on_ship_endgame` was not overridden")

## Called when a peer disconnects from the server.
##
## Minigames **must** override this.
func on_peer_left(_theirName: String):
	assert(false, "`on_peer_left` was not overridden")

## Called when a player outside the minigame disconnects from the server.
##
## Minigames **must** override this.
func on_non_peer_left(_theirName: String):
	assert(false, "`on_non_peer_left` was not overridden")

## == The following items are universal to all minigames. ==

## (GDScript does not have a native `Set` datatype, so we use `Dictionary` objects with dummy
## values instead.)

## A set containing the names of the other players in the minigame.
var _peer_names: Dictionary

## A set containing the names of peers that started in the minigame but have since disconnected.
var _dead_peer_names: Dictionary

## Tells the minigame which other players are in the minigame.
func init_peer_names(names: Array[String]):
	_peer_names = {}
	_dead_peer_names = {}

	# Underscore is to prevent shadowing.
	for theirName in names:
		# Add to the set.
		_peer_names[theirName] = null

## Returns a set containing the names of the other players currently in the minigame.
func peers() -> Dictionary:
	return _peer_names

## Returns a set containing the names of all of the players who started in the minigame but have
## since disconnected.
func disconnected_peers() -> Dictionary:
	return _dead_peer_names

## Moves the given peer from the connected set to the disconnected set. Panics if the peer is not
## initially in the connected set.
func _disconnect_peer(theirName: String):
	# Remove from the set of alive peers and add to the set of dead ones.
	assert(_peer_names.erase(theirName), "peer does not exist")
	_dead_peer_names[theirName] = null

## Called when the server sends us a `ship_peer_left` message.
func on_player_left_ship(theirName: String):
	if _peer_names.has(theirName):
		_disconnect_peer(theirName)
		on_peer_left(theirName)

		return

	assert(!_dead_peer_names.has(theirName), "peer disconnected twice")

	# Not a peer.
	on_non_peer_left(theirName)
