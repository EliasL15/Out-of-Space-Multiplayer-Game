# Shooter Minigame

## Outline

Components:

* Players
  * Initially five hitpoints each
  * Lose one hitpoint every time a bullet hits (regardless of which team shot the bullet)
  * Feel gravity
  * Collide with walls without bouncing
  * Can jump (single only) and move left/right
* Bullets
  * Do not feel gravity
  * Move at a constant speed
  * Collide with walls and bounce
  * Annihilate with each other
  * Despawn upon hitting a player
  * Despawn five seconds after they are created
* Walls
  * Enclose playing space so that players/bullets cannot escape
  * Only ever perfectly horizontal or vertical

Win condition:

* When timer finishes
  * Winner is determined by which team has a higher hitpoint sum
  * If teams have the same hitpoint sum, the tie is broken (or not) by the global tiebreak rules
* When one team eliminates the other
  * Winner is the team that killed the other team
  * Game ends immediately so that remaining bullets cannot harm remaining players

## Protocol

### Game Start

Each client receives the following message.

```json
{
  "type": "shooter_welcome",
  "health_initial": 5,
  "your_spawn": {
    "x": 0,
    "y": 0
  },
  "peer_spawns": {
    "OpponentUsername": {
      "x": 50,
      "y": 0
    }
  }
}
```

### Player Updates

To avoid synchronisation issues and to keep things simple, physics simulation is left to each client. Each client is
responsible for simulating and reporting the behaviour of its own player and the bullets that they have fired.

```json
{
  "type": "shooter_physics_report",
  "player_position": {
    "x": 0,
    "y": 0
  },
  "arm_rotation": 15.0,
  "bullet_positions": [
    {
      "x": 5,
      "y": 5
    },
    {
      "x": 10,
      "y": 7
    }
  ]
}
```

This message should only be sent when there has been a change to one of the components. That is, a client should
_not_ send a physics report if all of the following are true.

* The player's position has not changed since the last physics report was sent;
* The player's arm has not changed rotation since the last physics report was sent; and
* There are no bullets belonging to this player.

This ensures that updates are only sent when they are actually needed.

When the server receives a physics report from one client, it will be forwarded to all other clients using a message of
the following format.

```json
{
  "type": "shooter_peer_physics_report",
  "their_name": "SomeUsername",
  "their_position": {
    "x": 0,
    "y": 0
  },
  "their_arm_rotation": 15.0,
  "their_bullets": [
    {
      "x": 5,
      "y": 5
    },
    {
      "x": 10,
      "y": 7
    }
  ]
}
```

### Post-death Messaging

When a player dies, their client should stop sending physics updates as soon as all of the player's bullets have
despawned. Any physics updates sent after the player has died should include only the list of bullets, and no other
information. (Position information is no longer useful because dead players are not considered to be on the map.) So, a
post-death physics update would look like this:

```json
{
  "type": "shooter_physics_report",
  "bullet_positions": [
    {
      "x": 5,
      "y": 5
    },
    {
      "x": 10,
      "y": 7
    }
  ]
}
```

The server will continue to forward these messages to peers, but here, too, the invalid fields are removed:

```json
{
  "type": "shooter_peer_physics_report",
  "their_name": "SomeUsername",
  "their_bullets": [
    {
      "x": 5,
      "y": 5
    },
    {
      "x": 10,
      "y": 7
    }
  ]
}
```

This means that clients should not expect that every peer physics report will have every field.

If the client of a dead player sends a physics report that includes fields that are no longer valid, this is treated as
an error (because it may suggest that the client is not aware that the player is dead). The server responds accordingly.

```json
{
  "type": "shooter_invalid_dead_physics_report"
}
```

### Bullet Events

For now, clients are responsible for detecting collisions between their own bullets and other players' bullets and
characters. This could be exploited by cheaters, but in the interest of simplicity we will ignore that risk at the moment.

#### Bullet-to-player Collisions

When a client detects a collision between one of its own bullets and another player, it should send the following
message to the server.

```json
{
  "type": "shooter_bullet_player_hit",
  "victim": "UsernameOfPlayerWhoGotHit"
}
```

The victim will then receive the following message from the server.

```json
{
  "type": "shooter_you_got_hit",
  "shooter": "UsernameOfPlayerWhoFiredTheBullet",
  "remaining_health": 4
}
```

If the game is being played with more than two players total (that is, if this is a 2v2 or 3v3 game) then all other
players will receive the following message from the server.

```json
{
  "type": "shooter_someone_shot_someone",
  "shooter": "UsernameOfPlayerWhoFiredTheBullet",
  "victim": "UsernameOfPlayerWhoGotHit",
  "victim_remaining_health": 4
}
```

Neither the shooter nor the victim will receive this message.

In a 1v1 game, the game simply ends as soon as one player's health reaches zero. However, with more than one player per
team, the game will only end when the final player's health reaches zero. When a player's health reaches zero but the
game does not end, the server will reject any incoming messages from them. They will continue to receive all messages
so that the player can watch until the end of the game.

When other clients see that a player's health has reached zero, they should stop checking for collisions with that
player. The server will emit the following error when a player is referred to after they have died.

```json
{
  "type": "shooter_player_dead_error",
  "name": "UsernameOfPlayerWhoIsDead"
}
```

#### Bullet-to-bullet Collisions

The server does not care about collisions between bullets. It is the responsibility of each client to check for
collisions between its own bullets and those belonging to others, and despawn locally as required. Clients should trust
that the client that owns the bullet being collided with will be doing the same thing, which would mean that both
bullets despawn.
