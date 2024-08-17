extends MinigameBase
@onready var top_score
@onready var bird_scene = preload("res://bird_char.tscn")

var bird
var beat
# Called when the node enters the scene tree for the first time.
func _ready():
	
	pass


# Called every frame. 'delta' is the elapsed time since the previous frame.
func _process(delta):
	if bird.current_score != null:
		$currentscore.text = "Score to Beat: "+ str(beat) +"\nCurrent Score: " + str(bird.current_score)+ "\nTimer:" + str($countdown_timer.time_left)
	
func assign_top_score(msg):
	top_score = msg["top_score"]

func on_message(data: Dictionary) -> bool:
	match data["type"]:
		"bird_welcome": welcome(data["to_beat"])
		# We have to return `false` if we didn't handle the message.
		_: return false

	return true
	
func on_minigame_end():
	# In this method you should do anything that you need to do before your minigame ends. For
	# example, if we had something running in the background, we'd want to stop it.
	bird.queue_free()
	print("Cookie Cliker ending")
	
func welcome(to_beat):
	bird = bird_scene.instantiate()
	add_child(bird)
	bird.position = Vector2(86,71)
	beat = to_beat
	$currentscore.text = "Score to Beat: "+ str(to_beat) +"\nCurrent Score: " + str(0)


	
