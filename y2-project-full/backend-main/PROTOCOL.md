# Protocol

Forget everything you knew about "player IDs". Players are identified by their name. When a
client sends a message they do not need to include their name, because the server already knows
who they are.

## Activities

An "activity" might be

* The lobby;
* The main game (the spaceship); or
* A minigame.

Each activity has a frontend and backend component. The backend component is responsible for
handling messages from the client. In all activities, players are identified by name.

## "Lobby" Activity

The "lobby" activity actually includes the time before the user joins a lobby, and
the time when the user is in a lobby. It ends when the game begins, at which
point all players are changed to the `"main_game"` activity.

### Creating a Lobby

Client sends

```json
{
  "type": "lobby_create"
}
```

Server sends back

```json
{
  "type": "lobby_welcome",
  "your_name": "RandomUsernameGeneratedByServer",
  "your_team": 0,
  "lobby_id": "abcd1234"
}
```

### Joining a Lobby

Client sends

```json
{
  "type": "lobby_join",
  "lobby_id": "abcd1234"
}
```

Server sends back

```json
{
  "type": "lobby_welcome",
  "your_name": "OtherUser",
  "your_team": 1,
  "lobby_id": "abcd1234",
  "peer_teams": {
    "RandomUsernameGeneratedByServer": 0
  }
}
```

or

```json
{
  "type": "lobby_not_found"
}
```

or

```json
{
  "type": "lobby_full"
}
```

Server also notifies peers:

```json
{
  "type": "lobby_peer_joined",
  "their_name": "OtherUser",
  "their_team": 1
}
```

### Changing Team

Client sends

```json
{
  "type": "lobby_team_change",
  "team": 1
}
```

Server sends to all other players

```json
{
  "type": "lobby_peer_team_change",
  "their_name": "OtherUser",
  "team": 1
}
```

### Leaving a Lobby

Leaving can be implied by closing tab (which closes websocket connection) or by the user
actively choosing to click a button to leave the lobby.

In the second case, client sends

```json
{
  "type": "lobby_bye"
}
```

Server sends nothing back.

In either case, server notifies peers:

```json
{
  "type": "lobby_peer_left",
  "their_name": "OtherUser"
}
```

If the lobby is now empty, the server will delete it.

### Readiness

For the game to begin, all six players must mark themselves as "ready". A toggle should be
provided for this. When that toggle is used, client sends

```json
{
  "type": "lobby_ready_change",
  "ready": true
}
```

(Of course, `"ready"` would be `false` if the player marked themselves as not ready.)

When a player updates their ready status, there are two possible outcomes:

* If after the change, all six players are marked as ready, the game begins immediately.
* Otherwise, the peers will be notified of the change, as below.

To report a readiness change, server sends

```json
{
  "type": "lobby_peer_ready_change",
  "their_name": "OtherUser",
  "ready": true
}
```

A player **cannot** be ready if the teams are not ready – that is, individual players cannot
assert that they are ready to begin if the teams are not set up as 3v3. If a player was marked
as ready and the teams change (either by them changing team or by another player changing team),
the client should force their ready status back to not ready.

When all six players are ready, the activity will change to `"main_game"`.

## Database operations

For security reasons, we don't want to connect directly to a database from the frontend. We can use
the library send a request to the server to access the database.

### Leaderboard

When the user accesses the leaderboard a simple request will be sent:

```json
{
  "type": "leaderboard_get"
}
```

This will execute a `SELECT * FROM ...` on the database.

The data is then sorted by score and the name is extracted for each MVP of the lobby on the server
and the request is sent:

```json
{
  "type": "leaderboard_data",
  "data": [
    {
      "mvpScore": "3000",
      "mvpName": "test"
    },
    {
      "mvpScore": "2800",
      "mvpName": "player1"
    },
    {
      "mvpScore": "2200",
      "mvpName": "player2"
    },
    {
      "mvpScore": "2000",
      "mvpName": "player3"
    },
    {
      "mvpScore": "1900",
      "mvpName": "player4"
    },
    "...",
    {
      "mvpScore": "560",
      "mvpName": "player24"
    }
  ]
}
```

Then the leaderboard can be shown to the user.

The rest of the database operations on the server, e.g. storing playtest data can be done server
side.

## "Main Game" Activity

