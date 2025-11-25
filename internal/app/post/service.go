package post

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcutil/bech32"

	"noscli/internal/nostr"
)

// Request represents a post request.
type Request struct {
	Relay   string
	Content string
	ReplyTo string
}

// Client exposes the subset of nostr client functionality needed by the post service.
type Client interface {
	Publish(ctx context.Context, relay string, evt nostr.Event) error
}

// Service sends text note events to a relay.
type Service struct {
	client Client
	logger *slog.Logger
}

// NewService creates a Service that relies on the given nostr client.
func NewService(client Client, logger *slog.Logger) *Service {
	return &Service{client: client, logger: logger}
}

// Run executes the post request and writes a short result to w.
func (s *Service) Run(ctx context.Context, req Request, w io.Writer) error {
	if strings.TrimSpace(req.Relay) == "" {
		return errors.New("relay is required")
	}
	content := strings.TrimSpace(req.Content)
	if content == "" {
		return errors.New("content is empty")
	}

	priv, pub, err := loadKeysFromEnv()
	if err != nil {
		return err
	}

	evt := nostr.Event{
		PubKey:    pub,
		CreatedAt: time.Now().Unix(),
		Kind:      nostr.KindTextNote,
		Tags:      [][]string{},
		Content:   content,
	}
	if req.ReplyTo != "" {
		evt.Tags = append(evt.Tags, []string{"e", req.ReplyTo})
	}

	if err := nostr.SignEvent(&evt, priv); err != nil {
		return fmt.Errorf("sign event: %w", err)
	}

	if err := s.client.Publish(ctx, req.Relay, evt); err != nil {
		return err
	}

	prefixForPreview := evt.ID
	if len(prefixForPreview) > 8 {
		prefixForPreview = prefixForPreview[:8]
	}
	if _, err := fmt.Fprintf(w, "published: id:%s relay:%s\n", prefixForPreview, req.Relay); err != nil {
		return err
	}

	return nil
}

// loadKeysFromEnv reads NOSTR_NSEC and returns the raw private key and public key (both 32-byte hex).
func loadKeysFromEnv() ([]byte, string, error) {
	nsec := strings.TrimSpace(os.Getenv("NOSTR_NSEC"))
	if nsec == "" {
		return nil, "", errors.New("NOSTR_NSEC is not set")
	}
	priv, err := decodeNsec(nsec)
	if err != nil {
		return nil, "", fmt.Errorf("decode NOSTR_NSEC: %w", err)
	}

	// Derive public key using the same curve as verification.
	pubHex, err := derivePubKeyHex(priv)
	if err != nil {
		return nil, "", err
	}

	return priv, pubHex, nil
}

func derivePubKeyHex(priv []byte) (string, error) {
	if len(priv) != 32 {
		return "", fmt.Errorf("invalid private key length: %d", len(priv))
	}

	sk, _ := btcec.PrivKeyFromBytes(priv)
	if sk == nil {
		return "", errors.New("invalid private key")
	}
	pubKeyBytes := schnorr.SerializePubKey(sk.PubKey())
	return hex.EncodeToString(pubKeyBytes), nil
}

// decodeNsec decodes a NIP-19 nsec bech32 string and returns the raw 32-byte private key.
func decodeNsec(nsec string) ([]byte, error) {
	hrp, data, err := bech32.Decode(nsec)
	if err != nil {
		return nil, err
	}
	if hrp != "nsec" {
		return nil, fmt.Errorf("unexpected HRP: %s", hrp)
	}

	// Convert 5-bit groups back to 8-bit bytes.
	eightBits, err := convertBits(data, 5, 8, false)
	if err != nil {
		return nil, err
	}
	if len(eightBits) != 32 {
		return nil, fmt.Errorf("unexpected nsec length: %d", len(eightBits))
	}

	// For nsec, payload is just the 32-byte private key.
	return eightBits, nil
}

// convertBits converts a slice of data where each element is fromBits wide into
// a slice where each element is toBits wide. It is used for bech32 encoding/decoding.
func convertBits(data []byte, fromBits, toBits uint, pad bool) ([]byte, error) {
	var ret []byte
	var acc uint
	var bits uint
	maxv := uint((1 << toBits) - 1)
	maxAcc := uint((1 << (fromBits + toBits - 1)) - 1)

	for _, value := range data {
		v := uint(value)
		if v>>fromBits != 0 {
			return nil, fmt.Errorf("invalid data range: %d", value)
		}
		acc = ((acc << fromBits) | v) & maxAcc
		bits += fromBits
		for bits >= toBits {
			bits -= toBits
			ret = append(ret, byte((acc>>bits)&maxv))
		}
	}

	if pad {
		if bits > 0 {
			ret = append(ret, byte((acc<<(toBits-bits))&maxv))
		}
	} else if bits >= fromBits {
		return nil, fmt.Errorf("illegal zero padding")
	} else if ((acc << (toBits - bits)) & maxv) != 0 {
		return nil, fmt.Errorf("non-zero padding")
	}

	return ret, nil
}
