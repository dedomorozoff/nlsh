package cli

import (
	"fmt"
	"io"

	"github.com/nlsh/nlsh/internal/policy"
	"github.com/nlsh/nlsh/internal/prompt"
)

func renderResponse(w io.Writer, resp prompt.Response, _ policy.Decision) {
	switch resp.Intent {
	case prompt.IntentRunCommand:
		if resp.Explanation != "" {
			fmt.Fprintf(w, "  %s\n", resp.Explanation)
		}
		fmt.Fprintf(w, "  $ %s\n", resp.Command)
	case prompt.IntentExplain:
		fmt.Fprintln(w, resp.Explanation)
	case prompt.IntentAskClarification:
		fmt.Fprintf(w, "  %s\n", resp.Question)
	}
}