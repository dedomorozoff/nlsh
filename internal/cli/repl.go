package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/nlsh/nlsh/internal/executor"
	"github.com/nlsh/nlsh/internal/prompt"
	"github.com/spf13/cobra"
)

func newReplCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "repl",
		Short: "Интерактивный режим",
		RunE: func(cmd *cobra.Command, _ []string) error {
			s, err := newSession(rf.cfg)
			if err != nil {
				return err
			}
			defer s.close()

			out := cmd.OutOrStdout()
			in := cmd.InOrStdin()
			fmt.Fprintln(out, "nlsh repl — введи запрос на естественном языке. /help, /exit")

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			scanner := bufio.NewScanner(in)
			scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
			for {
				fmt.Fprint(out, "\n> ")
				if !scanner.Scan() {
					return scanner.Err()
				}
				line := strings.TrimSpace(scanner.Text())
				if line == "" {
					continue
				}
				if strings.HasPrefix(line, "/") {
					if stop := handleSlash(line, out); stop {
						return nil
					}
					continue
				}
				if err := handleTurn(ctx, s, rf, line, in, out, cmd.ErrOrStderr()); err != nil {
					if errors.Is(err, context.Canceled) {
						return nil
					}
					fmt.Fprintln(cmd.ErrOrStderr(), "ошибка:", err)
				}
			}
		},
	}
}

func handleSlash(line string, out io.Writer) (stop bool) {
	switch {
	case line == "/exit", line == "/quit":
		return true
	case line == "/help":
		fmt.Fprintln(out, "/help        — эта справка")
		fmt.Fprintln(out, "/exit        — выход")
		fmt.Fprintln(out, "просто пиши  — модель предложит команду, можно подтвердить выполнение")
	default:
		fmt.Fprintln(out, "неизвестная команда:", line)
	}
	return false
}

func handleTurn(ctx context.Context, s *session, rf *rootFlags, input string, in io.Reader, out, errW io.Writer) error {
	resp, raw, err := s.ask(ctx, "run", input)
	if err != nil {
		if raw != "" {
			fmt.Fprintln(errW, "raw output:")
			fmt.Fprintln(errW, raw)
		}
		return err
	}
	dec := evaluatePolicy(resp)
	renderResponse(out, resp, dec)

	if resp.Intent != prompt.IntentRunCommand {
		return nil
	}
	if rf.cfg.DryRun {
		fmt.Fprintln(out, "(dry-run: команда не запущена)")
		return nil
	}
	if !dec.Allowed {
		fmt.Fprintln(out, "(команда заблокирована политикой безопасности)")
		return nil
	}
	if dec.Risk != prompt.RiskLow || resp.NeedsConfirmation {
		if !confirm(in, out, "выполнить?") {
			fmt.Fprintln(out, "(отменено)")
			return nil
		}
	}
	res := executor.Run(ctx, rf.cfg.Shell, resp.Command)
	s.addRecent(resp.Command)
	if res.Stdout != "" {
		fmt.Fprint(out, res.Stdout)
	}
	if res.Stderr != "" {
		fmt.Fprint(errW, res.Stderr)
	}
	if res.Err != nil {
		return fmt.Errorf("exit %d: %w", res.ExitCode, res.Err)
	}
	return nil
}
