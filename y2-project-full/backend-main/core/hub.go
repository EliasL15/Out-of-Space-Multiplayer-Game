package core

import (
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"reflect"
	"runtime"
	"time"
)

var Logger = zap.Must(zap.NewDevelopment())

// A clientMessageIn represents a message received from a client.
type clientMessageIn struct {
	// c is the client which sent the message.
	c *Client

	// m is the parsed message payload.
	m *Message
}

// A Scheduler object can be used to execute code on the main thread.
type Scheduler struct {
	// events is the same event channel as used by the hub.
	event chan func() error
}

// Add schedules the given function to be called from the main thread.
func (s Scheduler) Add(fn func() error) {
	s.event <- fn
}

// A FunctionTimer retains information about a scheduler timer.
type FunctionTimer struct {
	// end is the time at which the timer will expire.
	End time.Time

	// quit can be used to end the timer prematurely. If a non-nil function pointer is provided,
	// that function will be called in place of the normal end function.
	quit chan func() error
}

// ExpiredTimer returns an expired timer.
func ExpiredTimer() FunctionTimer {
	return FunctionTimer{
		End:  time.Now(),
		quit: nil,
	}
}

// TickingTimer returns a function timer that calls `fn` every `interval` and then `onNormalEnd` at
// `end`. All calls are made using the given scheduler, which means that calls will likely not be
// made exactly on time.
func TickingTimer(
	s Scheduler,
	end time.Time,
	interval time.Duration,
	fn func() error,
	onNormalEnd func() error,
) FunctionTimer {
	quit := make(chan func() error)

	// Start a goroutine that uses a ticker to schedule fn to run regularly.
	go func() {
		ticker := time.NewTicker(interval)

		for {
			select {
			// Stop the ticker when we receive anything on the quit channel. If we don't
			// explicitly call Stop, the garbage collector won't clear up the ticker's resources.
			case f := <-quit:
				ticker.Stop()

				if f != nil {
					s.Add(f)
				}

				return

			case _ = <-ticker.C:
				// Schedule the function to run.
				s.Add(fn)
			}
		}
	}()

	// Start another goroutine that wakes up at the end time and tells the ticker to stop.
	go func() {
		time.Sleep(end.Sub(time.Now()))
		quit <- onNormalEnd
	}()

	return FunctionTimer{
		End:  end,
		quit: quit,
	}
}

// SingleTimer returns a function timer that will call `fn` on `s` at `end` unless ended
// prematurely.
func SingleTimer(
	s Scheduler,
	end time.Time,
	fn func() error,
) FunctionTimer {
	quit := make(chan func() error)

	go func() {
		timer := time.NewTimer(end.Sub(time.Now()))

		select {
		// If the timer ends naturally, call the function we were given to start with.
		case _ = <-timer.C:
			s.Add(fn)

		// If we're asked to end it prematurely, call the function we've been given (if any).
		case f := <-quit:
			// Stop the timer so we don't make a second call.
			timer.Stop()

			if f != nil {
				s.Add(f)
			}
		}
	}()

	return FunctionTimer{
		End:  end,
		quit: quit,
	}
}

// HasEnded returns true if the current time is past the timer's end time or if the timer has been
// stopped prematurely.
func (timer FunctionTimer) HasEnded() bool {
	return timer.quit == nil || time.Now().After(timer.End)
}

// WasStopped returns true if and only if the timer was forced to end.
func (timer FunctionTimer) WasStopped() bool {
	return timer.quit == nil
}

// TimeLeft returns the amount of time left on the timer, or zero if the timer has ended.
func (timer FunctionTimer) TimeLeft() time.Duration {
	if timer.quit == nil {
		return 0
	}

	now := time.Now()

	if timer.End.Before(now) {
		// Never return a negative time.
		return 0
	}

	return timer.End.Sub(now)
}

// StopWith ends the timer prematurely by calling the given function.
func (timer FunctionTimer) StopWith(fn func() error) {
	if timer.HasEnded() {
		return
	}

	timer.quit <- fn
	timer.quit = nil
}

// Stop ends the timer prematurely without calling any function.
func (timer FunctionTimer) Stop() {
	timer.StopWith(nil)
}

// The Hub is responsible for sending messages to clients and notifying activities of events.
type Hub struct {
	// lobbyMgr is the lobby manager used for all clients managed by this hub.
	lobbyMgr *LobbyManager

	// out is the channel along which outbound messages are sent.
	out chan ClientMessageOut

	// event is the channel along which event functions are sent.
	event chan func() error

	// in is the channel along which inbound messages are sent.
	in chan clientMessageIn

	// kill is the channel along which we can send a client pointer in order to kill that client.
	kill chan *Client
}

// NewHub returns a new hub with no lobbies.
func NewHub(minigames map[string]MinigamePrototype) *Hub {
	Logger.Info("creating hub")

	event := make(chan func() error, 10)

	return &Hub{
		lobbyMgr: NewLobbyManager(
			Scheduler{
				event: event,
			},

			minigames,
		),

		out:   make(chan ClientMessageOut, 10),
		event: event,
		in:    make(chan clientMessageIn, 10),
		kill:  make(chan *Client, 20),
	}
}