The ship serves as a hub which allows players to access minigame activities. The players will be
in the ship for a set amount of time. This will likely be only a few minutes. We will use ten
minutes as an example here, but clients should not assume that ten minutes will be used in reality.

### Starting/Rejoining the Ship

When the players enter the ship from the lobby, all clients will receive the following message.

```json
{
  "type": "ship_welcome",
  "your_spawn": {
    "x": 5.0,
    "y": -2.0
  },
  "peer_spawns": {
    "SomeName": {
      "x": 99.8,
      "y": 42.3
    },
    "BlahName": {
      "x": -22,
      "y": -33.4
    },
    "CoolName": {
      "x": 80,
      "y": 40
    }
  },
  "flags": {
    "some_flag_id": {
      "pos": {
        "x": 14,
        "y": 14
      },
      "minigame": "shooter",
      "worth": 10,
      "cooldown": 5
    },
    "some_other_flag_id": {
      "pos": {
        "x": 0,
        "y": 0
      },
      "minigame": "rock_paper_scissors",
      "worth": 5,
      "cooldown": 10
    }
  }
}
```

When a minigame finishes, the player(s) who was/were participating in it will be put back into 
the ship. While in a minigame, players do not receive messages that are relevant only for 
players who are in the ship. This means that when leaving a minigame and coming back to the ship,
players will receive a message of the following format to bring them back up to date.

```json
{
  "type": "ship_welcome_back",
  "your_spawn": {
    "x": 19,
    "y": 0
  },
  "peer_positions": {
    "SomeName": {
      "x": 90,
      "y": 40.1
    }
  },
  "flag_states": {
    "some_flag_id": {
      "capture_team": 0,
      "cooldown_left": 5.4
    },
    "some_other_flag_id": {
      "ongoing_players": ["BlahName", "CoolName"]
    },
    "yet_another_flag_id": {
      "capture_team": 1,
      "locked_players": ["SomeName"]
    }
  }
}
```

The player is given a spawn position (`"your_spawn"`). They are also told the positions of the 
players who are currently in the ship (`"peer_positions"`). Entries are not included for players 
who are not in the ship (that is, players currently in minigames). A map, `"flag_states"`, gives 
information on which teams own which flags. In the above example, `"some_flag_id"` has been 
captured by team `0` and has a remaining cooldown time of `5.4`. If a flag 
has a 
game ongoing, 
an array of the players in that game will be given – above we see that `"some_other_flag_id"` 
has an ongoing game between players `"BlahName"` and `"CoolName"`. If a flag does not yet have a 
game ongoing, but has been activated, all players locked to it will be listed under 
`"locked_players"`. For instance, notice above that `"SomeName"` is locked to 
`"yet_another_flag_id"`. The flag states map will not include entries for flags with no capture 
team, no active cooldown, no locked players and no ongoing game.

Players who are already in the ship will receive a message of the following format when a peer 
rejoins.

```json
{
  "type": "ship_welcome_back_peer",
  "their_name": "ThePlayer",
  "spawn": {
    "x": 19,
    "y": 0
  }
}
```

### While in the Ship

#### Flags

A flag is an in-game object that players can interact with. Flags act like gateways to minigames.
To capture a flag, the associated minigame must be won.

Each flag is referred to with a unique ID. These IDs are first given in the `ship_welcome` message.

##### Activation

Any player can attempt to activate a flag. To do this, they must press a certain key. The client
should send the following message whenever the key is pressed.

```json
{
  "type": "ship_flag_activate"
}
```

The server will use the last known position of the player to find the nearest flag. (For this
reason it is important that the client ensures that the server-side position of the player is
up-to-date before issuing an activation request.) If the server determines that there are no
flags within reach of the player, it will send back the following message.

```json
{
  "type": "ship_no_flags_in_reach"
}
```

If the player is locked to a game already (read below in "Player Locking" for details), the
following message will be sent.

```json
{
  "type": "ship_player_locked"
}
```

From this point onwards, activation responses include the ID of the flag that was identified as
being the nearest.

If the nearest flag already belongs to the team of the player who attempted to activate it, the
following message will be sent.

```json
{
  "type": "ship_flag_already_captured",
  "flag_id": "abcd1234"
}
```

