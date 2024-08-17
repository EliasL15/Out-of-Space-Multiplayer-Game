extends CharacterBody2D

@export var move_speed : float = 100

@export var starting_direction : Vector2 = Vector2(0,1)
@export var new_position: Vector2
@onready var animation_tree = $AnimationTree
@onready var state_machine = animation_tree.get("parameters/playback")
var time_stationary = 0

func _ready():
	update_animation_parameters(starting_direction)
	EventLib.peer_movement.connect(_update_pos)
# Determine state of remote player.
func _process(delta):
	if new_position == position:
		#add delta to ensure faster frame rates dont always show idle state
		time_stationary += delta
		if(time_stationary>0.05):
			state_machine.travel("idle")
		return
	time_stationary=0
	var moveDir = (new_position - position)
	#print(moveDir)
	
	
	animation_tree.set("parameters/walk/blend_position", moveDir)
	animation_tree.set("parameters/idle/blend_position", moveDir)
	
	state_machine.travel("walk")
	
	position = new_position
	
# Change movement dir on pos update.		
func update_animation_parameters(move_input : Vector2):
	if(move_input != Vector2.ZERO):
		animation_tree.set("parameters/walk/blend_position", move_input)
		animation_tree.set("parameters/idle/blend_position", move_input)
# Called when position update is sent from the server.

func _update_pos(data):
	var client_id = data["their_name"]
	if(client_id == name):
		new_position = Vector2(data["x"],data["y"])
	
