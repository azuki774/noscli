package nostr

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
)

// KindTextNote corresponds to NIP-01 kind 1 events.
const KindTextNote = 1

// Event represents a Nostr event structure.
type Event struct {
	ID        string     `json:"id"`
	PubKey    string     `json:"pubkey"`
	CreatedAt int64      `json:"created_at"`
	Kind      int        `json:"kind"`
	Tags      [][]string `json:"tags"`
	Content   string     `json:"content"`
	Sig       string     `json:"sig"`
	Relay     string     `json:"-"`
}

// CreatedAtTime converts the timestamp to time.Time.
func (e Event) CreatedAtTime() time.Time {
	return time.Unix(e.CreatedAt, 0).UTC()
}

// Verify ensures the event ID and signature are valid.
func (e Event) Verify() error {
	hash, err := hashEvent(e)
	if err != nil {
		return err
	}

	expected := hex.EncodeToString(hash[:])
	if !strings.EqualFold(expected, e.ID) {
		return errors.New("event id mismatch")
	}

	pubKeyBytes, err := hex.DecodeString(e.PubKey)
	if err != nil {
		return fmt.Errorf("pubkey decode: %w", err)
	}
	if len(pubKeyBytes) != 32 {
		return fmt.Errorf("invalid pubkey length: %d", len(pubKeyBytes))
	}

	sigBytes, err := hex.DecodeString(e.Sig)
	if err != nil {
		return fmt.Errorf("signature decode: %w", err)
	}

	signature, err := schnorr.ParseSignature(sigBytes)
	if err != nil {
		return fmt.Errorf("signature parse: %w", err)
	}

	pubKey, err := schnorr.ParsePubKey(pubKeyBytes)
	if err != nil {
		return fmt.Errorf("pubkey parse: %w", err)
	}

	if !signature.Verify(hash[:], pubKey) {
		return errors.New("signature verification failed")
	}

	return nil
}

// hashEvent calculates the event hash as specified in NIP-01.
func hashEvent(e Event) ([32]byte, error) {
	payload := []any{0, e.PubKey, e.CreatedAt, e.Kind, e.Tags, e.Content}
	serialized, err := json.Marshal(payload)
	if err != nil {
		return [32]byte{}, fmt.Errorf("marshal payload: %w", err)
	}
	return sha256.Sum256(serialized), nil
}

// SignEvent computes the event ID and signature using the given private key.
// privKey is a 32-byte secret key.
func SignEvent(e *Event, privKey []byte) error {
	if len(privKey) != 32 {
		return fmt.Errorf("invalid private key length: %d", len(privKey))
	}

	hash, err := hashEvent(*e)
	if err != nil {
		return err
	}
	e.ID = hex.EncodeToString(hash[:])

	sk, _ := btcec.PrivKeyFromBytes(privKey)
	sig, err := schnorr.Sign(sk, hash[:])
	if err != nil {
		return fmt.Errorf("sign: %w", err)
	}
	e.Sig = hex.EncodeToString(sig.Serialize())

	return nil
}
