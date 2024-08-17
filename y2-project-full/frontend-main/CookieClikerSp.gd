extends MinigameBase

var newscore = 0



func _process(delta):
	$TimerLabel.text = "Timer: "+ ( "%s" % int($Timer.time_left))
	

func on_welcome(duration,stb):
	$Timer.wait_time = duration +1
	$Timer.start()
	$STB.text = "Score To Beat: " + str(stb)

func on_message(data: Dictionary) -> bool:
	match data["type"]:
		"cps_welcome": on_welcome(
		data["duration"],data["score_to_beat"])
		
		"cps_timeout":send_score()
		# We have to return `false` if we didn't handle the message.
		_: return false

	return true
	

func update_score(score):
	newscore = score


func send_score():
	EventLib.message_send({
		"type": "cps_report",
		"clicks": newscore
		})


func on_minigame_end():
	# In this method you should do anything that you need to do before your minigame ends. For
	# example, if we had something running in the background, we'd want to stop it.
	
	print("Cookie Cliker ending")
