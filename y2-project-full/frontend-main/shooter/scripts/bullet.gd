extends CharacterBody2D


const SPEED = 700.0

var direction
signal hit(player)

func _ready():
	direction = Vector2(1,0).rotated(rotation)
	var timer : Timer = $Timer
	velocity = SPEED * direction



func _physics_process(delta):
	
	#if not is_on_floor():
		#velocity.y += gravity * delta
	#move_and_slide()
	var collision_info = move_and_collide(velocity * delta)
	if collision_info:
		var object_hit = collision_info.get_collider()
		if object_hit is CharacterBody2D:
			pass
		else:
			var temp = collision_info.get_remainder().bounce(collision_info.get_normal())
			velocity = velocity.bounce(collision_info.get_normal())
			move_and_collide(temp)
		if object_hit.is_in_group("bullet"):
			queue_free()

func _on_timer_timeout():
	queue_free()


func _on_area_2d_body_entered(body):
	
	if body.is_in_group("player"):
		hit.emit(body.uname)
		#EventLib.message_send({
			#"type": "shooter_bullet_player_hit",
			#"victim": "lol"
			#})
