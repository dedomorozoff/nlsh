package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/nlsh/nlsh/internal/config"
	"github.com/nlsh/nlsh/internal/model"
	"github.com/spf13/cobra"
)

func newModelCmd(rf *rootFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Управление моделями",
		Long:  "Скачать или выбрать модель для nlsh",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Показать доступные модели",
		RunE: func(cmd *cobra.Command, _ []string) error {
			d := model.New("")
			models := model.RecommendedModels

			fmt.Fprintln(cmd.OutOrStdout(), "=== Рекомендуемые модели ===")
			for i, m := range models {
				status := "[ ]"
				if d.Exists(m.Name) {
					status = "[*]"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%d. %s %s (%d MB, RAM ~%dGB)\n    %s\n    %s\n",
					i+1, status, m.Name, m.SizeMB, m.MinRAM, m.Description, m.URL)
			}
			return nil
		},
	})

	downloadCmd := &cobra.Command{
		Use:   "download [номер или имя]",
		Short: "Скачать модель",
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := cmd.Flags().Args()
			if len(args) == 0 {
				m := model.RecommendModel()
				args = append(args, m.Name)
			}

			target := args[0]
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
				return fmt.Errorf("модель %q не найдена, используй: nlsh model list", target)
			}

			d := model.New("")
			if d.Exists(info.Name) {
				fmt.Fprintf(cmd.OutOrStdout(), "Модель %s уже скачана: %s\n", info.Name, d.ModelPath(info.Name))
				return nil
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Скачиваю %s (%d MB)...\n", info.Name, info.SizeMB)

			path, err := d.Download(info, func(dl, total int) {
				if total > 0 {
					pct := dl * 100 / total
					fmt.Fprintf(cmd.OutOrStdout(), "\r  %d%% (%d / %d MB)", pct, dl/1024/1024, total/1024/1024)
				}
			})
			if err != nil {
				return fmt.Errorf("скачивание не удалось: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "\n\nГотово: %s\n", path)

			if cmd.Flags().Changed("set-default") {
				cfg, _ := config.Load()
				cfg.DefaultModel = info.Name
				if err := config.Save(cfg); err != nil {
					fmt.Fprintf(cmd.OutOrStdout(), "Не удалось сохранить в конфиг: %v\n", err)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Модель установлена по умолчанию\n")
				}
			}

			return nil
		},
	}
	downloadCmd.Flags().Bool("set-default", false, "Установить как модель по умолчанию")

	cmd.AddCommand(downloadCmd)

	return cmd
}

func addModelCommand(root *cobra.Command, rf *rootFlags) {
	modelCmd := newModelCmd(rf)
	modelCmd.Aliases = []string{"models"}
	root.AddCommand(modelCmd)

	root.AddCommand(&cobra.Command{
		Use:   "pull",
		Short: "Скачать модель (shortcut для model download)",
		RunE:  modelCmd.Commands()[1].RunE,
	})
}

func sortModelList(list []model.ModelInfo) {
	sort.Slice(list, func(i, j int) bool {
		return list[i].SizeMB < list[j].SizeMB
	})
}

func getOrDownloadModel(cmd *cobra.Command, cfg config.Config, dl func(string)) (string, error) {
	if cfg.ModelPath != "" {
		if _, err := os.Stat(cfg.ModelPath); err == nil {
			return cfg.ModelPath, nil
		}
	}

	if cfg.DefaultModel != "" {
		d := model.New("")
		if d.Exists(cfg.DefaultModel) {
			return d.ModelPath(cfg.DefaultModel), nil
		}
	}

	d := model.New("")
	available := d.ListModels()
	if len(available) > 0 {
		return d.ModelPath(available[0].Name), nil
	}

	recommended := model.RecommendModel()

	fmt.Fprintf(cmd.OutOrStdout(), `
nlsh: модель не найдена!

Рекомендуемая модель: %s (%d MB)
%s

Скачать? [Y/n]: `, recommended.Name, recommended.SizeMB, recommended.Description)

	var answer string
	fmt.Fscan(cmd.InOrStdin(), &answer)

	if answer == "" || answer == "y" || answer == "Y" {
		fmt.Fprintf(cmd.OutOrStdout(), "Скачиваю...\n")
		path, err := d.Download(recommended, func(dl, total int) {
			if total > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "\r  %d%%", dl*100/total)
			}
		})
		if err != nil {
			return "", fmt.Errorf("download failed: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), "\nГотово!")

		cfg.DefaultModel = recommended.Name
		config.Save(cfg)

		return path, nil
	}

	return "", fmt.Errorf("модель не выбрана, используй: nlsh model download")
}