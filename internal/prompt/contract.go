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
		if strings.TrimSpace(r.Question) == "" && strings.TrimSpace(r.Explanation) == "" {
			return errors.New("intent=ask_clarification, but question and explanation are empty")
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
	objects := extractAllJSONObjects(raw)
	if len(objects) == 0 {
		return Response{}, fmt.Errorf("no JSON object found in model output")
	}

	var lastErr error
	for _, jsonText := range objects {
		var resp Response
		dec := json.NewDecoder(strings.NewReader(jsonText))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&resp); err != nil {
			if err2 := json.Unmarshal([]byte(jsonText), &resp); err2 != nil {
				lastErr = fmt.Errorf("decode json: %w", err)
				continue
			}
		}
		if err := resp.Validate(); err != nil {
			lastErr = err
			continue
		}
		return resp, nil
	}
	return Response{}, fmt.Errorf("invalid response: %w", lastErr)
}

// extractAllJSONObjects извлекает все сбалансированные JSON-объекты из строки.
func extractAllJSONObjects(s string) []string {
	var results []string
	for {
		start := strings.IndexByte(s, '{')
		if start == -1 {
			break
		}
		s = s[start:]
		depth := 0
		inString := false
		escaped := false
		for i := 0; i < len(s); i++ {
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
					results = append(results, s[:i+1])
					s = s[i+1:]
					break
				}
			}
		}
		break
	}
	return results
}
