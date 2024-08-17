extends MinigameBase

@onready var mole_scene = preload("res://molecharacter.tscn")


var xlist = [27.5,43.5,59.5,75.5,91.5,107.5,123.5,139.5,155.5,171.5,187.5]
var ylist=[148-115,212-115,276-115]
var used_positions = []
var moles = []
var score=0



var positions = [
	Vector2(27.5, 33),
	Vector2(27.5, 97),
	Vector2(27.5, 161),
	Vector2(43.5, 33),
	Vector2(43.5, 97),
	Vector2(43.5, 161),
	Vector2(59.5, 33),
	Vector2(59.5, 97),
	Vector2(59.5, 161),
	Vector2(75.5, 33),
	Vector2(75.5, 97),
	Vector2(75.5, 161),
	Vector2(91.5, 33),
	Vector2(91.5, 97),
	Vector2(91.5, 161),
	Vector2(107.5, 33),
	Vector2(107.5, 97),
	Vector2(107.5, 161),
	Vector2(123.5, 33),
	Vector2(123.5, 97),
	Vector2(123.5, 161),
	Vector2(139.5, 33),
	Vector2(139.5, 97),
	Vector2(139.5, 161),
	Vector2(155.5, 33),
	Vector2(155.5, 97),
	Vector2(155.5, 161),
	Vector2(171.5, 33),
	Vector2(171.5, 97),
	Vector2(171.5, 161),
	Vector2(187.5, 33),
	Vector2(187.5, 97),
	Vector2(187.5, 161),
]

func _ready():
	for i in range(33):
		var mole_position = positions[i]
		var mole = instance_mole(mole_position)
		moles.append(mole)
		#var x = xlist[randi() % xlist.size()]
		#var y = ylist[randi() % ylist.size()]
		#var mole_position = Vector2(x, y)
		#while mole_position in used_positions:
			#x = xlist[randi() % xlist.size()]
			#y = ylist[randi() % ylist.size()]
			#mole_position = Vector2(x, y)
		#used_positions.append(mole_position)
		

	print(moles)
	move_down()



func instance_mole(mole_position):
	var mole = mole_scene.instantiate()
	add_child(mole)
	mole.position = mole_position
	return mole



func move_down():
	for mole in moles:
		if mole.visible==true:
			mole.visible = false


func mole_clicked(mole):
	if mole.visible:
		print("Mole was clicked:", mole)
		EventLib.message_send({
		"type": "mole_hit",
		"location": moles.find(mole)
		})


func move_up(mole1,mole2):
	moles[mole1].visible = true
	moles[mole2].visible = true


func on_welcome(duration_seconds,interval_seconds,initial_moles,stb):
	move_down()
	$Timer.wait_time = int(duration_seconds)
	$Timer.start()
	new_moles(initial_moles)
	$STB.text = "Score To Beat: " + str(stb)
	
func new_moles(newmolespositions):
	move_down()
	move_up(newmolespositions[0],newmolespositions[1])


func _process(delta):
	$TimeRemaining.text = "Timer: "+ ( "%s" % int($Timer.time_left))


func valid_hit(newscore,newmolespositions):
	var score_label = get_node("Label") 
	score = newscore
	score_label.text = str("Score: "+ str(score))
	new_moles(newmolespositions)


func on_minigame_end():
	# In this method you should do anything that you need to do before your minigame ends. For
	# example, if we had something running in the background, we'd want to stop it.
	
	print("demo minigame is ending")

func on_message(data: Dictionary) -> bool:
	match data["type"]:
		"mole_welcome": on_welcome(
		data["duration_seconds"],
		data["interval_seconds"],
		data["initial_moles"],
		data["score_to_beat"]
		) #array of 2 integers
		
		"mole_timeout": new_moles(data["locations"])
		
		"mole_hit_valid": valid_hit(data["score"],data["new_moles"])

		# We have to return `false` if we didn't handle the message.
		_: return false

	return true
	
	
	
