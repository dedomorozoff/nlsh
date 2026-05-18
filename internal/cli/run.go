package cli

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/dedomorozoff/nlsh/internal/executor"
	"github.com/dedomorozoff/nlsh/internal/policy"
	"github.com/dedomorozoff/nlsh/internal/prompt"
	"github.com/spf13/cobra"
)

func newRunCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "run <запрос>",
		Short: "Предложить и выполнить команду по запросу",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := strings.Join(args, " ")
			if strings.TrimSpace(input) == "" {
				return errors.New("пустой запрос")
			}
			s, err := newSession(rf.cfg)
			if err != nil {
				return err
			}
			defer s.close()

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}
	resp, err := askWithFollowUp(ctx, s, "run", input, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	dec := evaluatePolicy(resp)
			renderResponse(cmd.OutOrStdout(), resp, dec)

			if resp.Intent != prompt.IntentRunCommand {
				return nil
			}
			if rf.cfg.DryRun {
				fmt.Fprintln(cmd.OutOrStdout(), "(dry-run: команда не запущена)")
				return nil
			}
			if !dec.Allowed {
				fmt.Fprintln(cmd.OutOrStdout(), "(команда заблокирована политикой безопасности)")
				return nil
			}
			if dec.Risk != prompt.RiskLow || resp.NeedsConfirmation {
				if !confirm(cmd.InOrStdin(), cmd.OutOrStdout(), "выполнить?") {
					fmt.Fprintln(cmd.OutOrStdout(), "(отменено)")
					return nil
				}
			}
			res := executor.Run(ctx, rf.cfg.Shell, resp.Command)
			s.addRecent(resp.Command)
			if res.Stdout != "" {
				fmt.Fprint(cmd.OutOrStdout(), res.Stdout)
			}
			if res.Stderr != "" {
				fmt.Fprint(cmd.ErrOrStderr(), res.Stderr)
			}
			if res.Err != nil {
				return fmt.Errorf("exit %d: %w", res.ExitCode, res.Err)
			}
			return nil
		},
	}
}

// evaluatePolicy — обёртка над policy.Evaluate, удобная для render-слоя.
func evaluatePolicy(resp prompt.Response) policy.Decision {
	if resp.Intent != prompt.IntentRunCommand {
		return policy.Decision{Allowed: true, Risk: prompt.RiskLow}
	}
	return policy.Evaluate(resp.Command, resp.Risk)
}
