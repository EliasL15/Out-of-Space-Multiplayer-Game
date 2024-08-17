extends Label



# Called when the node enters the scene tree for the first time.
var time_left = 60

func _ready():
	update_timer_label()
	start_timer()
	
func start_timer():
	while time_left >0:
		await get_tree().create_timer(1.0).timeout
		time_left -= 1
		update_timer_label()
		
func update_timer_label():
	text = str(time_left).pad_zeros(2)
