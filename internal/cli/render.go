package cli

import (
	"fmt"
	"io"

	"github.com/nlsh/nlsh/internal/policy"
	"github.com/nlsh/nlsh/internal/prompt"
)

func renderResponse(w io.Writer, resp prompt.Response, _ policy.Decision) {
	fmt.Fprintf(w, "%s[ai]%s ", cyan, reset)
	switch resp.Intent {
	case prompt.IntentRunCommand:
		if resp.Explanation != "" {
			fmt.Fprintf(w, "%s%s%s\n", cyan, resp.Explanation, reset)
		}
		fmt.Fprintf(w, "%s$%s %s\n", gray, reset, resp.Command)
	case prompt.IntentExplain:
		fmt.Fprintf(w, "%s%s%s\n", cyan, resp.Explanation, reset)
	case prompt.IntentAskClarification:
		if resp.Question != "" {
			fmt.Fprintf(w, "%s%s%s\n", cyan, resp.Question, reset)
		} else if resp.Explanation != "" {
			fmt.Fprintf(w, "%s%s%s\n", cyan, resp.Explanation, reset)
		}
	}
}