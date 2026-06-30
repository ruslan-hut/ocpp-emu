package ocpp

import (
	"encoding/json"
	"fmt"
)

// SendCall builds a Call for action with the given payload, sends it via send
// (when non-nil), and returns the Call. It is the shared body for the protocol
// handlers' Send* methods; action is reused in the error messages.
func SendCall(send func(stationID string, data []byte) error, stationID, action string, payload interface{}) (*Call, error) {
	call, err := NewCall(action, payload)
	if err != nil {
		return nil, fmt.Errorf("failed to create %s call: %w", action, err)
	}

	data, err := call.ToBytes()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s: %w", action, err)
	}

	if send != nil {
		if err := send(stationID, data); err != nil {
			return nil, fmt.Errorf("failed to send %s: %w", action, err)
		}
	}

	return call, nil
}

// HandleRequest decodes a Call payload into *Req and dispatches it to cb. When cb
// is nil it returns defaultResp (the protocol's default reply for an unhandled
// action). action is reused in the error message.
func HandleRequest[Req any, Resp any](stationID, action string, call *Call, cb func(string, *Req) (*Resp, error), defaultResp *Resp) (*Resp, error) {
	var req Req
	if err := json.Unmarshal(call.Payload, &req); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s request: %w", action, err)
	}

	if cb == nil {
		return defaultResp, nil
	}

	return cb(stationID, &req)
}

// UnmarshalResult decodes a CallResult payload into *T. action is reused in the
// error message.
func UnmarshalResult[T any](result *CallResult, action string) (*T, error) {
	var resp T
	if err := json.Unmarshal(result.Payload, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s response: %w", action, err)
	}
	return &resp, nil
}
