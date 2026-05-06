package llm

import (
	"context"
	"errors"
)

// Params — настройки загрузки модели и контекста.
type Params struct {
	ModelPath string
	Threads   int
	CtxSize   int
	GPULayers int
}

// SamplingOptions — параметры генерации одного запроса.
type SamplingOptions struct {
	MaxTokens   int
	Temperature float32
	TopP        float32
	StopTokens  []string
	// Seed=0 -> случайный.
	Seed uint32
}

// Engine — обобщённый интерфейс инференс-движка. Реальная реализация
// (CGO над llama.cpp) и stub соблюдают его одинаково, что позволяет
// собирать бинарь без CGO для тестирования прочей логики.
type Engine interface {
	// Generate выполняет один запрос. systemPrompt и userPrompt модель
	// получит как чат: system role + user role. Реализация сама форматирует
	// под токенайзер модели.
	Generate(ctx context.Context, systemPrompt, userPrompt string, opts SamplingOptions) (string, error)

	// Stream — потоковая генерация. tokens закрывается, когда генерация
	// завершилась штатно или была отменена через ctx.
	Stream(ctx context.Context, systemPrompt, userPrompt string, opts SamplingOptions, tokens chan<- string) error

	Close() error
}

// ErrNotBuiltWithCGO возвращается stub-реализацией, когда бинарь собран
// без тега `llama` и реальный движок недоступен.
var ErrNotBuiltWithCGO = errors.New("llm: this build has no llama.cpp linked; rebuild with `-tags llama`")
