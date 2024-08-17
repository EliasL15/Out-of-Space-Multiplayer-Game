extends Node


# Called when the node enters the scene tree for the first time.
func _ready():
	var sfx_die = load("res://FlappyBirdAssets/audio/sfx_die.ogg")
	var sfx_hit = load("res://FlappyBirdAssets/audio/sfx_hit.ogg")
	var sfx_point = load("res://FlappyBirdAssets/audio/sfx_point.ogg")
	var sfx_swooshing = load("res://FlappyBirdAssets/audio/sfx_swooshing.ogg")
	var sfx_wing = load("res://FlappyBirdAssets/audio/sfx_wing.ogg")
	
	$sfx_die.stream = sfx_die
	$sfx_hit.stream = sfx_hit
	$sfx_point.stream = sfx_point
	$sfx_swooshing.stream = sfx_swooshing
	$sfx_wing.stream = sfx_wing
	
	$sfx_wing.play()
	$sfx_hit.play()
	$sfx_point.play()
	$sfx_swooshing.play()
	$sfx_wing.play()
	pass # Replace with function body.


# Called every frame. 'delta' is the elapsed time since the previous frame.

	
	
