extends RigidBody2D

# Movement parameters
var horizontal_speed = 1000  # Adjust the horizontal speed to your liking
var vertical_speed = 1000  # Adjust the vertical speed to your liking

func _ready():
# Set the linear damping for smoother movement directly
	self.linear_damp = 5.0  # Adjust the damping to make the movement smoother

func _integrate_forces(state):
	var local_velocity = state.get_linear_velocity()
	var applied_force = Vector2.ZERO

  # Handling left and right movement
	if Input.is_action_pressed("ui_right"):
		applied_force.x += horizontal_speed
	if Input.is_action_pressed("ui_left"):
		applied_force.x -= horizontal_speed

	# Handling up and down movement
	if Input.is_action_pressed("ui_down"):
		applied_force.y += vertical_speed
	if Input.is_action_pressed("ui_up"):
		applied_force.y -= vertical_speed

	# Applying the movement force
	local_velocity += applied_force * state.get_step()

	# Prevent rotation on collisions
	local_velocity.x = local_velocity.x  # This sets the x-axis velocity directly

	state.set_linear_velocity(local_velocity)
