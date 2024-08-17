extends CharacterBody2D

@export var move_speed : float = 100

@export var starting_direction : Vector2 = Vector2(0,1)
@export var player_id : String
@onready var animation_tree = $AnimationTree
@onready var state_machine = animation_tree.get("parameters/playback")

func _ready():
	var player_interaction = get_node("./Area2D")
	player_interaction.display_instruction_text.connect(_on_display_instruction_text)
	update_animation_parameters(starting_direction)
	
func _physics_process(_delta):
	#get input direction
	var input_direction =  Vector2(
		Input.get_action_strength("right") - Input.get_action_strength("left"),
		Input.get_action_strength("down") - Input.get_action_strength("up")
	)
	#print("player: input_direction:"+str(input_direction))
	
	update_animation_parameters(input_direction)
	
	#update velocity
	velocity = input_direction * move_speed
	#move and slide function updates character on map
	move_and_slide()
	
	pick_new_state()

func update_animation_parameters(move_input : Vector2):
	if(move_input != Vector2.ZERO):
		animation_tree.set("parameters/walk/blend_position", move_input)
		animation_tree.set("parameters/idle/blend_position", move_input)
	
func pick_new_state():
	if(velocity != Vector2.ZERO):
		state_machine.travel("walk")
		EventLib.update_client_pos(global_position)
	else:
		state_machine.travel("idle")

func _on_display_instruction_text(data):
	var pressFBtn = get_node("pressFBtn")
	if(data==true):
		pressFBtn.visible = true
	else:
		pressFBtn.visible = false
