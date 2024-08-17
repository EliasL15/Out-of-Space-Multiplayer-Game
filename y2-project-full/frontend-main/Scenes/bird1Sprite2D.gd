extends RigidBody2D

class_name Bird

signal score_changed(score: int)
signal bird_died

var current_score = 0
var highest_score = 0

# Called when the node enters the scene tree for the first time.
func _ready():
	pass # Replace with function body.


# Called every frame. 'delta' is the elapsed time since the previous frame.
func _physics_process(delta):
	if Input.is_mouse_button_pressed(1):
		linear_velocity = Vector2.UP*500
		angular_velocity = -3.0
	if rotation_degrees < -30:
		rotation_degrees = -30
		angular_velocity = 0
		
	if linear_velocity.y > 0.0:
		angular_velocity = 1.5
		if rotation_degrees > 50:
			rotation_degrees = 50
			angular_velocity = 0

	

func add_score():
	current_score += 1
	score_changed.emit(current_score)
	print(current_score)

func died():
	bird_died.emit()
