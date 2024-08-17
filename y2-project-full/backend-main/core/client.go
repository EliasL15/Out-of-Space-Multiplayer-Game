package core

import (
	"github.com/gorilla/websocket"
)

// A ClientMessageOut combines a message payload with a pointer to the client to whom it is being
// sent.
type ClientMessageOut struct {
	// C is the relevant client.
	C *Client

	// M is the message data.
	M []byte
}

// A Client represents a connection to the frontend.
type Client struct {
	// lobbyMgr points to the object which manages lobbies for this client.
	lobbyMgr *LobbyManager

	// Player is a pointer to the player object for this client, if there is one.
	//
	// A client can only have a player once the user is in a lobby.
	Player *Player

	// out is the channel along which outgoing messages are sent.
	out chan ClientMessageOut

	// Conn is the WebSocket connection for this client.
	conn *websocket.Conn
}

// Send encodes and sends m to the client.
func (c *Client) Send(m *Message) error {
	data, err := m.Encode()

	if err != nil {
		return err
	}

	// Combine the client and message and send.
	c.out <- ClientMessageOut{C: c, M: data}

	return nil
}

// doLobbyCreate handles a lobby creation message from the client.
func (c *Client) doLobbyCreate() error {
	// All validation happens further down the call chain.
	return c.lobbyMgr.HandleLobbyCreate(c)
}

// doLobbyJoin handles a lobby join message from the client.
func (c *Client) doLobbyJoin(message *Message) error {
	id, err := message.GetString("lobby_id")

	if err != nil {
		return err
	}

	// Get the lobby activity, which is responsible for lobby messaging.
	act := c.lobbyMgr.GetActivity(id)

	if act == nil {
		return c.Send(NewMessage("lobby_not_found"))
	}

	return act.HandleJoinRequest(c)
}

// Receive processes a message received from the client.
func (c *Client) Receive(m *Message) error {
	switch m.Type {
	case "lobby_create":
		return c.doLobbyCreate()

	case "lobby_join":
		return c.doLobbyJoin(m)
	}

	// If the client has a player, forward the message to their current activity.
	if c.Player != nil {
		return c.Player.Activity.HandleMessage(c.Player, m)
	}

	return c.Send(NewMessage("client_unknown_non_lobby_message"))
}
