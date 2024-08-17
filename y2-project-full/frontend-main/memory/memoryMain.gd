extends MinigameBase

var spawn :Array = []
var clickable : Array = []
var nodes : Array = []
var clicked 
var card = []
var matched = false

var normal = preload("res://memory/img/white back.jpg")
var red = preload("res://memory/img/red.jpg")
var yellow = preload("res://memory/img/yellow.jpg")
var blue = preload("res://memory/img/blue.jpg")
var pink = preload("res://memory/img/pink.jpg")
var green = preload("res://memory/img/green.jpg")
var cyan = preload("res://memory/img/cyan.jpg")
var orange = preload("res://memory/img/orange.jpg")
var purple = preload("res://memory/img/purple.jpg")


###############################################################################


func on_message(data: Dictionary) -> bool:
	match data["type"]:

		"match_welcome":
			var duration = data["duration_seconds"]
		
		"match_tick":
			var time = data["seconds_left"]
			update_timer(time) 
			if time <=0:
				var nodes = get_children()
				for i in nodes:
					if not i is Timer:
						i.visible = false
				if get_child_count() >3:
					$Label2.text = "You Lose"
				else:
					$Label2.text = "You Win"
		
		"match_first_flip":
			if len(card) < 2:
				card.append(clicked)
				show_card(card[0], data["pattern"])
			
		"match_second_flip":
			if len(card) < 2:
				card.append(clicked)
				matched = data["match_found"]
				show_card(card[1], data["pattern"], matched)
		
		"match_flipped_twice_error":
			pass
		
		"match_already_matched_error":
			pass
		
		"match_flip_invalid_x_error":
			pass
		
		"match_flip_invalid_y_error":
			pass
		
		"match_unknown_message_type_error":
			pass


		# We have to return `false` if we didn't handle the message.
		_: return false

	return true

func on_minigame_end():
	# In this method you should do anything that you need to do before your minigame ends. For
	# example, if we had something running in the background, we'd want to stop it.
	pass

func on_ship_endgame():
	pass

func on_peer_left(_theirName: String):
	pass

func on_non_peer_left(_theirName: String):
	pass

################################################################################

	
# spawns all the cards, from x = 125 and y = 175
# x increments by 100 and y increments by 150
# anchor is on the middle

func _ready():
	for i in range (0,4):
		var y = 175
		y += i*150
		var x = 125
		for j in range(0,8):
			x += 100
			spawn.append(x)
			spawn.append(y)
			clickable.append(Rect2(x-36,y-64,72,128))
	Input.set_mouse_mode(Input.MOUSE_MODE_VISIBLE)
	for i in range (0,64, 2):
		var node = Node2D.new()
		var sfx = Sprite2D.new()
		sfx.texture = normal
		sfx.scale = Vector2(0.1,0.1)
		node.position.x = spawn[i]
		node.position.y = spawn[i+1]
		node.add_child(sfx)
		add_child(node)
	for i in get_children():
		if i is Node2D:
			nodes.append(i)


# will send request if mouse click is on a card
# probably a bunch of if else for what the card the server tells it is
# queue_free() is server returns true
func _input(event):
	if event is InputEventMouseButton and event.pressed:
		if event.button_index == MOUSE_BUTTON_LEFT:
			var mouse_pos = get_global_mouse_position()
			for i in range(0, clickable.size(), 1):
				if clickable[i].has_point(mouse_pos):
					if len(card) < 2:
						EventLib.message_send({
							"type": "match_flip",
							"card_x": i%8,
							"card_y": i/8
						})
						if len(nodes[i+1].get_children()) > 0:
							clicked = nodes[i+1].get_children()[0]

# update the timer, will send request (change delta?)func _process(_delta):
func update_timer(seconds_left):
	$Label.text =  "%02d" %seconds_left

func show_card(clicked, pattern, matched = false):
	if pattern == 0:
		clicked.texture = red
	elif pattern == 1:
		clicked.texture = yellow
	elif pattern == 2:
		clicked.texture = blue
	elif pattern == 3:
		clicked.texture = pink
	elif pattern == 4:
		clicked.texture = green
	elif pattern == 5:
		clicked.texture = cyan
	elif pattern == 6:
		clicked.texture = orange
	elif pattern == 7:
		clicked.texture = purple
	
	if matched == true:
		$TimerDel.start()
	elif matched == false and len(card)==2:
		revert(matched)
		
func revert(matched):
		$TimerNorm.start()

func _on_timer_del_timeout():
		card[1].queue_free()
		card[0].queue_free()
		card = []
		matched = false

func _on_timer_norm_timeout():
	if len(card) == 2:
		card[1].texture = normal
		card[0].texture = normal
		card = []
