#this script defines what the banana does when the player interacts with it
extends Interactable

@onready var animation_tree = get_node("../AnimationTree")
@onready var state_machine = animation_tree.get("parameters/playback")
@onready var state = 1
@onready var tilemap = get_node("/root/game_level/TileMap")
@onready var GameLevel = get_node("/root/game_level")

func _ready():
	EventLib.add_portal_tiles.connect(_add_portal_tiles)
	Globalvar.add_portal_tiles_gl.connect(_add_portal_tiles)

func interact(user: Node2D):
	state = state + 1

	EventLib.message_send({
		"type": "ship_flag_activate"
	})

	#_add_portal_tiles("red", Vector2(0,0))

#func stop_interaction(user: Node2D):  --- temporarily not in use as we figure out game logic
	#code for a stop interaction  --- temporarily not in use as we figure out game logic

func _add_portal_tiles(colour, portal):
	#portal.name)
	#print(get_parent().name)
	if (portal.name != get_parent().name):
		return

	var centre = tilemap.local_to_map(tilemap.to_local(portal.position))
	var surrounding_tiles_arr = surrounding_tiles(centre)
	var portal_pos = portal.position
	#testing the state transitions
	if (colour == "purple"):
		state_machine.travel("purple_portal_idle")
		for vector in surrounding_tiles_arr:
			tilemap.set_cell(0, vector, 0, Vector2i(4, 1))
		#state=2
	elif (colour == "red"):
		state_machine.travel("red_portal_idle")
		for vector in surrounding_tiles_arr:
			tilemap.set_cell(0, vector, 0, Vector2i(3, 1))
		#state=3
	else:
		state_machine.travel("blue_portal_idle")
		for vector in surrounding_tiles_arr:
			tilemap.set_cell(0, vector, 0, Vector2i(5, 1))

		#state=1

func surrounding_tiles(centre_coord):
	var surrounding_tiles = []
	surrounding_tiles.append(Vector2(centre_coord[0] + 1, centre_coord[1] - 1))
	surrounding_tiles.append(Vector2(centre_coord[0], centre_coord[1] - 1))
	surrounding_tiles.append(Vector2(centre_coord[0] - 1, centre_coord[1] - 1))
	surrounding_tiles.append(Vector2(centre_coord[0] + 1, centre_coord[1]))
	surrounding_tiles.append(Vector2(centre_coord[0] + 1, centre_coord[1] + 1))
	surrounding_tiles.append(Vector2(centre_coord[0] - 1, centre_coord[1]))
	surrounding_tiles.append(Vector2(centre_coord[0] - 1, centre_coord[1] + 1))
	surrounding_tiles.append(Vector2(centre_coord[0] - 1, centre_coord[1] + 1))
	surrounding_tiles.append(Vector2(centre_coord[0], centre_coord[1] + 1))
	return surrounding_tiles
