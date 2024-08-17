# frontend
## eventLib documentation
### Import

EventLib is set to autoload in the project settings so just call
```python
EventLib.test_function()
```
to test if it is imported correctly.
### Lobby
#### `Lobby` object
##### Atributes
- ` playerTeam` 3D array storing `[[player name, team, ready status],...]`
- `lobbyId` Stores the Lobby Id of the lobby the client is connected to
##### Methods
- `search_player(name)` Takes in a player name as a string, returns position of that player in the list `playerTeam` else returns -1.
- `addPlayer(name, team)` Adds a player to the lobby object given the name and team.
- `removePlayer(name, team)` Removes a player from the lobby object given the name and team.
- `setReady (name, ready)` Changes ready status of a specific player given by it's name.
- `changeTeam (name):` Changes team of the player given by it's from 0 to 1 or from 1 to 0.
#### Lobby methods.
- `create_lobby()`: Creates a new `Lobby` object and sends a new lobby request to the server. Returns a `Lobby` object if succesful, otherwise an error.
- `join_lobby(code)`: Takes in a server code, returns a `Lobby` object if succesful, otherwise an error.
- `leave_lobby()`: Resets lobby object, sends `lobby_bye` request to server. Returns nothing.
- `client_team_change()`: Switches client's team from 0 to 1 or from 1 to 0.
- `client_ready_change(new_ready)`: Takes in boolean `true` or `false`, updates the player's ready status accordingly. Returns nothing.



