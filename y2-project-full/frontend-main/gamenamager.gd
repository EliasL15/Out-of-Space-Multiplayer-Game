extends Node

@export var endscreen: PackedScene

var bird1: Bird1
var bird2: Bird2
# Called when the node enters the scene tree for the first time.
func _ready():
	var bird1 = get_tree().get_first_node_in_group("bird1")
	bird1.bird1_died.connect(_add_end_screen)
	var bird2 = get_tree().get_first_node_in_group("bird2")
	bird2.bird2_died.connect(_add_end_screen)
	pass # Replace with function body.


# Called every frame. 'delta' is the elapsed time since the previous frame.
func _process(delta): 
	pass

func _add_end_screen():
	var endscreen_instance = endscreen.instantiate()
	add_child(endscreen_instance)
