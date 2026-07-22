package heartbeat

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestEncode(t *testing.T) {
	payload := validPayload()
	original := payload
	data, err := Encode(payload)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	if bytes.HasSuffix(data, []byte("\n")) {
		t.Fatal("encoded payload ends with a newline")
	}
	if !reflect.DeepEqual(payload, original) {
		t.Fatal("Encode() mutated the payload")
	}

	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	wantFields := []string{"schema_version", "agent_id", "agent_name", "agent_version", "sent_at", "sequence"}
	if len(decoded) != len(wantFields) {
		t.Fatalf("field count = %d, want %d", len(decoded), len(wantFields))
	}
	for _, field := range wantFields {
		if _, ok := decoded[field]; !ok {
			t.Errorf("field %q is missing", field)
		}
	}
	if decoded["sent_at"] != "2026-07-22T12:00:00Z" {
		t.Errorf("sent_at = %v", decoded["sent_at"])
	}
	if _, ok := decoded["sequence"].(float64); !ok {
		t.Errorf("sequence type = %T, want JSON number", decoded["sequence"])
	}
}

func TestEncodeRejectsInvalidPayload(t *testing.T) {
	payload := validPayload()
	payload.Sequence = 0
	if _, err := Encode(payload); err == nil || !strings.Contains(err.Error(), "validate heartbeat payload") {
		t.Fatalf("Encode() error = %v", err)
	}
}

func TestDecodeAndRoundTrip(t *testing.T) {
	original := validPayload()
	data, err := Encode(original)
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	decoded, err := Decode(append(data, ' ', '\n', '\t'))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if !reflect.DeepEqual(decoded, original) {
		t.Fatalf("decoded = %#v, want %#v", decoded, original)
	}
}

func TestDecodeRejectsInvalidJSON(t *testing.T) {
	valid := `{"schema_version":"1","agent_id":"` + testAgentID + `","agent_name":"app-server-01","agent_version":"dev","sent_at":"2026-07-22T12:00:00Z","sequence":1}`
	tests := []struct {
		name string
		data string
		want string
	}{
		{name: "unknown field", data: strings.TrimSuffix(valid, "}") + `,"extra":true}`, want: "unknown field"},
		{name: "missing schema", data: `{"agent_id":"` + testAgentID + `","agent_name":"app-server-01","agent_version":"dev","sent_at":"2026-07-22T12:00:00Z","sequence":1}`, want: "schema version"},
		{name: "missing agent ID", data: `{"schema_version":"1","agent_name":"app-server-01","agent_version":"dev","sent_at":"2026-07-22T12:00:00Z","sequence":1}`, want: "agent ID"},
		{name: "missing agent name", data: `{"schema_version":"1","agent_id":"` + testAgentID + `","agent_version":"dev","sent_at":"2026-07-22T12:00:00Z","sequence":1}`, want: "agent name"},
		{name: "missing agent version", data: `{"schema_version":"1","agent_id":"` + testAgentID + `","agent_name":"app-server-01","sent_at":"2026-07-22T12:00:00Z","sequence":1}`, want: "agent version"},
		{name: "missing timestamp", data: `{"schema_version":"1","agent_id":"` + testAgentID + `","agent_name":"app-server-01","agent_version":"dev","sequence":1}`, want: "sent timestamp"},
		{name: "missing sequence", data: `{"schema_version":"1","agent_id":"` + testAgentID + `","agent_name":"app-server-01","agent_version":"dev","sent_at":"2026-07-22T12:00:00Z"}`, want: "sequence"},
		{name: "wrong schema", data: strings.Replace(valid, `"schema_version":"1"`, `"schema_version":"2"`, 1), want: "schema version"},
		{name: "invalid UUID", data: strings.Replace(valid, testAgentID, "invalid", 1), want: "agent ID"},
		{name: "invalid timestamp", data: strings.Replace(valid, "2026-07-22T12:00:00Z", "not-a-time", 1), want: "cannot parse"},
		{name: "zero sequence", data: strings.Replace(valid, `"sequence":1`, `"sequence":0`, 1), want: "sequence"},
		{name: "negative sequence", data: strings.Replace(valid, `"sequence":1`, `"sequence":-1`, 1), want: "cannot unmarshal"},
		{name: "fractional sequence", data: strings.Replace(valid, `"sequence":1`, `"sequence":1.5`, 1), want: "cannot unmarshal"},
		{name: "string sequence", data: strings.Replace(valid, `"sequence":1`, `"sequence":"1"`, 1), want: "cannot unmarshal"},
		{name: "overflow sequence", data: strings.Replace(valid, `"sequence":1`, `"sequence":18446744073709551616`, 1), want: "cannot unmarshal"},
		{name: "empty", data: "", want: "input is empty"},
		{name: "whitespace", data: " \n\t", want: "input is empty"},
		{name: "null", data: "null", want: "schema version"},
		{name: "array", data: "[]", want: "cannot unmarshal"},
		{name: "two objects", data: valid + valid, want: "multiple JSON values"},
		{name: "trailing content", data: valid + " trailing", want: "trailing heartbeat data"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Decode([]byte(test.data))
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Decode() error = %v, want context %q", err, test.want)
			}
		})
	}
}

func FuzzDecode(f *testing.F) {
	valid, err := Encode(validPayload())
	if err != nil {
		f.Fatalf("Encode() error = %v", err)
	}
	f.Add(valid)
	f.Add([]byte{})
	f.Add([]byte(`{"schema_version":"1","unknown":true}`))
	f.Add([]byte(`{"sent_at":"not-a-time"}`))

	f.Fuzz(func(t *testing.T, data []byte) {
		payload, err := Decode(data)
		if err != nil {
			return
		}
		if err := Validate(payload); err != nil {
			t.Fatalf("Decode() returned invalid payload: %v", err)
		}
		if _, err := Encode(payload); err != nil {
			t.Fatalf("Encode() failed after successful Decode(): %v", err)
		}
	})
}

func TestMinimumTimestampAccepted(t *testing.T) {
	payload, err := New(testAgentID, "app-server-01", "dev", time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC), 1)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := Validate(payload); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
