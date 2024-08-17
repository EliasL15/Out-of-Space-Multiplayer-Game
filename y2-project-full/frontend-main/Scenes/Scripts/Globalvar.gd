extends Node

signal add_portal_tiles_gl
signal populate_leaderboard

var go_main_game = false

var playerArray = [[],[],[],[],[],[]]  #the first three elements are blue team and last three are red team
var serverObj

#boolean values to run methods
var remove_ready = false
var add_player  = false
var add_remote_players = false

#player variables
var lobby_code = ""
var current_player_name = ""  #this is the current player's name
var current_player_team = 0  #this is the current player's team
var current_player_ready = false
var current_player_position = Vector2()
var ship_init_data
var player_layer_no = 0
var playerNodes = {} # Stores the id of the node and the username.

#game_level message handle variables
var flag_cooldown_dict = {}
var ongoing_players = []
var locked_players = []
var flags_copy = {}

var peers_in_minigame = {}


