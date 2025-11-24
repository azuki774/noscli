package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"

	"noscli/internal/app/timeline"
	"noscli/internal/nostr"
)

type timelineOptions struct {
	relay string
}

func newTimelineCommand() *cobra.Command {
	opts := &timelineOptions{}

	cmd := &cobra.Command{
		Use:   "timeline",
		Short: "Nostr テキストノートをストリーム表示する",
		Long:  "WebSocket でリレーに接続し、Ctrl+C などで中断するまでイベントを受信し続けます。",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			logger := getLogger()

			relay := opts.relay
			if relay == "" {
				relay = cfg.Timeline.Relay
			}
			if relay == "" {
				return errors.New("リレーが指定されていません (--relay または NOSCLI_RELAY)")
			}

			req := timeline.Request{
				Relays: []string{relay},
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			svc := timeline.NewService(nostr.NewClient(logger), logger)
			return svc.Run(ctx, req, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&opts.relay, "relay", "", "リレー URL")

	return cmd
}
