package ocpp

import (
	"encoding/json"
	"fmt"
)

// MessageEncoder provides utilities for encoding OCPP messages
type MessageEncoder struct{}

// NewMessageEncoder creates a new message encoder
func NewMessageEncoder() *MessageEncoder {
	return &MessageEncoder{}
}

// EncodeCall encodes a Call message to JSON bytes
func (e *MessageEncoder) EncodeCall(action string, payload interface{}) ([]byte, error) {
	call, err := NewCall(action, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create call: %w", err)
	}

	return call.ToBytes()
}

// EncodeCallResult encodes a CallResult message to JSON bytes
func (e *MessageEncoder) EncodeCallResult(uniqueID string, payload interface{}) ([]byte, error) {
	callResult, err := NewCallResult(uniqueID, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create call result: %w", err)
	}

	return callResult.ToBytes()
}

// EncodeCallError encodes a CallError message to JSON bytes
func (e *MessageEncoder) EncodeCallError(uniqueID string, errorCode ErrorCode, errorDesc string, errorDetails interface{}) ([]byte, error) {
	callError, err := NewCallError(uniqueID, errorCode, errorDesc, errorDetails)
	if err != nil {
		return nil, fmt.Errorf("failed to create call error: %w", err)
	}

	return callError.ToBytes()
}

// MessageDecoder provides utilities for decoding OCPP messages
type MessageDecoder struct{}

// NewMessageDecoder creates a new message decoder
func NewMessageDecoder() *MessageDecoder {
	return &MessageDecoder{}
}

// Decode decodes a raw OCPP message and returns the appropriate type
func (d *MessageDecoder) Decode(data []byte) (interface{}, error) {
	return ParseMessage(data)
}

// DecodeCall decodes a Call message and unmarshals its payload into the provided type
func (d *MessageDecoder) DecodeCall(data []byte, payloadDest interface{}) (*Call, error) {
	msg, err := ParseMessage(data)
	if err != nil {
		return nil, err
	}

	call, ok := msg.(*Call)
	if !ok {
		return nil, fmt.Errorf("expected Call message, got %T", msg)
	}

	if payloadDest != nil {
		if err := json.Unmarshal(call.Payload, payloadDest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
	}

	return call, nil
}

// DecodeCallResult decodes a CallResult message and unmarshals its payload into the provided type
func (d *MessageDecoder) DecodeCallResult(data []byte, payloadDest interface{}) (*CallResult, error) {
	msg, err := ParseMessage(data)
	if err != nil {
		return nil, err
	}

	callResult, ok := msg.(*CallResult)
	if !ok {
		return nil, fmt.Errorf("expected CallResult message, got %T", msg)
	}

	if payloadDest != nil {
		if err := json.Unmarshal(callResult.Payload, payloadDest); err != nil {
			return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
		}
	}

	return callResult, nil
}

// DecodeCallError decodes a CallError message
func (d *MessageDecoder) DecodeCallError(data []byte) (*CallError, error) {
	msg, err := ParseMessage(data)
	if err != nil {
		return nil, err
	}

	callError, ok := msg.(*CallError)
	if !ok {
		return nil, fmt.Errorf("expected CallError message, got %T", msg)
	}

	return callError, nil
}

// ValidateMessage validates that a raw message conforms to OCPP message structure
func ValidateMessage(data []byte) error {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return fmt.Errorf("invalid JSON array: %w", err)
	}

	if len(arr) < 3 {
		return fmt.Errorf("message array too short: expected at least 3 elements, got %d", len(arr))
	}

	// Parse message type
	var msgType MessageType
	if err := json.Unmarshal(arr[0], &msgType); err != nil {
		return fmt.Errorf("invalid message type: %w", err)
	}

	// Validate based on message type
	switch msgType {
	case MessageTypeCall:
		if len(arr) != 4 {
			return fmt.Errorf("Call message must have 4 elements, got %d", len(arr))
		}
	case MessageTypeCallResult:
		if len(arr) != 3 {
			return fmt.Errorf("CallResult message must have 3 elements, got %d", len(arr))
		}
	case MessageTypeCallError:
		if len(arr) != 5 {
			return fmt.Errorf("CallError message must have 5 elements, got %d", len(arr))
		}
	default:
		return fmt.Errorf("unknown message type: %d", msgType)
	}

	return nil
}

// PrettyPrint returns a pretty-printed JSON string of any OCPP message
func PrettyPrint(msg interface{}) (string, error) {
	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}
	return string(data), nil
}

// CompactPrint returns a compact JSON string of any OCPP message
func CompactPrint(msg interface{}) (string, error) {
	data, err := json.Marshal(msg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}
	return string(data), nil
}

// GetMessageType extracts the message type from raw OCPP message bytes
func GetMessageType(data []byte) (MessageType, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return 0, fmt.Errorf("invalid JSON array: %w", err)
	}

	if len(arr) < 1 {
		return 0, fmt.Errorf("empty message array")
	}

	var msgType MessageType
	if err := json.Unmarshal(arr[0], &msgType); err != nil {
		return 0, fmt.Errorf("invalid message type: %w", err)
	}

	return msgType, nil
}

// GetMessageID extracts the unique message ID from raw OCPP message bytes
func GetMessageID(data []byte) (string, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return "", fmt.Errorf("invalid JSON array: %w", err)
	}

	if len(arr) < 2 {
		return "", fmt.Errorf("message array too short")
	}

	var uniqueID string
	if err := json.Unmarshal(arr[1], &uniqueID); err != nil {
		return "", fmt.Errorf("invalid message ID: %w", err)
	}

	return uniqueID, nil
}

// GetAction extracts the action from a Call message
func GetAction(data []byte) (string, error) {
	msgType, err := GetMessageType(data)
	if err != nil {
		return "", err
	}

	if msgType != MessageTypeCall {
		return "", fmt.Errorf("not a Call message")
	}

	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return "", fmt.Errorf("invalid JSON array: %w", err)
	}

	if len(arr) < 3 {
		return "", fmt.Errorf("message array too short")
	}

	var action string
	if err := json.Unmarshal(arr[2], &action); err != nil {
		return "", fmt.Errorf("invalid action: %w", err)
	}

	return action, nil
}
