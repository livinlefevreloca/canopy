package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/livinlefevreloca/canopy/internal/backend"
	"github.com/livinlefevreloca/canopy/internal/ipc"
	"github.com/livinlefevreloca/canopy/internal/logging"
	"github.com/livinlefevreloca/canopy/internal/tui"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:   "canopy",
		Short: "Canopy is a TUI for aws",
		Long: `Canopy is a terminal user interface for managing and monitoring your aws resources.
				It is meant as a replacement for AWS console providing funcationality that us missing
				from the console but is already available in the CLI. It is meant to be used as a
				daily driver for managing your aws resources.`,
		Run: RunRootCmd,
	}
	rootArgs struct {
		Profile string
		Region  string
	}
)

func RunRootCmd(cmd *cobra.Command, args []string) {

	os.Rename("./.canopy.log", fmt.Sprintf("./.canopy.log.bak-%d", time.Now().Unix())) // Backup previous log file if it exists

	logging.ConfigureLogger(logging.LoggingConfig{
		LogLevel: "DEBUG",
		LogFile:  "./.canopy.log",
		Json:     false,
	})

	tx := make(chan ipc.Trigger, 100) // Buffered channel for outgoing triggers
	server := backend.NewServer(&tx, rootArgs.Profile, rootArgs.Region)
	go server.Run()
	requestHandler := ipc.NewTriggerHandler(&tx)
	tui := tui.NewTui(requestHandler)
	err := tui.Run()
	if err != nil {
		slog.Error("Failed to run TUI", "error", err)
		return
	}

}

func Run() error {
	rootCmd.PersistentFlags().StringVarP(&rootArgs.Profile, "profile", "p", "", "AWS profile to use")
	rootCmd.PersistentFlags().StringVarP(&rootArgs.Region, "region", "r", "", "AWS region to use")

	if err := rootCmd.Execute(); err != nil {
		return err
	}
	return nil
}
