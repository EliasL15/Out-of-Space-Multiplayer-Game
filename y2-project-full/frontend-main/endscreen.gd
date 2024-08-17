extends CanvasLayer

@onready var currentscore_label = %currentscore_label
@onready var highestscore_label = %highestscore_label
@onready var quitbutton = %quitbutton

# Called when the node enters the scene tree for the first time.
func _ready():
	quitbutton.pressed.connect(_on_quit_button_pressed)
	get_tree().paused = true
	get_scores()
	pass # Replace with function body.

func get_scores():
	var bird = get_tree().get_first_node_in_group("bird") as Bird
	currentscore_label.text = str(bird.current_score)
	if bird.highest_score < bird.current_score:
		highestscore_label.text = str(bird.current_score)
	else:
		highestscore_label.text = str(bird.highest_score)
# Called every frame. 'delta' is the elapsed time since the previous frame.

func _on_quit_button_pressed():
	get_tree().quit()
	pass

func _on_restart_button_pressed():
	get_tree().paused = false
	get_tree().reload_current_scene()
