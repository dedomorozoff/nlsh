package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
)

// runInteractive запускает REPL как дефолтное поведение root-команды.
func runInteractive(cmd *cobra.Command, rf *rootFlags) error {
	s, err := newSession(rf.cfg)
	if err != nil {
		return err
	}
	defer s.close()

	out := cmd.OutOrStdout()
	in := cmd.InOrStdin()

	banner := fmt.Sprintf("%s%s.nlsh%s — Natural Language Shell\n%sНапиши запрос или /help для справки%s\n\n",
		bold, cyan, reset, gray, reset)
	fmt.Fprint(out, banner)

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	return replLoop(ctx, s, rf, in, out, cmd.ErrOrStderr())
}

