#this is the script to define how the players interact with objects
extends Area2D

@export var interactor : Node2D
@export var interaction_action : StringName = "interact"

var selected_interactable : Interactable

signal display_instruction_text(data)

func _ready():
	#assign signals to methods
	area_entered.connect(_on_area_entered)
	area_exited.connect(_on_area_exited)
	
func _process(delta):
	pass
	
#will run when the player presses the interaction_action (in this case "F")
func _input(event):
	if(event.is_action_pressed(interaction_action)):
		#make sure there actually is a interactable object nearby
		if(selected_interactable != null):
			selected_interactable.interact(interactor)

#this will run if the player detects an interactive object
func _on_area_entered(object_area : Area2D):
	#"entered area")
	#print(object_area)
	if(object_area is Interactable):
		#print("is interactable")
		#make sure this variable is null to avoid overwritting data
		if(selected_interactable == null):
			selected_interactable = object_area
			#ADD TEXT TO SAY "PRESS F TO JOIN GAME"
		#	print("EMIT SIGNAL")
			display_instruction_text.emit(true)
			
#this will run if the interactive object leaves the player's area
func _on_area_exited(object_area : Area2D):
	#print("left area")
	if(object_area is Interactable):
		#print("is interactable")
		#will stop interactions (such as readying up for a minigame) if they leave the area radius   --- temporarily not in use as we figure out game logic
		#selected_interactable.stop_interaction(interactor)  --- temporarily not in use as we figure out game logic
		selected_interactable = null    #for now, we just need to set this to null since the interactive object is no longer in the area
		display_instruction_text.emit(false)
