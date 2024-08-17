extends MinigameBase

@onready var car_scence = preload("res://car_character.tscn")
@onready  var car_opp_scene = preload("res://car_character_opp.tscn")

var startIsActive = false
var othercheckpointIsActive = true
var lapCount=-1
var car
var lap
var charscript
var laps
var timeout_time
var oppcars = {}

# Called when the node enters the scene tree for the first time.
func _ready():
	pass


# Called every frame. 'delta' is the elapsed time since the previous frame.
func _process(delta):
	lap =  car.get_child(3)
	lap.text = "Lap Count: " +str(lapCount) + str("/3") + "\nTimer: " + str(149 - int($Timer.time_left))
	if car.position== null || car.stop:
		pass
	else:
		EventLib.message_send({
			"type": "race_pos_changed",
			"pos": {"x":car.position.x,"y":car.position.y}
		}
		)
	

func instance_car(car_position):
	print(true)
	var car = car_scence.instantiate()
	add_child(car)
	car.position = car_position
	return car
	
func instance_opp_car(car_position):
	print(false)
	var newcar = car_opp_scene.instantiate()
	add_child(newcar)
	newcar.position = car_position
	var image = newcar.get_child(0)
	print(image.rotation)
	image.rotation = 60

	return newcar



func _on_start_checkpoint_body_entered(body):
	if startIsActive:
		othercheckpointIsActive=!othercheckpointIsActive
		startIsActive = !startIsActive
		lapCount+=1
		lap =  car.get_child(3)
		lap.text = "Lap Count: " +str(lapCount) + str("/3") + "\nTimer: " + str(timeout_time - $Timer.time_left)
		
		if lapCount >=1:
			if lapCount==3:
				stop()
			await get_tree().create_timer(0.5).timeout  # Wait for 0.5 seconds
			EventLib.message_send({
				"type": "race_completed_lap",
				})

func _on_start_checkpoint_2_body_entered(body):
	if othercheckpointIsActive:
		othercheckpointIsActive=!othercheckpointIsActive
		startIsActive = !startIsActive
		
func stop():
	car.stop = true

func on_message(data: Dictionary) -> bool:
	match data["type"]:
		"race_welcome": on_welcome(
		data["your_spawn"],
		data["laps"],
		data["timeout"],
		data["peer_spawns"]                                
		) 
		
		"race_you_finished": on_finish(
			data["time_taken"],
			data["points_earned"]
			)
			
			
		"race_peer_pos_changed": update_pos(
			data["their_name"],
			data["their_pos"]
		)
		
		"race_peer_completed_lap":pass
		
		"race_you_finished":pass
		
		"race_peer_finished": pass
		
		"race_pos_change_for_finished_player_error": stop()
		
		# We have to return `false` if we didn't handle the message.
		_: return false

	return true


func update_pos(name, pos):
	var new_pos = Vector2(pos["X"], pos["Y"])
	
	var image = oppcars[name].get_child(0)

	var oppcar = oppcars[name]
	if oppcar.position != new_pos:
		var direction = new_pos - oppcar.position
		var angle = direction.angle() + 1.571

		# Convert the angle to degrees and add 90 degrees to match the sprite's orientation
		#angle += 90

		image.rotation = angle
		oppcar.position = new_pos


func on_welcome(spwanp,all_laps,timeout,peer_spawns):

	var x = spwanp["x"]
	var y = spwanp["y"]
	for key in peer_spawns:
		var pos = (peer_spawns[key])
		oppcars[key] = instance_opp_car(Vector2(pos["x"],pos["y"]))
	car = instance_car(Vector2(x,y))
	laps = all_laps
	timeout_time = timeout
	$Timer.wait_time = timeout
	pass


func on_finish(time_taken,points_earned):
	pass
	
	
func on_minigame_end():
	# In this method you should do anything that you need to do before your minigame ends. For
	# example, if we had something running in the background, we'd want to stop it.
	print("Race ending")

