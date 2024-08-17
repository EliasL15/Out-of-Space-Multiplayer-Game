extends Node

const INIT_POSITION:int = 1000
@export var pipe: PackedScene
@onready var timer = $Timer

# Called when the node enters the scene tree for the first time.
func _ready():
	timer.timeout.connect(_on_timer_timeout)
	
	pass # Replace with function body.

# Called every frame. 'delta' is the elapsed time since the previous frame.
func _on_timer_timeout():
	timer.start()
	var pipe_instance = pipe.instantiate() as Node2D
	add_child(pipe_instance)
	pipe_instance.position.x = INIT_POSITION 
	pipe_instance.position.y = randf_range(-150, 150)
	pass
