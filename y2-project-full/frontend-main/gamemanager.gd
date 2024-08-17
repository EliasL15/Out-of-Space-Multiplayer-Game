extends Node

const SAVE_PATH = "user://save_game.dat"
@export var end_screen_scene: PackedScene

var bird: Bird
# Called when the node enters the scene tree for the first time.
func _ready():
	bird = get_tree().get_first_node_in_group("bird")
	bird.bird_died.connect(add_end_screen)
	bird.bird_died.connect(_on_bird_died)
	pass # Replace with function body.


func _on_bird_died():
	add_end_screen()
	save_score(bird.current_score)
# Called every frame. 'delta' is the elapsed time since the previous frame.
func add_end_screen():
	var end_screen_instance = end_screen_scene.instantiate()
	add_child(end_screen_instance)
	pass

func save_score(score: int):
	var highest_score = read_score()
	if highest_score < score :
		var file = FileAccess.open(SAVE_PATH, FileAccess.WRITE)
		var data = {"higest score" : score}
		file.store_var(data)
	

func read_score():
	var score = 0
	if FileAccess.file_exists(SAVE_PATH):
		var file = FileAccess.open(SAVE_PATH, FileAccess.READ) 
		var data = file.get_var()
		bird.highest_score = data["higest score"]
		return data["higest score"]
	return score


