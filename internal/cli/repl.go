package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/dedomorozoff/nlsh/internal/executor"
	"github.com/dedomorozoff/nlsh/internal/feedback"
	"github.com/dedomorozoff/nlsh/internal/prompt"
	"github.com/spf13/cobra"
)

var (
	reset  = "\033[0m"
	bold   = "\033[1m"
	cyan   = "\033[36m"
	green  = "\033[32m"
	yellow = "\033[33m"
	red    = "\033[31m"
	gray   = "\033[90m"
)

func flushOutput(w io.Writer) {
	if f, ok := w.(*os.File); ok {
		os.Stderr.Sync()
		if f == os.Stdout || f == os.Stderr {
			os.Stdout.Sync()
		}
	}
}

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

			banner := fmt.Sprintf("%s%s.nlsh%s — Natural Language Shell (%srepl%s mode)\n%sНапиши запрос или /help для справки%s\n\n",
				bold, cyan, reset, green, reset, gray, reset)
			fmt.Fprint(out, banner)

			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			// Try readline, fallback to bufio if not TTY
			if isTTY := isTerminal(in); isTTY {
				return replLoopReadline(ctx, s, rf, out, cmd.ErrOrStderr())
			}
			return replLoop(ctx, s, rf, in, out, cmd.ErrOrStderr())
		},
	}
}

func replLoop(ctx context.Context, s *session, rf *rootFlags, in io.Reader, out, errW io.Writer) error {
	usr, _ := user.Current()
	hostname, _ := os.Hostname()

	isTTY := isTerminal(in)

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for {
		cwd, _ := os.Getwd()
		promptStr := buildPrompt(usr.Username, hostname, cwd, isTTY)

		fmt.Fprint(out, promptStr)
		fmt.Fprint(out, " ")  // space after prompt
		flushOutput(out)

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(errW, "scanner error: %v\n", err)
				return err
			}
			fmt.Fprintln(out, "\n[EOF - exit]")
			return nil
		}

		line := scanner.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "/") {
			if stop := handleSlash(line, out); stop {
				return nil
			}
			continue
		}

		if err := handleTurn(ctx, s, rf, line, in, out, errW); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			fmt.Fprintf(errW, "%s%s%s\n", red, err, reset)
		}
	}
}

func buildPrompt(username, hostname, cwd string, isTTY bool) string {
	_ = username // suppress unused warning
	short := shortPath(cwd)
	return fmt.Sprintf("%s[%s]%s> ", gray, short, reset)
}

func shortPath(p string) string {
	home, _ := os.UserHomeDir()
	if home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}

	if runtime.GOOS == "windows" {
		if len(p) > 3 {
			return p
		}
		return p
	}

	if len(p) > 40 {
		return "..." + p[len(p)-37:]
	}
	return p
}

func isTerminal(r io.Reader) bool {
	if f, ok := r.(*os.File); ok {
		info, err := f.Stat()
		if err != nil {
			return false
		}
		return (info.Mode() & os.ModeCharDevice) != 0
	}
	return false
}

