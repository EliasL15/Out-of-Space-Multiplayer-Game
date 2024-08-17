extends Control


#@onready 
#@onready var itemList = lobby_scene.get_node("BlueTeamContainer/ItemListBlue")

#called when the node enters the scene tree for the first time
func _ready():
	pass
	
#called every frame. 'delta' is elapsed time since the previous frame
func _process(_delta):
	pass

#runs when the button is pressed
func _on_pressed():
	print("creating lobby")
	await EventLib.create_lobby()       #create lobby library
	print(EventLib.the_lobby.playerTeam)
	var player_arr = EventLib.the_lobby.playerTeam
	Globalvar.lobby_code  = EventLib.the_lobby.lobbyId
	
	#used for updating the lobby
	Globalvar.serverObj = EventLib.the_lobby
	
	#add 1 player to the blue team for the initial lobby creation
	var currentPlayer = player_arr[0]     
	#Globalvar.playerArray[0]=(currentPlayer)
	#test players
	#Globalvar.serverObj.playerTeam.append(["test1",0,false])  
	#Globalvar.serverObj.playerTeam.append(["test2",0,false])
	#Globalvar.serverObj.playerTeam.append(["test3",1,false])
	#Globalvar.serverObj.playerTeam.append(["test4",1,false])
	#Globalvar.serverObj.playerTeam.append(["test5",1,false])
	EventLib.client_uname = currentPlayer[0]
	Globalvar.current_player_team = currentPlayer[1]
	Globalvar.current_player_ready = currentPlayer[2]
	
	#change scene
	get_tree().change_scene_to_file("res://Scenes/lobby.tscn")
	pass #enter code to create a lobby 
	
