package ocpp

import "time"

// DateTime is an OCPP timestamp that marshals to and from an RFC3339 string.
// The protocol version packages alias this type so the wire format stays shared.
type DateTime struct {
	time.Time
}

// MarshalJSON renders the timestamp as a quoted RFC3339 string.
func (dt DateTime) MarshalJSON() ([]byte, error) {
	return []byte(`"` + dt.Time.Format(time.RFC3339) + `"`), nil
}

// UnmarshalJSON parses a quoted RFC3339 string into the timestamp.
func (dt *DateTime) UnmarshalJSON(data []byte) error {
	str := string(data[1 : len(data)-1])
	t, err := time.Parse(time.RFC3339, str)
	if err != nil {
		return err
	}
	dt.Time = t
	return nil
}
