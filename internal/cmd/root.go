package cmd

import (
	"log/slog"
	"os"
	"sync"

	"github.com/spf13/cobra"

	"noscli/internal/config"
	"noscli/internal/logging"
)

var (
	verbose bool

	configOnce sync.Once
	cfg        config.Config

	loggerOnce sync.Once
	logger     *slog.Logger
)

var rootCmd = &cobra.Command{
	Use:           "noscli",
	Short:         "Nostr CLI client",
	SilenceUsage:  true,
	SilenceErrors: false,
	RunE: func(cmd *cobra.Command, args []string) error {
		return cmd.Help()
	},
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "詳細ログを表示する")
	rootCmd.AddCommand(newTimelineCommand())
}

// Execute runs the root command.
func Execute() error {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)
	return rootCmd.Execute()
}

func loadConfig() config.Config {
	configOnce.Do(func() {
		cfg = config.Load()
	})
	return cfg
}

func getLogger() *slog.Logger {
	loggerOnce.Do(func() {
		logger = logging.New(verbose)
	})
	return logger
}
