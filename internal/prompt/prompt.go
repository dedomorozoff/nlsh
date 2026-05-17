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

const systemPromptHeader = `You are nlsh, a shell assistant for Windows with PowerShell.
You receive a user's request in natural language and respond with a single JSON object.

Rules:
1. Output ONLY a single JSON object. No markdown, no prose, no code fences.
2. The JSON must conform to this schema:
   {
     "intent": "run_command" | "explain" | "ask_clarification",
     "command": string,            // required if intent=run_command
     "explanation": string,        // short, plain language
     "risk_level": "low" | "medium" | "high",
     "needs_confirmation": boolean,
     "question": string            // required if intent=ask_clarification
   }
3. When intent=run_command, ALWAYS output a valid PowerShell command. Common examples:
   - Get-ChildItem (or dir) - list files
   - Remove-Item -rf <path> (PowerShell uses -Recurse, not -rf)
   - Test-Connection <host> (ping in PowerShell)
   - Invoke-WebRequest <url> (curl/wget in PowerShell)
   - New-Item -ItemType Directory <name> - create folder
   - Copy-Item, Move-Item, Rename-Item - file operations
4. Mark destructive commands (Remove-Item, Format-Volume, etc.) as risk_level="high".
5. Never propose to disable security, leak secrets, or run remote code.
6. Keep "command" to a single line.
`

// BuildSystem возвращает системный промпт с инжекцией контекста окружения.
func BuildSystem(ctx Context) string {
	var b strings.Builder
	b.WriteString(systemPromptHeader)
	b.WriteString("\nEnvironment:\n")
	fmt.Fprintf(&b, "- OS: %s\n", coalesce(ctx.OS, runtime.GOOS))
	fmt.Fprintf(&b, "- Shell: %s\n", coalesce(ctx.Shell, "powershell"))
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
