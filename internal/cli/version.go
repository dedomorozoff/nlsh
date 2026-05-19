package cli

import (
	"fmt"

	"github.com/dedomorozoff/nlsh/internal/config"
	"github.com/spf13/cobra"
)

// Version is set via -ldflags at build time.
var Version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), Version)
		},
	}
}

func newInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show system and configuration info",
		Run: func(cmd *cobra.Command, _ []string) {
			hw := config.DetectHardware()
			cfg, _ := config.Load()

			fmt.Fprintln(cmd.OutOrStdout(), "=== System Information ===")
			fmt.Fprintf(cmd.OutOrStdout(), "CPU Cores:    %d\n", hw.CPUCores)
			fmt.Fprintf(cmd.OutOrStdout(), "RAM:          %d GB\n", hw.RAMGB)
			fmt.Fprintf(cmd.OutOrStdout(), "GPU:          %s\n", hw.GPUName)
			fmt.Fprintf(cmd.OutOrStdout(), "GPU Type:     %s\n", hw.GPUType)
			fmt.Fprintf(cmd.OutOrStdout(), "GPU Layers:   %d\n", hw.GPULayers)
			fmt.Fprintln(cmd.OutOrStdout(), "")
			fmt.Fprintln(cmd.OutOrStdout(), "=== Current Config ===")
			fmt.Fprintf(cmd.OutOrStdout(), "Threads:      %d\n", cfg.Threads)
			fmt.Fprintf(cmd.OutOrStdout(), "Ctx Size:     %d\n", cfg.CtxSize)
			fmt.Fprintf(cmd.OutOrStdout(), "GPU Layers:   %d\n", cfg.GPULayers)
			fmt.Fprintf(cmd.OutOrStdout(), "Max Tokens:   %d\n", cfg.MaxTokens)
			fmt.Fprintf(cmd.OutOrStdout(), "Temperature:  %.2f\n", cfg.Temperature)
			fmt.Fprintf(cmd.OutOrStdout(), "Top P:        %.2f\n", cfg.TopP)
			fmt.Fprintf(cmd.OutOrStdout(), "Shell:        %s\n", cfg.Shell)
			fmt.Fprintf(cmd.OutOrStdout(), "Dry Run:      %t\n", cfg.DryRun)
		},
	}
}
