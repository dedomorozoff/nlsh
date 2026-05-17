package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Config описывает рантайм-настройки nlsh. Поля сознательно плоские,
// чтобы их легко было пробрасывать из флагов CLI и из JSON-файла.
type Config struct {
	ModelPath   string  `json:"model_path"`
	DefaultModel string `json:"default_model"`
	Threads     int     `json:"threads"`
	CtxSize     int     `json:"ctx_size"`
	GPULayers   int     `json:"gpu_layers"`
	MaxTokens   int     `json:"max_tokens"`
	Temperature float32 `json:"temperature"`
	TopP        float32 `json:"top_p"`
	Shell       string  `json:"shell"`
	HistoryFile string  `json:"history_file"`
	DryRun      bool    `json:"dry_run"`
}

// Default возвращает дефолтную конфигурацию. Значения подобраны под
// маленькие instruct-модели 3B-8B в Q4_K_M и слабое железо.
func Default() Config {
	return Config{
		Threads:     runtime.NumCPU(),
		CtxSize:     4096,
		GPULayers:   0,
		MaxTokens:   512,
		Temperature: 0.2,
		TopP:        0.9,
		Shell:       defaultShell(),
		HistoryFile: defaultHistoryFile(),
		DryRun:      false,
	}
}

// Load читает конфиг из ~/.config/nlsh/config.json, если он есть, и
// мерджит его поверх дефолтов. Отсутствие файла — не ошибка.
func Load() (Config, error) {
	cfg := Default()
	path, err := userConfigPath()
	if err != nil {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read config: %w", err)
	}
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %s: %w", path, err)
	}
	return cfg, nil
}

// Save сохраняет конфигурацию в ~/.config/nlsh/config.json
func Save(cfg Config) error {
	path, err := userConfigPath()
	if err != nil {
		return fmt.Errorf("config path: %w", err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func userConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "nlsh", "config.json"), nil
}

func defaultShell() string {
	if runtime.GOOS == "windows" {
		return "cmd"
	}
	if sh := os.Getenv("SHELL"); sh != "" {
		return sh
	}
	return "/bin/sh"
}

func defaultHistoryFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "nlsh", "history.jsonl")
}
