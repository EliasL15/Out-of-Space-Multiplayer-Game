extends Node2D

const SPEED: int = 500
@onready var pipe_area = %pipearea
@onready var score_area = %scorearea
@onready var timer = $Timer

var bird: Bird

# Called when the node enters the scene tree for the first time.
func _ready():
	pipe_area.body_entered.connect(_on_pipe_area_entered)
	score_area.body_exited.connect(_on_score_area_exited)
	timer.timeout.connect(_on_timer_timeout)
	bird = get_tree().get_first_node_in_group("bird")
	pass # Replace with function body.


# Called every frame. 'delta' is the elapsed time since the previous frame.
func _process(delta):
	position.x -= SPEED * delta
	pass

func _on_pipe_area_entered(body: Node2D):
	bird.died()
	print("damn")
	
func _on_score_area_exited(body: Node2D):
	bird.add_score()
	print("yeah")
	
	

func _on_timer_timeout():
	queue_free()
