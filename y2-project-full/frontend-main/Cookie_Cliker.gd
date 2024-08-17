extends Sprite2D

var score = 0
	
func _input(event):
	if event is InputEventMouseButton and event.is_pressed() and event.button_index == MOUSE_BUTTON_LEFT:
		var mouse_position = get_global_mouse_position()
		if get_rect().has_point(to_local(mouse_position)):
			print("clicked")
			score+=1
			$ClickerScore.text = "Score: " + str(score)
			var game = self.get_parent()
			game.update_score(score)
			
			

