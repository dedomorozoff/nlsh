package cli

import (
	"context"
	"errors"
	"fmt"
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
			resp, raw, err := s.ask(ctx, "ask", input)
			if err != nil {
				if raw != "" {
					fmt.Fprintln(cmd.ErrOrStderr(), "raw output:")
					fmt.Fprintln(cmd.ErrOrStderr(), raw)
				}
				return err
			}
			renderResponse(cmd.OutOrStdout(), resp, evaluatePolicy(resp))
			return nil
		},
	}
}