If the nearest flag is currently in use (that is, if there is an instance of the associated
minigame ongoing) the following message will be sent.

```json
{
  "type": "ship_flag_in_use",
  "flag_id": "abcd1234"
}
```

If the flag is currently cooling down, the following message will be sent.

```json
{
  "type": "ship_flag_cooling_down",
  "flag_id": "abcd1234"
}
```

The remaining cooldown time is not included. See the section below on cooldowns to see how this
information is received.

If the flag has already been activated, the following message will be sent.

```json
{
  "type": "ship_flag_already_activated",
  "flag_id": "abcd1234"
}
```

If none of the above cases apply, the player will become locked to the flag. The protocol for 
this is detailed below.

##### Player Locking

A player can be "locked" to either one or zero flags at a time. If a player is not locked to any 
flag, they are allowed to attempt to activate any flag. (Although the activation may not succeed,
as we will discuss later.) If a player is locked to a flag, they are committed to playing the 
game for that flag. The game will begin once enough players are locked to that same flag for the 
minigame to have enough players.

All players inside the ship are able to run around freely, regardless of whether they are 
locked to a flag or not.

Locking is skipped for single-player games, because we do not need to wait for other players to 
become available.

When a player becomes locked to a flag, they will receive the following message.

```json
{
  "type": "ship_player_lock_set",
  "flag_id": "abcd1234"
}
```

Peers will receive the following message.

```json
{
  "type": "ship_peer_lock_set",
  "their_name": "OtherUser",
  "flag_id": "abcd1234"
}
```

Players become unlocked when they reenter the ship after finishing a game. This can be inferred 
by all clients without needing an explicit message.

##### Cooldowns

Each flag has a cooldown period. This period begins when the flag minigame finishes. During the 
cooldown period, the flag cannot be activated by members of either team.

Syncing clocks over a network is a notoriously tricky problem, so this protocol will simply take
a "good enough" approach to communicating cooldown information to clients.

Clients should assume that the cooldown begins as soon as the `"ship_minigame_finished"`
message is received. From this point onwards, clients should only update the remaining cooldown
time upon reception of the following message.

```json
{
  "type": "ship_flag_cooldown_tick",
  "flag_id": "abcd1234",
  "time_left": 9.2
}
```

`"time_left"` will always be less than the original `"cooldown"` value. The final cooldown
update will have a `"time_left"` value of `0`. Clients should treat the cooldown as having
finished as soon as this message is received. There is no specific "cooldown end" message.

The time units used will likely be seconds on the server-side, but the imprecise nature of this
system means that clients should treat the units as unknown and irrelevant.

#### Movement

Clients should send messages periodically to notify the server of updates to their players'
positions. A position update sent from a client to the server is as follows.

```json
{
  "type": "position_update",
  "x": 7.0,
  "y": -3.0
}
```

Other clients will then receive a notification from the server so that they may update their
local positions for the player who moved. This notification is as follows.

```json
{
  "type": "peer_position_update",
  "their_name": "OtherUser",
  "x": 7.0,
  "y": -3.0
}
```

If the server detects an issue with a player's position, it may choose to reset their position
to a certain value. A position reset message is as follows.

```json
{
  "type": "position_reset",
  "x": 7.0,
  "y": -5.0
}
```

Other clients will be notified as follows.

```json
{
  "type": "peer_position_reset",
  "their_name": "OtherUser",
  "x": 7.0,
  "y": -5.0
}
```

#### Minigame Entry and Exit

The server starts a minigame as soon as enough players are locked to the flag for that minigame. 
(For single-player games, the minigame will start as soon as the player activates the flag.)

When a player is placed into a minigame, the server will send their client a message of the 
following format.

```json
{
  "type": "ship_minigame_join",
  "flag_id": "abcd1234",
  "peers": [
    "SomeUsername"
  ]
}
```

Players who are remaining in the ship (i.e. those who are not joining this minigame) will 
receive a message of the following format for every player who is joining the minigame.

```json
{
  "type": "ship_peer_minigame_join",
  "flag_id": "abcd1234",
  "their_name": "OtherUser"
}
```

When a minigame finishes, all clients will receive the following message.

```json
{
  "type": "ship_minigame_finished",
  "flag_id": "abcd1234",
  "winning_team": 0
}
```
