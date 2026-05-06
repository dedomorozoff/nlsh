package cli

import (
	"github.com/nlsh/nlsh/internal/config"
	"github.com/spf13/cobra"
)

// rootFlags хранит общие флаги, разделяемые подкомандами.
type rootFlags struct {
	cfg       config.Config
	configMod bool
}

// NewRootCmd собирает корневую cobra-команду.
func NewRootCmd() *cobra.Command {
	cfg, _ := config.Load()
	rf := &rootFlags{cfg: cfg}

	cmd := &cobra.Command{
		Use:           "nlsh",
		Short:         "Natural Language Shell — общайся с системой по-человечески",
		Long:          "nlsh — это локальный LLM-ассистент для shell. Локальная модель через llama.cpp, без облака и без HTTP-сервера.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pf := cmd.PersistentFlags()
	pf.StringVar(&rf.cfg.ModelPath, "model", rf.cfg.ModelPath, "путь к GGUF-модели")
	pf.IntVar(&rf.cfg.Threads, "threads", rf.cfg.Threads, "число потоков для инференса")
	pf.IntVar(&rf.cfg.CtxSize, "ctx-size", rf.cfg.CtxSize, "размер контекста токенов")
	pf.IntVar(&rf.cfg.GPULayers, "gpu-layers", rf.cfg.GPULayers, "количество слоёв на GPU (0 = только CPU)")
	pf.IntVar(&rf.cfg.MaxTokens, "max-tokens", rf.cfg.MaxTokens, "максимум токенов в ответе модели")
	pf.Float32Var(&rf.cfg.Temperature, "temperature", rf.cfg.Temperature, "температура сэмплирования")
	pf.Float32Var(&rf.cfg.TopP, "top-p", rf.cfg.TopP, "top-p сэмплирование")
	pf.StringVar(&rf.cfg.Shell, "shell", rf.cfg.Shell, "shell для исполнения команд")
	pf.BoolVar(&rf.cfg.DryRun, "dry-run", rf.cfg.DryRun, "не выполнять команды, только показывать")

	cmd.AddCommand(newAskCmd(rf))
	cmd.AddCommand(newRunCmd(rf))
	cmd.AddCommand(newReplCmd(rf))
	cmd.AddCommand(newVersionCmd())

	return cmd
}
