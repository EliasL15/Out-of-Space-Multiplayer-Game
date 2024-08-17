extends CharacterBody2D


const SPEED = 300.0
const JUMP_VELOCITY = -600.0
@export var vitality = 5
var dead = false
# Get the gravity from the project settings to be synced with RigidBody nodes.
var gravity = ProjectSettings.get_setting("physics/2d/default_gravity")
const bullet = preload("res://shooter/scene/bullet.tscn")
var uname 
var og_pos
var og_rot
var og_bull

func _physics_process(delta):
			if dead == false:
				if not is_on_floor():
					velocity.y += gravity * delta
				
				$GunRotation.look_at(get_global_mouse_position())
				#update_health.rpc()
				update_health()
				if vitality <= 0:
					#var dead = multiplayer.get_unique_id()
					await get_tree().create_timer(0.01).timeout
					dead = true
					$Icon.visible = false
					$GunRotation/gun.visible = false
					$ProgressBar.visible = false
					$player_name.visible = false
					$CollisionShape2D.disabled = true
					$Area2D/CollisionShape2D.disabled = true
					#GameManager.death.append(dead)
					#queue_free()
				# Handle jump.
				if Input.is_action_just_pressed("up") and is_on_floor():
					velocity.y = JUMP_VELOCITY

				if Input.is_action_just_pressed("shoot"):
					#fire.rpc()
					fire()
				# Get the input direction and handle the movement/deceleration.
				# As good practice, you should replace UI actions with custom gameplay actions.
				var direction = Input.get_axis("left", "right")
				if direction:
					velocity.x = direction * SPEED
				else:
					velocity.x = move_toward(velocity.x, 0, SPEED)
				if Input.is_action_pressed("right"):
					$Icon.flip_h = false
				elif Input.is_action_pressed("left"):
					$Icon.flip_h = true
				
				if get_global_mouse_position().x >= $Icon.global_position.x:
					$GunRotation/gun.flip_v = false
				else:
					$GunRotation/gun.flip_v = true
					
				move_and_slide()
				var i = get_tree().root.get_children()
				#for j in i:
					#if j is CharacterBody2D:
						#print(j.global_position)
				if get_tree().root.get_node("main").end == false:
					var bullet_coor = []
					#print(get_tree().get_nodes_in_group(uname))
					for b in get_tree().get_nodes_in_group(uname.to_lower()):
						bullet_coor.append({
						"x": b.global_position.x,
						"y": b.global_position.y
							})
							
					#print(bullet_coor)
					if global_position != og_pos or $GunRotation.rotation != og_rot or og_bull != get_tree().get_nodes_in_group(uname.to_lower()):
						EventLib.message_send({
							"type": "shooter_physics_report",
							"position":  {
												"x": self.global_position.x,
												"y": self.global_position.y
											  },
							"arm": $GunRotation.rotation,
							"bullets": bullet_coor
							})
						og_pos = global_position
					else:
						##print(get_tree().get_nodes_in_group(uname))
						#for b in get_tree().get_nodes_in_group(uname.to_lower()):
							#bullet_coor.append({
							#"x": b.global_position.x,
							#"y": b.global_position.y
								#})
						EventLib.message_send({
								"type": "shooter_physics_report",
								"position":  {
													"x": "",
													"y": ""
												  },
								"arm": "",
								"bullets": bullet_coor
								})
					og_bull = get_tree().get_nodes_in_group(uname.to_lower())
					
				
					
				
func bullet_hit(victim):
	if get_tree().root.get_node("main").end == false:
		EventLib.message_send({
				"type": "shooter_bullet_player_hit",
				"victim": victim
				})
func _ready():
	og_pos = global_position
	og_rot = $GunRotation.rotation
	og_bull = get_tree().get_nodes_in_group(uname.to_lower())
	$player_name.text = uname
			#for i in get_tree().root.get_children():
				#if i is CharacterBody2D:
					#if i.Eventlib.client_uname != EventLib.client_uname:
						#i.set_script("res://shooter/scripts/not_player.gd")

	
func fire():
	var b = bullet.instantiate()
	b.global_position = $GunRotation/bulletSpawn.global_position
	b.rotation_degrees = $GunRotation.rotation_degrees
	get_tree().root.add_child(b)
	b.add_to_group(uname.to_lower())
	b.connect("hit", bullet_hit)
	
	#add_child(b)

#@rpc("any_peer", "call_local")
func update_health():
	var health = $ProgressBar
	health.value = vitality

func _on_area_2d_body_entered(body):
	if body.is_in_group("bullet"):
		#vitality -= 1
		body.queue_free()
