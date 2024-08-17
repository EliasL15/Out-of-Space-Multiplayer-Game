extends CanvasLayer

@onready var label = $Label

var bird: Bird

# Called when the node enters the scene tree for the first time.
func _ready():
	bird = get_tree().get_first_node_in_group("bird")
	bird.score_changed.connect(_on_score_changed)
	label.text = "0"
	pass # Replace with function body.


# Called every frame. 'delta' is the elapsed time since the previous frame.
func set_score(score: int):
	label.text = score

func _on_score_changed(score: int):
	label.text = str(score)
