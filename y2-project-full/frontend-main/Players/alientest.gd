extends Sprite2D


# Get the gravity from the project settings to be synced with RigidBody nodes.


func _ready():
		$AlienBar.show_percentage = false
		
		$AlienBar.min_value = 0  # Set the minimum value for the progress bar
		$AlienBar.max_value = 0  # Set the maximum value for the progress bar
		$AlienBar.value = 0  # Set the initial value for the progress bar
		
		
		$AstraunautBar.show_percentage = false
		$AstraunautBar.min_value = 0  # Set the minimum value for the progress bar
		$AstraunautBar.max_value = 0  # Set the maximum value for the progress bar
		$AstraunautBar.value = 0  # Set the initial value for the progress bar
		
		
		update_progress_bar(4,"alien")
		
		update_progress_bar(2,"astronaut")
		
		
		
func update_progress_bar(score,team):
	$AlienBar.max_value += score
	$AstraunautBar.max_value +=score
	
	if team =="alien":
		$AlienBar.value+=score
	if team== "astronaut":
		$AstraunautBar.value +=score


