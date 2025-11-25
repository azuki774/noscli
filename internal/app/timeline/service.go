package timeline

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"noscli/internal/nostr"
)

// Request represents timeline filters and rendering options.
type Request struct {
	Relays []string
}

// Client exposes the subset of nostr client functionality needed by the timeline service.
type Client interface {
	Stream(ctx context.Context, relay string, filter nostr.Filter) (<-chan nostr.Event, <-chan error)
}

// Service fetches and renders timeline events.
type Service struct {
	client Client
	logger *slog.Logger
}

// NewService creates a Service that relies on the given nostr client.
func NewService(client Client, logger *slog.Logger) *Service {
	return &Service{client: client, logger: logger}
}

// Run executes the timeline request and writes results to w.
func (s *Service) Run(ctx context.Context, req Request, w io.Writer) error {
	if len(req.Relays) == 0 {
		return errors.New("relay is required")
	}

	// 単一リレーのみを処理する。将来的に複数リレー対応時はここでルーティングを追加する。
	relay := req.Relays[0]

	filter := nostr.Filter{
		Kinds: []int{nostr.KindTextNote},
	}

	events, errs := s.client.Stream(ctx, relay, filter)

	for {
		select {
		case <-ctx.Done():
			return nil
		case evt, ok := <-events:
			if !ok {
				events = nil
				if errs == nil {
					return nil
				}
				continue
			}
			if err := renderPlainEvent(w, evt); err != nil {
				return err
			}
		case err, ok := <-errs:
			if !ok {
				errs = nil
				if events == nil {
					return nil
				}
				continue
			}
			if err == nil || errors.Is(err, context.Canceled) {
				continue
			}
			s.logger.Warn("timeline stream error", "error", err)
		}
	}
}

func renderPlainEvent(w io.Writer, evt nostr.Event) error {
	ts := time.Unix(evt.CreatedAt, 0).Local().Format("2006-01-02 15:04:05")
	author := truncateHex(evt.PubKey)
	summary := sanitizeContent(evt.Content)
	prefixForPreview := evt.ID
	if len(prefixForPreview) > 8 {
		prefixForPreview = prefixForPreview[:8]
	}
	_, err := fmt.Fprintf(w, "[%s] %s: %s (id:%s relay:%s)\n", ts, author, summary, prefixForPreview, evt.Relay)
	return err
}

func truncateHex(in string) string {
	if len(in) <= 12 {
		return in
	}
	return fmt.Sprintf("%s...%s", in[:6], in[len(in)-4:])
}

func sanitizeContent(in string) string {
	trimmed := strings.TrimSpace(in)
	if trimmed == "" {
		return "(no content)"
	}
	trimmed = strings.ReplaceAll(trimmed, "\n", " ")
	trimmed = strings.ReplaceAll(trimmed, "\r", " ")
	return trimmed
}
