extends Node
@onready var portal_scene = load("res://Interactive_objects/portal.tscn")
class Lobby:
	var playerTeam: Array
	var lobbyId: String
	func _init(p, l):
		playerTeam = p
		lobbyId = l
	func search_player(name):
		for index in range(len(playerTeam)):
			#print(len(playerTeam))
			if playerTeam[index][0] == name:
				return index
		push_error("[EventLib] Player {p} doesn't exist.".format({"p": name}))
		return - 1

	func addPlayer(name, team):
		#player attributes: name, team , ready status and individual scores
		playerTeam.append([name, team, false, 0])
	func removePlayer(name): # Find player and delete it.
		var index = search_player(name)
		if index != - 1:
			playerTeam.pop_at(index)
	func setReady(name, ready):
		var index = search_player(name)
		if index != - 1:
			playerTeam[search_player(name)][2] = ready
	func changeTeam(name):
		var index = search_player(name)
		if index != - 1: # Change team from 0 -> 1,or 1->0
			playerTeam[search_player(name)][1] = 1 - playerTeam[search_player(name)][1]

class Ship:
	var ship_lobby: Lobby
	var minigames: Dictionary
	func _init(sl, m):
		ship_lobby = sl
		minigames = m
	func get_minigame(name):
		return minigames[name]
	func add_minigame(data):
		minigames.merge(data)

class Minigame:
	var game_name: String
	var pos: Vector2
	var players: String # 1v1,2v2 or 3v3
	var expired: bool
	var cooldown: float
	var cooldown_active: bool
	func _init(n, po, pl, c):
		game_name = n
		pos = po
		players = pl
		expired = false
		cooldown = c
		cooldown_active = false

var the_lobby = Lobby.new([], "") # Make new lobby object which is stored
var client_uname = "" # Used to idenify the client.
var playerRefs: Array # Associates the username with the spawned player obj.
# We need one signal for every event that other scripts need to be able to
# respond to.
signal go_to_main(the_data)
signal update_player_tables
signal update_lobby_code
signal portal_ready_to_spawn
signal add_portal_tiles
signal flag_cooldown_timeout(flagID)
signal toggle_ready_btn

signal data_ready(the_data) # A signal which is emited after a response is sent
# to the server and the data has been processed
signal peer_movement(the_data)

static func test_function():
	print("Library has been imported sucessfully.")

func handle_response(request_string): # Takes in a JSON request and
	# calls library specific functions accordingly
	var json = JSON.new()
	var err = json.parse(request_string) # Parse and validate JSON
	if err == OK: # If JSON is valid...
		var data = json.data
		if "type" in data.keys(): # Type is needed in the JSON request.
			match data["type"]: # Case statement matching different response types.
				"lobby_welcome": _lobby_welcome(data)
				"lobby_peer_joined": _lobby_peer_joined(data)
				"lobby_peer_team_change": _lobby_peer_team_change(data)
				"lobby_peer_left": _lobby_peer_left(data)
				"lobby_peer_ready_change": _lobby_peer_ready_change(data)
				"client_already_in_lobby_error": _client_already_in_lobby()
				"lobby_not_found": _lobby_not_found()
				"lobby_full": _lobby_full()
				"ship_welcome": _handle_welcome(data)
				"ship_mov_peer_position_update": _peer_position_update(data)
				_: data_ready.emit(data) # Emit arbitrary response if applicable
# ----------- LOBBY METHODS --------------------#

# ----------- CLIENT REQUEST METHODS -----------#

func create_lobby(): # Send lobby create request to server.
	Server.socket.send_text(JSON.stringify({"type": "lobby_create"}))
	var response = await data_ready
	return response

func join_lobby(code):
	Server.socket.send_text(JSON.stringify({"type": "lobby_join", "lobby_id": code}))
	var response = await data_ready
	# Wait for response to be received and variables updated
	return response # Return class if sucessful, else return error.

func leave_lobby():
	Server.socket.send_text(JSON.stringify({"type": "lobby_bye"}))
	the_lobby = Lobby.new([], "") # Overwrite old lobby object

func update_lobby_visual():
	update_player_tables.emit()
	update_lobby_code.emit()

func update_client_pos(newPos):
	Server.socket.send_text(JSON.stringify({"type": "ship_mov_position_update",
	"x": newPos[0], "y": newPos[1]}))

