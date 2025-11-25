package post

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcutil/bech32"

	"noscli/internal/nostr"
)

type mockClient struct {
	calls []publishCall
	err   error
}

type publishCall struct {
	relay string
	evt   nostr.Event
}

func (m *mockClient) Publish(_ context.Context, relay string, evt nostr.Event) error {
	m.calls = append(m.calls, publishCall{relay: relay, evt: evt})
	return m.err
}

func TestServiceRun(t *testing.T) {
	privKey := bytes.Repeat([]byte{0x01}, 32)
	nsec, err := encodeNsec(privKey)
	if err != nil {
		t.Fatalf("encodeNsec: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	tests := []struct {
		name            string
		req             Request
		setNsec         bool
		nsecValue       string
		clientErr       error
		wantErr         bool
		wantErrContains string
		wantCalls       int
	}{
		{
			name: "missing relay",
			req: Request{
				Relay:   "",
				Content: "hello",
			},
			setNsec:         true,
			nsecValue:       nsec,
			wantErr:         true,
			wantErrContains: "relay is required",
			wantCalls:       0,
		},
		{
			name: "empty content",
			req: Request{
				Relay:   "wss://relay.example.com",
				Content: "   ",
			},
			setNsec:         true,
			nsecValue:       nsec,
			wantErr:         true,
			wantErrContains: "content is empty",
			wantCalls:       0,
		},
		{
			name: "NOSTR_NSEC not set",
			req: Request{
				Relay:   "wss://relay.example.com",
				Content: "hello",
			},
			setNsec:         false,
			wantErr:         true,
			wantErrContains: "NOSTR_NSEC is not set",
			wantCalls:       0,
		},
		{
			name: "invalid NOSTR_NSEC format",
			req: Request{
				Relay:   "wss://relay.example.com",
				Content: "hello",
			},
			setNsec:         true,
			nsecValue:       "invalid",
			wantErr:         true,
			wantErrContains: "decode NOSTR_NSEC",
			wantCalls:       0,
		},
		{
			name: "publish error",
			req: Request{
				Relay:   "wss://relay.example.com",
				Content: "hello",
			},
			setNsec:         true,
			nsecValue:       nsec,
			clientErr:       errors.New("publish failed"),
			wantErr:         true,
			wantErrContains: "publish failed",
			wantCalls:       1,
		},
		{
			name: "success",
			req: Request{
				Relay:   "wss://relay.example.com",
				Content: "hello nostr",
				ReplyTo: "abcdef",
			},
			setNsec:   true,
			nsecValue: nsec,
			wantErr:   false,
			wantCalls: 1,
		},
		{
			name: "success without reply-to",
			req: Request{
				Relay:   "wss://relay.example.com",
				Content: "hello nostr",
				ReplyTo: "",
			},
			setNsec:   true,
			nsecValue: nsec,
			wantErr:   false,
			wantCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setNsec {
				t.Setenv("NOSTR_NSEC", tt.nsecValue)
			} else {
				t.Setenv("NOSTR_NSEC", "")
			}

			client := &mockClient{err: tt.clientErr}
			svc := NewService(client, logger)

			var buf bytes.Buffer
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := svc.Run(ctx, tt.req, &buf)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Run() expected error")
				}
				if tt.wantErrContains != "" && !strings.Contains(err.Error(), tt.wantErrContains) {
					t.Fatalf("Run() error %q does not contain %q", err.Error(), tt.wantErrContains)
				}
			} else if err != nil {
				t.Fatalf("Run() unexpected error: %v", err)
			}

			if got := len(client.calls); got != tt.wantCalls {
				t.Fatalf("Publish calls = %d, want %d", got, tt.wantCalls)
			}

			if !tt.wantErr && tt.wantCalls == 1 {
				call := client.calls[0]
				if call.relay != tt.req.Relay {
					t.Fatalf("relay = %s, want %s", call.relay, tt.req.Relay)
				}
				evt := call.evt
				if evt.Kind != nostr.KindTextNote {
					t.Fatalf("Kind = %d, want %d", evt.Kind, nostr.KindTextNote)
				}
				if evt.Content != tt.req.Content {
					t.Fatalf("Content = %q, want %q", evt.Content, tt.req.Content)
				}
				if tt.req.ReplyTo != "" {
					if len(evt.Tags) == 0 || evt.Tags[0][0] != "e" || evt.Tags[0][1] != tt.req.ReplyTo {
						t.Fatalf("expected reply-to tag for %q, got %#v", tt.req.ReplyTo, evt.Tags)
					}
				} else {
					if evt.Tags == nil || len(evt.Tags) != 0 {
						t.Fatalf("expected empty non-nil tags, got %#v", evt.Tags)
					}
				}
				if evt.ID == "" || evt.Sig == "" {
					t.Fatalf("expected ID and Sig to be set, got ID=%q Sig=%q", evt.ID, evt.Sig)
				}
				if err := evt.Verify(); err != nil {
					t.Fatalf("Verify() failed for signed event: %v", err)
				}
			}
		})
	}
}

func encodeNsec(priv []byte) (string, error) {
	if len(priv) != 32 {
		return "", errors.New("invalid private key length")
	}

	data, err := convertBits(priv, 8, 5, true)
	if err != nil {
		return "", err
	}
	return bech32.Encode("nsec", data)
}
