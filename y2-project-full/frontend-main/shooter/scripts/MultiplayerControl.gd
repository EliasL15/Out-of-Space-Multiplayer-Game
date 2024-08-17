extends Control

@export var address = "127.0.0.1"
@export var port = 8910
var peer
func _ready():
	multiplayer.peer_connected.connect(peer_connected)
	multiplayer.peer_disconnected.connect(peer_disconnected)
	multiplayer.connected_to_server.connect(connected_to_server)
	multiplayer.connection_failed.connect(connection_failed)
	if "--server" in OS.get_cmdline_args():
		hostGame()
# called on server and client
func peer_connected(id):
	print("player connected " + str(id))

#called n server and client
func peer_disconnected(id):
	print("player disconnected " + str(id))
	GameManager.players.erase(id)
	var players = get_tree().get_nodes_in_group("player")
	for i in players:
		i.queue_free()


#called only on clients
func connected_to_server():
	print("connected to server")
	SendPlayerInfo.rpc_id(1, $LineEdit.text, multiplayer.get_unique_id())

#called only on clients
func connection_failed():
	print("Can't connect")
	
@rpc("any_peer")
func SendPlayerInfo(name, id):
	if !GameManager.players.has(id):
		GameManager.players[id] = {
			"name" : name,
			"id" : id,
			"score" : 0
		}
	if multiplayer.is_server():
		for i in GameManager.players:
			SendPlayerInfo.rpc(GameManager.players[i].name, i)

@rpc("any_peer", "call_local")
func StartGame():
	var scene = load("res://scene/main.tscn").instantiate()
	get_tree().root.add_child(scene)
	self.hide()

func hostGame():
	peer = ENetMultiplayerPeer.new()
	var error = peer.create_server(port, 2)
	if error != OK:
		print("cannot host" + str(error))
		return
	peer.get_host().compress(ENetConnection.COMPRESS_RANGE_CODER)
	
	multiplayer.set_multiplayer_peer(peer)
	print("waiting for players")
	
	
func _on_join_button_down():
	peer = ENetMultiplayerPeer.new()
	peer.create_client(address, port)
	peer.get_host().compress(ENetConnection.COMPRESS_RANGE_CODER)
	multiplayer.set_multiplayer_peer(peer)


func _on_start_button_down():
	StartGame.rpc()
	pass # Replace with function body.


func _on_host_button_down():
	hostGame()
	SendPlayerInfo($LineEdit.text, multiplayer.get_unique_id())
