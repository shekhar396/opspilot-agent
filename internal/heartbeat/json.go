package heartbeat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Encode validates and encodes one compact heartbeat JSON object.
func Encode(payload Payload) ([]byte, error) {
	if err := Validate(payload); err != nil {
		return nil, fmt.Errorf("validate heartbeat payload: %w", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode heartbeat payload: %w", err)
	}
	return data, nil
}

// Decode strictly decodes and validates exactly one heartbeat JSON object.
func Decode(data []byte) (Payload, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return Payload{}, fmt.Errorf("decode heartbeat payload: input is empty")
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var payload Payload
	if err := decoder.Decode(&payload); err != nil {
		return Payload{}, fmt.Errorf("decode heartbeat payload: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err != nil {
			return Payload{}, fmt.Errorf("decode trailing heartbeat data: %w", err)
		}
		return Payload{}, fmt.Errorf("decode heartbeat payload: multiple JSON values are not allowed")
	}
	if err := Validate(payload); err != nil {
		return Payload{}, fmt.Errorf("validate heartbeat payload: %w", err)
	}
	return payload, nil
}
