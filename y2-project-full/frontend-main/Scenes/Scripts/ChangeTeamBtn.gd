extends Button

var player_position = 0

#called when the node enters the scene tree for the first time
func _ready():
	pass
	
#called every frame. 'delta' is elapsed time since the previous frame
func _process(_delta):
	pass

#runs when the button is pressed
func _on_pressed():
	
	#unready all the players
	Globalvar.remove_ready = true
	
	
	############################# update the server so the other players know they switched teams  ##############################
	
	await EventLib.client_team_change()
	
	pass

