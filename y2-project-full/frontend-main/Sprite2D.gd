extends Sprite2D

func _input(event):
	if event is InputEventMouseButton and event.is_pressed() and event.button_index == MOUSE_BUTTON_LEFT:
		var mouse_position = get_global_mouse_position()
		if get_rect().has_point(to_local(mouse_position)):
			print("clicked", self.get_parent())
			var game = get_tree().get_root().get_node("/root/minigame/")  # Replace with the path to your TileMap node
			game.mole_clicked(self.get_parent())