func (hub *Hub) logQueueLengths() {
	if len(hub.in) > 1 {
		Logger.Info("hub is n incoming msg(s) behind", zap.Int("n", len(hub.in)))
	}

	if len(hub.out) > 1 {
		Logger.Info("hub is n outgoing msg(s) behind", zap.Int("n", len(hub.out)))
	}

	if len(hub.event) > 1 {
		Logger.Info("hub is n event(s) behind", zap.Int("n", len(hub.event)))
	}

	if len(hub.kill) > 1 {
		Logger.Info("hub has n clients waiting for death", zap.Int("n", len(hub.kill)))
	}
}

// handleIncoming handles an incoming message from a client.
func handleIncoming(msg clientMessageIn) {
	l := Logger.With(
		zap.Stringer("from", msg.c.conn.RemoteAddr()),
		zap.Any("msg", *msg.m),
	)

	l.Debug("handling incoming message")

	err := msg.c.Receive(msg.m)

	if err == nil {
		return
	}

	l.Error("error handling incoming message", zap.Error(err))
}

// handleEvent handles an event.
func handleEvent(fn func() error) {
	// See https://stackoverflow.com/a/68857530

	// Get the function address (i.e. the PC value for the first instruction).
	pc := reflect.ValueOf(fn).Pointer()

	// Get the filename and line number for the function.
	filename, line := runtime.FuncForPC(pc).FileLine(pc)

	l := Logger.With(
		zap.String("file", filename),
		zap.Int("line", line),
	)

	l.Debug("calling event function")

	// Call the event function.
	err := fn()

	if err == nil {
		return
	}

	l.Error("error from event function", zap.Error(err))
}

// killClient kills the given client.
func killClient(client *Client) {
	l := Logger.With(zap.Stringer("addr", client.conn.RemoteAddr()))

	l.Info("killing client")

	// Close the connection. This should cause the listening goroutine to exit if it
	// hasn't already.
	err := client.conn.Close()

	if err != nil {
		l.Warn("error closing client connection", zap.Error(err))
	}

	if client.Player == nil {
		// logger.Infoln("client died outside lobby")

		return
	}

	l.Info("dead client was inside lobby")

	// Simulate a soft leave.
	err = client.Receive(NewMessage("lobby_bye"))

	if err != nil {
		l.Error("error handling lobby_bye", zap.Error(err))
	}

	if client.Player != nil {
		l.Panic("lobby_bye did not remove player pointer")
	}
}

// runMainGameLoop runs the loop which processes incoming messages,
// kills clients and calls event functions.
func (hub *Hub) runMainGameLoop() {
	Logger.Info("starting main loop")

	for {
		hub.logQueueLengths()

		select {
		case msg := <-hub.in:
			handleIncoming(msg)

		case fn := <-hub.event:
			handleEvent(fn)

		case client := <-hub.kill:
			killClient(client)
		}
	}
}

// runOutboundLoop runs the loop which sends outbound messages to the frontend.
func (hub *Hub) runOutboundLoop() {
	for {
		msg := <-hub.out

		l := Logger.With(
			zap.Stringer("addr", msg.C.conn.RemoteAddr()),
			zap.ByteString("msg", msg.M),
		)

		l.Debug("sending message")

		err := msg.C.conn.WriteMessage(websocket.TextMessage, msg.M)

		if err == nil {
			continue
		}

		l.Error("error sending message", zap.Error(err))

		// If we failed to write, we assume the client is unreachable and kill it.
		hub.kill <- msg.C
	}
}

// Start starts running hub processes in the background.
func (hub *Hub) Start() {
	go hub.runMainGameLoop()
	go hub.runOutboundLoop()
}

// clientListen starts a reading loop for client.
// When a message is read from the client's WebSocket,
// it will be parsed and sent along the inbound message channel for processing by the hub.
func (hub *Hub) clientListen(client *Client) {
	for {
		mt, body, err := client.conn.ReadMessage()

		l := Logger.With(
			zap.Stringer("addr", client.conn.RemoteAddr()),
			zap.Int("type", mt),
		)

		if err != nil {
			l.Error("error reading message", zap.Error(err))

			// Fatal error (to the client). Kill the client and quit the loop.
			hub.kill <- client

			return
		}

		if mt != websocket.TextMessage {
			l.Error("message was not text")

			// Protocol-level error only.
			_ = client.Send(NewMessage("ws_non_text_error"))

			continue
		}

		l = l.With(zap.ByteString("body", body))

		l.Debug("read message")

		msg, ok := ParseMessage(body)

		if !ok {
			l.Error("invalid message JSON")

			// Protocol-level error.
			_ = client.Send(NewMessage("ws_json_format_error"))

			continue
		}

		hub.in <- clientMessageIn{
			c: client,
			m: msg,
		}
	}
}

// AddConnection creates a client for the given WebSocket connection and starts interacting with it.
func (hub *Hub) AddConnection(ws *websocket.Conn) {
	Logger.Info("adding client", zap.Stringer("addr", ws.RemoteAddr()))

	client := &Client{
		lobbyMgr: hub.lobbyMgr,
		Player:   nil,
		out:      hub.out,
		conn:     ws,
	}

	go hub.clientListen(client)
}
