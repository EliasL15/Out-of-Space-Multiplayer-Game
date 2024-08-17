extends Node2D



var timer : Timer


	
func _on_timer_timeout():
	queue_free()
	get_tree().quit()

