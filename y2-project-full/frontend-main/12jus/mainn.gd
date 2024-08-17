# This is a single player version, for multiplayer, remove cpu components

extends MinigameBase

var choice = ""
var playerscore = 0
var cpuscore = 0

var selection_duration
var post_round_duration
var target_win_count
var result
var opp_sel

##########################################

class _Player:
	## The colours we use for players from different teams. Team number should be used as index.
	static var _teamColours: Array[Color] = [
		"red",
		"blue",
	]

	var colour: Color

	func _init(team: int):
		colour = _teamColours[team]

## The object we use to represent the local player.
var _localPlayer: _Player = null

## The position of the local player when we last reported their position to the server.
var _lastReportedPosition: Vector2

## The nodes we use to represent the remote players. Keys are player names; values are `_Player`
## objects.
var _remotePlayers: Dictionary = {}

## Called when we receive a message telling us where the local player has been spawned.
func on_spawn():
	_localPlayer = _Player.new(local_team())

## Called when we receive a message telling us where a peer has been spawned.
func on_peer_spawn(theName: String):
	var remotePlayer = _Player.new(player_team(theName))

	_remotePlayers[theName] = remotePlayer
	
func player_team(theName: String) -> int:
	# Based on code from `game_level.gd`.
	var index = EventLib.the_lobby.search_player(theName)
	return EventLib.the_lobby.playerTeam[index][1]

## Returns the team number of the local player.
func local_team() -> int:
	return player_team(EventLib.client_uname)

func on_message(data: Dictionary) -> bool:
	match data["type"]:

		"rps_welcome":
			selection_duration = data["selection_secs"]
			post_round_duration = data["post_round_secs"]
			target_win_count = data["target_win_count"]
			$aniTimer.wait_time = post_round_duration
			$Timer.wait_time = selection_duration
			$player1.text = EventLib.client_uname
			for i in _peer_names:
				if i != EventLib.client_uname:
					$player2.text = i
					break
			$Timer.start()
			peers()
			## Round ends, image animation plays
			## if no choice has been made, then all buttons disabled
		"rps_round_end":
			result = data["result"]
			if result == "win":
				playerscore += 1
			elif result == "draw":
				pass
			else:
				cpuscore += 1
			opp_sel = data["opponent_selection"]
			if choice == "rock":
				$AnimationPlayer.play("rock")
			elif choice == "paper":
				$AnimationPlayer.play("paper")
			elif choice == "scissors":
				$AnimationPlayer.play("scissors")
			else: 
				$rock.disabled = true
				$paper.disabled = true
				$scissors.disabled = true
			
			if opp_sel == "rock":
				$AnimationPlayer2.play("rock2")
			elif opp_sel == "paper":
				$AnimationPlayer2.play("paper2")
			elif opp_sel == "scissors":
				$AnimationPlayer2.play("scissors2")

			$aniTimer.start()

			## round starts, all images back to i=original position, 
			## all buttons enabled and visible
		"rps_selection_start":
			choice = ""
			$rock.disabled = false
			$paper.disabled = false
			$scissors.disabled = false
			$rock.visible = true
			$paper.visible = true
			$scissors.visible = true
			$Rock.position = $spawn.position
			$Scissors.position = $spawn.position
			$Paper.position = $spawn.position
			
			#cpu 
			$Rock2.position = $spawn2.position
			$Scissors2.position = $spawn2.position
			$Paper2.position = $spawn2.position
			$Timer.start()
		"rps_selection_too_late_error":
			pass
		"rps_selection_already_made_error":
			pass
		
		"rps_selection_invalid_string_error":
			pass
		
		"rps_invalid_selection_error":
			pass
		
		"rps_unexpected_message_type_error":
			pass

		# We have to return `false` if we didn't handle the message.
		_: return false

	return true

func on_minigame_end():
	# In this method you should do anything that you need to do before your minigame ends. For
	# example, if we had something running in the background, we'd want to stop it.
	pass

func on_ship_endgame():
	pass

func on_peer_left(_theirName: String):
	pass

func on_non_peer_left(_theirName: String):
	pass
	
func _process(_delta):
	var choosing = $Timer.time_left+1
	var waiting = $aniTimer.time_left+1
	$round.text = "Choose in"
	if $Timer.time_left == selection_duration:
		$countdown.text = str("%01d" %selection_duration)
	else:
		$countdown.text = str("%01d" %choosing)
	if $Timer.time_left == 0:
		$round.text = "Next Round"
		if playerscore == 3:
			$round.text = "Winner"
		if cpuscore == 3:
			$round.text = "Loser"
		if$aniTimer.time_left == post_round_duration:
			$countdown.text = str("%01d" %post_round_duration)
		else:
			$countdown.text = str("%01d" %waiting)
	$score.text = (str(playerscore) + " : " + str(cpuscore))

func _on_rock_pressed():
		choice = "rock"
		EventLib.message_send({
							"type": "rps_selection",
							"element": "rock"
						})
		$paper.disabled = true
		$rock.disabled = true
		$scissors.disabled = true
		$paper.visible = false
		$scissors.visible = false

func _on_paper_pressed():
		choice = "paper"
		EventLib.message_send({
							"type": "rps_selection",
							"element": "paper"
						})
		$paper.disabled = true
		$rock.disabled = true
		$scissors.disabled = true
		$rock.visible = false
		$scissors.visible = false

func _on_scissors_pressed():
		choice = "scissors"
		EventLib.message_send({
							"type": "rps_selection",
							"element": "scissors"
						})
		$rock.disabled = true
		$paper.disabled = true
		$scissors.disabled = true
		$rock.visible = false
		$paper.visible = false

func _on_timer_timeout():
	if choice == "":
		EventLib.message_send({
				"type": "rps_selection",
				"element": ""
				})