// replLoopReadline — REPL с readline-подобными хоткеями
func replLoopReadline(ctx context.Context, s *session, rf *rootFlags, out, errW io.Writer) error {
	usr, _ := user.Current()
	hostname, _ := os.Hostname()

	// History file path
	historyPath := filepath.Join(filepath.Dir(rf.cfg.HistoryFile), "readline_history")
	_ = os.MkdirAll(filepath.Dir(historyPath), 0755)

	// Initialize readline with history
	config := &readline.Config{
		Prompt:          buildPrompt(usr.Username, hostname, "", false) + " ",
		HistoryFile:     historyPath,
		HistoryLimit:    1000,
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	}

	// Filter input for special keys
	config.FuncFilterInputRune = func(r rune) (rune, bool) {
		// Ctrl+L (0x0c) - clear screen
		if r == 0x0c {
			clearScreen(out)
			return 0, false
		}
		return r, true
	}

	rl, err := readline.NewEx(config)
	if err != nil {
		// Fallback to basic mode if readline fails
		fmt.Fprintf(errW, "%sreadline initialization failed: %v, using basic mode%s\n", yellow, err, reset)
		return replLoop(ctx, s, rf, os.Stdin, out, errW)
	}
	defer rl.Close()

	for {
		cwd, _ := os.Getwd()
		rl.SetPrompt(buildPrompt(usr.Username, hostname, cwd, true) + " ")

		line, err := rl.Readline()
		if err != nil {
			if errors.Is(err, readline.ErrInterrupt) {
				fmt.Fprintln(out, "\n^C")
				continue
			}
			if errors.Is(err, io.EOF) {
				fmt.Fprintln(out, "\n[EOF - exit]")
				return nil
			}
			fmt.Fprintf(errW, "%sreadline error: %v%s\n", red, err, reset)
			return err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Add to history
		_ = rl.SaveHistory(line)

		if strings.HasPrefix(line, "/") {
			if stop := handleSlash(line, out); stop {
				return nil
			}
			continue
		}

		if err := handleTurn(ctx, s, rf, line, os.Stdin, out, errW); err != nil {
			if errors.Is(err, context.Canceled) {
				return nil
			}
			fmt.Fprintf(errW, "%s%s%s\n", red, err, reset)
		}
	}
}

func handleSlash(line string, out io.Writer) (stop bool) {
	switch {
	case line == "/exit", line == "/quit", line == "exit", line == "quit":
		fmt.Fprintln(out, "до встречи!")
		return true
	case line == "/help", line == "help":
		showHelp(out)
	case strings.HasPrefix(line, "/cd "):
		target := strings.TrimPrefix(line, "/cd ")
		target = strings.TrimSpace(target)
		if err := os.Chdir(target); err != nil {
			fmt.Fprintf(out, "%s%s%s\n", red, err, reset)
		}
	case line == "/cd":
		if home, err := os.UserHomeDir(); err == nil {
			if err := os.Chdir(home); err != nil {
				fmt.Fprintf(out, "%s%s%s\n", red, err, reset)
			}
		}
	case line == "/clear", line == "clear":
		clearScreen(out)
	case line == "/pwd", line == "pwd":
		wd, _ := os.Getwd()
		fmt.Fprintln(out, wd)
	case line == "/history", line == "history":
		fmt.Fprintln(out, "история... (используйте Ctrl+R для поиска)")
	case line == "/bind", line == "/bind keys":
		showKeyBindings(out)
	default:
		if strings.HasPrefix(line, "!") {
			cmd := strings.TrimSpace(strings.TrimPrefix(line, "!"))
			fmt.Fprintf(out, "%s$ %s%s\n", cyan, reset, cmd)
			return false
		}
		fmt.Fprintf(out, "%sнеизвестная команда: %s%s\n", red, line, reset)
	}
	return false
}

func showKeyBindings(out io.Writer) {
	fmt.Fprintf(out, "%s%s=== Доступные хоткеи ===%s\n\n", bold, cyan, reset)
	fmt.Fprintf(out, "%sОсновные:%s\n", bold, reset)
	fmt.Fprintf(out, "  %sCtrl+A%s     — начало строки\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+E%s     — конец строки\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+U%s     — удалить до начала строки\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+K%s     — удалить до конца строки\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+L%s     — очистить экран\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+R%s     — обратный поиск по истории\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+S%s     — прямой поиск по истории\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+P%s     — предыдущая команда\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+N%s     — следующая команда\n", yellow, reset)
	fmt.Fprintf(out, "  %sAlt+B%s      — назад по слову\n", yellow, reset)
	fmt.Fprintf(out, "  %sAlt+F%s      — вперед по слову\n", yellow, reset)
	fmt.Fprintf(out, "  %sAlt+D%s      — удалить слово вперед\n", yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+W%s     — удалить слово назад\n", yellow, reset)
	fmt.Fprintf(out, "\n%sСпециальные:%s\n", bold, reset)
	fmt.Fprintf(out, "  %s/exit%s      — выйти из REPL\n", yellow, reset)
	fmt.Fprintf(out, "  %s/cd%s путь   — сменить директорию\n", yellow, reset)
	fmt.Fprintf(out, "  %s/clear%s     — очистить экран\n", yellow, reset)
	fmt.Fprintf(out, "  %s/pwd%s       — показать текущую директорию\n", yellow, reset)
	fmt.Fprintf(out, "  %s/history%s   — показать историю\n", yellow, reset)
	fmt.Fprintf(out, "  %s/bind keys%s — показать этот список\n", yellow, reset)
	fmt.Fprintf(out, "  %s!команда%s   — выполнить команду напрямую\n", yellow, reset)
}

func showHelp(out io.Writer) {
	fmt.Fprintf(out, "%s%s=== nlsh справка ===%s\n\n", bold, cyan, reset)
	fmt.Fprintf(out, "%sОписание:%s\n  nlsh — оболочка с естественным языком. Пишешь \"покажи файлы\",\n  а он выполняет \"ls -la\".\n\n", bold, reset)
	fmt.Fprintf(out, "%sКоманды:%s\n  просто текст    — отправить запрос LLM\n  %s!команда%s     — выполнить команду напрямую\n  %s/cd%s путь     — сменить директорию\n  %s/clear%s       — очистить экран\n  %s/pwd%s         — показать текущую директорию\n  %s/history%s     — показать историю\n  %s/exit%s        — выйти\n\n", bold, reset,
		yellow, reset, yellow, reset, yellow, reset, yellow, reset, yellow, reset, yellow, reset)
	fmt.Fprintf(out, "%sХоткеи (bash-стиль):%s\n", bold, reset)
	fmt.Fprintf(out, "  %sCtrl+A%s     — начало строки    %sCtrl+E%s     — конец строки\n", yellow, reset, yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+R%s     — поиск истории    %sCtrl+S%s     — поиск вперёд\n", yellow, reset, yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+P%s     — предыдущая       %sCtrl+N%s     — следующая\n", yellow, reset, yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+U%s     — удалить до начала %sCtrl+K%s     — удалить до конца\n", yellow, reset, yellow, reset)
	fmt.Fprintf(out, "  %sAlt+B%s      — назад по слову    %sAlt+F%s      — вперед по слову\n", yellow, reset, yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+W%s     — удалить слово назад %sAlt+D%s    — удалить слово вперед\n", yellow, reset, yellow, reset)
	fmt.Fprintf(out, "  %sCtrl+L%s     — очистить экран    %s/exit%s      — выйти\n\n", yellow, reset, yellow, reset)
	fmt.Fprintf(out, "%sПримеры:%s\n  покажи все txt файлы\n  найди ошибки в логах\n  запусти docker\n\n", bold, reset)
	fmt.Fprintf(out, "%s Режим по умолчанию: %sdry-run%s (команды не выполняются).\n  Используй --dry-run=false чтобы включить.\n\n", bold, green, reset)
}

func clearScreen(out io.Writer) {
	fmt.Fprint(out, "\033[H\033[2J\033[3J")
}

func handleTurn(ctx context.Context, s *session, rf *rootFlags, input string, in io.Reader, out, errW io.Writer) error {
	if strings.HasPrefix(strings.TrimSpace(input), "!") {
		raw := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(input), "!"))
		if handled, shouldExit, err := runBuiltin(raw, out, errW, s.recent); handled {
			if err != nil {
				return err
			}
			if shouldExit {
				return context.Canceled
			}
			s.addRecentAndHistory(raw, "direct")
			return nil
		}
		res := executor.Run(ctx, rf.cfg.Shell, raw)
		s.addRecentAndHistory(raw, "direct")
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

	resp, err := askWithFollowUp(ctx, s, "run", input, in, out, errW)
	if err != nil {
		return err
	}

	if resp.Intent != prompt.IntentRunCommand {
		return nil
	}
	if rf.cfg.DryRun {
		fmt.Fprintln(out, "(dry-run: команда не запущена)")
		return nil
	}
	return runCommandWithCorrection(ctx, s, rf, resp, in, out, errW)
}

// spin — простой спиннер, работающий в горутине, пока не будет остановлен.
type spin struct {
	stopCh chan struct{}
	wg     sync.WaitGroup
}

func startSpin(w io.Writer) *spin {
	s := &spin{stopCh: make(chan struct{})}
	frames := []string{"\U0001f311", "\U0001f312", "\U0001f313", "\U0001f314", "\U0001f315", "\U0001f316", "\U0001f317", "\U0001f318"}
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		i := 0
		ticker := time.NewTicker(120 * time.Millisecond)
		defer ticker.Stop()
		fmt.Fprintf(w, "  \U0001f4ad") // initial
		for {
			select {
			case <-ticker.C:
				fmt.Fprintf(w, "\r\033[K%s", frames[i%len(frames)])
				flushOutput(w)
				i++
			case <-s.stopCh:
				fmt.Fprintf(w, "\r\033[K")
				flushOutput(w)
				return
			}
		}
	}()
	return s
}

func (s *spin) stop() {
	close(s.stopCh)
	s.wg.Wait()
}

// askWithFollowUp вызывает модель и, если в ответе есть question, задаёт его
// пользователю, передаёт ответ обратно модели и повторяет, пока вопросов больше нет.
func askWithFollowUp(ctx context.Context, s *session, mode, input string, in io.Reader, out, errW io.Writer) (prompt.Response, error) {
	for {
		resp, raw, err := s.askStream(ctx, mode, input, out)
		if err != nil {
			if raw != "" {
				fmt.Fprintln(errW, "raw output:")
				fmt.Fprintln(errW, raw)
			}
			return resp, err
		}
		if strings.TrimSpace(resp.Question) == "" {
			return resp, nil
		}

		fmt.Fprintf(out, "%s[nlsh]%s %s%s%s\n", cyan, reset, cyan, resp.Question, reset)
		fmt.Fprintf(out, "%s>%s ", yellow, reset)
		flushOutput(out)

		sc := bufio.NewScanner(in)
		if !sc.Scan() {
			return resp, nil
		}
		answer := strings.TrimSpace(sc.Text())

		input = input + "\n" + answer
	}
}

// runCommandWithCorrection выполняет команду, и в случае ошибки запрашивает автоисправление у LLM.
func runCommandWithCorrection(ctx context.Context, s *session, rf *rootFlags, resp prompt.Response, in io.Reader, out, errW io.Writer) error {
	dec := evaluatePolicy(resp)
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

	if handled, shouldExit, err := runBuiltin(resp.Command, out, errW, s.recent); handled {
		if err != nil {
			return err
		}
		s.addRecentAndHistory(resp.Command, "llm")
		if shouldExit {
			return context.Canceled
		}
		return nil
	}

	res := executor.Run(ctx, rf.cfg.Shell, resp.Command)
	s.addRecentAndHistory(resp.Command, "llm")

	fb := feedback.Analyze(resp.Command, res.Stdout, res.Stderr, res.ExitCode)
	if res.Stdout != "" {
		fmt.Fprint(out, res.Stdout)
	}

	if fb.Success {
		if hint := fb.Format(); hint != "" {
			fmt.Fprintf(out, "\n%s[nlsh]%s %s%s%s\n", green, reset, green, hint, reset)
		}
		return nil
	}

	// Команда завершилась ошибкой. Запрашиваем исправление.
	stderr := res.Stderr
	if stderr == "" && res.Err != nil {
		stderr = res.Err.Error()
	}

	fmt.Fprintf(out, "\n%s[nlsh]%s Обнаружена ошибка (код %d). Запрашиваю автоисправление у LLM...\n", yellow, reset, res.ExitCode)

	correctionInput := fmt.Sprintf("Команда '%s' завершилась с ошибкой.\nКод выхода: %d\nStderr:\n%s\n\nПожалуйста, исправь команду, чтобы она выполнилась успешно в текущей ОС.", resp.Command, res.ExitCode, stderr)

	corrResp, _, err := s.askStream(ctx, "run", correctionInput, out)
	if err != nil {
		return err
	}

	if corrResp.Intent != prompt.IntentRunCommand {
		return nil
	}

	decCorr := evaluatePolicy(corrResp)
	if !decCorr.Allowed {
		fmt.Fprintln(out, "(исправленная команда заблокирована политикой безопасности)")
		return nil
	}

	if !confirm(in, out, "выполнить исправленную команду?") {
		fmt.Fprintln(out, "(отменено)")
		return nil
	}

	resCorr := executor.Run(ctx, rf.cfg.Shell, corrResp.Command)
	s.addRecentAndHistory(corrResp.Command, "llm")

	fbCorr := feedback.Analyze(corrResp.Command, resCorr.Stdout, resCorr.Stderr, resCorr.ExitCode)
	if resCorr.Stdout != "" {
		fmt.Fprint(out, resCorr.Stdout)
	}

	if hint := fbCorr.Format(); hint != "" {
		if fbCorr.Success {
			fmt.Fprintf(out, "\n%s[nlsh]%s %s%s%s\n", green, reset, green, hint, reset)
		} else {
			fmt.Fprintf(out, "\n%s[nlsh]%s %s%s%s\n", yellow, reset, yellow, hint, reset)
		}
	}

	if resCorr.Err != nil && !fbCorr.Success {
		return fmt.Errorf("exit %d: %w", resCorr.ExitCode, resCorr.Err)
	}
	return nil
}
