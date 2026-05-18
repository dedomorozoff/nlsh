package cli

import (
	"context"
	"errors"
	"strings"

	"github.com/spf13/cobra"
)

func newAskCmd(rf *rootFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "ask <запрос>",
		Short: "Объяснить, как сделать, ничего не выполняя",
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
			resp, err := askWithFollowUp(ctx, s, "ask", input, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return err
			}
			renderResponse(cmd.OutOrStdout(), resp, evaluatePolicy(resp))
			return nil
		},
	}
}
