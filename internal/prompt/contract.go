package prompt

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Intent перечисляет типы намерений, которые модель может вернуть.
type Intent string

const (
	IntentRunCommand       Intent = "run_command"
	IntentExplain          Intent = "explain"
	IntentAskClarification Intent = "ask_clarification"
)

// Risk — категория риска предлагаемой команды.
type Risk string

const (
	RiskLow    Risk = "low"
	RiskMedium Risk = "medium"
	RiskHigh   Risk = "high"
)

// Response — это контракт, по которому LLM обязана возвращать ответ.
// Мы парсим строго это и ничего больше; всё остальное — ошибка формата.
type Response struct {
	Intent            Intent `json:"intent"`
	Command           string `json:"command,omitempty"`
	Explanation       string `json:"explanation,omitempty"`
	Risk              Risk   `json:"risk_level,omitempty"`
	NeedsConfirmation bool   `json:"needs_confirmation,omitempty"`
	Question          string `json:"question,omitempty"`
}

// Validate проверяет внутреннюю согласованность ответа и нормализует поля.
func (r *Response) Validate() error {
	switch r.Intent {
	case IntentRunCommand:
		if strings.TrimSpace(r.Command) == "" {
			return errors.New("intent=run_command, but command is empty")
		}
		if r.Risk == "" {
			r.Risk = RiskMedium
		}
		if r.Risk == RiskMedium || r.Risk == RiskHigh {
			r.NeedsConfirmation = true
		}
	case IntentExplain:
		if strings.TrimSpace(r.Explanation) == "" {
			return errors.New("intent=explain, but explanation is empty")
		}
	case IntentAskClarification:
		if strings.TrimSpace(r.Question) == "" {
			return errors.New("intent=ask_clarification, but question is empty")
		}
	default:
		return fmt.Errorf("unknown intent: %q", r.Intent)
	}
	return nil
}

// Parse извлекает первый JSON-объект из произвольного текста модели и
// разбирает его в Response. Толерантен к префиксам/суффиксам типа
// "Sure, here is the JSON:" и тройных бэктиков.
func Parse(raw string) (Response, error) {
	jsonText, ok := extractJSONObject(raw)
	if !ok {
		return Response{}, fmt.Errorf("no JSON object found in model output")
	}
	var resp Response
	dec := json.NewDecoder(strings.NewReader(jsonText))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&resp); err != nil {
		// Делаем второй проход без strict-режима: модели часто добавляют
		// лишние поля, и нам важнее извлечь валидное ядро.
		if err2 := json.Unmarshal([]byte(jsonText), &resp); err2 != nil {
			return Response{}, fmt.Errorf("decode json: %w", err)
		}
	}
	if err := resp.Validate(); err != nil {
		return resp, fmt.Errorf("invalid response: %w", err)
	}
	return resp, nil
}

// extractJSONObject ищет первый сбалансированный JSON-объект в строке.
// Учитывает кавычки и экранирование, чтобы скобки в строках не сбивали баланс.
func extractJSONObject(s string) (string, bool) {
	start := strings.IndexByte(s, '{')
	if start == -1 {
		return "", false
	}
	depth := 0
	inString := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1], true
			}
		}
	}
	return "", false
}
