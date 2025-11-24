package nostr

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

// Client talks to Nostr relays via WebSocket.
type Client struct {
	dialer      *websocket.Dialer
	logger      *slog.Logger
	readTimeout time.Duration
	backoff     time.Duration
}

// NewClient creates a Client with sane defaults.
func NewClient(logger *slog.Logger) *Client {
	dialer := *websocket.DefaultDialer
	dialer.Proxy = http.ProxyFromEnvironment

	return &Client{
		dialer:      &dialer,
		logger:      logger,
		readTimeout: 30 * time.Second,
		backoff:     3 * time.Second,
	}
}

// Stream subscribes to a single relay and emits events until ctx is done.
func (c *Client) Stream(ctx context.Context, relay string, filter Filter) (<-chan Event, <-chan error) {
	events := make(chan Event, 64)
	errs := make(chan error, 1)

	go func() {
		defer close(events)
		defer close(errs)

		backoff := c.backoff
		for {
			if ctx.Err() != nil {
				return
			}

			conn, _, err := c.dialer.DialContext(ctx, relay, nil)
			if err != nil {
				c.emitError(errs, fmt.Errorf("dial %s: %w", relay, err))
				if !c.wait(ctx, backoff) {
					return
				}
				continue
			}

			c.logger.Info("connected to relay", "relay", relay)
			err = c.runSubscription(ctx, conn, relay, filter, events)
			conn.Close()

			if err != nil {
				if errors.Is(err, context.Canceled) {
					return
				}
				c.emitError(errs, fmt.Errorf("relay %s: %w", relay, err))
				if !c.wait(ctx, backoff) {
					return
				}
				continue
			}
		}
	}()

	return events, errs
}

func (c *Client) runSubscription(ctx context.Context, conn *websocket.Conn, relay string, filter Filter, events chan<- Event) error {
	subID := randomSubID()

	filterCopy := filter
	now := time.Now()
	filterCopy.Since = &now

	req := []any{"REQ", subID, filterCopy.toRequest()}
	if err := conn.WriteJSON(req); err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteJSON([]any{"CLOSE", subID})
			return ctx.Err()
		default:
		}

		_ = conn.SetReadDeadline(time.Now().Add(c.readTimeout))
		_, data, err := conn.ReadMessage()
		if err != nil {
			return err
		}

		var payload []json.RawMessage
		if err := json.Unmarshal(data, &payload); err != nil {
			continue
		}
		if len(payload) == 0 {
			continue
		}

		var msgType string
		if err := json.Unmarshal(payload[0], &msgType); err != nil {
			continue
		}

		switch msgType {
		case "EVENT":
			if len(payload) < 3 {
				continue
			}
			var recvSub string
			if err := json.Unmarshal(payload[1], &recvSub); err != nil {
				continue
			}
			if recvSub != subID {
				continue
			}
			var evt Event
			if err := json.Unmarshal(payload[2], &evt); err != nil {
				continue
			}
			if err := evt.Verify(); err != nil {
				c.logger.Debug("ignore invalid event", "relay", relay, "error", err)
				continue
			}
			evt.Relay = relay
			select {
			case events <- evt:
			case <-ctx.Done():
				return ctx.Err()
			}
		case "EOSE":
			// keep the subscription open for streaming; no action needed.
			continue
		case "NOTICE":
			if len(payload) > 1 {
				var notice string
				if err := json.Unmarshal(payload[1], &notice); err == nil {
					c.logger.Warn("relay notice", "relay", relay, "notice", notice)
				}
			}
		}
	}
}

func (c *Client) emitError(errs chan<- error, err error) {
	select {
	case errs <- err:
	default:
	}
}

func (c *Client) wait(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func randomSubID() string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("noscli-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
