extends Control

#called when the node enters the scene tree for the first time
func _ready():
	pass
	
#called every frame. 'delta' is elapsed time since the previous frame
func _process(_delta):
	pass

#runs when the button is pressed
func _on_pressed():
	print("joining lobby")
	
	
	#switch scenes to enter a code
	get_tree().change_scene_to_file("res://Scenes/join_lobby.tscn")
	
	
	pass #enter code to join a lobby 
