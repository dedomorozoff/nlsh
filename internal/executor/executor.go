package executor

import (
	"context"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// Result — итог выполнения команды.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Err      error
}

// Run исполняет одну shell-командную строку, проксируя её в системный shell.
// Через shell, а не argv-разделение, чтобы корректно работали пайпы и `&&`.
func Run(ctx context.Context, shell, command string) Result {
	if strings.TrimSpace(command) == "" {
		return Result{ExitCode: -1, Err: errEmpty}
	}
	if runtime.GOOS == "windows" && strings.Contains(strings.ToLower(shell), "powershell") {
		command = translateToPowerShell(command)
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
// Linux/macOS: <shell> -c "<command>"; Windows: powershell -Command или cmd /C.
func shellArgs(shell string) []string {
	if shell == "" {
		if runtime.GOOS == "windows" {
			return []string{"powershell", "-NoProfile", "-Command"}
		}
		return []string{"/bin/sh", "-c"}
	}
	low := strings.ToLower(shell)
	switch {
	case strings.Contains(low, "powershell"), strings.Contains(low, "pwsh"):
		return []string{shell, "-NoProfile", "-Command"}
	case strings.HasSuffix(low, "cmd"), strings.HasSuffix(low, "cmd.exe"):
		return []string{shell, "/C"}
	default:
		return []string{shell, "-c"}
	}
}

type errEmptyCommand struct{}

func (errEmptyCommand) Error() string { return "empty command" }

var errEmpty = errEmptyCommand{}

func translateToPowerShell(cmd string) string {
	replacer := strings.NewReplacer(
		"rm -rf", "Remove-Item -Recurse -Force",
		"rm -r", "Remove-Item -Recurse -Force",
		"rm -f", "Remove-Item -Force",
		"rm ", "Remove-Item ",
		"mkdir ", "New-Item -ItemType Directory ",
		"touch ", "New-Item -ItemType File ",
		"cat ", "Get-Content ",
		"ls ", "Get-ChildItem ",
		"ls", "Get-ChildItem",
		"cp ", "Copy-Item ",
		"mv ", "Move-Item ",
		"pwd", "Get-Location",
		"echo ", "Write-Output ",
		"ping -c ", "Test-Connection -Count ",
		"curl ", "Invoke-WebRequest ",
		"wget ", "Invoke-WebRequest ",
	)
	return replacer.Replace(cmd)
}
