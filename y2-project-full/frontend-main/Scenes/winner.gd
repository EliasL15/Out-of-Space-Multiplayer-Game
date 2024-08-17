extends Control

@onready var lbl = get_node("winningTeamLbl")
@onready var increase = true
@onready var pulse_speed = 90.0

func _process(delta):
	var speed_adjusted = pulse_speed * delta
	print("speed_adjusted:"+str(speed_adjusted))
	if increase:
		lbl.set("theme_override_font_sizes/font_size", lbl.get("theme_override_font_sizes/font_size")+(speed_adjusted))
		if lbl.get("theme_override_font_sizes/font_size") > 100:
			increase = false
	else:
		lbl.set("theme_override_font_sizes/font_size", lbl.get("theme_override_font_sizes/font_size")-(speed_adjusted))
		if lbl.get("theme_override_font_sizes/font_size") < 90:
			increase = true
