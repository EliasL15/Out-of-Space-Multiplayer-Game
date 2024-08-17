extends CharacterBody2D


@export var vitality = 5
var dead = false
# Get the gravity from the project settings to be synced with RigidBody nodes.
var gravity = ProjectSettings.get_setting("physics/2d/default_gravity")
const bullet = preload("res://shooter/scene/bullet.tscn")
var uname
@onready var og_pos :float = global_position.x

#func _ready():

	
func _physics_process(delta):
			if dead == false:
				if og_pos != null:
					if global_position.x > og_pos:
						$Icon.flip_h = false
					elif global_position.x < og_pos:
						$Icon.flip_h = true
					og_pos = global_position.x
				
				if $GunRotation/bulletSpawn.global_position.x >= $Icon.global_position.x:
					$GunRotation/gun.flip_v = false
				else:
					$GunRotation/gun.flip_v = true
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
					#queue_free()



func _ready():
			$player_name.text = uname
			#og_pos = global_position
			$Area2D.body_entered.connect(_on_area_2d_body_entered)
			#for i in get_tree().root.get_node_in_group("player"):
				#if i.EventLib.client_uname != EventLib.client_uname:


#@rpc("any_peer", "call_local")
func update_health():
	
	var health = $ProgressBar
	health.value = vitality

func _on_area_2d_body_entered(body):
	if body.is_in_group("bullet"):
		#vitality -= 1
		body.queue_free()
