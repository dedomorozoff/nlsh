package executor

import (
	"context"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

func init() {
	if runtime.GOOS == "windows" {
		// Устанавливаем кодировку UTF-8 для консоли Windows
		os.Setenv("CHCP", "65001")
	}
}

// Result — итог выполнения команды.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// Run исполняет одну shell-командную строку, проксируя её в системный shell.
func Run(ctx context.Context, shell, command string) Result {
	if strings.TrimSpace(command) == "" {
		return Result{ExitCode: -1, Err: errEmpty}
	}
	if runtime.GOOS == "windows" {
		command = translateToWindows(command)
		// Добавляем установку UTF-8 кодировки для PowerShell
		if strings.Contains(strings.ToLower(shell), "powershell") || shell == "" {
			command = "[Console]::OutputEncoding = [System.Text.Encoding]::UTF8; " + command
		}
	}
	args := shellArgs(shell)
	cmd := exec.CommandContext(ctx, args[0], append(args[1:], command)...)

	var stdout, stderr strings.Builder
	cmd.Stdout = io.MultiWriter(&stdout)
	cmd.Stderr = io.MultiWriter(&stderr)

	err := cmd.Run()
	res := Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Err:    err,
	}
	if cmd.ProcessState != nil {
		res.ExitCode = cmd.ProcessState.ExitCode()
	} else if err != nil {
		res.ExitCode = -1
	}
	return res
}

// shellArgs подбирает интерпретатор и флаг для одиночной команды.
func shellArgs(shell string) []string {
	if shell == "" {
		if runtime.GOOS == "windows" {
			return []string{"powershell", "-NoProfile", "-NoLogo", "-Command"}
		}
		return []string{"/bin/sh", "-c"}
	}
	low := strings.ToLower(shell)
	switch {
	case strings.Contains(low, "powershell"), strings.Contains(low, "pwsh"):
		return []string{shell, "-NoProfile", "-NoLogo", "-Command"}
	case strings.HasSuffix(low, "cmd"), strings.HasSuffix(low, "cmd.exe"):
		return []string{shell, "/C"}
	default:
		return []string{shell, "-c"}
	}
}

type errEmptyCommand struct{}

func (errEmptyCommand) Error() string { return "empty command" }

var errEmpty = errEmptyCommand{}

// translateToWindows конвертирует Unix-подобные команды в Windows cmd эквиваленты.
func translateToWindows(cmd string) string {
	// Не переводим если уже выглядит как Windows/PowerShell команда
	lower := strings.ToLower(cmd)
	if strings.Contains(lower, "-item") || strings.Contains(lower, "get-") ||
		strings.Contains(lower, "set-") || strings.Contains(lower, "invoke-") ||
		strings.Contains(lower, "test-") || strings.Contains(lower, "out-") ||
		strings.Contains(lower, "where-") || strings.Contains(lower, "select-") ||
		strings.Contains(lower, "forfiles") || strings.Contains(lower, "reg ") ||
		strings.Contains(lower, "sc ") || strings.Contains(lower, "netsh") ||
		strings.Contains(lower, ".exe") || strings.Contains(lower, "choco ") ||
		strings.Contains(lower, "winget ") || strings.Contains(lower, "pip ") ||
		strings.Contains(lower, "python") || strings.Contains(lower, "node") ||
		strings.Contains(lower, "git ") || strings.Contains(lower, "sqlite") ||
		strings.Contains(lower, "curl ") || strings.Contains(lower, "wget ") ||
		strings.Contains(lower, "ssh ") || strings.Contains(lower, "docker") {
		return cmd
	}

	// Простые Unix -> Windows cmd переводы
	translations := []struct {
		from string
		to   string
	}{
		{"rm -rf ", "rmdir /s /q "},
		{"rm -r ", "rmdir /s /q "},
		{"rm -f ", "del /f "},
		{"rm ", "del /f "},
		{"mkdir ", "mkdir "},
		{"touch ", "echo. > "},
		{"cat ", "type "},
		{"ls -la", "dir"},
		{"ls -l", "dir"},
		{"ls", "dir"},
		{"cp ", "copy "},
		{"mv ", "move "},
		{"pwd", "cd"},
		{"echo ", "echo "},
		{"clear", "cls"},
		{"which ", "where "},
		{"head -n ", "more +1 /"},
		{"tail -n ", "for /f "},
		{"grep ", "findstr "},
		{"find . -name", "dir /s /b"},
		{"wc -l", "find /c /v \"\""},
		{"uname -a", "ver"},
		{"date", "date /t"},
		{"whoami", "whoami"},
		{"hostname", "hostname"},
		{"ps aux", "tasklist"},
		{"kill ", "taskkill /f /pid "},
	}

	result := cmd
	for _, t := range translations {
		if strings.HasPrefix(strings.ToLower(result), strings.ToLower(t.from)) ||
			strings.Contains(result, " "+t.from) || strings.HasPrefix(result, t.from) {
			result = strings.Replace(result, t.from, t.to, 1)
			break
		}
	}

	return result
}