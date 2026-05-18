package cli

import (
	"io"

	"github.com/dedomorozoff/nlsh/internal/policy"
	"github.com/dedomorozoff/nlsh/internal/prompt"
)

func renderResponse(w io.Writer, resp prompt.Response, _ policy.Decision) {
	// Стриминг выводит всё в реальном времени, поэтому renderResponse больше не нужен.
}
