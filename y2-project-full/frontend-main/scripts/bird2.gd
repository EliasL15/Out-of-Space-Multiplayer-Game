extends RigidBody2D



signal score_changed(score: int)
signal bird_died

# Called when the node enters the scene tree for the first time.
func _ready():
	pass # Replace with function body.


# Called every frame. 'delta' is the elapsed time since the previous frame.
func _physics_process(delta):
	if delta is InputEventKey and delta.pressed:
		if delta.keycode == KEY_SPACE:
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
	pass


func died():
	bird_died.emit()
