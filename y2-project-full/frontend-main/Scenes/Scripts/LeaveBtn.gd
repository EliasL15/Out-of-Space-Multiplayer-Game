extends Button

#called when the node enters the scene tree for the first time
func _ready():
	pass
	
#called every frame. 'delta' is elapsed time since the previous frame
func _process(_delta):
	pass

#runs when the button is pressed
func _on_pressed():
	
	#remove the player from the array of players
	#var player_to_remove = Globalvar.current_player_name   #placeholder 
	#for i in range(len(Globalvar.playerArray)):
	#	if(Globalvar.playerArray[i]!=[]):
	#		print("asd:"+str(Globalvar.playerArray[i][0]))
	#		print("rem:"+player_to_remove)
	#		if(Globalvar.playerArray[i][0] == player_to_remove):
	#			Globalvar.playerArray[i] == [] #remove the player name from the list
	
	EventLib.leave_lobby()
	
	get_tree().change_scene_to_file("res://Scenes/mainmenu.tscn")
	pass
	
