package policy

import (
	"regexp"
	"runtime"
	"strings"

	"github.com/dedomorozoff/nlsh/internal/prompt"
)

// Decision — результат проверки команды политикой.
type Decision struct {
	Allowed bool
	Risk    prompt.Risk
	Reason  string
}

type pattern struct {
	re     *regexp.Regexp
	reason string
}

var dangerPatternsUnix = []pattern{
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

var suspiciousPatternsUnix = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bsudo\b`),
	regexp.MustCompile(`(?i)\bapt(-get)?\s+(install|remove|purge|upgrade)\b`),
	regexp.MustCompile(`(?i)\b(yum|dnf|pacman)\s+`),
	regexp.MustCompile(`(?i)\bsystemctl\s+(start|stop|restart|enable|disable)\b`),
	regexp.MustCompile(`(?i)\bgit\s+push\b.*--force`),
	regexp.MustCompile(`(?i)\bdocker\s+(rm|rmi|system\s+prune)\b`),
}

var dangerPatternsWindows = []pattern{
	{regexp.MustCompile(`(?i)\bRemove-Item\b.*\b[A-Z]:[\\/]*(\s|$|["'])`), "Remove-Item на корне"},
	{regexp.MustCompile(`(?i)\bformat\s+[A-Z]:\s*(/fs:[a-z]+\s*)?(/q\s*)?(/y\s*)?`), "Форматирование диска"},
	{regexp.MustCompile(`(?i)\bClear-Item\s+HKLM:\\`), "Удаление системного реестра"},
	{regexp.MustCompile(`(?i)\bRemove-ItemProperty\s+HKLM:\\`), "Удаление ключей системного реестра"},
	{regexp.MustCompile(`(?i)\bStop-Computer\b|\bRestart-Computer\b`), "shutdown/reboot"},
	{regexp.MustCompile(`(?i)\bSet-ExecutionPolicy\s+(Bypass|Unrestricted)\b`), "Снятие ограничений PowerShell"},
	{regexp.MustCompile(`(?i)\bDisable-NetFirewallRule\b|\bSet-NetFirewallProfile\s+.*-Enabled\s+False\b`), "Отключение брандмауэра"},
}

var suspiciousPatternsWindows = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bInvoke-WebRequest\b.*\bInvoke-Expression\b`),
	regexp.MustCompile(`(?i)\biex\s*\(irm\b`),
	regexp.MustCompile(`(?i)\bStop-Service\b`),
	regexp.MustCompile(`(?i)\bRemove-Item\s+.*-Recurse\b`),
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

	dangerPats := dangerPatternsUnix
	suspiciousPats := suspiciousPatternsUnix
	if runtime.GOOS == "windows" {
		dangerPats = dangerPatternsWindows
		suspiciousPats = suspiciousPatternsWindows
	}

	for _, p := range dangerPats {
		if p.re.MatchString(cmd) {
			return Decision{Allowed: false, Risk: prompt.RiskHigh, Reason: p.reason}
		}
	}

	risk := suggested
	if risk == "" {
		risk = prompt.RiskLow
	}

	for _, re := range suspiciousPats {
		if re.MatchString(cmd) {
			if risk == prompt.RiskLow {
				risk = prompt.RiskMedium
			}
			break
		}
	}
	return Decision{Allowed: true, Risk: risk}
}
