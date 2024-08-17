extends Button

@onready var not_found_lbl = get_node("/root/JoinLobby/no_lobby_found_lbl")
@onready var full_lbl = get_node("/root/JoinLobby/lobby_full_lbl")
@onready var show_no_lobby_lbl = false
@onready var animation_count = 0

@onready var show_full_lobby_lbl = false

func _on_pressed():
	#get code from line edit node
	var tf = get_node("/root/JoinLobby/BackBtn/CodeTF")
	print(tf.text)
	
	var lobby_data = await EventLib.join_lobby(tf.text)
	#handle error messages
	if(str(lobby_data)=="lobby_not_found"):
		print("lobby not found so don't do anything")
		show_no_lobby_lbl=true
		
	elif(str(lobby_data)=="lobby_full"):
		print("lobby is full")
		show_full_lobby_lbl=true
		
	
	else:
		#used for updating
		Globalvar.serverObj = EventLib.the_lobby
		
		print(EventLib.the_lobby.playerTeam)
		var player_arr = EventLib.the_lobby.playerTeam
		Globalvar.lobby_code  = EventLib.the_lobby.lobbyId
		
		#add 1 player to the blue team for the initial lobby creation
		#find the index of the current player
		var index = find_index(player_arr)
		var currentPlayer = player_arr[index]     
		
		#EventLib.client_uname = currentPlayer[0]
		Globalvar.current_player_team = currentPlayer[1]
		Globalvar.current_player_ready = currentPlayer[2]
		
		#change scene
		get_tree().change_scene_to_file("res://Scenes/lobby.tscn")
		
	pass # Replace with function body.

func _process(_delta):
	#animate the error messages
	if(show_no_lobby_lbl and (animation_count<120)):
		show_full_lobby_lbl=false  #hide the other label if they manage to try a new server too fast
		not_found_lbl.show()
		not_found_lbl.position.y += 0.1
		animation_count+=1
	elif(show_full_lobby_lbl and (animation_count<120)):
		show_no_lobby_lbl=false  #hide the other label if they manage to try a new server too fast
		full_lbl.show()
		full_lbl.position.y += 0.1
		animation_count+=1
	else:
		full_lbl.position.y = 348
		not_found_lbl.position.y = 348
		show_full_lobby_lbl=false
		show_no_lobby_lbl=false
		animation_count=0
		full_lbl.hide()
		not_found_lbl.hide()
	
func find_index(player_arr):
	var index = 0
	for i in range(len(player_arr)):
		if(EventLib.client_uname==player_arr[i][0]):
			index = i
	return index
		
