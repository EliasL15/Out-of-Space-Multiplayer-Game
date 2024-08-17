extends MinigameBase

@onready var car_scence = preload("res://car_character.tscn")

var startIsActive = false
var othercheckpointIsActive = true
var lapCount=-1
var car
var lap
var charscript
var laps
var score_to_beat
var timeout_time

# Called when the node enters the scene tree for the first time.
func _ready():
	pass


# Called every frame. 'delta' is the elapsed time since the previous frame.
func _process(delta):
	lap =  car.get_child(3)
	lap.text = "Lap Count: " +str(lapCount) + str("/3") + "\nTime To Beat: " + str(score_to_beat) + "\nTimer: " + str(149 - int($Timer.time_left))


func instance_car(car_position):
	var car = car_scence.instantiate()
	add_child(car)
	car.position = car_position
	return car



func _on_start_checkpoint_body_entered(body):
	if startIsActive:
		othercheckpointIsActive=!othercheckpointIsActive
		startIsActive = !startIsActive
		lapCount+=1
		lap =  car.get_child(3)
		lap.text = "Lap Count: " +str(lapCount) + str("/3") + "\nTime To Beat: " + str(score_to_beat) + "\nTimer: " + str(timeout_time - $Timer.time_left)
		if lapCount >=1:
			EventLib.message_send({
				"type": "race_completed_lap",
				})

func _on_start_checkpoint_2_body_entered(body):
	if othercheckpointIsActive:
		othercheckpointIsActive=!othercheckpointIsActive
		startIsActive = !startIsActive
		


func on_message(data: Dictionary) -> bool:
	match data["type"]:
		"race_welcome": on_welcome(
		data["to_beat"],
		data["your_spawn"],
		data["laps"],
		data["timeout"]
		) 
		
		"race_you_finished": on_finish(
			data["time_taken"],
			data["points_earned"]
			)
		# We have to return `false` if we didn't handle the message.
		_: return false

	return true


func on_welcome(to_beat, spwanp,all_laps,timeout):
	var x = spwanp["x"]
	var y = spwanp["y"]
	car = instance_car(Vector2(x,y))
	laps = all_laps
	timeout_time = timeout
	$Timer.wait_time = timeout
	
	score_to_beat = int(to_beat)
	pass


func on_finish(time_taken,points_earned):
	pass
	
	
func on_minigame_end():
	# In this method you should do anything that you need to do before your minigame ends. For
	# example, if we had something running in the background, we'd want to stop it.
	print("Race ending")

