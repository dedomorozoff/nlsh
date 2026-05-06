package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// confirmer спрашивает пользователя y/n. Возвращает true только при явном yes.
func confirm(in io.Reader, out io.Writer, prompt string) bool {
	fmt.Fprintf(out, "%s [y/N]: ", prompt)
	sc := bufio.NewScanner(in)
	if !sc.Scan() {
		return false
	}
	answer := strings.ToLower(strings.TrimSpace(sc.Text()))
	return answer == "y" || answer == "yes"
}
