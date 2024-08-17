extends RigidBody2D

class_name Bird1
signal bird1_died

func _physics_process(delta):
	if Input.is_action_pressed("fly1"):
		linear_velocity = Vector2.UP*500
		angular_velocity = -3.0
	if rotation_degrees < -30:
		rotation_degrees = -30
		angular_velocity = 0
	if linear_velocity.y > 0.0:
		angular_velocity = 1.5

func bird1died():
	bird1_died.emit()
