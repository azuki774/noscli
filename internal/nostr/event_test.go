package nostr

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

func TestEventVerify(t *testing.T) {
	base := mustValidEvent(t)

	tests := []struct {
		name     string
		mutate   func(t *testing.T, evt *Event)
		wantErr  bool
		contains string
	}{
		{
			name: "valid event",
		},
		{
			name: "id mismatch",
			mutate: func(t *testing.T, evt *Event) {
				evt.ID = strings.Repeat("0", len(evt.ID))
			},
			wantErr:  true,
			contains: "event id mismatch",
		},
		{
			name: "invalid pubkey length",
			mutate: func(t *testing.T, evt *Event) {
				evt.PubKey = "abcd"
				hash := computeEventHash(t, *evt)
				evt.ID = hex.EncodeToString(hash[:])
			},
			wantErr:  true,
			contains: "invalid pubkey length",
		},
		{
			name: "signature verification failed",
			mutate: func(t *testing.T, evt *Event) {
				sigBytes, err := hex.DecodeString(evt.Sig)
				if err != nil {
					panic(err)
				}
				sigBytes[0] ^= 0x01
				evt.Sig = hex.EncodeToString(sigBytes)
			},
			wantErr:  true,
			contains: "signature verification failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := base
			if tt.mutate != nil {
				tt.mutate(t, &evt)
			}
			err := evt.Verify()
			if !tt.wantErr {
				if err != nil {
					t.Fatalf("Verify() unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Verify() expected error")
				return
			}
			if tt.contains != "" && !strings.Contains(err.Error(), tt.contains) {
				t.Fatalf("Verify() error %q does not contain %q", err, tt.contains)
			}
		})
	}
}

func mustValidEvent(t *testing.T) Event {
	t.Helper()

	keyBytes := bytes.Repeat([]byte{0x01}, 32)
	priv, _ := btcec.PrivKeyFromBytes(keyBytes)
	pubKeyBytes := schnorr.SerializePubKey(priv.PubKey())

	evt := Event{
		PubKey:    hex.EncodeToString(pubKeyBytes),
		CreatedAt: 1_700_000_000,
		Kind:      KindTextNote,
		Tags:      [][]string{{"t", "nostr"}},
		Content:   "hello nostr",
	}

	hash := computeEventHash(t, evt)
	evt.ID = hex.EncodeToString(hash[:])

	sig, err := schnorr.Sign(priv, hash[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	evt.Sig = hex.EncodeToString(sig.Serialize())

	return evt
}

func computeEventHash(t *testing.T, evt Event) [32]byte {
	t.Helper()

	payload := []any{0, evt.PubKey, evt.CreatedAt, evt.Kind, evt.Tags, evt.Content}
	serialized, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	return sha256.Sum256(serialized)
}

func TestSignEvent(t *testing.T) {
	keyBytes := bytes.Repeat([]byte{0x01}, 32)
	priv, _ := btcec.PrivKeyFromBytes(keyBytes)
	pubKeyBytes := schnorr.SerializePubKey(priv.PubKey())

	base := Event{
		PubKey:    hex.EncodeToString(pubKeyBytes),
		CreatedAt: 1_700_000_000,
		Kind:      KindTextNote,
		Tags:      [][]string{{"t", "nostr"}},
		Content:   "hello nostr",
	}

	tests := []struct {
		name       string
		privKey    []byte
		wantErr    bool
		errMessage string
	}{
		{
			name:    "valid private key",
			privKey: keyBytes,
		},
		{
			name:       "invalid private key length",
			privKey:    []byte{0x01, 0x02},
			wantErr:    true,
			errMessage: "invalid private key length",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := base
			err := SignEvent(&evt, tt.privKey)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("SignEvent() expected error")
				}
				if tt.errMessage != "" && !strings.Contains(err.Error(), tt.errMessage) {
					t.Fatalf("SignEvent() error %q does not contain %q", err.Error(), tt.errMessage)
				}
				return
			}

			if err != nil {
				t.Fatalf("SignEvent() unexpected error: %v", err)
			}
			if evt.ID == "" {
				t.Fatalf("expected ID to be set")
			}
			if evt.Sig == "" {
				t.Fatalf("expected Sig to be set")
			}

			hash := computeEventHash(t, evt)
			expectedID := hex.EncodeToString(hash[:])
			if !strings.EqualFold(expectedID, evt.ID) {
				t.Fatalf("unexpected ID: got %s, want %s", evt.ID, expectedID)
			}

			if err := evt.Verify(); err != nil {
				t.Fatalf("Verify() failed after SignEvent: %v", err)
			}
		})
	}
}
