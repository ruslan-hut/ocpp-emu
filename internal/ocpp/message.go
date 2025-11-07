package ocpp

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// MessageType represents the OCPP message type
type MessageType int

const (
	// MessageTypeCall represents a Call message (request from client to server)
	// Format: [2, "uniqueId", "Action", {payload}]
	MessageTypeCall MessageType = 2

	// MessageTypeCallResult represents a CallResult message (response from server to client)
	// Format: [3, "uniqueId", {payload}]
	MessageTypeCallResult MessageType = 3

	// MessageTypeCallError represents a CallError message (error response)
	// Format: [4, "uniqueId", "ErrorCode", "ErrorDescription", {errorDetails}]
	MessageTypeCallError MessageType = 4
)

// ErrorCode represents OCPP error codes
type ErrorCode string

const (
	// ErrorCodeNotImplemented - Requested action is not known by receiver
	ErrorCodeNotImplemented ErrorCode = "NotImplemented"

	// ErrorCodeNotSupported - Requested action is recognized but not supported by the receiver
	ErrorCodeNotSupported ErrorCode = "NotSupported"

	// ErrorCodeInternalError - An internal error occurred and the receiver was not able to process the request
	ErrorCodeInternalError ErrorCode = "InternalError"

	// ErrorCodeProtocolError - Payload for Action is incomplete
	ErrorCodeProtocolError ErrorCode = "ProtocolError"

	// ErrorCodeSecurityError - During processing of Action a security issue occurred
	ErrorCodeSecurityError ErrorCode = "SecurityError"

	// ErrorCodeFormationViolation - Payload for Action is syntactically incorrect
	ErrorCodeFormationViolation ErrorCode = "FormationViolation"

	// ErrorCodePropertyConstraintViolation - Payload is syntactically correct but violates constraints
	ErrorCodePropertyConstraintViolation ErrorCode = "PropertyConstraintViolation"

	// ErrorCodeOccurrenceConstraintViolation - Payload for Action is syntactically correct but at least one field contains an invalid value
	ErrorCodeOccurrenceConstraintViolation ErrorCode = "OccurrenceConstraintViolation"

	// ErrorCodeTypeConstraintViolation - Payload for Action is syntactically correct but at least one of the fields violates data type constraints
	ErrorCodeTypeConstraintViolation ErrorCode = "TypeConstraintViolation"

	// ErrorCodeGenericError - Any other error not covered by the previous ones
	ErrorCodeGenericError ErrorCode = "GenericError"
)

