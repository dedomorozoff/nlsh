package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dedomorozoff/nlsh/internal/config"
	"github.com/dedomorozoff/nlsh/internal/model"
	"github.com/spf13/cobra"
)

func newModelCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Model management",
		Long:  "Download or select a model for nlsh",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List available models",
		RunE: func(cmd *cobra.Command, _ []string) error {
			d := model.New("")
			out := cmd.OutOrStdout()

			fmt.Fprintln(out, "=== Recommended ===")
			for i, m := range model.RecommendedModels {
				status := "[ ]"
				if d.Exists(m.Name) {
					status = "[*]"
				}
				fmt.Fprintf(out, "%d. %s %s (%d MB)\n    %s\n    %s\n",
					i+1, status, m.Name, m.SizeMB, m.Description, m.URL)
			}

			all, err := d.ListAllModels()
			if err != nil {
				fmt.Fprintf(out, "scan error: %v\n", err)
			}
			if len(all) > 0 {
				fmt.Fprintln(out, "\n=== Downloaded ===")
				for _, m := range all {
					size := ""
					if fi, err := os.Stat(d.ModelPath(m.Name)); err == nil {
						size = fmt.Sprintf(" (%d MB)", fi.Size()/1024/1024)
					}
					fmt.Fprintf(out, "  %s%s\n", m.Name, size)
				}
			}
			return nil
		},
	})

	downloadCmd := &cobra.Command{
		Use:   "download [number, name, or URL]",
		Short: "Download a model (URL or from list)",
		Long: `Downloads a GGUF model. You can specify:
  a number from the list (nlsh model list)
  a model name from the list
  a direct URL to a .gguf file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) > 0 {
				target = args[0]
			}

			d := model.New("")

			if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
				if strings.HasSuffix(strings.ToLower(target), ".gguf") {
					if d.Exists(target) {
						fmt.Fprintf(cmd.OutOrStdout(), "Already downloaded: %s\n", d.ModelPath(target))
						return nil
					}
					return downloadURL(cmd, d, target)
				}
				return fmt.Errorf("URL must point to a .gguf file")
			}

			if target == "" {
				m := model.RecommendModel()
				target = m.Name
			}

			models := model.RecommendedModels
			var info model.ModelInfo
			found := false

			if num, err := strconv.Atoi(target); err == nil && num > 0 && num <= len(models) {
				info = models[num-1]
				found = true
			} else {
				for _, m := range models {
					if m.Name == target {
						info = m
						found = true
						break
					}
				}
			}

			if !found {
				errMsg := fmt.Sprintf("model %q not found in the list.\n", target)
				errMsg += "  Use: nlsh model list\n"
				errMsg += "  Or provide a direct URL to a .gguf file"
				return fmt.Errorf(errMsg)
			}

			if d.Exists(info.Name) {
				fmt.Fprintf(cmd.OutOrStdout(), "Model %s already downloaded: %s\n", info.Name, d.ModelPath(info.Name))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Downloading %s (%d MB)...\n", info.Name, info.SizeMB)
			path, err := d.Download(info, progressFn(cmd.OutOrStdout()))
			if err != nil {
				return fmt.Errorf("download failed: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n\nDone: %s\n", path)

			if cmd.Flags().Changed("set-default") {
				setDefault(cmd, info.Name)
			}
			return nil
		},
	}
	downloadCmd.Flags().Bool("set-default", false, "Set as default model")

	cmd.AddCommand(downloadCmd)

	cmd.AddCommand(&cobra.Command{
		Use:   "path [name]",
		Short: "Show path to a downloaded model",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d := model.New("")
			if !d.Exists(args[0]) {
				return fmt.Errorf("model %q not found", args[0])
			}
			fmt.Fprintln(cmd.OutOrStdout(), d.ModelPath(args[0]))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "use [name]",
		Short: "Set a downloaded model as default",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			d := model.New("")
			if !d.Exists(args[0]) {
				return fmt.Errorf("model %q not found in %s", args[0], d.ModelPath(""))
			}
			setDefault(cmd, args[0])
			return nil
		},
	})

	return cmd
}

func downloadURL(cmd *cobra.Command, d *model.Downloader, url string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "Downloading %s ...\n", url)
	path, err := d.DownloadURL(url, progressFn(cmd.OutOrStdout()))
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "\n\nDone: %s\n", path)

	if cmd.Flags().Changed("set-default") {
		name := path
		if idx := strings.LastIndexAny(name, "/\\"); idx >= 0 {
			name = name[idx+1:]
		}
		setDefault(cmd, name)
	}
	return nil
}

func setDefault(cmd *cobra.Command, name string) {
	cfg, _ := config.Load()
	cfg.DefaultModel = name
	if err := config.Save(cfg); err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Failed to save config: %v\n", err)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Model set as default\n")
	}
}

func progressFn(out io.Writer) func(dl, total int) {
	return func(dl, total int) {
		if total > 0 {
			pct := dl * 100 / total
			fmt.Fprintf(out, "\r  %d%% (%d / %d MB)", pct, dl/1024/1024, total/1024/1024)
		}
	}
}

func addModelCommand(root *cobra.Command, rf *rootFlags) {
	modelCmd := newModelCmd(rf)
	modelCmd.Aliases = []string{"models"}
	root.AddCommand(modelCmd)

	root.AddCommand(&cobra.Command{
		Use:   "pull",
		Short: "Download a model (shortcut for model download)",
		RunE:  modelCmd.Commands()[1].RunE,
	})

	root.AddCommand(newInfoCmd())
}
