#this is the script for the interactable class

class_name Interactable #makes it easier to see which nodes extend this class
extends Area2D

func interact(user: Node2D):
	#push a warning as the interactable object does not have it's own interact method defined so it is resorting to the base interact method
	push_warning("Not implemented")
	
#func stop_interaction(user: Node2D):   --- temporarily not in use as we figure out game logic
	#push warning if reached this base method as each subclass should have it's own version of this method  
#	push_warning("Not implemented")
	


