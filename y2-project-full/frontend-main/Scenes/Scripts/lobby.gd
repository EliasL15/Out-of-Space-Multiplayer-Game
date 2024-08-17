extends Control

@onready var itemListBlue = get_node("BlueTeamContainer/ItemListBlue")
@onready var itemListRed = get_node("RedTeamContainer/ItemListRed")
@onready var code_lbl = get_node("GameCode")

@onready var temp_server = Globalvar.serverObj

func _ready():
	EventLib.go_to_main.connect(_on_go_to_main)
	EventLib.update_player_tables.connect(_on_update_player_tables)
	EventLib.update_lobby_code.connect(_on_update_lobby_code)
	EventLib.toggle_ready_btn.connect(_toggle_ready_btn)
	#var readyBtn = get_node("./ReadyBtn")
	#readyBtn.unready_players.connect(_unready_all_players)

	# Initial update
	_on_update_lobby_code()
	_on_update_player_tables()

#called every frame. 'delta' is elapsed time since the previous frame
func _process(_delta):
	pass

func _on_go_to_main(data):
	print("going to main game")

	#Add player into game (goes to game_level scene)
	Globalvar.add_player = true
	Globalvar.ship_init_data = data
	#add the rest of the players as remote players ie. different script and detach player interaction script
	Globalvar.add_remote_players = true
	############################################

	# Transition to the game scene
	get_tree().change_scene_to_file("res://Scenes/game_level.tscn")

func _on_update_lobby_code():
	code_lbl.text = (Globalvar.lobby_code)

func _on_update_player_tables():
	print("updating tables")
	itemListBlue.clear()
	itemListRed.clear()
	
	for n in range(len(EventLib.the_lobby.playerTeam)): # 6 is the size of the player array
		if (EventLib.the_lobby.playerTeam[n] != []):
			print(EventLib.the_lobby.playerTeam)
			if (EventLib.the_lobby.playerTeam[n][1] == 0):
				#on blue team
				if (EventLib.the_lobby.playerTeam[n][2] == false):
					#player ready
					itemListBlue.add_item(EventLib.the_lobby.playerTeam[n][0], null, false)
					#ready_btn.set_text("Ready")
				else:
					#player ready
					itemListBlue.add_item(EventLib.the_lobby.playerTeam[n][0] + "(ready)", null, false)
					#ready_btn.set_text("Unready")
			else:
				if (EventLib.the_lobby.playerTeam[n][2] == false):
					#player ready
					itemListRed.add_item(EventLib.the_lobby.playerTeam[n][0], null, false)
					#ready_btn.set_text("Ready")
				else:
					#player ready
					itemListRed.add_item(EventLib.the_lobby.playerTeam[n][0] + "(ready)", null, false)
					

func _toggle_ready_btn():
	var ready_btn = get_node("ReadyBtn")
	ready_btn.set_text("Ready")

## Called when the lobby code label detects input.
func _on_game_code_gui_input(event: InputEvent):
	# WARNING! This is just a convenience feature for use during development. IT IS NOT ACCESSIBLE!
	# In the real world, some people do not use LMB as their primary mouse button.

	if !event.is_pressed()||event.button_index != MOUSE_BUTTON_LEFT:
		# Not a left-click.
		return

	print("copying lobby ID to clipboard")

	DisplayServer.clipboard_set(Globalvar.lobby_code)
