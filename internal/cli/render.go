package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/nlsh/nlsh/internal/policy"
	"github.com/nlsh/nlsh/internal/prompt"
)

// renderResponse печатает разобранный ответ модели в человекочитаемом виде.
// Намеренно без цветов — добавим в этапе UX.
func renderResponse(w io.Writer, resp prompt.Response, dec policy.Decision) {
	switch resp.Intent {
	case prompt.IntentRunCommand:
		fmt.Fprintln(w, "command:")
		fmt.Fprintln(w, "  "+resp.Command)
		if resp.Explanation != "" {
			fmt.Fprintln(w, "why:")
			fmt.Fprintln(w, "  "+indent(resp.Explanation, "  "))
		}
		fmt.Fprintf(w, "risk: %s\n", riskLabel(dec.Risk))
		if dec.Reason != "" {
			fmt.Fprintf(w, "policy: %s\n", dec.Reason)
		}
	case prompt.IntentExplain:
		fmt.Fprintln(w, resp.Explanation)
	case prompt.IntentAskClarification:
		fmt.Fprintln(w, "уточни:")
		fmt.Fprintln(w, "  "+resp.Question)
	}
}

func riskLabel(r prompt.Risk) string {
	switch r {
	case prompt.RiskLow:
		return "low"
	case prompt.RiskMedium:
		return "medium"
	case prompt.RiskHigh:
		return "HIGH"
	default:
		return string(r)
	}
}

func indent(s, prefix string) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	for i, line := range lines {
		if i == 0 {
			continue
		}
		lines[i] = prefix + line
	}
	return strings.Join(lines, "\n")
}