func client_team_change():
	var new_team = 1 - the_lobby.playerTeam[the_lobby.search_player(client_uname)][1]
	# Get client index in list and switch team from 0 -> 1 or 1 -> 0
	Server.socket.send_text(JSON.stringify({"type": "lobby_team_change", "team": new_team}))
	# Change team on the backend side.
	the_lobby.changeTeam(client_uname)
	
	# update ready status
	for player in the_lobby.playerTeam:
		player[2] = false
	
	toggle_ready_btn.emit()
	# Change team on the frontend side.
	update_lobby_visual()

func client_ready_change(new_ready):
	Server.socket.send_text(JSON.stringify({"type": "lobby_ready_change", "ready": new_ready}))
	the_lobby.setReady(client_uname, new_ready) # Update ready on the frontend.
	update_lobby_visual()

func message_send(request):
	Server.socket.send_text(JSON.stringify(request))

# ----------- SERVER RESPONSE METHODS -----------#
func _lobby_welcome(data): # Called when lobby_welcome response is recieved.
	# I.e. the user joins the lobby
	the_lobby.addPlayer(data["your_name"], data['your_team'])
	client_uname = data["your_name"] # Store the client's username.s
	# Add current player & lobby id to the_lobby object
	the_lobby.lobbyId = data["lobby_id"]
	if "peer_teams" in data.keys(): # Are other players already in lobby?
		for key in data["peer_teams"]:
			the_lobby.addPlayer(key, data["peer_teams"][key])
			# E.g. {"peer_teams": {{"user": 0}}} will add ["user",0, false] to
			# playerTeam.
	push_warning("[EventLib] Joined lobby with code: " + data["lobby_id"])
	data_ready.emit(the_lobby)
	# Emit the updated lobby class.

func _lobby_peer_joined(data): # Peer joined so add their name to the lobby obj
	the_lobby.addPlayer(data["their_name"], data["their_team"])
	data_ready.emit(true)
	update_lobby_visual()

func _lobby_peer_team_change(data): # Peer joined so update their team stored in the lobby obj.
	the_lobby.changeTeam(data["their_name"])
	data_ready.emit(true)
	
	#unready all players
	for player in the_lobby.playerTeam:
		player[2] = false
	
	toggle_ready_btn.emit()
	update_lobby_visual()

func _lobby_peer_ready_change(data): # Peer changed ready so update ready in lobby obj.
	the_lobby.setReady(data["their_name"], data["ready"])
	data_ready.emit(true)
	update_lobby_visual()

func _lobby_peer_left(data): # Peer left, so  remove them from the lobby obj.
	the_lobby.removePlayer(data["their_name"])
	data_ready.emit(true)
	update_lobby_visual()

# Various error responses for the lobby
func _client_already_in_lobby():
	data_ready.emit("client_already_in_lobby")
	push_warning("[EventLib] client_already_in_lobby")
	data_ready.emit("client_already_in_lobby")

	# Emit error.
func _lobby_not_found():
	data_ready.emit("lobby_not_found")
	push_warning("[EventLib] lobby_not_found")

func _lobby_full():
	data_ready.emit("lobby_full")
	push_warning("[EventLib] lobby_full")
# ---------------- MOVEMENT METHODS ---------------------------#
func _peer_position_update(data):
	peer_movement.emit(data)

# ---------------- SHIP METHODS --------------------------------#
func _handle_welcome(data):
	go_to_main.emit(data) # Switch scene
	print(data)
	await portal_ready_to_spawn # Wait for the scene to be created.
	# Now add the flags.
	var the_ship = Ship.new(the_lobby, {})
	Globalvar.flags_copy = data["flags"]
	
	for flag in data["flags"]:
		# Create minigame object for each minigame.
		var a_minigame = Minigame.new(data["flags"][flag]["minigame"],
		Vector2(data["flags"][flag]["pos"]["x"],
		data["flags"][flag]["pos"]["y"]),
		"1v1", data["flags"][flag]["cooldown"])
		the_ship.add_minigame({data["flags"][flag]["minigame"]:a_minigame})
		var portal_instance = portal_scene.instantiate().get_node(".")
		portal_instance.position = a_minigame.pos
		var nameLbl = portal_instance.get_node("nameLbl")
		nameLbl.text = data["flags"][flag]["minigame"]
		var area_2d = portal_instance.get_node("./Area2D")
		get_tree().get_root().get_node("game_level").add_child(portal_instance)
		area_2d.set_script(load("res://Interactive_objects/portal_interactable.gd"))
		
		#add a attribute to each flag to say which team owns which flag
		Globalvar.flags_copy[flag]["owned_by"] = -1
		
		#set name of portal 
		portal_instance.name = flag
		

func is_valid_message(object) -> bool:
	return typeof(object) == TYPE_DICTIONARY&&object.has("type")
