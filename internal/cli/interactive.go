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
	fmt.Fprintln(out, "nlsh interactive mode — введи запрос. /help, /exit")

	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	return replLoop(ctx, s, rf, in, out, cmd.ErrOrStderr())
}

