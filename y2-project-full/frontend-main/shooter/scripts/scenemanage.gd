extends MinigameBase

@export var PlayerScene : PackedScene
var humans = preload("res://shooter/scene/humantscn.tscn")
var peer_script = preload("res://shooter/scripts/not_player.gd")
var bullet = preload("res://shooter/scene/bullet.tscn")
var duration : int
var end = false
var human_team = []
var alien_team = []




func on_minigame_end():
	# In this method you should do anything that you need to do before your minigame ends. For
	# example, if we had something running in the background, we'd want to stop it.
	for i in get_tree().get_nodes_in_group("bullet"):
		i.queue_free()
	print("demo minigame is ending")

func on_peer_left(theirName: String):
	# Most minigames won't need to implement this method beyond `pass`, because the server will
	# automatically end the minigame when one of its participants leaves. (So there won't be much
	# time to tell the player what's happened before they get removed from the minigame anyway.)

	print("player '", theirName, "' (who was in this minigame) has disconnected")

# For the following two methods, you might want to show the user a notification, because the game
# will not necessarily end immediately. (Here we just print, because this is just the demo
# minigame.)

func on_ship_endgame():
	print("ship is entering endgame state")

func on_non_peer_left(theirName: String):
	print("player '", theirName, "' (who was not in this minigame) has disconnected")

## == End required functions ==

func on_message(data: Dictionary) -> bool:
	match data["type"]:

		"shooter_welcome":
			var health = data["health_initial"]
			var local_spawn = data["your_spawn"]
			var peer_spawn = data["peer_spawns"]
			duration = data["duration"]
			$Label.text = str("%02d" %duration)
			print("this is local spawn" + str(local_spawn))
			print("this is peer spawn" + str(peer_spawn))
			
			var spawn = $spawn.get_children()
			var team_humans = []
			var team_aliens = []
			
			var players = []
			for i in peer_spawn.keys():
				players.append(i)
			players.append(EventLib.client_uname)
			for player in EventLib.the_lobby.playerTeam:
				if player[0] not in players:
					pass
				else: 
					if player[1] == 1:
						team_humans.append(player)
					else:
						team_aliens.append(player)
			team_humans.sort_custom(func(a, b): return a[0] < b[0])
			team_aliens.sort_custom(func(a, b): return a[0] < b[0])
			print(team_aliens)
			print(team_humans)
			for i in range(len(team_aliens)):
						var player
						player = PlayerScene.instantiate()
						player.uname = team_aliens[i][0]
						add_child(player)
						if player.uname != EventLib.client_uname:
							player.set_script(peer_script)
						player.uname = team_aliens[i][0]
						player.set_name(team_aliens[i][0])
						var node = get_node("spawn/" + str(i))
						player.global_position = node.global_position
						alien_team.append(player)
			for i in range(len(team_humans)):
						var player
						player = humans.instantiate()
						player.uname = team_humans[i][0]
						add_child(player)
						if player.uname != EventLib.client_uname:
							player.set_script(peer_script)
						player.uname = team_humans[i][0]
						player.set_name(team_humans[i][0])
						var node = get_node("spawn/" + str(i+len(team_humans)))
						player.global_position = node.global_position
						human_team.append(player)
			
		"shooter_peer_physics_report":
			var peer_pos
			var peer_gun
			var peer : String = data["their_name"]
			if data.has("their_position"):
				peer_pos = data["their_position"]
			var peer_bullets = data["their_bullets"]
			if data.has("their_arm"):
				peer_gun = data["their_arm"]
			if peer_gun != null:
				var gun = get_node(peer).get_node("GunRotation")
				gun.rotation = peer_gun
			if peer_pos != null:
				get_node(peer).global_position.x = peer_pos.x
				get_node(peer).global_position.y = peer_pos.y
			for b in get_tree().get_nodes_in_group(peer.to_lower()):
				b.queue_free()
			for b in peer_bullets:
				spawn_peer_bullets(b["X"], b["Y"], peer.to_lower())
			
			
		"shooter_you_got_hit":
			var shooter : String = data["shooter"]
			var remaining_health = data["remaining_health"]
			get_node(EventLib.client_uname).vitality = remaining_health
			
		"shooter_you_hit_someone":
			var victim = data["victim"]
			var peer_rem_health = data["victim_remaining_health"]
					
			get_node(victim).vitality = peer_rem_health
		"shooter_someone_got_hit":
			var shooter = data["shooter"]
			var victim = data["victim"]
			var victim_rem_health = data["victim_remaining_health"]
			
			get_node(victim).vitality = victim_rem_health
			
		"shooter_physics_report_position_invalid_error":
			print("shooter_physics_report_position_invalid_error")
			pass
		
		"shooter_physics_report_invalid_arm_error":
			print("shooter_physics_report_invalid_arm_error")
			
		"shooter_physics_report_invalid_bullets_error":
			print("shooter_physics_report_invalid_bullets_error")
			pass
		
		"shooter_physics_report_dead_data_error":
			print("shooter_physics_report_dead_data_error")
			pass
			
		"shooter_bullet_hit_invalid_victim_name_error":
			print("shooter_bullet_hit_invalid_victim_name_error")
			pass
			
		"shooter_bullet_hit_no_such_alive_player_error":
			print("shooter_bullet_hit_no_such_alive_player_error")
			pass
			
		"shooter_unknown_message_error":
			print("shooter_unknown_message_error")
			pass


		# We have to return `false` if we didn't handle the message.
		_: return false

	return true

	
func spawn_peer_bullets(x,y,p):
	var b = bullet.instantiate()
	b.set_script(null)
	b.global_position.x = x
	b.global_position.y = y
	get_tree().root.add_child(b)
	b.add_to_group(p)
	

func _process(delta):
	if duration >= 10:
		$Label.text = str("%02d" %duration)
	else:
		$Label.text = str("%01d" %duration)
	if duration <= 0:
		end = true
		
	var wiped = false
	var alien_num = 0
	var human_num = 0
	for i in alien_team:
		if i.vitality == 0:
			alien_num += 1
	if alien_num == len(alien_team):
		wiped = true
	
	for i in human_team:
		if i.vitality == 0:
			human_num += 1
	if human_num == len(human_team):
		wiped = true
		
	if wiped == true:
		end = true
	
	

	#await get_tree().create_timer(2).timeout
	#if _localPlayer == null:
		#return
#
	#var direction = input_direction()
#
	#if direction == Vector2.ZERO:
		## Nothing to do.
		#return

	## Tell the server.
	#send_position_update(_localPlayer.position)



func _on_timer_timeout():
	duration -= 1
	
	pass # Replace with function body.
