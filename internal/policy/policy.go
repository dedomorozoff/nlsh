package policy

import (
	"regexp"
	"strings"

	"github.com/nlsh/nlsh/internal/prompt"
)

// Decision — результат проверки команды политикой.
type Decision struct {
	Allowed bool
	Risk    prompt.Risk
	Reason  string
}

// dangerPatterns — регекспы команд, которые мы считаем заведомо опасными.
// Это deny-list нижнего уровня: даже если модель сказала risk=low,
// мы поднимаем risk до high и требуем подтверждение.
var dangerPatterns = []struct {
	re     *regexp.Regexp
	reason string
}{
	{regexp.MustCompile(`(?i)\brm\s+(-[a-z]*r[a-z]*f|-[a-z]*f[a-z]*r)\b.*\s/(\s|$)`), "rm -rf на корне"},
	{regexp.MustCompile(`(?i)\brm\s+-[a-z]*r[a-z]*f?\s+/\*`), "rm -rf /*"},
	{regexp.MustCompile(`(?i)\bmkfs(\.[a-z0-9]+)?\b`), "форматирование ФС"},
	{regexp.MustCompile(`(?i)\bdd\s+if=.+\s+of=/dev/`), "dd на блочное устройство"},
	{regexp.MustCompile(`(?i)>\s*/dev/sd[a-z]`), "запись в блочное устройство"},
	{regexp.MustCompile(`:\(\)\s*\{\s*:\|\s*:&\s*\}\s*;\s*:`), "fork-бомба"},
	{regexp.MustCompile(`(?i)\bchmod\s+-R\s+0*777\b`), "chmod -R 777"},
	{regexp.MustCompile(`(?i)\bchown\s+-R\s+.*\s+/\b`), "chown -R на корне"},
	{regexp.MustCompile(`(?i)curl\s+[^|]*\|\s*(sudo\s+)?(ba)?sh\b`), "curl | sh"},
	{regexp.MustCompile(`(?i)wget\s+[^|]*\|\s*(sudo\s+)?(ba)?sh\b`), "wget | sh"},
	{regexp.MustCompile(`(?i)\bshutdown\b|\breboot\b|\bhalt\b|\bpoweroff\b`), "shutdown/reboot"},
	{regexp.MustCompile(`(?i)\biptables\s+-F\b`), "сброс iptables"},
}

// suspiciousPatterns — не блокируем, но повышаем risk до medium.
var suspiciousPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bsudo\b`),
	regexp.MustCompile(`(?i)\bapt(-get)?\s+(install|remove|purge|upgrade)\b`),
	regexp.MustCompile(`(?i)\b(yum|dnf|pacman)\s+`),
	regexp.MustCompile(`(?i)\bsystemctl\s+(start|stop|restart|enable|disable)\b`),
	regexp.MustCompile(`(?i)\bgit\s+push\b.*--force`),
	regexp.MustCompile(`(?i)\bdocker\s+(rm|rmi|system\s+prune)\b`),
}

// Evaluate проверяет команду и возвращает решение. Если команда есть в
// dangerPatterns — Allowed=false: shell обязан спросить подтверждение
// (или, в --strict, отказать вовсе). Иначе мы лишь корректируем risk.
func Evaluate(cmd string, suggested prompt.Risk) Decision {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return Decision{Allowed: false, Risk: prompt.RiskHigh, Reason: "пустая команда"}
	}
	for _, p := range dangerPatterns {
		if p.re.MatchString(cmd) {
			return Decision{Allowed: false, Risk: prompt.RiskHigh, Reason: p.reason}
		}
	}
	risk := suggested
	if risk == "" {
		risk = prompt.RiskLow
	}
	for _, re := range suspiciousPatterns {
		if re.MatchString(cmd) {
			if risk == prompt.RiskLow {
				risk = prompt.RiskMedium
			}
			break
		}
	}
	return Decision{Allowed: true, Risk: risk}
}
