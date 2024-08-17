extends Button

@onready var code_text_field = get_node("CodeTF")

func _on_pressed():
	#clear text field
	code_text_field.clear()
	
	#go back to main menu
	get_tree().change_scene_to_file("res://Scenes/mainmenu.tscn")
	pass # Replace with function body.
