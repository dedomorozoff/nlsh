package prompt

import (
	"fmt"
	"runtime"
	"strings"
)

// Context — это окружение, которое мы инжектим в системный промпт.
type Context struct {
	OS          string
	Shell       string
	CWD         string
	RecentCmds  []string
	UserRequest string
	// Mode задаёт ожидаемый intent: ask -> только explain/ask_clarification,
	// run -> предпочтительно run_command. Пусто = без ограничения.
	Mode string
}

const systemPromptBase = `You are nlsh, an intelligent natural language shell assistant.
You receive a user's request in natural language and respond with a single JSON object.

Rules:
1. Output ONLY a single JSON object. No markdown, no prose, no code fences.
2. The JSON must conform to this schema:
   {
     "intent": "run_command" | "explain" | "ask_clarification",
     "command": string,
     "explanation": string,
     "risk_level": "low" | "medium" | "high",
     "needs_confirmation": boolean,
     "question": string
   }
3. Mark destructive commands as risk_level="high".
4. Never propose to disable security or run remote code.
5. Keep "command" to a single line.
`

const windowsSpecifics = `Target OS: Windows. Shell: PowerShell.
Use PowerShell native commands where possible:
- Files/Dirs: New-Item, Remove-Item, Get-ChildItem, Get-Content, Copy-Item, Move-Item.
- Processes: Get-Process, Stop-Process.
`

const unixSpecifics = `Target OS: Unix-like (Linux/macOS). Shell: bash/zsh.
Use standard POSIX utilities where possible:
- Files/Dirs: touch, mkdir, rm, ls, cat, cp, mv.
- Processes: ps, grep, kill.
`

// BuildSystem возвращает системный промпт с инжекцией контекста окружения.
func BuildSystem(ctx Context) string {
	var b strings.Builder
	b.WriteString(systemPromptBase)
	
	targetOS := coalesce(ctx.OS, runtime.GOOS)
	if targetOS == "windows" {
		b.WriteString("\n" + windowsSpecifics)
	} else {
		b.WriteString("\n" + unixSpecifics)
	}
	
	b.WriteString("\nEnvironment:\n")
	fmt.Fprintf(&b, "- OS: %s\n", targetOS)
	fmt.Fprintf(&b, "- Shell: %s\n", coalesce(ctx.Shell, getDefaultShell(targetOS)))
	if ctx.CWD != "" {
		fmt.Fprintf(&b, "- CWD: %s\n", ctx.CWD)
	}
	if len(ctx.RecentCmds) > 0 {
		b.WriteString("- Recent commands:\n")
		for _, c := range ctx.RecentCmds {
			fmt.Fprintf(&b, "  * %s\n", c)
		}
	}
	if ctx.Mode != "" {
		fmt.Fprintf(&b, "\nMode hint: %s\n", ctx.Mode)
	}
	return b.String()
}

func getDefaultShell(os string) string {
	if os == "windows" {
		return "powershell"
	}
	return "bash"
}

// BuildUser форматирует запрос пользователя для модели.
func BuildUser(ctx Context) string {
	return strings.TrimSpace(ctx.UserRequest)
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
