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
     "command": string,
     "explanation": string,
     "risk_level": "low" | "medium" | "high",
     "needs_confirmation": boolean,
     "question": string
   }
3. For creating files, use PowerShell: New-Item -ItemType File -Path "filename" OR echo $null > filename
   For creating directories: New-Item -ItemType Directory -Path "dirname"
   For removing files: Remove-Item -Path "filename" -Force
   For removing directories: Remove-Item -Path "dirname" -Recurse -Force
   For listing files: Get-ChildItem OR dir
   For reading file: Get-Content "filename"
   For copying: Copy-Item -Path "src" -Destination "dst"
   For moving: Move-Item -Path "src" -Destination "dst"
4. Mark destructive commands as risk_level="high".
5. Never propose to disable security or run remote code.
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
