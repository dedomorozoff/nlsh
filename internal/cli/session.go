package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/dedomorozoff/nlsh/internal/config"
	"github.com/dedomorozoff/nlsh/internal/llm"
	"github.com/dedomorozoff/nlsh/internal/model"
	"github.com/dedomorozoff/nlsh/internal/prompt"
)

// session инкапсулирует движок и собранный для него контекст промпта.
// Один session живёт всё время REPL или одной CLI-команды.
type session struct {
	cfg    config.Config
	engine llm.Engine
	recent []string
}

func newSession(cfg config.Config) (*session, error) {
	modelPath, err := resolveModelPath(cfg)
	if err != nil {
		return nil, err
	}
	cfg.ModelPath = modelPath

	eng, err := llm.New(llm.Params{
		ModelPath: cfg.ModelPath,
		Threads:   cfg.Threads,
		CtxSize:   cfg.CtxSize,
		GPULayers: cfg.GPULayers,
	})
	if err != nil {
		return nil, fmt.Errorf("загрузка модели: %w", err)
	}
	return &session{cfg: cfg, engine: eng}, nil
}

func resolveModelPath(cfg config.Config) (string, error) {
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

	return "", errors.New("модель не найдена, выполни: nlsh model download")
}

func (s *session) close() {
	if s.engine != nil {
		_ = s.engine.Close()
	}
}

// ask отправляет запрос модели и пытается вытащить из ответа Response.
// Один retry в случае невалидного JSON: добавляем repair-инструкцию.
func (s *session) ask(ctx context.Context, mode, userInput string) (prompt.Response, string, error) {
	cwd, _ := os.Getwd()
	pctx := prompt.Context{
		OS:          osName(),
		Shell:       s.cfg.Shell,
		CWD:         cwd,
		RecentCmds:  s.recent,
		UserRequest: userInput,
		Mode:        mode,
	}
	system := prompt.BuildSystem(pctx)
	user := prompt.BuildUser(pctx)

	opts := llm.SamplingOptions{
		MaxTokens:   s.cfg.MaxTokens,
		Temperature: s.cfg.Temperature,
		TopP:        s.cfg.TopP,
		StopTokens:  []string{"<|im_end|>", "</s>"},
	}

	raw, err := s.engine.Generate(ctx, system, user, opts)
	if err != nil {
		return prompt.Response{}, raw, err
	}
	resp, perr := prompt.Parse(raw)
	if perr == nil {
		return resp, raw, nil
	}

	repair := user + "\n\nПредыдущий ответ был не валидным JSON. Верни строго один JSON-объект по схеме без любого текста вокруг."
	raw2, err := s.engine.Generate(ctx, system, repair, opts)
	if err != nil {
		return prompt.Response{}, raw, err
	}
	resp2, perr2 := prompt.Parse(raw2)
	if perr2 != nil {
		return prompt.Response{}, raw + "\n---\n" + raw2, fmt.Errorf("не удалось распарсить ответ модели: %w", perr2)
	}
	return resp2, raw2, nil
}

func (s *session) addRecent(cmd string) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return
	}
	s.recent = append(s.recent, cmd)
	if len(s.recent) > 10 {
		s.recent = s.recent[len(s.recent)-10:]
	}
}
