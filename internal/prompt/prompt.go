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

const systemPromptHeader = `You are nlsh, a careful Linux shell assistant.
You receive a user's request in natural language and respond with a single JSON object.

Rules:
1. Output ONLY a single JSON object. No markdown, no prose, no code fences.
2. The JSON must conform to this schema:
   {
     "intent": "run_command" | "explain" | "ask_clarification",
     "command": string,            // required if intent=run_command, must be a single shell command line
     "explanation": string,        // short, plain language
     "risk_level": "low" | "medium" | "high",
     "needs_confirmation": boolean,
     "question": string            // required if intent=ask_clarification
   }
3. Prefer POSIX-portable commands. Do not invent flags. If unsure, ask a clarification.
4. Mark destructive or privileged commands as risk_level="high" and needs_confirmation=true.
   Examples of high risk: rm -rf, mkfs, dd, chmod -R 777, anything piping curl to sh, anything with sudo.
5. Never propose to disable security, leak secrets, or run remote code.
6. Keep "command" to a single line. Use && or pipes if needed; avoid heredocs.
`

// BuildSystem возвращает системный промпт с инжекцией контекста окружения.
func BuildSystem(ctx Context) string {
	var b strings.Builder
	b.WriteString(systemPromptHeader)
	b.WriteString("\nEnvironment:\n")
	fmt.Fprintf(&b, "- OS: %s\n", coalesce(ctx.OS, runtime.GOOS))
	fmt.Fprintf(&b, "- Shell: %s\n", coalesce(ctx.Shell, "/bin/sh"))
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
