package nostr

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestParseOKMessage(t *testing.T) {
	tests := []struct {
		name        string
		data        any
		wantErr     bool
		errContains string
		want        okResult
	}{
		{
			name: "valid OK true",
			data: []any{"OK", "event-id", true, "accepted"},
			want: okResult{
				EventID: "event-id",
				OK:      true,
				Message: "accepted",
			},
		},
		{
			name: "valid OK false",
			data: []any{"OK", "event-id", false, "reason"},
			want: okResult{
				EventID: "event-id",
				OK:      false,
				Message: "reason",
			},
		},
		{
			name:        "invalid json payload",
			data:        "{not-json",
			wantErr:     true,
			errContains: "unmarshal OK payload",
		},
		{
			name:        "too short array",
			data:        []any{"OK", "event-id", true},
			wantErr:     true,
			errContains: "invalid OK message",
		},
		{
			name:        "unexpected message type",
			data:        []any{"EVENT", "event-id", true, "msg"},
			wantErr:     true,
			errContains: "unexpected message type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw []byte
			var err error

			switch v := tt.data.(type) {
			case string:
				raw = []byte(v)
			default:
				raw, err = json.Marshal(v)
				if err != nil {
					t.Fatalf("marshal test data: %v", err)
				}
			}

			got, err := parseOKMessage(raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseOKMessage() expected error")
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("parseOKMessage() error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseOKMessage() unexpected error: %v", err)
			}

			if got.EventID != tt.want.EventID || got.OK != tt.want.OK || got.Message != tt.want.Message {
				t.Fatalf("parseOKMessage() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
