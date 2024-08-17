# CALLED AUTOMATICALLY WHEN THE GAME IS FIRST OPENED.
extends Node

var socket = WebSocketPeer.new()

# y2.squ1dd13.ink   <-- our server
# 127.0.0.1         <-- your computer
@export var ip_address = "y2.squ1dd13.ink"
@export var port = "443"

func _ready():
	print("wss://{ip}:{port}/ws".format({"ip": ip_address, "port": port}))
	socket.connect_to_url("wss://{ip}:{port}/ws".format({"ip": ip_address, "port": port})) # Connect to WS server.

func _process(_delta):
	socket.poll() # Poll the socket.
	var state = socket.get_ready_state()
	if state == WebSocketPeer.STATE_OPEN: # If connection is open...
		#print("web socket opened")
		while socket.get_available_packet_count(): # For each remaining incoming packet...
			EventLib.handle_response(socket.get_packet().get_string_from_utf8())
	elif state == WebSocketPeer.STATE_CLOSING:
		# Keep polling to achieve proper close.
		pass
	elif state == WebSocketPeer.STATE_CLOSED:
		var code = socket.get_close_code()
		var reason = socket.get_close_reason()
		print("WebSocket closed with code: %d, reason %s. Clean: %s" % [code, reason, code != - 1])
		set_process(false) # Stop processing.
