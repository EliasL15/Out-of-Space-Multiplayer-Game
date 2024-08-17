extends Control

#called when the node enters the scene tree for the first time
func _ready():
	pass
	
#called every frame. 'delta' is elapsed time since the previous frame
func _process(_delta):
	pass

#runs when the button is pressed   ---this method can be used after all players ready up
func _on_pressed():
	#print("going to main game")
	
	# Transition to the game scene
	get_tree().change_scene_to_file("res://Scenes/game_level.tscn")
	
	#Add player into game (goes to game_level scene)
	Globalvar.add_player = true

	#add the rest of the players as remote players ie. different script and detach player interaction script
	Globalvar.add_remote_players = true
	
	pass