// Message represents a generic OCPP message
type Message struct {
	MessageTypeID MessageType     `json:"messageTypeId"`
	UniqueID      string          `json:"uniqueId"`
	Action        string          `json:"action,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	ErrorCode     ErrorCode       `json:"errorCode,omitempty"`
	ErrorDesc     string          `json:"errorDescription,omitempty"`
	ErrorDetails  json.RawMessage `json:"errorDetails,omitempty"`

	// Metadata (not part of wire format)
	Timestamp time.Time `json:"-"`
	StationID string    `json:"-"`
}

// Call represents an OCPP Call message (request)
type Call struct {
	MessageTypeID MessageType     `json:"-"`
	UniqueID      string          `json:"-"`
	Action        string          `json:"-"`
	Payload       json.RawMessage `json:"-"`
}

// CallResult represents an OCPP CallResult message (response)
type CallResult struct {
	MessageTypeID MessageType     `json:"-"`
	UniqueID      string          `json:"-"`
	Payload       json.RawMessage `json:"-"`
}

// CallError represents an OCPP CallError message (error response)
type CallError struct {
	MessageTypeID MessageType     `json:"-"`
	UniqueID      string          `json:"-"`
	ErrorCode     ErrorCode       `json:"-"`
	ErrorDesc     string          `json:"-"`
	ErrorDetails  json.RawMessage `json:"-"`
}

// NewCall creates a new Call message
func NewCall(action string, payload interface{}) (*Call, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &Call{
		MessageTypeID: MessageTypeCall,
		UniqueID:      GenerateMessageID(),
		Action:        action,
		Payload:       payloadBytes,
	}, nil
}

// NewCallResult creates a new CallResult message
func NewCallResult(uniqueID string, payload interface{}) (*CallResult, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return &CallResult{
		MessageTypeID: MessageTypeCallResult,
		UniqueID:      uniqueID,
		Payload:       payloadBytes,
	}, nil
}

// NewCallError creates a new CallError message
func NewCallError(uniqueID string, errorCode ErrorCode, errorDesc string, errorDetails interface{}) (*CallError, error) {
	var detailsBytes json.RawMessage
	if errorDetails != nil {
		var err error
		detailsBytes, err = json.Marshal(errorDetails)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal error details: %w", err)
		}
	} else {
		detailsBytes = json.RawMessage("{}")
	}

	return &CallError{
		MessageTypeID: MessageTypeCallError,
		UniqueID:      uniqueID,
		ErrorCode:     errorCode,
		ErrorDesc:     errorDesc,
		ErrorDetails:  detailsBytes,
	}, nil
}

// MarshalJSON marshals a Call message to JSON in OCPP format
func (c *Call) MarshalJSON() ([]byte, error) {
	arr := []interface{}{
		MessageTypeCall,
		c.UniqueID,
		c.Action,
		c.Payload,
	}
	return json.Marshal(arr)
}

// UnmarshalJSON unmarshals a Call message from JSON in OCPP format
func (c *Call) UnmarshalJSON(data []byte) error {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) != 4 {
		return fmt.Errorf("invalid Call message format: expected 4 elements, got %d", len(arr))
	}

	// Parse message type
	var msgType MessageType
	if err := json.Unmarshal(arr[0], &msgType); err != nil {
		return err
	}
	if msgType != MessageTypeCall {
		return fmt.Errorf("invalid message type: expected %d, got %d", MessageTypeCall, msgType)
	}
	c.MessageTypeID = msgType

	// Parse unique ID
	if err := json.Unmarshal(arr[1], &c.UniqueID); err != nil {
		return err
	}

	// Parse action
	if err := json.Unmarshal(arr[2], &c.Action); err != nil {
		return err
	}

	// Store raw payload
	c.Payload = arr[3]

	return nil
}

// MarshalJSON marshals a CallResult message to JSON in OCPP format
func (cr *CallResult) MarshalJSON() ([]byte, error) {
	arr := []interface{}{
		MessageTypeCallResult,
		cr.UniqueID,
		cr.Payload,
	}
	return json.Marshal(arr)
}

// UnmarshalJSON unmarshals a CallResult message from JSON in OCPP format
func (cr *CallResult) UnmarshalJSON(data []byte) error {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) != 3 {
		return fmt.Errorf("invalid CallResult message format: expected 3 elements, got %d", len(arr))
	}

	// Parse message type
	var msgType MessageType
	if err := json.Unmarshal(arr[0], &msgType); err != nil {
		return err
	}
	if msgType != MessageTypeCallResult {
		return fmt.Errorf("invalid message type: expected %d, got %d", MessageTypeCallResult, msgType)
	}
	cr.MessageTypeID = msgType

	// Parse unique ID
	if err := json.Unmarshal(arr[1], &cr.UniqueID); err != nil {
		return err
	}

	// Store raw payload
	cr.Payload = arr[2]

	return nil
}

// MarshalJSON marshals a CallError message to JSON in OCPP format
func (ce *CallError) MarshalJSON() ([]byte, error) {
	arr := []interface{}{
		MessageTypeCallError,
		ce.UniqueID,
		ce.ErrorCode,
		ce.ErrorDesc,
		ce.ErrorDetails,
	}
	return json.Marshal(arr)
}

// UnmarshalJSON unmarshals a CallError message from JSON in OCPP format
func (ce *CallError) UnmarshalJSON(data []byte) error {
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return err
	}

	if len(arr) != 5 {
		return fmt.Errorf("invalid CallError message format: expected 5 elements, got %d", len(arr))
	}

	// Parse message type
	var msgType MessageType
	if err := json.Unmarshal(arr[0], &msgType); err != nil {
		return err
	}
	if msgType != MessageTypeCallError {
		return fmt.Errorf("invalid message type: expected %d, got %d", MessageTypeCallError, msgType)
	}
	ce.MessageTypeID = msgType

	// Parse unique ID
	if err := json.Unmarshal(arr[1], &ce.UniqueID); err != nil {
		return err
	}

	// Parse error code
	if err := json.Unmarshal(arr[2], &ce.ErrorCode); err != nil {
		return err
	}

	// Parse error description
	if err := json.Unmarshal(arr[3], &ce.ErrorDesc); err != nil {
		return err
	}

	// Store raw error details
	ce.ErrorDetails = arr[4]

	return nil
}

// ToBytes converts the Call message to bytes
func (c *Call) ToBytes() ([]byte, error) {
	return json.Marshal(c)
}

// ToBytes converts the CallResult message to bytes
func (cr *CallResult) ToBytes() ([]byte, error) {
	return json.Marshal(cr)
}

// ToBytes converts the CallError message to bytes
func (ce *CallError) ToBytes() ([]byte, error) {
	return json.Marshal(ce)
}

// ParseMessage parses a raw OCPP message and determines its type
func ParseMessage(data []byte) (interface{}, error) {
	// First, check the message type
	var arr []json.RawMessage
	if err := json.Unmarshal(data, &arr); err != nil {
		return nil, fmt.Errorf("invalid message format: %w", err)
	}

	if len(arr) < 3 {
		return nil, fmt.Errorf("invalid message format: expected at least 3 elements, got %d", len(arr))
	}

	// Parse message type
	var msgType MessageType
	if err := json.Unmarshal(arr[0], &msgType); err != nil {
		return nil, fmt.Errorf("failed to parse message type: %w", err)
	}

	// Parse based on type
	switch msgType {
	case MessageTypeCall:
		var call Call
		if err := json.Unmarshal(data, &call); err != nil {
			return nil, fmt.Errorf("failed to parse Call message: %w", err)
		}
		return &call, nil

	case MessageTypeCallResult:
		var callResult CallResult
		if err := json.Unmarshal(data, &callResult); err != nil {
			return nil, fmt.Errorf("failed to parse CallResult message: %w", err)
		}
		return &callResult, nil

	case MessageTypeCallError:
		var callError CallError
		if err := json.Unmarshal(data, &callError); err != nil {
			return nil, fmt.Errorf("failed to parse CallError message: %w", err)
		}
		return &callError, nil

	default:
		return nil, fmt.Errorf("unknown message type: %d", msgType)
	}
}

// GenerateMessageID generates a unique message ID
func GenerateMessageID() string {
	return uuid.New().String()
}

// ValidateMessageID validates a message ID format
func ValidateMessageID(id string) bool {
	// OCPP allows any string as message ID, but we enforce UUID format
	_, err := uuid.Parse(id)
	return err == nil
}
