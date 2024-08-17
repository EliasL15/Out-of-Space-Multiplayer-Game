package core

import (
	"encoding/json"
	"fmt"
	"math"
)

// A Message is a transmission between a client and the server, in either direction.
type Message struct {
	// Type is the `type` field from the message object.
	Type string

	// payload is a map containing the parsed payload without the `type` field.
	payload map[string]interface{}
}

// NewMessage returns a pointer to a new message with the given type.
func NewMessage(typ string) *Message {
	return &Message{
		Type:    typ,
		payload: map[string]interface{}{},
	}
}

// ParseMessage attempts to parse the given data into a Message.
func ParseMessage(data []byte) (*Message, bool) {
	var parsed map[string]interface{}

	if json.Unmarshal(data, &parsed) != nil {
		// We don't care what the error is. We just know that the message is invalid.
		return nil, false
	}

	typeVal, ok := parsed["type"]

	if !ok {
		// Missing "type" field.
		return nil, false
	}

	typeStr, ok := typeVal.(string)

	if !ok {
		// "type" field is not a string.
		return nil, false
	}

	// Delete the "type" field so we get just the payload.
	delete(parsed, "type")

	return &Message{
		Type:    typeStr,
		payload: parsed,
	}, true
}

// Encode turns the message into something suitable for sending over the network.
func (msg *Message) Encode() ([]byte, error) {
	// Ensure "type" field was not set in payload.
	if existingType, ok := msg.payload["type"]; ok {
		return nil, fmt.Errorf("found 'type' in payload: '%v'", existingType)
	}

	copied := map[string]interface{}{"type": msg.Type}

	// Copy the message body into our new object.
	for k, v := range msg.payload {
		copied[k] = v
	}

	return json.Marshal(copied)
}

// Add adds the given key-value pair to the message payload and returns a pointer to the
// message again.
func (msg *Message) Add(key string, value interface{}) *Message {
	msg.payload[key] = value
	return msg
}

// TryGet returns a pointer to the value for the given field in the message payload,
// or nil if the field does not exist.
func (msg *Message) TryGet(key string) *interface{} {
	if v, ok := msg.payload[key]; ok {
		return &v
	}

	return nil
}

// GetString finds the value for the given field and casts it to a string.
// It returns an error if the key is not found or if the cast fails.
func (msg *Message) GetString(key string) (string, error) {
	v := msg.TryGet(key)

	if v == nil {
		return "", fmt.Errorf("key %v does not exist", key)
	}

	if s, ok := (*v).(string); ok {
		return s, nil
	} else {
		return "", fmt.Errorf("cannot convert '%v' value %v to string", key, *v)
	}
}

// GetNumber finds the value for the given field and casts it to a float64.
// It returns an error if the key is not found or if the cast fails.
func (msg *Message) GetNumber(key string) (float64, error) {
	v := msg.TryGet(key)

	if v == nil {
		return 0, fmt.Errorf("key %v does not exist", key)
	}

	if f, ok := (*v).(float64); ok {
		return f, nil
	} else {
		return 0, fmt.Errorf("cannot convert '%v' value %v to float64", key, *v)
	}
}

// GetInt finds the value for the given field and casts it to an integer.
// It returns an error if the key is not found or if the value found cannot be converted to an
// integer.
func (msg *Message) GetInt(key string) (int, error) {
	float, floatErr := msg.GetNumber(key)

	if floatErr != nil {
		return 0, floatErr
	}

	// Make sure we're not losing data.
	if math.Trunc(float) != float {
		return 0, fmt.Errorf("float %v is not an integer", float)
	}

	return int(float), nil
}
