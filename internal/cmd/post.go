package cmd

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"noscli/internal/app/post"
	"noscli/internal/nostr"
)

type postOptions struct {
	relay   string
	message string
	replyTo string
}

func newPostCommand() *cobra.Command {
	opts := &postOptions{}

	cmd := &cobra.Command{
		Use:   "post",
		Short: "Nostr テキストノートを投稿する",
		Long:  "kind 1 のテキストノートイベントを単一リレーに送信します。メッセージは -m または標準入力から指定します。",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := loadConfig()
			logger := getLogger()

			relay := opts.relay
			if relay == "" {
				relay = cfg.Timeline.Relay
			}
			if strings.TrimSpace(relay) == "" {
				return errors.New("リレーが指定されていません (--relay または NOSCLI_RELAY)")
			}

			content := strings.TrimSpace(opts.message)
			if content == "" {
				b, err := io.ReadAll(cmd.InOrStdin())
				if err != nil {
					return err
				}
				content = strings.TrimSpace(string(b))
			}
			if content == "" {
				return errors.New("投稿内容が空です (-m または標準入力で指定してください)")
			}

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			req := post.Request{
				Relay:   relay,
				Content: content,
				ReplyTo: strings.TrimSpace(opts.replyTo),
			}

			svc := post.NewService(nostr.NewClient(logger), logger)
			return svc.Run(ctx, req, cmd.OutOrStdout())
		},
	}

	cmd.Flags().StringVar(&opts.relay, "relay", "", "リレー URL")
	cmd.Flags().StringVarP(&opts.message, "message", "m", "", "投稿するテキスト本文")
	cmd.Flags().StringVar(&opts.replyTo, "reply-to", "", "返信先イベント ID")

	return cmd
}
